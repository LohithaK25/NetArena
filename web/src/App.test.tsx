import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { runSimulation } from './api'

vi.mock('./api', () => ({
  runSimulation: vi.fn(),
}))

const mockRunSimulation = vi.mocked(runSimulation)

const successfulResult = {
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

describe('NetArena dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the default scenario form and empty results state', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: 'Scenario setup' })).toBeInTheDocument()
    expect(screen.getByLabelText('SCENARIO NAME')).toHaveValue('api-smoke-test')
    expect(screen.getByRole('spinbutton', { name: /SERVICE CAPACITY/ })).toHaveValue(1)
    expect(screen.getByRole('spinbutton', { name: /SERVICE DURATION/ })).toHaveValue(5)
    expect(screen.getByRole('button', { name: /Run simulation/ })).toBeEnabled()
    expect(screen.getByText('AWAITING EXECUTION')).toBeInTheDocument()
  })

  it('submits objective thresholds using the API fraction format', async () => {
    const user = userEvent.setup()
    mockRunSimulation.mockResolvedValue(successfulResult)

    render(<App />)

    await user.clear(screen.getByRole('spinbutton', { name: /MINIMUM SUCCESS RATE/ }))
    await user.type(screen.getByRole('spinbutton', { name: /MINIMUM SUCCESS RATE/ }), '95')
    await user.clear(screen.getByRole('spinbutton', { name: /MAXIMUM AVERAGE LATENCY/ }))
    await user.type(screen.getByRole('spinbutton', { name: /MAXIMUM AVERAGE LATENCY/ }), '20')
    await user.click(screen.getByRole('button', { name: /Run simulation/ }))

    await waitFor(() => {
      expect(mockRunSimulation).toHaveBeenCalledWith(expect.objectContaining({
        objectives: {
          minSuccessRate: 0.95,
          maxAverageLatency: 20,
        },
      }))
    })
  })

  it.each([
    {
      label: 'passing',
      passed: true,
      status: 'ALL PASSED',
      outcomeMessage: 'Success rate 100.0%; minimum required 95.0%',
    },
    {
      label: 'failing',
      passed: false,
      status: 'NOT MET',
      outcomeMessage: 'Success rate 50.0%; minimum required 95.0%',
    },
  ])('renders $label objective evaluation', async ({ passed, status, outcomeMessage }) => {
    const user = userEvent.setup()
    mockRunSimulation.mockResolvedValue({
      ...successfulResult,
      objectives: {
        configured: true,
        passed,
        outcomes: [
          {
            name: 'minimum_success_rate',
            passed,
            actual: passed ? 1 : 0.5,
            threshold: 0.95,
            message: outcomeMessage,
          },
        ],
      },
    })

    render(<App />)
    await user.click(screen.getByRole('button', { name: /Run simulation/ }))

    expect(await screen.findByText(status)).toBeInTheDocument()
    expect(screen.getByText('minimum_success_rate')).toBeInTheDocument()
    expect(screen.getByText(passed ? 'PASS' : 'FAIL')).toBeInTheDocument()
    expect(screen.getByText(outcomeMessage)).toBeInTheDocument()
  })
  it('submits the scenario and displays the returned metrics', async () => {
    const user = userEvent.setup()
    mockRunSimulation.mockResolvedValue(successfulResult)

    render(<App />)
    await user.click(screen.getByRole('button', { name: /Run simulation/ }))

    await waitFor(() => {
      expect(mockRunSimulation).toHaveBeenCalledWith({
        name: 'api-smoke-test',
        serviceCapacity: 1,
        serviceDuration: 5,
        arrivals: [
          { requestID: 'request-1', at: 0 },
          { requestID: 'request-2', at: 1 },
        ],
      })
    })

    expect(await screen.findByRole('heading', { name: 'api-smoke-test' })).toBeInTheDocument()
    expect(screen.getByText('COMPLETED')).toBeInTheDocument()
    expect(screen.getByText('Virtual simulation duration').parentElement).toHaveTextContent('10')
    expect(screen.getByText('Total requests').parentElement).toHaveTextContent('2')
    expect(screen.getByText('Succeeded').parentElement).toHaveTextContent('2')
    expect(screen.getByText('Failed').parentElement).toHaveTextContent('0')
    expect(screen.getByText('Cancelled').parentElement).toHaveTextContent('0')
    expect(screen.getByText('7.00')).toBeInTheDocument()
    expect(screen.getByText('5.00')).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Worker utilization' })).toHaveAttribute('aria-valuenow', '75')
  })

  it('disables submission while a simulation is in progress', async () => {
    const user = userEvent.setup()

    let resolveSimulation!: (value: typeof successfulResult) => void
    mockRunSimulation.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveSimulation = resolve
        }),
    )

    render(<App />)

    const submitButton = screen.getByRole('button', {
      name: /Run simulation/,
    })

    await user.click(submitButton)

    expect(
      screen.getByRole('button', { name: /Running simulation/ }),
    ).toBeDisabled()

    expect(mockRunSimulation).toHaveBeenCalledTimes(1)

    resolveSimulation(successfulResult)

    expect(await screen.findByText('COMPLETED')).toBeInTheDocument()
  })


  it('renders route trace hops with endpoints, tick, latency, and status', async () => {
    const user = userEvent.setup()

    mockRunSimulation.mockResolvedValue({
      ...successfulResult,
      routeTraces: {
        'request-1': [
          {
            from: 'client-1',
            to: 'service-1',
            at: 2,
            latency: 3,
            status: 'forwarded',
          },
        ],
      },
    })

    render(<App />)
    await user.click(screen.getByRole('button', { name: /Run simulation/ }))

    expect(await screen.findByRole('heading', { name: 'Request route traces' })).toBeInTheDocument()
    expect(screen.getByText('request-1')).toBeInTheDocument()
    expect(screen.getByText('client-1')).toBeInTheDocument()
    expect(screen.getByText('service-1')).toBeInTheDocument()
    expect(screen.getByText('Tick 2')).toBeInTheDocument()
    expect(screen.getByText('Latency 3')).toBeInTheDocument()
    expect(screen.getByText('forwarded')).toBeInTheDocument()
  })

  it('shows an API error and permits a retry', async () => {
    const user = userEvent.setup()
    mockRunSimulation
      .mockRejectedValueOnce(new Error('scenario rejected by API'))
      .mockResolvedValueOnce(successfulResult)

    render(<App />)
    await user.click(screen.getByRole('button', { name: /Run simulation/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent('scenario rejected by API')

    await user.click(screen.getByRole('button', { name: /Run simulation/ }))

    expect(await screen.findByText('COMPLETED')).toBeInTheDocument()
    expect(mockRunSimulation).toHaveBeenCalledTimes(2)
  })


  it('requires topology when shortest-path routing is selected', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.selectOptions(
      screen.getByLabelText('ROUTING ALGORITHM'),
      'shortest_path',
    )

    await user.click(screen.getByRole('button', { name: /run simulation/i }))

    expect(
      await screen.findByText(
        'Shortest-path routing requires network topology. Enable topology first.',
      ),
    ).toBeInTheDocument()
    expect(mockRunSimulation).not.toHaveBeenCalled()
  })

  it('rejects duplicate topology node IDs without calling the API', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(
      screen.getByRole('checkbox', { name: /enable network topology/i }),
    )

    const nodeCards = document.querySelectorAll('.topology-node')
    expect(nodeCards).toHaveLength(3)

    const secondNodeID = nodeCards[1].querySelector('input')!
    await user.clear(secondNodeID)
    await user.type(secondNodeID, 'client-1')

    expect(
      document.querySelectorAll('.topology-node')[1].querySelector('input'),
    ).toHaveValue('client-1')

    await user.click(screen.getByRole('button', { name: /run simulation/i }))

    expect(
      await screen.findByText('Topology node IDs must be unique.'),
    ).toBeInTheDocument()
    expect(mockRunSimulation).not.toHaveBeenCalled()
  })


  it('adds and removes topology nodes without losing row identity', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(
      screen.getByRole('checkbox', { name: /enable network topology/i }),
    )

    const initialRows = Array.from(
      document.querySelectorAll('.topology-node'),
    )
    expect(initialRows).toHaveLength(3)

    const workerID = initialRows[2].querySelector('input')!
    await user.clear(workerID)
    await user.type(workerID, 'worker-renamed')

    expect(
      document.querySelectorAll('.topology-node')[2].querySelector('input'),
    ).toHaveValue('worker-renamed')

    await user.click(screen.getByRole('button', { name: /add node/i }))
    expect(document.querySelectorAll('.topology-node')).toHaveLength(4)

    await user.click(
      screen.getByRole('button', { name: 'Remove node service-1' }),
    )
    expect(document.querySelectorAll('.topology-node')).toHaveLength(3)

    expect(
      document.querySelectorAll('.topology-node')[1].querySelector('input'),
    ).toHaveValue('worker-renamed')
  })

  it('adds and removes request arrivals', async () => {
    const user = userEvent.setup()
    render(<App />)

    expect(screen.getByLabelText('Request 1 ID')).toHaveValue('request-1')
    expect(screen.getByLabelText('Request 2 ID')).toHaveValue('request-2')

    await user.click(screen.getByRole('button', { name: /Add request/ }))
    expect(screen.getByLabelText('Request 3 ID')).toHaveValue('request-3')
    expect(screen.getByText('3 REQUESTS')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Remove request 2' }))
    expect(screen.queryByLabelText('Request 3 ID')).not.toBeInTheDocument()
    expect(screen.getByText('2 REQUESTS')).toBeInTheDocument()
  })
})
