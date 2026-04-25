import service, { requestWithRetry } from './index'

/** Start async report generation. */
export const generateReport = (data) =>
  requestWithRetry(() => service.post('/api/report/generate', data), 2, 2000)

/**
 * Poll report generation status.
 * Accepts either report_id or simulation_id as a query param.
 * @param {{ report_id?: string, simulation_id?: string }} params
 */
export const getReportStatus = (params) =>
  service({ url: '/api/report/generate/status', method: 'get', params, timeout: 10000 })

/** Fetch a completed report by its ID. */
export const getReport = (reportId) =>
  service({ url: `/api/report/${reportId}`, method: 'get', timeout: 15000 })

/** Delete a report by its ID. */
export const deleteReport = (reportId) =>
  service({ url: `/api/report/${reportId}`, method: 'delete', timeout: 10000 })

/** Get report generation progress (section-level). */
export const getReportProgress = (reportId) =>
  service({ url: `/api/report/${reportId}/progress`, method: 'get', timeout: 10000 })

/** Get all completed report sections as an array. */
export const getReportSections = (reportId) =>
  service({ url: `/api/report/${reportId}/sections`, method: 'get', timeout: 15000 })

/** Get a single report section by index. */
export const getReportSection = (reportId, index) =>
  service({ url: `/api/report/${reportId}/section/${index}`, method: 'get', timeout: 10000 })

/** Download the full report as a Markdown file. */
export const downloadReport = (reportId) =>
  service({ url: `/api/report/${reportId}/download`, method: 'get', timeout: 30000 })

/** Get the LLM agent log for a report (polling). */
export const getAgentLog = (reportId, fromLine = 0) =>
  service({ url: `/api/report/${reportId}/agent-log`, method: 'get', params: { from_line: fromLine }, timeout: 10000 })

/** Get the console/execution log for a report (polling). */
export const getConsoleLog = (reportId, fromLine = 0) =>
  service({ url: `/api/report/${reportId}/console-log`, method: 'get', params: { from_line: fromLine }, timeout: 10000 })

/** Get all agent log lines at once (streaming snapshot). */
export const getAgentLogStream = (reportId) =>
  service({ url: `/api/report/${reportId}/agent-log/stream`, method: 'get', timeout: 10000 })

/** Get all console log lines at once (streaming snapshot). */
export const getConsoleLogStream = (reportId) =>
  service({ url: `/api/report/${reportId}/console-log/stream`, method: 'get', timeout: 10000 })

/** Chat with the report agent (RAG over the simulation graph). */
export const chatWithReport = (data) =>
  requestWithRetry(() => service.post('/api/report/chat', data), 2, 2000)

/** Get the report for a simulation (by-simulation lookup). */
export const getReportBySimulation = (simulationId) =>
  service({ url: `/api/report/by-simulation/${simulationId}`, method: 'get', timeout: 10000 })

/**
 * Check whether a report exists for a simulation (lightweight — no full payload).
 * Returns: { has_report, report_status, report_id, interview_unlocked }
 */
export const checkReportForSimulation = (simulationId) =>
  service({ url: `/api/report/check/${simulationId}`, method: 'get', timeout: 8000 })

/** List reports, optionally filtered by simulation_id. */
export const listReports = (simulationId = null, limit = 50) =>
  service({ url: '/api/report/list', method: 'get', params: simulationId ? { simulation_id: simulationId, limit } : { limit }, timeout: 10000 })

/** Graph search tool (used internally by report agent; also callable from UI). */
export const searchGraph = (graphId, query, limit = 10) =>
  service({ url: '/api/report/tools/search', method: 'post', data: { graph_id: graphId, query, limit }, timeout: 15000 })

/** Graph statistics tool. */
export const getGraphStatistics = (graphId) =>
  service({ url: '/api/report/tools/statistics', method: 'post', data: { graph_id: graphId }, timeout: 10000 })
