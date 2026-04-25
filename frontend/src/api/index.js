import axios from 'axios'
import i18n from '../i18n'
import { getRuntimeApiBaseUrl } from '../composables/runtimeTarget'

const service = axios.create({
  baseURL: getRuntimeApiBaseUrl(),
  timeout: 300000,
})

service.interceptors.request.use(
  config => {
    config.baseURL = getRuntimeApiBaseUrl()
    config.headers['Accept-Language'] = i18n.global.locale.value
    // For FormData (file uploads) let the browser set Content-Type with the
    // correct multipart boundary. For everything else default to JSON.
    if (config.data instanceof FormData) {
      delete config.headers['Content-Type']
    } else if (!config.headers['Content-Type']) {
      config.headers['Content-Type'] = 'application/json'
    }
    return config
  },
  error => {
    console.error('Request error:', error)
    return Promise.reject(error)
  }
)

service.interceptors.response.use(
  response => {
    const res = response.data
    if (res && res.success === false) {
      const msg = res.error || res.message || 'Unknown API error'
      console.error('API Error:', msg)
      return Promise.reject(new Error(msg))
    }
    return res
  },
  error => {
    // Extract the server-side error message from the JSON body when available.
    const serverMsg =
      error.response?.data?.error ||
      error.response?.data?.message ||
      null

    if (serverMsg) {
      const wrapped = new Error(serverMsg)
      wrapped.status = error.response?.status
      wrapped.response = error.response
      console.error('API Error:', serverMsg)
      return Promise.reject(wrapped)
    }

    if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
      console.error('Request timeout')
    } else if (error.message === 'Network Error') {
      console.error('Network error — check gateway is running (make up)')
    } else {
      console.error('Response error:', error.message)
    }

    return Promise.reject(error)
  }
)

/**
 * Retry a request function up to maxRetries times.
 * Only retries on network errors, timeouts, and 5xx responses.
 * 4xx errors are not retried — they are client errors that won't fix themselves.
 */
export const requestWithRetry = async (requestFn, maxRetries = 3, delay = 1000) => {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await requestFn()
    } catch (error) {
      const status = error.status || error.response?.status
      // Do not retry client-side errors (4xx).
      if (status >= 400 && status < 500) throw error
      if (i === maxRetries - 1) throw error
      console.warn(`Request failed (${error.message}), retrying (${i + 1}/${maxRetries})…`)
      await new Promise(resolve => setTimeout(resolve, delay * Math.pow(2, i)))
    }
  }
}

export default service
