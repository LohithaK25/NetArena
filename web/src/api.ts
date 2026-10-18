
export interface SimulationArrival {
  requestID: string
  at: number
}


export type RoutingAlgorithm = 'round_robin' | 'shortest_path'

export type TopologyNodeType = 1 | 2 | 3 | 4

export interface TopologyNode {
  id: string
  type: TopologyNodeType
  name: string
  capacity: number
}

export interface TopologyConnection {
  from: string
  to: string
  latency: number
  dropPackets: boolean
  packetLossProbability: number
}

export interface SimulationTopology {
  nodes: TopologyNode[]
  connections: TopologyConnection[]
}

export interface ScenarioObjectives {
  minSuccessRate?: number
  maxAverageLatency?: number
}

export interface ObjectiveOutcome {
  name: string
  passed: boolean
  actual: number
  threshold: number
  message: string
}

export interface ObjectiveEvaluation {
  configured: boolean
  passed: boolean
  outcomes: ObjectiveOutcome[]
}
export interface SimulationScenario {
  name: string
  serviceCapacity: number
  serviceDuration: number
  arrivals: SimulationArrival[]
  routingAlgorithm?: RoutingAlgorithm
  topology?: SimulationTopology
  objectives?: ScenarioObjectives
}

export interface LatencyMetrics {
  count: number
  min: number
  max: number
  average: number
}

export interface SimulationSummary {
  totalRequests: number
  createdRequests: number
  queuedRequests: number
  retryWaitingRequests: number
  inFlightRequests: number
  succeededRequests: number
  failedRequests: number
  cancelledRequests: number
  timedOutRequests: number
  completedRequests: number
  retries: number
  endToEndLatency: LatencyMetrics
  serviceLatency: LatencyMetrics
}

export interface WorkerMetrics {
  capacity: number
  busyWorkers: number
  waitingRequests: number
  utilization: number
  online: boolean
  circuitState: number
  consecutiveFailures: number
  probeInFlight: boolean
}

export interface SimulationMetrics {
  totalRequests: number
  createdRequests: number
  queuedRequests: number
  inFlightRequests: number
  succeededRequests: number
  failedRequests: number
  cancelledRequests: number
  timedOutRequests: number
  busyWorkers: number
  serviceCapacity: number
  waitingRequests: number
  workerUtilization: number
  workers: Record<string, WorkerMetrics>
}

export type RouteHopStatus =
  | 'forwarded'
  | 'dropped'
  | 'cycle_detected'

export interface RouteTraceHop {
  from: string
  to: string
  at: number
  latency: number
  status: RouteHopStatus
}

export type RouteTraces = Record<string, RouteTraceHop[]>

export interface SimulationResult {
  name: string
  duration: number
  summary: SimulationSummary
  metrics: SimulationMetrics
  routeTraces: RouteTraces
  objectives: ObjectiveEvaluation
}

interface ApiError {
  error?: string
}

const SIMULATION_TIMEOUT_MS = 30_000

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function hasFiniteNumbers(
  value: Record<string, unknown>,
  fields: string[],
): boolean {
  return fields.every(
    (field) =>
      typeof value[field] === 'number' &&
      Number.isFinite(value[field]),
  )
}

function isLatencyMetrics(value: unknown): value is LatencyMetrics {
  return (
    isRecord(value) &&
    hasFiniteNumbers(value, ['count', 'min', 'max', 'average'])
  )
}

function isSimulationSummary(value: unknown): value is SimulationSummary {
  return (
    isRecord(value) &&
    hasFiniteNumbers(value, [
      'totalRequests',
      'createdRequests',
      'queuedRequests',
      'retryWaitingRequests',
      'inFlightRequests',
      'succeededRequests',
      'failedRequests',
      'cancelledRequests',
      'timedOutRequests',
      'completedRequests',
      'retries',
    ]) &&
    isLatencyMetrics(value.endToEndLatency) &&
    isLatencyMetrics(value.serviceLatency)
  )
}

