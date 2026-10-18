import { afterEach, describe, expect, it, vi } from 'vitest'
import { runSimulation, type SimulationResult } from './api'

const validResult: SimulationResult = {
  name: 'api-smoke-test',
  duration: 10,
  summary: {
    totalRequests: 2,
    createdRequests: 2,
    queuedRequests: 0,
    retryWaitingRequests: 0,
    inFlightRequests: 0,
    succeededRequests: 2,
    failedRequests: 0,
    cancelledRequests: 0,
    timedOutRequests: 0,
    completedRequests: 2,
    retries: 0,
    endToEndLatency: { count: 2, min: 5, max: 9, average: 7 },
    serviceLatency: { count: 2, min: 5, max: 5, average: 5 },
  },
  metrics: {
    totalRequests: 2,
    createdRequests: 2,
    queuedRequests: 0,
    inFlightRequests: 0,
    succeededRequests: 2,
    failedRequests: 0,
    cancelledRequests: 0,
    timedOutRequests: 0,
    busyWorkers: 0,
    serviceCapacity: 1,
    waitingRequests: 0,
    workerUtilization: 0.75,
    workers: {},
  },
  routeTraces: {},
  objectives: {
    configured: false,
    passed: true,
    outcomes: [],
  },
}

function mockResponse(
  body: unknown,
  status = 200,
): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: vi.fn().mockResolvedValue(body),
  } as unknown as Response
}

describe('runSimulation', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('returns a valid simulation result', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockResponse(validResult)))

    await expect(
      runSimulation({
        name: 'api-smoke-test',
        serviceCapacity: 1,
        serviceDuration: 5,
        arrivals: [{ requestID: 'request-1', at: 0 }],
      }),
    ).resolves.toEqual(validResult)
  })

  it('rejects malformed objective evaluations', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockResponse({
      ...validResult,
      objectives: {
        configured: true,
        passed: true,
        outcomes: [{
          name: 'minimum_success_rate',
          passed: true,
          actual: 'not-a-number',
          threshold: 0.95,
          message: 'invalid actual value',
        }],
      },
    })))

    await expect(
      runSimulation({
        name: 'api-smoke-test',
        serviceCapacity: 1,
        serviceDuration: 5,
        arrivals: [],
      }),
    ).rejects.toThrow(/invalid simulation response/i)
  })
  it('rejects a successful response with invalid JSON data shape', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockResponse({
      name: 'api-smoke-test',
      duration: 'not-a-number',
      summary: {},
      metrics: null,
    })))

    await expect(
      runSimulation({
        name: 'api-smoke-test',
        serviceCapacity: 1,
        serviceDuration: 5,
        arrivals: [],
      }),
    ).rejects.toThrow(/invalid simulation response/i)
  })

  it('rejects a successful response that is not an object', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockResponse(null)))

    await expect(
      runSimulation({
        name: 'api-smoke-test',
        serviceCapacity: 1,
        serviceDuration: 5,
        arrivals: [],
      }),
    ).rejects.toThrow(/invalid simulation response/i)
  })

  it('preserves API error messages for unsuccessful responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(mockResponse({ error: 'scenario rejected by API' }, 400)),
    )

    await expect(
      runSimulation({
        name: 'api-smoke-test',
        serviceCapacity: 1,
        serviceDuration: 5,
        arrivals: [],
      }),
    ).rejects.toThrow('scenario rejected by API')
  })

  it('aborts a simulation request when the timeout expires', async () => {
    vi.useFakeTimers()

    const fetchMock = vi.fn(
      (_input: RequestInfo | URL, init?: RequestInit) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            reject(new DOMException('The operation was aborted', 'AbortError'))
          })
        }),
    )

    vi.stubGlobal('fetch', fetchMock)

    const request = runSimulation({
      name: 'api-smoke-test',
      serviceCapacity: 1,
      serviceDuration: 5,
      arrivals: [],
    })

    const assertion = expect(request).rejects.toThrow(/timed out/i)

    await vi.advanceTimersByTimeAsync(30_000)
    await assertion

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/simulations',
      expect.objectContaining({
        signal: expect.any(AbortSignal),
      }),
    )
  })

})
