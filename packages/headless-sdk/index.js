import axios from 'axios'

function isFormDataLike(value) {
  return typeof FormData !== 'undefined' && value instanceof FormData
}

export async function requestWithRetry(requestFn, maxRetries = 3, delayMs = 1000) {
  for (let i = 0; i < maxRetries; i += 1) {
    try {
      return await requestFn()
    } catch (error) {
      const status = error.status || error.response?.status
      if (status >= 400 && status < 500) {
        throw error
      }
      if (i === maxRetries - 1) {
        throw error
      }
      await new Promise((resolve) => setTimeout(resolve, delayMs * Math.pow(2, i)))
    }
  }
  throw new Error('requestWithRetry exhausted without a terminal response')
}

export function createTransport({
  baseURL,
  timeout = 300000,
  acceptLanguage = 'en',
  headers = {},
} = {}) {
  if (!baseURL || typeof baseURL !== 'string') {
    throw new Error('baseURL is required')
  }

  const service = axios.create({
    baseURL,
    timeout,
    headers: { ...headers },
  })

  service.interceptors.request.use(
    (config) => {
      config.baseURL = baseURL
      config.headers = config.headers || {}
      if (acceptLanguage) {
        config.headers['Accept-Language'] = acceptLanguage
      }
      if (isFormDataLike(config.data)) {
        delete config.headers['Content-Type']
      } else if (!config.headers['Content-Type']) {
        config.headers['Content-Type'] = 'application/json'
      }
      return config
    },
    (error) => Promise.reject(error)
  )

  service.interceptors.response.use(
    (response) => {
      const res = response.data
      if (res && res.success === false) {
        const msg = res.error || res.message || 'Unknown API error'
        return Promise.reject(new Error(msg))
      }
      return res
    },
    (error) => {
      const serverMsg =
        error.response?.data?.error ||
        error.response?.data?.message ||
        null

      if (serverMsg) {
        const wrapped = new Error(serverMsg)
        wrapped.status = error.response?.status
        wrapped.response = error.response
        return Promise.reject(wrapped)
      }

      return Promise.reject(error)
    }
  )

  return service
}