function isWorkerMetrics(value: unknown): value is WorkerMetrics {
  return (
    isRecord(value) &&
    hasFiniteNumbers(value, [
      'capacity',
      'busyWorkers',
      'waitingRequests',
      'utilization',
      'circuitState',
      'consecutiveFailures',
    ]) &&
    typeof value.online === 'boolean' &&
    typeof value.probeInFlight === 'boolean'
  )
}

function isSimulationMetrics(value: unknown): value is SimulationMetrics {
  if (
    !isRecord(value) ||
    !hasFiniteNumbers(value, [
      'totalRequests',
      'createdRequests',
      'queuedRequests',
      'inFlightRequests',
      'succeededRequests',
      'failedRequests',
      'cancelledRequests',
      'timedOutRequests',
      'busyWorkers',
      'serviceCapacity',
      'waitingRequests',
      'workerUtilization',
    ]) ||
    !isRecord(value.workers)
  ) {
    return false
  }

  return Object.values(value.workers).every(isWorkerMetrics)
}

function isRouteTraceHop(value: unknown): value is RouteTraceHop {
  return (
    isRecord(value) &&
    typeof value.from === 'string' &&
    typeof value.to === 'string' &&
    hasFiniteNumbers(value, ['at', 'latency']) &&
    ['forwarded', 'dropped', 'cycle_detected'].includes(
      String(value.status),
    )
  )
}

function isRouteTraces(value: unknown): value is RouteTraces {
  return (
    isRecord(value) &&
    Object.values(value).every(
      (hops) => Array.isArray(hops) && hops.every(isRouteTraceHop),
    )
  )
}

function isObjectiveEvaluation(
  value: unknown,
): value is ObjectiveEvaluation {
  return (
    isRecord(value) &&
    typeof value.configured === 'boolean' &&
    typeof value.passed === 'boolean' &&
    Array.isArray(value.outcomes) &&
    value.outcomes.every(
      (outcome) =>
        isRecord(outcome) &&
        typeof outcome.name === 'string' &&
        typeof outcome.passed === 'boolean' &&
        typeof outcome.actual === 'number' &&
        Number.isFinite(outcome.actual) &&
        typeof outcome.threshold === 'number' &&
        Number.isFinite(outcome.threshold) &&
        typeof outcome.message === 'string',
    )
  )
}

function isSimulationResult(value: unknown): value is SimulationResult {
  return (
    isRecord(value) &&
    typeof value.name === 'string' &&
    typeof value.duration === 'number' &&
    Number.isFinite(value.duration) &&
    isSimulationSummary(value.summary) &&
    isSimulationMetrics(value.metrics) &&
    isRouteTraces(value.routeTraces) &&
    isObjectiveEvaluation(value.objectives)
)
}

export async function runSimulation(
  scenario: SimulationScenario,
): Promise<SimulationResult> {
  const controller = new AbortController()
  let timedOut = false

  const timeout = setTimeout(() => {
    timedOut = true
    controller.abort()
  }, SIMULATION_TIMEOUT_MS)

  try {
    const response = await fetch('/api/v1/simulations', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(scenario),
      signal: controller.signal,
    })

    const payload: unknown = await response.json().catch(() => null)

    if (!response.ok) {
      const message =
        payload &&
        typeof payload === 'object' &&
        'error' in payload &&
        typeof (payload as ApiError).error === 'string'
          ? (payload as ApiError).error!
          : `Simulation request failed (${response.status})`

      throw new Error(message)
    }

    if (!isSimulationResult(payload)) {
      throw new Error('Invalid simulation response received from API')
    }

    return payload
  } catch (error) {
    if (timedOut) {
  throw new Error(
    'Simulation request timed out. Please try again.',
    { cause: error },
  )
}

throw error
  } finally {
    clearTimeout(timeout)
  }
}
