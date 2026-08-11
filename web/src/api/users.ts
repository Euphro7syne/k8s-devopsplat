import { request } from './http'

export interface UserRecord {
  id: number
  username: string
  email: string
  provider: string
  status: 'active' | 'disabled'
  created_at: string
  roles: string[]
}

export interface RoleRecord {
  id: number
  name: string
}

export interface CreateUserPayload {
  username: string
  email: string
  password: string
  roles: string[]
}

export function listUsers() {
  return request<UserRecord[]>({
    method: 'GET',
    url: '/users'
  })
}

export function listRoles() {
  return request<RoleRecord[]>({
    method: 'GET',
    url: '/roles'
  })
}

export function createUser(payload: CreateUserPayload) {
  return request<UserRecord>({
    method: 'POST',
    url: '/users',
    data: payload
  })
}

export function updateUserStatus(id: number, status: UserRecord['status']) {
  return request<UserRecord>({
    method: 'PUT',
    url: `/users/${id}/status`,
    data: { status }
  })
}

export function updateUserRoles(id: number, roles: string[]) {
  return request<UserRecord>({
    method: 'PUT',
    url: `/users/${id}/roles`,
    data: { roles }
  })
}
