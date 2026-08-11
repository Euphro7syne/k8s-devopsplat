import { request } from './http'

export interface Principal {
  user_id: number
  username: string
  email: string
  roles: string[]
  mfa_enabled: boolean
  mfa_verified: boolean
}

export interface LoginPayload {
  email: string
  password: string
}

export interface LoginResult {
  access_token?: string
  refresh_token?: string
  token_type?: string
  expires_in?: number
  user: Principal
  mfa_required: boolean
  mfa_setup_required: boolean
  mfa_token?: string
}

export interface MFASetupResult {
  secret: string
  provisioning_uri: string
  mfa_token: string
  expires_in: number
}

export interface MFAStatus {
  enabled: boolean
  required: boolean
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

export function setupMFA(mfaToken: string) {
  return request<MFASetupResult>({
    method: 'POST',
    url: '/auth/mfa/setup',
    data: { mfa_token: mfaToken }
  })
}

export function verifyMFA(mfaToken: string, code: string) {
  return request<LoginResult>({
    method: 'POST',
    url: '/auth/mfa/verify',
    data: { mfa_token: mfaToken, code }
  })
}

export function getMFAStatus() {
  return request<MFAStatus>({
    method: 'GET',
    url: '/auth/mfa/status'
  })
}

export function startMFAEnrollment() {
  return request<MFASetupResult>({
    method: 'POST',
    url: '/auth/mfa/enrollment'
  })
}

export function enableMFA(mfaToken: string, code: string) {
  return request<LoginResult>({
    method: 'POST',
    url: '/auth/mfa/enable',
    data: { mfa_token: mfaToken, code }
  })
}

export function disableMFA(password: string, code: string) {
  return request<LoginResult>({
    method: 'POST',
    url: '/auth/mfa/disable',
    data: { password, code }
  })
}
