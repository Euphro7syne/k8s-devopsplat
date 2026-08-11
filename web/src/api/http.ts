import axios, { type AxiosRequestConfig } from 'axios'

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
  const response = await client.request<ApiEnvelope<T>>(config)
  const body = response.data
  if (body.code !== 0) {
    throw new Error(body.message || 'request failed')
  }
  return body.data
}

export default client
