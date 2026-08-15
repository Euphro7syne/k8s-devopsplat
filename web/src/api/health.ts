import { request } from './http'

export interface HealthStatus {
  status: 'ok' | 'degraded'
  database: 'ok' | 'error'
  kubernetes: 'configured' | 'unavailable'
  resource_cache: 'ready' | 'not_ready' | 'disabled' | 'unavailable'
  time: string
}

export function getHealthz() {
  return request<HealthStatus>({
    method: 'GET',
    url: '/healthz'
  })
}
