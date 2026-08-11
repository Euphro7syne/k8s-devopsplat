import { request } from './http'

export interface HealthStatus {
  status: 'ok' | 'degraded'
  database: 'ok' | 'error'
  time: string
}

export function getHealthz() {
  return request<HealthStatus>({
    method: 'GET',
    url: '/healthz'
  })
}