export function createHeadlessSDK(options = {}) {
  const service = createTransport(options)

  const graph = {
    generateOntology(formData) {
      return requestWithRetry(
        () =>
          service({
            url: '/api/graph/ontology/generate',
            method: 'post',
            data: formData,
            timeout: 600000,
          }),
        2,
        2000
      )
    },
    buildGraph(data) {
      return requestWithRetry(() => service({ url: '/api/graph/build', method: 'post', data }))
    },
    getTaskStatus(taskId) {
      return service({ url: `/api/graph/task/${taskId}`, method: 'get', timeout: 10000 })
    },
    getGraphData(graphId) {
      return service({ url: `/api/graph/data/${graphId}`, method: 'get', timeout: 30000 })
    },
    getProject(projectId) {
      return service({ url: `/api/graph/project/${projectId}`, method: 'get', timeout: 10000 })
    },
    listProjects(limit = 20) {
      return service({ url: '/api/graph/project/list', method: 'get', params: { limit }, timeout: 10000 })
    },
    deleteProject(projectId) {
      return service({ url: `/api/graph/project/${projectId}`, method: 'delete', timeout: 10000 })
    },
  }

  const simulation = {
    createSimulation(data) {
      return requestWithRetry(() => service.post('/api/simulation/create', data), 3, 1000)
    },
    prepareSimulation(data) {
      return requestWithRetry(() => service.post('/api/simulation/prepare', data), 2, 2000)
    },
    getPrepareStatus(data) {
      return service({ url: '/api/simulation/prepare/status', method: 'post', data, timeout: 10000 })
    },
    getSimulation(simulationId) {
      return service({ url: `/api/simulation/${simulationId}`, method: 'get', timeout: 10000 })
    },
    getSimulationProfiles(simulationId, platform = 'reddit') {
      return service({
        url: `/api/simulation/${simulationId}/profiles`,
        method: 'get',
        params: { platform },
        timeout: 15000,
      })
    },
    getSimulationProfilesRealtime(simulationId, platform = 'reddit') {
      return service({
        url: `/api/simulation/${simulationId}/profiles/realtime`,
        method: 'get',
        params: { platform },
        timeout: 15000,
      })
    },
    getSimulationConfig(simulationId) {
      return service({ url: `/api/simulation/${simulationId}/config`, method: 'get', timeout: 10000 })
    },
    getSimulationConfigRealtime(simulationId) {
      return service({
        url: `/api/simulation/${simulationId}/config/realtime`,
        method: 'get',
        timeout: 10000,
      })
    },
    listSimulations(projectId) {
      return service({
        url: '/api/simulation/list',
        method: 'get',
        params: projectId ? { project_id: projectId } : {},
        timeout: 10000,
      })
    },
    startSimulation(data) {
      return requestWithRetry(() => service.post('/api/simulation/start', data), 2, 2000)
    },
    stopSimulation(data) {
      return service({ url: '/api/simulation/stop', method: 'post', data, timeout: 15000 })
    },
    deleteSimulation(data) {
      return service({ url: '/api/simulation/delete', method: 'post', data, timeout: 15000 })
    },
    getRunStatus(simulationId) {
      return service({ url: `/api/simulation/${simulationId}/run-status`, method: 'get', timeout: 8000 })
    },
    getRunStatusDetail(simulationId) {
      return service({ url: `/api/simulation/${simulationId}/run-status/detail`, method: 'get', timeout: 10000 })
    },
    getSimulationPosts(simulationId, platform = 'reddit', limit = 50, offset = 0) {
      return service({
        url: `/api/simulation/${simulationId}/posts`,
        method: 'get',
        params: { platform, limit, offset },
        timeout: 15000,
      })
    },
    getSimulationComments(simulationId, platform = 'reddit', limit = 50, offset = 0) {
      return service({
        url: `/api/simulation/${simulationId}/comments`,
        method: 'get',
        params: { platform, limit, offset },
        timeout: 15000,
      })
    },
    getSimulationTimeline(simulationId, startRound = 0, endRound = null) {
      const params = { start_round: startRound }
      if (endRound !== null) {
        params.end_round = endRound
      }
      return service({ url: `/api/simulation/${simulationId}/timeline`, method: 'get', params, timeout: 15000 })
    },
    getAgentStats(simulationId) {
      return service({ url: `/api/simulation/${simulationId}/agent-stats`, method: 'get', timeout: 15000 })
    },
    getSimulationActions(simulationId, params = {}) {
      return service({ url: `/api/simulation/${simulationId}/actions`, method: 'get', params, timeout: 15000 })
    },
    closeSimulationEnv(data) {
      return service({ url: '/api/simulation/close-env', method: 'post', data, timeout: 30000 })
    },
    getEnvStatus(data) {
      return service({ url: '/api/simulation/env-status', method: 'post', data, timeout: 10000 })
    },
    interviewAgents(data) {
      return requestWithRetry(() => service.post('/api/simulation/interview/batch', data), 2, 2000)
    },
    getSimulationHistory(limit = 20) {
      return service({ url: '/api/simulation/history', method: 'get', params: { limit }, timeout: 10000 })
    },
    generateProfiles(data) {
      return service({ url: '/api/simulation/generate-profiles', method: 'post', data, timeout: 60000 })
    },
  }

  const report = {
    generateReport(data) {
      return requestWithRetry(() => service.post('/api/report/generate', data), 2, 2000)
    },
    getReportStatus(params) {
      return service({ url: '/api/report/generate/status', method: 'get', params, timeout: 10000 })
    },
    getReport(reportId) {
      return service({ url: `/api/report/${reportId}`, method: 'get', timeout: 15000 })
    },
    deleteReport(reportId) {
      return service({ url: `/api/report/${reportId}`, method: 'delete', timeout: 10000 })
    },
    getReportProgress(reportId) {
      return service({ url: `/api/report/${reportId}/progress`, method: 'get', timeout: 10000 })
    },
    getReportSections(reportId) {
      return service({ url: `/api/report/${reportId}/sections`, method: 'get', timeout: 15000 })
    },
    getReportSection(reportId, index) {
      return service({ url: `/api/report/${reportId}/section/${index}`, method: 'get', timeout: 10000 })
    },
    downloadReport(reportId) {
      return service({ url: `/api/report/${reportId}/download`, method: 'get', timeout: 30000 })
    },
    getAgentLog(reportId, fromLine = 0) {
      return service({
        url: `/api/report/${reportId}/agent-log`,
        method: 'get',
        params: { from_line: fromLine },
        timeout: 10000,
      })
    },
    getConsoleLog(reportId, fromLine = 0) {
      return service({
        url: `/api/report/${reportId}/console-log`,
        method: 'get',
        params: { from_line: fromLine },
        timeout: 10000,
      })
    },
    getAgentLogStream(reportId) {
      return service({ url: `/api/report/${reportId}/agent-log/stream`, method: 'get', timeout: 10000 })
    },
    getConsoleLogStream(reportId) {
      return service({ url: `/api/report/${reportId}/console-log/stream`, method: 'get', timeout: 10000 })
    },
    chatWithReport(data) {
      return requestWithRetry(() => service.post('/api/report/chat', data), 2, 2000)
    },
    getReportBySimulation(simulationId) {
      return service({ url: `/api/report/by-simulation/${simulationId}`, method: 'get', timeout: 10000 })
    },
    checkReportForSimulation(simulationId) {
      return service({ url: `/api/report/check/${simulationId}`, method: 'get', timeout: 8000 })
    },
    listReports(simulationId = null, limit = 50) {
      return service({
        url: '/api/report/list',
        method: 'get',
        params: simulationId ? { simulation_id: simulationId, limit } : { limit },
        timeout: 10000,
      })
    },
    searchGraph(graphId, query, limit = 10) {
      return service({
        url: '/api/report/tools/search',
        method: 'post',
        data: { graph_id: graphId, query, limit },
        timeout: 15000,
      })
    },
    getGraphStatistics(graphId) {
      return service({
        url: '/api/report/tools/statistics',
        method: 'post',
        data: { graph_id: graphId },
        timeout: 10000,
      })
    },
  }

  const system = {
    getHealth() {
      return service({ url: '/health', method: 'get', timeout: 5000 })
    },
    getReady() {
      return service({ url: '/ready', method: 'get', timeout: 5000 })
    },
    getMetrics() {
      return service({ url: '/metrics', method: 'get', timeout: 10000 })
    },
    getProviders() {
      return service({ url: '/api/providers', method: 'get', timeout: 10000 })
    },
  }

  return {
    raw: service,
    graph,
    simulation,
    report,
    system,
  }
}

export default createHeadlessSDK
