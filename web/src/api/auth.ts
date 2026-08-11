import { request } from './http'

export interface Principal {
  user_id: number
  username: string
  email: string
  roles: string[]
}

export interface LoginPayload {
  email: string
  password: string
}

export interface LoginResult {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  user: Principal
}

export function login(payload: LoginPayload) {
  return request<LoginResult>({
    method: 'POST',
    url: '/auth/login',
    data: payload
  })
}

export function profile() {
  return request<Principal>({
    method: 'GET',
    url: '/auth/profile'
  })
}
