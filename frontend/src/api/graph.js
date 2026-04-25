import service, { requestWithRetry } from './index'

/**
 * Upload seed documents and generate the project ontology via the Go gateway.
 * Must use FormData — do NOT set Content-Type manually (multipart boundary is set by the browser).
 * @param {FormData} formData - fields: project_name, simulation_requirement, files[]
 */
export function generateOntology(formData) {
  return requestWithRetry(
    () =>
      service({
        url: '/api/graph/ontology/generate',
        method: 'post',
        data: formData,
        timeout: 600000, // 10 min — LLM ontology generation can be slow
      }),
    2,   // max retries (not 4xx)
    2000 // base delay ms
  )
}

/**
 * Start an async graph-build task for a project.
 * @param {{ project_id: string, chunk_size?: number, chunk_overlap?: number }} data
 */
export function buildGraph(data) {
  return requestWithRetry(() =>
    service({
      url: '/api/graph/build',
      method: 'post',
      data,
    })
  )
}

/**
 * Poll a single graph/build task by ID.
 * @param {string} taskId
 */
export function getTaskStatus(taskId) {
  return service({
    url: `/api/graph/task/${taskId}`,
    method: 'get',
    timeout: 10000,
  })
}

/**
 * Fetch the full node/edge graph data for a graph ID (from Zep).
 * @param {string} graphId
 */
export function getGraphData(graphId) {
  return service({
    url: `/api/graph/data/${graphId}`,
    method: 'get',
    timeout: 30000,
  })
}

/**
 * Get a single project record (status, ontology, graph_id, etc.).
 * @param {string} projectId
 */
export function getProject(projectId) {
  return service({
    url: `/api/graph/project/${projectId}`,
    method: 'get',
    timeout: 10000,
  })
}

/**
 * List recent projects.
 * @param {number} [limit=20]
 */
export function listProjects(limit = 20) {
  return service({
    url: '/api/graph/project/list',
    method: 'get',
    params: { limit },
    timeout: 10000,
  })
}

/**
 * Delete a project and its associated artifacts.
 * @param {string} projectId
 */
export function deleteProject(projectId) {
  return service({
    url: `/api/graph/project/${projectId}`,
    method: 'delete',
    timeout: 10000,
  })
}
