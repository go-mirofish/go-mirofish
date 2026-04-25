import service, { requestWithRetry } from './index'

/** Create a simulation record for a completed graph project. */
export const createSimulation = (data) =>
  requestWithRetry(() => service.post('/api/simulation/create', data), 3, 1000)

/**
 * Prepare a simulation: load entities from graph, generate profiles, write simulation_config.json.
 * Starts async; poll via getPrepareStatus.
 */
export const prepareSimulation = (data) =>
  requestWithRetry(() => service.post('/api/simulation/prepare', data), 2, 2000)

/**
 * Poll prepare-task status.
 * @param {{ task_id?: string, simulation_id?: string }} data
 */
export const getPrepareStatus = (data) =>
  service({ url: '/api/simulation/prepare/status', method: 'post', data, timeout: 10000 })

/** Get the full simulation state (control_state.json). */
export const getSimulation = (simulationId) =>
  service({ url: `/api/simulation/${simulationId}`, method: 'get', timeout: 10000 })

/** Get agent profiles for the simulation. */
export const getSimulationProfiles = (simulationId, platform = 'reddit') =>
  service({ url: `/api/simulation/${simulationId}/profiles`, method: 'get', params: { platform }, timeout: 15000 })

/** Get real-time agent profiles (includes live round data). */
export const getSimulationProfilesRealtime = (simulationId, platform = 'reddit') =>
  service({ url: `/api/simulation/${simulationId}/profiles/realtime`, method: 'get', params: { platform }, timeout: 15000 })

/** Get the generated simulation config JSON. */
export const getSimulationConfig = (simulationId) =>
  service({ url: `/api/simulation/${simulationId}/config`, method: 'get', timeout: 10000 })

/** Get real-time simulation config with live stats overlaid. */
export const getSimulationConfigRealtime = (simulationId) =>
  service({ url: `/api/simulation/${simulationId}/config/realtime`, method: 'get', timeout: 10000 })

/** List simulations, optionally filtered by project. */
export const listSimulations = (projectId) =>
  service({ url: '/api/simulation/list', method: 'get', params: projectId ? { project_id: projectId } : {}, timeout: 10000 })

/** Start a simulation run. */
export const startSimulation = (data) =>
  requestWithRetry(() => service.post('/api/simulation/start', data), 2, 2000)

/** Stop a running simulation. */
export const stopSimulation = (data) =>
  service({ url: '/api/simulation/stop', method: 'post', data, timeout: 15000 })

/** Delete a simulation and its artifacts. */
export const deleteSimulation = (data) =>
  service({ url: '/api/simulation/delete', method: 'post', data, timeout: 15000 })

/** Get the lightweight run-status for a simulation (current round, progress %). */
export const getRunStatus = (simulationId) =>
  service({ url: `/api/simulation/${simulationId}/run-status`, method: 'get', timeout: 8000 })

/** Get detailed run-status including per-platform action counts. */
export const getRunStatusDetail = (simulationId) =>
  service({ url: `/api/simulation/${simulationId}/run-status/detail`, method: 'get', timeout: 10000 })

/** Get simulation posts (reddit or twitter). */
export const getSimulationPosts = (simulationId, platform = 'reddit', limit = 50, offset = 0) =>
  service({ url: `/api/simulation/${simulationId}/posts`, method: 'get', params: { platform, limit, offset }, timeout: 15000 })

/** Get simulation comments (reddit or twitter). */
export const getSimulationComments = (simulationId, platform = 'reddit', limit = 50, offset = 0) =>
  service({ url: `/api/simulation/${simulationId}/comments`, method: 'get', params: { platform, limit, offset }, timeout: 15000 })

/** Get per-round timeline for a simulation. */
export const getSimulationTimeline = (simulationId, startRound = 0, endRound = null) => {
  const params = { start_round: startRound }
  if (endRound !== null) params.end_round = endRound
  return service({ url: `/api/simulation/${simulationId}/timeline`, method: 'get', params, timeout: 15000 })
}

/** Get per-agent action stats for a simulation. */
export const getAgentStats = (simulationId) =>
  service({ url: `/api/simulation/${simulationId}/agent-stats`, method: 'get', timeout: 15000 })

/** Get paginated action log entries. */
export const getSimulationActions = (simulationId, params = {}) =>
  service({ url: `/api/simulation/${simulationId}/actions`, method: 'get', params, timeout: 15000 })

/** Close/shutdown a simulation's worker environment. */
export const closeSimulationEnv = (data) =>
  service({ url: '/api/simulation/close-env', method: 'post', data, timeout: 30000 })

/** Get the current env/worker health status. */
export const getEnvStatus = (data) =>
  service({ url: '/api/simulation/env-status', method: 'post', data, timeout: 10000 })

/** Batch-interview a set of simulation agents. */
export const interviewAgents = (data) =>
  requestWithRetry(() => service.post('/api/simulation/interview/batch', data), 2, 2000)

/** Get recent simulations (history view). */
export const getSimulationHistory = (limit = 20) =>
  service({ url: '/api/simulation/history', method: 'get', params: { limit }, timeout: 10000 })

/** Generate agent profiles for a specific graph (standalone — not tied to prepare). */
export const generateProfiles = (data) =>
  service({ url: '/api/simulation/generate-profiles', method: 'post', data, timeout: 60000 })
