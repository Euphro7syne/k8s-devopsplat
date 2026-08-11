import axios, { isAxiosError, type AxiosRequestConfig } from 'axios'

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('ops-token')
  if (token) {
    config.headers = config.headers ?? {}
    ;(config.headers as Record<string, string>).Authorization = `Bearer ${token}`
  }
  return config
})

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  try {
    const response = await client.request<ApiEnvelope<T>>(config)
    const body = response.data
    if (body.code !== 0) {
      throw new Error(body.message || 'request failed')
    }
    return body.data
  } catch (error) {
    if (isAxiosError<ApiEnvelope<unknown>>(error)) {
      if (shouldClearSession(error.response?.status, config.url)) {
        clearStoredSession()
        if (window.location.pathname !== '/login') {
          window.location.replace('/login')
        }
      }
      throw new Error(error.response?.data?.message || error.message || 'request failed')
    }
    throw error
  }
}

function shouldClearSession(status: number | undefined, url: string | undefined) {
  if (status !== 401 || !localStorage.getItem('ops-token')) return false
  return !['/auth/mfa/verify', '/auth/mfa/enable', '/auth/mfa/disable'].includes(url ?? '')
}

function clearStoredSession() {
  for (const key of [
    'ops-user-id',
    'ops-token',
    'ops-refresh-token',
    'ops-username',
    'ops-email',
    'ops-roles',
    'ops-mfa-enabled',
    'ops-mfa-verified'
  ]) {
    localStorage.removeItem(key)
  }
}

export default client
