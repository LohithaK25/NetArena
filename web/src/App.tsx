import { useRef, useState, type FormEvent } from 'react'
import {
  runSimulation,
  type RoutingAlgorithm,
  type SimulationArrival,
  type SimulationResult,
  type SimulationTopology,
  type TopologyNode,
  type TopologyConnection,
} from './api'
import './App.css'

function App() {
  const [name, setName] = useState('api-smoke-test')
  const [capacity, setCapacity] = useState(1)
  const [serviceDuration, setServiceDuration] = useState(5)
  const [minSuccessRate, setMinSuccessRate] = useState('')
  const [maxAverageLatency, setMaxAverageLatency] = useState('')
  const [arrivals, setArrivals] = useState<SimulationArrival[]>([
    { requestID: 'request-1', at: 0 },
    { requestID: 'request-2', at: 1 },
  ])
  const [result, setResult] = useState<SimulationResult | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const [routingAlgorithm, setRoutingAlgorithm] =
    useState<RoutingAlgorithm>('round_robin')
  const [topologyEnabled, setTopologyEnabled] = useState(false)
  const [nodes, setNodes] = useState<TopologyNode[]>([
    { id: 'client-1', type: 1, name: 'Client', capacity: 0 },
    { id: 'service-1', type: 3, name: 'Service', capacity: 0 },
    { id: 'worker-1', type: 4, name: 'Worker', capacity: 1 },
  ])
  const [nodeKeys, setNodeKeys] = useState([
    'node-key-1',
    'node-key-2',
    'node-key-3',
  ])
  const nextNodeKey = useRef(4)
  const [connections, setConnections] = useState<TopologyConnection[]>([
    {
      from: 'client-1',
      to: 'service-1',
      latency: 1,
      dropPackets: false,
      packetLossProbability: 0,
    },
    {
      from: 'service-1',
      to: 'worker-1',
      latency: 1,
      dropPackets: false,
      packetLossProbability: 0,
    },
  ])
  const [sourceNode, setSourceNode] = useState('client-1')

  function updateArrival(
    index: number,
    field: keyof SimulationArrival,
    value: string,
  ) {
    setArrivals((current) =>
      current.map((arrival, i) =>
        i === index
          ? {
            ...arrival,
            [field]: field === 'at' ? Number(value) : value,
          }
          : arrival,
      ),
    )
  }

  function addArrival() {
    setArrivals((current) => [
      ...current,
      { requestID: `request-${current.length + 1}`, at: 0 },
    ])
  }

  function removeArrival(index: number) {
    setArrivals((current) => current.filter((_, i) => i !== index))
  }


  function updateNode(index: number, field: keyof TopologyNode, value: string) {
    setNodes((current) =>
      current.map((node, i) =>
        i === index
          ? {
            ...node,
            [field]:
              field === 'type' || field === 'capacity'
                ? Number(value)
                : value,
          }
          : node,
      ),
    )
  }

  function addNode() {
    const id = `node-${nodes.length + 1}`
    const key = `node-key-${nextNodeKey.current++}`

    setNodes((current) => [
      ...current,
      { id, type: 3, name: 'Service', capacity: 0 },
    ])
    setNodeKeys((current) => [...current, key])
  }

  function removeNode(index: number) {
    const removed = nodes[index]
    const remaining = nodes.filter((_, i) => i !== index)
    setNodes(remaining)
    setNodeKeys((current) => current.filter((_, i) => i !== index))
    setConnections((current) =>
      current.filter(
        (connection) =>
          connection.from !== removed.id && connection.to !== removed.id,
      ),
    )
    if (sourceNode === removed.id) {
      const nextClient = remaining.find((node) => node.type === 1)
      setSourceNode(nextClient?.id ?? remaining[0]?.id ?? '')
    }
  }

  function updateConnection(
    index: number,
    field: keyof TopologyConnection,
    value: string | boolean,
  ) {
    setConnections((current) =>
      current.map((connection, i) =>
        i === index
          ? {
            ...connection,
            [field]:
              field === 'dropPackets'
                ? Boolean(value)
                : field === 'from' || field === 'to'
                  ? String(value)
                  : Number(value),
          }
          : connection,
      ),
    )
  }

  function addConnection() {
    if (nodes.length < 2) return
    setConnections((current) => [
      ...current,
      {
        from: nodes[0].id,
        to: nodes[1].id,
        latency: 1,
        dropPackets: false,
        packetLossProbability: 0,
      },
    ])
  }


  function validateTopology(): string | null {
    if (!topologyEnabled) {
      if (routingAlgorithm === 'shortest_path') {
        return 'Shortest-path routing requires network topology. Enable topology first.'
      }
      return null
    }

    if (nodes.length === 0) {
      return 'Topology must contain at least one node.'
    }

    const nodeIDs = nodes.map((node) => node.id.trim())
    if (nodeIDs.some((id) => !id)) {
      return 'Every topology node must have an ID.'
    }

    if (new Set(nodeIDs).size !== nodeIDs.length) {
      return 'Topology node IDs must be unique.'
    }

    if (!nodeIDs.includes(sourceNode.trim())) {
      return 'Select a valid request source node.'
    }

    for (const node of nodes) {
      if (![1, 2, 3, 4].includes(node.type)) {
        return `Node "${node.id}" has an invalid type.`
      }
      if (node.type !== 4 && node.capacity !== 0) {
        return `Only worker nodes may define capacity: "${node.id}".`
      }
      if (!Number.isInteger(node.capacity) || node.capacity < 0) {
        return `Node "${node.id}" must have a non-negative integer capacity.`
      }
    }

    const nodeIDSet = new Set(nodeIDs)

    for (const connection of connections) {
      if (!nodeIDSet.has(connection.from.trim()) || !nodeIDSet.has(connection.to.trim())) {
        return 'Every connection must reference existing nodes.'
      }
      if (connection.from.trim() === connection.to.trim()) {
        return `Connection "${connection.from}" cannot point to itself.`
      }
      if (!Number.isFinite(connection.latency) || connection.latency < 0) {
        return 'Connection latency must be a non-negative number.'
      }
      if (
        !Number.isFinite(connection.packetLossProbability) ||
        connection.packetLossProbability < 0 ||
        connection.packetLossProbability > 1
      ) {
        return 'Packet-loss probability must be between 0 and 1.'
      }
    }

    const connectionKeys = connections.map(
      (connection) => `${connection.from.trim()}\u0000${connection.to.trim()}`,
    )
    if (new Set(connectionKeys).size !== connectionKeys.length) {
      return 'Duplicate directed connections are not allowed.'
    }

    return null
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setResult(null)

    const validationError = validateTopology()
    if (validationError) {
      setError(validationError)
      return
    }

    setLoading(true)

    try {
      const simulation = await runSimulation({
        name: name.trim(),
        serviceCapacity: capacity,
        serviceDuration,
        arrivals: arrivals.map((arrival) => ({
          ...arrival,
          requestID: arrival.requestID.trim(),
          ...(topologyEnabled ? { sourceNode } : {}),
        })),
        ...(minSuccessRate.trim() !== '' || maxAverageLatency.trim() !== ''
          ? {
            objectives: {
              ...(minSuccessRate.trim() !== ''
                ? { minSuccessRate: Number(minSuccessRate) / 100 }
                : {}),
              ...(maxAverageLatency.trim() !== ''
                ? { maxAverageLatency: Number(maxAverageLatency) }
                : {}),
            },
          }
          : {}),
        ...(topologyEnabled
          ? {
            routingAlgorithm,
            topology: {
              nodes,
              connections,
            } satisfies SimulationTopology,
          }
          : {}),
      })
      setResult(simulation)
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : 'An unexpected error occurred while running the simulation.',
      )
    } finally {
      setLoading(false)
    }
  }

  const summary = result?.summary
  const metrics = result?.metrics

  return (
    <main className="app-shell">
      <header className="topbar">
        <a className="brand" href="#" aria-label="NetArena home">
          <span className="brand-mark" aria-hidden="true">
            N
          </span>
          <span>
            <strong>NetArena</strong>
            <small>NETWORK SYSTEMS LAB</small>
          </span>
        </a>
        <div className="topbar-status">
          <span className="status-dot" />
          SIMULATION WORKSPACE
        </div>
      </header>

      <section className="hero">
        <div className="eyebrow">
          <span>01</span> / SIMULATION ENGINE
        </div>
        <h1>
          Design it.
          <br />
          <span>Stress-test it.</span>
        </h1>
        <p>
          Configure incoming requests, tune service capacity, and observe how
          your system handles the workload. Every result comes from the NetArena
          simulation engine.
        </p>
        <div className="hero-meta">
          <span><span className="meta-dot" /> DETERMINISTIC SIMULATION</span>
          <span>VIRTUAL TIME · DISCRETE EVENTS</span>
        </div>
      </section>

      <div className="workspace">
        <section className="panel config-panel">
          <div className="panel-heading">
            <div>
              <div className="section-label">CONFIGURATION / 01</div>
              <h2>Scenario setup</h2>
            </div>
            <span className="panel-icon" aria-hidden="true">⌘</span>
          </div>

          <form onSubmit={handleSubmit}>
            <label className="field">
              <span>SCENARIO NAME</span>
              <input
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                maxLength={100}
                placeholder="e.g. traffic-spike"
              />
            </label>

            <div className="field-grid">
              <label className="field">
                <span>SERVICE CAPACITY</span>
                <input
                  type="number"
                  min={1}
                  max={1000}
                  value={capacity}
                  onChange={(event) => setCapacity(Number(event.target.value))}
                  required
                />
                <small>Concurrent service slots</small>
              </label>
              <label className="field">
                <span>SERVICE DURATION</span>
                <input
                  type="number"
                  min={1}
                  max={1000000}
                  value={serviceDuration}
                  onChange={(event) =>
                    setServiceDuration(Number(event.target.value))
                  }
                  required
                />
                <small>Ticks per request</small>
              </label>
            </div>

            <div className="arrivals-heading">
              <div>
                <div className="section-label">NETWORK / 01</div>
                <h3>Routing configuration</h3>
              </div>
            </div>

            <label className="field" style={{ marginTop: 18 }}>
              <span>ROUTING ALGORITHM</span>
              <select
                value={routingAlgorithm}
                onChange={(event) =>
                  setRoutingAlgorithm(event.target.value as RoutingAlgorithm)
                }
              >
                <option value="round_robin">Round robin</option>
                <option value="shortest_path">Shortest path</option>
              </select>
            </label>

            <div className="objective-fields">
              <div className="section-label">SUCCESS CRITERIA / OPTIONAL</div>

              <label className="field">
                <span>MINIMUM SUCCESS RATE (%)</span>
                <input
                  type="number"
                  min={0}
                  max={100}
                  step={1}
                  value={minSuccessRate}
                  onChange={(event) => setMinSuccessRate(event.target.value)}
                  placeholder="e.g. 95"
                />
              </label>

              <label className="field">
                <span>MAXIMUM AVERAGE LATENCY (TICKS)</span>
                <input
                  type="number"
                  min={0}
                  step={1}
                  value={maxAverageLatency}
                  onChange={(event) => setMaxAverageLatency(event.target.value)}
                  placeholder="e.g. 20"
                />
              </label>

              <p>
                Set either or both thresholds to evaluate the scenario against
                your targets.
              </p>
            </div>

            <label className="topology-toggle" style={{ marginTop: 18 }}>
              <input
                type="checkbox"
                checked={topologyEnabled}
                onChange={(event) => setTopologyEnabled(event.target.checked)}
              />
              <span>
                <strong>Enable network topology</strong>
                <small>Route requests through configured network nodes.</small>
              </span>
            </label>


            {topologyEnabled && (
              <section className="topology-editor">
                <div className="arrivals-heading">
                  <div>
                    <div className="section-label">NETWORK / TOPOLOGY</div>
                    <h3>Nodes</h3>
                  </div>
                  <span className="count-pill">{nodes.length} NODES</span>
                </div>

                {nodes.map((node, index) => (
                  <div className="topology-node" key={index}>
                    <label className="field">
                      <span>NODE ID</span>
                      <input
                        value={node.id}
                        onChange={(event) => {
                          updateNode(index, 'id', event.target.value)
                        }}
                        required
                      />
                    </label>

                    <label className="field">
                      <span>TYPE</span>
                      <select
                        value={node.type}
                        onChange={(event) =>
                          updateNode(index, 'type', event.target.value)
                        }
                      >
                        <option value={1}>Client</option>
                        <option value={2}>Load balancer</option>
                        <option value={3}>Service</option>
                        <option value={4}>Worker</option>
                      </select>
                    </label>

                    <label className="field">
                      <span>DISPLAY NAME</span>
                      <input
                        value={node.name}
                        onChange={(event) =>
                          updateNode(index, 'name', event.target.value)
                        }
                        required
                      />
                    </label>

                    {node.type === 4 && (
                      <label className="field">
                        <span>WORKER CAPACITY</span>
                        <input
                          type="number"
                          min={0}
                          value={node.capacity}
                          onChange={(event) =>
                            updateNode(index, 'capacity', event.target.value)
                          }
                          required
                        />
                      </label>
                    )}

                    <button
                      className="remove-button"
                      type="button"
                      aria-label={`Remove node ${node.id}`}
                      disabled={nodes.length <= 1}
                      onClick={() => removeNode(index)}
                    >
                      ×
                    </button>
                  </div>
                ))}

                <button className="add-arrival" type="button" onClick={addNode}>
                  <span aria-hidden="true">+</span> Add node
                </button>

                <label className="field" style={{ marginTop: 18 }}>
                  <span>REQUEST SOURCE NODE</span>
                  <select
                    value={sourceNode}
                    onChange={(event) => setSourceNode(event.target.value)}
                    required
                  >
                    {nodes.map((node, nodeIndex) => (
                      <option key={nodeKeys[nodeIndex]} value={node.id}>
                        {node.id} — {node.name}
                      </option>
                    ))}
                  </select>
                </label>

                <div className="arrivals-heading">
                  <div>
                    <div className="section-label">NETWORK / LINKS</div>
                    <h3>Directed connections</h3>
                  </div>
                  <span className="count-pill">
                    {connections.length} LINKS
                  </span>
                </div>

                {connections.map((connection, index) => (
                  <div className="topology-connection" key={index}>
                    <div className="field-grid">
                      <label className="field">
                        <span>FROM</span>
                        <select
                          value={connection.from}
                          onChange={(event) =>
                            updateConnection(index, 'from', event.target.value)
                          }
                          required
                        >
                          {nodes.map((node, nodeIndex) => (
                            <option key={nodeKeys[nodeIndex]} value={node.id}>
                              {node.id}
                            </option>
                          ))}
                        </select>
                      </label>

                      <label className="field">
                        <span>TO</span>
                        <select
                          value={connection.to}
                          onChange={(event) =>
                            updateConnection(index, 'to', event.target.value)
                          }
                          required
                        >
                          {nodes.map((node, nodeIndex) => (
                            <option key={nodeKeys[nodeIndex]} value={node.id}>
                              {node.id}
                            </option>
                          ))}
                        </select>
                      </label>

                      <label className="field">
                        <span>LATENCY (TICKS)</span>
                        <input
                          type="number"
                          min={0}
                          value={connection.latency}
                          onChange={(event) =>
                            updateConnection(index, 'latency', event.target.value)
                          }
                          required
                        />
                      </label>

                      <label className="field">
                        <span>PACKET LOSS (0–1)</span>
                        <input
                          type="number"
                          min={0}
                          max={1}
                          step={0.05}
                          value={connection.packetLossProbability}
                          onChange={(event) =>
                            updateConnection(
                              index,
                              'packetLossProbability',
                              event.target.value,
                            )
                          }
                          required
                        />
                      </label>
                    </div>

                    <label className="topology-toggle">
                      <input
                        type="checkbox"
                        checked={connection.dropPackets}
                        onChange={(event) =>
                          updateConnection(
                            index,
                            'dropPackets',
                            event.target.checked,
                          )
                        }
                      />
                      <span>
                        <strong>Drop packets on this link</strong>
                      </span>
                    </label>

                    <button
                      className="remove-button"
                      type="button"
                      aria-label={`Remove connection ${index + 1}`}
                      onClick={() =>
                        setConnections((current) =>
                          current.filter((_, i) => i !== index),
                        )
                      }
                    >
                      Remove connection
                    </button>
                  </div>
                ))}

                <button
                  className="add-arrival"
                  type="button"
                  onClick={addConnection}
                  disabled={nodes.length < 2}
                >
                  <span aria-hidden="true">+</span> Add connection
                </button>
              </section>
            )}

            <div className="arrivals-heading">
              <div>
                <div className="section-label">INPUT STREAM</div>
                <h3>Request arrivals</h3>
              </div>
              <span className="count-pill">{arrivals.length} REQUESTS</span>
            </div>

            <div className="arrival-list">
              {arrivals.map((arrival, index) => (
                <div className="arrival-row" key={`${index}-${arrival.requestID}`}>
                  <span className="arrival-index">
                    {String(index + 1).padStart(2, '0')}
                  </span>
                  <label className="arrival-input">
                    <span className="visually-hidden">Request ID</span>
                    <input
                      aria-label={`Request ${index + 1} ID`}
                      value={arrival.requestID}
                      onChange={(event) =>
                        updateArrival(index, 'requestID', event.target.value)
                      }
                      required
                      maxLength={100}
                      placeholder="request-id"
                    />
                  </label>
                  <label className="arrival-tick">
                    <span className="visually-hidden">Arrival tick</span>
                    <input
                      aria-label={`Request ${index + 1} arrival tick`}
                      type="number"
                      min={0}
                      max={1000000}
                      value={arrival.at}
                      onChange={(event) =>
                        updateArrival(index, 'at', event.target.value)
                      }
                      required
                    />
                  </label>
                  <button
                    className="remove-button"
                    type="button"
                    aria-label={`Remove request ${index + 1}`}
                    disabled={arrivals.length <= 1}
                    onClick={() => removeArrival(index)}
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>

            <button className="add-arrival" type="button" onClick={addArrival}>
              <span aria-hidden="true">+</span> Add request
            </button>

            <div className="form-note">
              <span aria-hidden="true">ⓘ</span>
              Requests are scheduled at their specified virtual ticks. IDs must
              be unique.
            </div>

            <button className="run-button" type="submit" disabled={loading}>
              <span>{loading ? 'Running simulation…' : 'Run simulation'}</span>
              <span className="run-arrow" aria-hidden="true">
                {loading ? '···' : '↗'}
              </span>
            </button>
          </form>
        </section>

        <section className="results-column" aria-live="polite">
          <div className="results-heading">
            <div>
              <div className="section-label">OUTPUT / 02</div>
              <h2>Simulation results</h2>
            </div>
            {result && <span className="result-badge">COMPLETED</span>}
          </div>

          {error && (
            <div className="error-panel" role="alert">
              <strong>Simulation failed</strong>
              <p>{error}</p>
              <small>Check the scenario and confirm the Go API is running.</small>
            </div>
          )}

          {!result && !error && (
            <div className="empty-state">
              <div className="empty-visual" aria-hidden="true">
                <div className="orbit orbit-one" />
                <div className="orbit orbit-two" />
                <div className="orbit-core">N</div>
                <span className="orbit-node node-one" />
                <span className="orbit-node node-two" />
                <span className="orbit-node node-three" />
              </div>
              <div className="section-label">AWAITING EXECUTION</div>
              <h3>Your system, under the microscope.</h3>
              <p>
                Configure your workload and run a simulation. Request outcomes,
                latency, and worker metrics will appear here.
              </p>
              <div className="empty-divider" />
              <div className="empty-facts">
                <span>01 <strong>Configure</strong></span>
                <span>02 <strong>Execute</strong></span>
                <span>03 <strong>Analyze</strong></span>
              </div>
            </div>
          )}

          {result && summary && metrics && (
            <div className="result-content">
              <div className="result-summary">
                <div className="result-summary-top">
                  <div>
                    <span className="section-label">SCENARIO EXECUTED</span>
                    <h3>{result.name}</h3>
                  </div>
                  <span className="success-mark" aria-label="Success">✓</span>
                </div>
                <div className="duration-line">
                  <span>Virtual simulation duration</span>
                  <strong>{result.duration} <small>ticks</small></strong>
                </div>
              </div>

              <section className="metric-panel objective-result-panel">
                <div className="metric-heading">
                  <div>
                    <div className="section-label">SCENARIO / OBJECTIVES</div>
                    <h3>Objective evaluation</h3>
                  </div>
                  {result.objectives.configured && (
                    <span
                      className={result.objectives.passed
                        ? 'objective-status objective-passed'
                        : 'objective-status objective-failed'}
                    >
                      {result.objectives.passed ? 'ALL PASSED' : 'NOT MET'}
                    </span>
                  )}
                </div>

                {!result.objectives.configured ? (
                  <p className="objective-empty">
                    No success criteria configured. Set optional thresholds and
                    run again to evaluate this scenario against your targets.
                  </p>
                ) : (
                  <div className="objective-outcomes">
                    {result.objectives.outcomes.map((outcome) => (
                      <article className="objective-outcome" key={outcome.name}>
                        <div className="objective-outcome-heading">
                          <strong>{outcome.name}</strong>
                          <span>{outcome.passed ? 'PASS' : 'FAIL'}</span>
                        </div>
                        <p>{outcome.message}</p>
                        <div className="objective-values">
                          <span>
                            Actual:{' '}
                            {outcome.name === 'minimum_success_rate'
                              ? `${(outcome.actual * 100).toFixed(1)}%`
                              : outcome.actual.toFixed(2)}
                          </span>
                          <span>
                            Threshold:{' '}
                            {outcome.name === 'minimum_success_rate'
                              ? `${(outcome.threshold * 100).toFixed(1)}%`
                              : outcome.threshold.toFixed(2)}
                          </span>

                        </div>
                      </article>
                    ))}
                  </div>
                )}
              </section>

              <div className="stat-grid">
                {Object.entries(result.routeTraces ?? {}).length > 0 && (
                  <section className="metric-panel route-trace-panel">
                    <div className="metric-heading">
                      <div>
                        <div className="section-label">NETWORK / ROUTING</div>
                        <h3>Request route traces</h3>
                      </div>
                      <span className="metric-unit">
                        {Object.keys(result.routeTraces).length} REQUESTS
                      </span>
                    </div>

                    {Object.entries(result.routeTraces).map(([requestID, hops]) => (
                      <div className="route-trace" key={requestID}>
                        <div className="route-trace-heading">
                          <strong>{requestID}</strong>
                          <span>{hops.length} hops</span>
                        </div>

                        {hops.length === 0 ? (
                          <p>No route hops recorded.</p>
                        ) : (
                          <ol className="route-hop-list">
                            {hops.map((hop, index) => (
                              <li
                                className={`route-hop route-hop-${hop.status}`}
                                key={`${requestID}-${index}`}
                              >
                                <div className="route-hop-path">
                                  <strong>{hop.from}</strong>
                                  <span aria-hidden="true">→</span>
                                  <strong>{hop.to}</strong>
                                </div>
                                <div className="route-hop-details">
                                  <span>Tick {hop.at}</span>
                                  <span>Latency {hop.latency}</span>
                                  <span>{hop.status.replaceAll('_', ' ')}</span>
                                </div>
                              </li>
                            ))}
                          </ol>
                        )}
                      </div>
                    ))}
                  </section>
                )}
                <article className="stat-card">
                  <span>Total requests</span>
                  <strong>{summary.totalRequests}</strong>
                  <small>Submitted workload</small>
                </article>
                <article className="stat-card stat-success">
                  <span>Succeeded</span>
                  <strong>{summary.succeededRequests}</strong>
                  <small>Completed successfully</small>
                </article>
                <article className="stat-card stat-failure">
                  <span>Failed</span>
                  <strong>{summary.failedRequests}</strong>
                  <small>Unsuccessful requests</small>
                </article>
                <article className="stat-card">
                  <span>Cancelled</span>
                  <strong>{summary.cancelledRequests}</strong>
                  <small>Cancelled requests</small>
                </article>
              </div>

              <section className="metric-panel">
                <div className="metric-heading">
                  <div>
                    <div className="section-label">PERFORMANCE</div>
                    <h3>Latency analysis</h3>
                  </div>
                  <span className="metric-unit">VIRTUAL TICKS</span>
                </div>
                <div className="latency-grid">
                  <div className="latency-item">
                    <span>End-to-end average</span>
                    <strong>{summary.endToEndLatency.average.toFixed(2)}</strong>
                    <small>
                      Min {summary.endToEndLatency.min} · Max{' '}
                      {summary.endToEndLatency.max}
                    </small>
                  </div>
                  <div className="latency-item">
                    <span>Service average</span>
                    <strong>{summary.serviceLatency.average.toFixed(2)}</strong>
                    <small>
                      Min {summary.serviceLatency.min} · Max{' '}
                      {summary.serviceLatency.max}
                    </small>
                  </div>
                </div>
              </section>

              <section className="metric-panel">
                <div className="metric-heading">
                  <div>
                    <div className="section-label">RESOURCE USAGE</div>
                    <h3>Worker utilization</h3>
                  </div>
                  <strong className="utilization-value">
                    {Math.round(metrics.workerUtilization * 100)}%
                  </strong>
                </div>
                <div
                  className="utilization-track"
                  role="progressbar"
                  aria-label="Worker utilization"
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={Math.max(
                    0,
                    Math.min(100, Math.round(metrics.workerUtilization * 100)),
                  )}
                >
                  <div
                    className="utilization-fill"
                    style={{
                      width: `${Math.max(
                        0,
                        Math.min(100, metrics.workerUtilization * 100),
                      )}%`,
                    }}
                  />
                </div>
                <div className="resource-foot">
                  <span>{metrics.busyWorkers} busy workers</span>
                  <span>{metrics.waitingRequests} waiting requests</span>
                </div>

                {Object.entries(metrics.workers).length > 0 && (
                  <div className="worker-list">
                    {Object.entries(metrics.workers).map(([id, worker]) => (
                      <div className="worker-row" key={id}>
                        <span className="worker-indicator" />
                        <strong>{id}</strong>
                        <span>{worker.online ? 'Online' : 'Offline'}</span>
                        <span>
                          {Math.round(worker.utilization * 100)}% utilized
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </section>

              <div className="result-footnote">
                Results returned by the NetArena engine. Latency and duration
                are measured in virtual ticks, not wall-clock time.
              </div>

              <button
                className="run-again-button"
                type="button"
                onClick={() => {
                  setResult(null)
                  setError('')
                }}
              >
                Configure another run <span aria-hidden="true">↗</span>
              </button>
            </div>
          )}
        </section>
      </div>

      <footer className="footer">
        <span>NETARENA <span className="footer-separator">/</span> SYSTEMS ENGINEERING SIMULATOR</span>
        <span>BUILT FOR EXPERIMENTATION.</span>
      </footer>
    </main>
  )
}

export default App
