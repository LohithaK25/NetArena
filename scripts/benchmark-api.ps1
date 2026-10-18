
param(
    [string]$BaseUrl = "http://127.0.0.1:18080",
    [int[]]$ConcurrencyLevels = @(1, 4, 8),
    [int]$RequestsPerLevel = 20,
    [int]$ArrivalsPerSimulation = 20,
    [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

if ($RequestsPerLevel -lt 1 -or
    $ArrivalsPerSimulation -lt 1 -or
    $TimeoutSeconds -lt 1) {
    throw "Counts and timeout must be positive."
}

if (@($ConcurrencyLevels | Where-Object { $_ -lt 1 -or $_ -gt 8 }).Count -gt 0) {
    throw "Concurrency levels must be between 1 and 8 for this initial benchmark."
}

$BaseUrl = $BaseUrl.TrimEnd("/")
$Health = Invoke-RestMethod -Uri "$BaseUrl/api/v1/health" -Method Get -TimeoutSec $TimeoutSeconds
if ($Health.status -ne "ok") {
    throw "Health check failed."
}

Add-Type -AssemblyName System.Net.Http

$script:HttpClient = New-Object System.Net.Http.HttpClient
$script:HttpClient.Timeout = [TimeSpan]::FromSeconds($TimeoutSeconds)

function New-PayloadJson {
    param([string]$Name, [int]$ArrivalCount)

    $arrivals = @(
        for ($j = 0; $j -lt $ArrivalCount; $j++) {
            @{
                requestId = "$Name-req-$j"
                at = $j
            }
        }
    )

    return @{
        name = $Name
        serviceCapacity = 2
        serviceDuration = 3
        arrivals = $arrivals
    } | ConvertTo-Json -Depth 6 -Compress
}

function Invoke-OneRequest {
    param([string]$Name)

    $json = New-PayloadJson -Name $Name -ArrivalCount $ArrivalsPerSimulation
    $content = New-Object System.Net.Http.StringContent(
        $json,
        [System.Text.Encoding]::UTF8,
        "application/json"
    )
    $watch = [System.Diagnostics.Stopwatch]::StartNew()

    try {
        $response = $script:HttpClient.PostAsync(
            "$BaseUrl/api/v1/simulations",
            $content
        ).GetAwaiter().GetResult()

        $text = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        $watch.Stop()

        $valid = $false
        $message = ""

        if ($response.IsSuccessStatusCode) {
            try {
                $body = $text | ConvertFrom-Json
                $valid = (
                    $body.name -eq $Name -and
                    $body.summary.totalRequests -eq $ArrivalsPerSimulation -and
                    $body.summary.succeededRequests -eq $ArrivalsPerSimulation -and
                    $body.summary.failedRequests -eq 0
                )
                if (-not $valid) {
                    $message = "Unexpected simulation response"
                }
            }
            catch {
                $message = "Invalid JSON response: $($_.Exception.Message)"
            }
        }
        else {
            $message = "HTTP $([int]$response.StatusCode): $text"
        }

        return [pscustomobject]@{
            Name = $Name
            Milliseconds = $watch.Elapsed.TotalMilliseconds
            HttpSuccess = [bool]$response.IsSuccessStatusCode
            ResultValid = [bool]$valid
            Error = $message
        }
    }
    catch {
        $watch.Stop()
        return [pscustomobject]@{
            Name = $Name
            Milliseconds = $watch.Elapsed.TotalMilliseconds
            HttpSuccess = $false
            ResultValid = $false
            Error = $_.Exception.Message
        }
    }
    finally {
        $content.Dispose()
    }
}

function Get-Percentile {
    param([double[]]$Values, [double]$Percentile)

    if ($Values.Count -eq 0) {
        return [double]::NaN
    }

    $sorted = @($Values | Sort-Object)
    $index = [Math]::Ceiling($Percentile * $sorted.Count) - 1
    $index = [Math]::Max(0, [Math]::Min($index, $sorted.Count - 1))
    return [double]$sorted[$index]
}


$allResults = @()

try {
    foreach ($concurrency in $ConcurrencyLevels) {
        Write-Host "`nConcurrency $concurrency - warm-up" -ForegroundColor Cyan

        $warmup = Invoke-OneRequest -Name "warmup-c$concurrency"
        if (-not $warmup.HttpSuccess -or -not $warmup.ResultValid) {
            throw "Warm-up failed: $($warmup.Error)"
        }

        $jobs = New-Object 'System.Collections.Generic.List[object]'
        $levelResults = New-Object 'System.Collections.Generic.List[object]'
        $batchWatch = [System.Diagnostics.Stopwatch]::StartNew()

        for ($i = 0; $i -lt $RequestsPerLevel; $i++) {
            while ($jobs.Count -ge $concurrency) {
                $completed = $false

                for ($j = $jobs.Count - 1; $j -ge 0; $j--) {
                    $job = $jobs[$j]
                    if ($job.Task.IsCompleted) {
                        $levelResults.Add($job.Task.GetAwaiter().GetResult())
                        $jobs.RemoveAt($j)
                        $completed = $true
                    }
                }

                if (-not $completed) {
                    Start-Sleep -Milliseconds 2
                }
            }

            $name = "bench-c$concurrency-r$i"
            $requestName = $name

            $task = [System.Threading.Tasks.Task[object]]::Factory.StartNew(
                [Func[object]]{
                    Invoke-OneRequest -Name $requestName
                }.GetNewClosure(),
                [System.Threading.CancellationToken]::None,
                [System.Threading.Tasks.TaskCreationOptions]::None,
                [System.Threading.Tasks.TaskScheduler]::Default
            )

            $jobs.Add([pscustomobject]@{
                Name = $name
                Task = $task
            })
        }

        foreach ($job in $jobs) {
            $levelResults.Add($job.Task.GetAwaiter().GetResult())
        }

        $batchWatch.Stop()
        $allResults += @($levelResults)

        $latencies = @(
            $levelResults |
                Where-Object { $_.HttpSuccess } |
                ForEach-Object { [double]$_.Milliseconds }
        )

        $httpFailures = @(
            $levelResults | Where-Object { -not $_.HttpSuccess }
        ).Count

        $invalidResults = @(
            $levelResults | Where-Object {
                $_.HttpSuccess -and -not $_.ResultValid
            }
        ).Count

        [pscustomobject]@{
            Concurrency = $concurrency
            Requests = $levelResults.Count
            ThroughputRPS = [Math]::Round(
                $levelResults.Count / $batchWatch.Elapsed.TotalSeconds, 2
            )
            P50Milliseconds = [Math]::Round(
                (Get-Percentile $latencies 0.50), 2
            )
            P95Milliseconds = [Math]::Round(
                (Get-Percentile $latencies 0.95), 2
            )
            HttpFailures = $httpFailures
            InvalidResults = $invalidResults
            ElapsedSeconds = [Math]::Round(
                $batchWatch.Elapsed.TotalSeconds, 2
            )
        } | Format-Table -AutoSize
    }

    Write-Host "`nIndividual failures:" -ForegroundColor Yellow
    $allResults |
        Where-Object { -not $_.HttpSuccess -or -not $_.ResultValid } |
        Select-Object Name, Milliseconds, HttpSuccess, ResultValid, Error |
        Format-Table -Wrap -AutoSize

    Write-Host "`nBenchmark complete. No report files were written."
}
finally {
    $script:HttpClient.Dispose()
}
