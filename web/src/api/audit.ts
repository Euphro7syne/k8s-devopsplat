import { request } from './http'

export interface AuditLog {
  id: number
  user_id?: number
  action: string
  resource_type: string
  resource_name: string
  namespace: string
  request_body: string
  ip: string
  created_at: string
}

export function listAuditLogs(params: { namespace?: string; action?: string; resource?: string; limit?: number }) {
  return request<{ items: AuditLog[] }>({
    method: 'GET',
    url: '/audit/logs',
    params
  })
}
