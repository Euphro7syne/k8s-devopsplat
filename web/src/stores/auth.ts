import { defineStore } from 'pinia'

import {
  enableMFA as enableMFAApi,
  login as loginApi,
  profile,
  verifyMFA as verifyMFAApi,
  type LoginResult,
  type Principal
} from '../api/auth'

interface LoginForm {
  email: string
  password: string
}

interface AuthState {
  userID: number
  token: string
  refreshToken: string
  username: string
  email: string
  roles: string[]
  mfaEnabled: boolean
  mfaVerified: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    userID: 0,
    token: '',
    refreshToken: '',
    username: '',
    email: '',
    roles: [],
    mfaEnabled: false,
    mfaVerified: false
  }),
  getters: {
    isAuthenticated: (state) => state.token.length > 0,
    hasAnyRole: (state) => (roles: string[]) => {
      if (roles.length === 0) return true
      if (state.roles.includes('admin')) return true
      return roles.some((role) => state.roles.includes(role))
    }
  },
  actions: {
    bootstrap() {
      if (!this.token) {
        this.userID = Number(localStorage.getItem('ops-user-id') ?? '0')
        this.token = localStorage.getItem('ops-token') ?? ''
        this.refreshToken = localStorage.getItem('ops-refresh-token') ?? ''
        this.username = localStorage.getItem('ops-username') ?? ''
        this.email = localStorage.getItem('ops-email') ?? ''
        this.roles = JSON.parse(localStorage.getItem('ops-roles') ?? '[]') as string[]
        this.mfaEnabled = localStorage.getItem('ops-mfa-enabled') === 'true'
        this.mfaVerified = localStorage.getItem('ops-mfa-verified') === 'true'
      }
    },
    async login(form: LoginForm) {
      const result = await loginApi(form)
      if (!result.mfa_required) {
        this.applyLoginResult(result)
      }
      return result
    },
    async verifyMFA(mfaToken: string, code: string) {
      const result = await verifyMFAApi(mfaToken, code)
      this.applyLoginResult(result)
      return result
    },
    async enableMFA(mfaToken: string, code: string) {
      const result = await enableMFAApi(mfaToken, code)
      this.applyLoginResult(result)
      return result
    },
    async loadProfile() {
      const principal = await profile()
      this.applyUser(principal)
    },
    logout() {
      this.userID = 0
      this.token = ''
      this.refreshToken = ''
      this.username = ''
      this.email = ''
      this.roles = []
      this.mfaEnabled = false
      this.mfaVerified = false
      localStorage.removeItem('ops-user-id')
      localStorage.removeItem('ops-token')
      localStorage.removeItem('ops-refresh-token')
      localStorage.removeItem('ops-username')
      localStorage.removeItem('ops-email')
      localStorage.removeItem('ops-roles')
      localStorage.removeItem('ops-mfa-enabled')
      localStorage.removeItem('ops-mfa-verified')
    },
    applyLoginResult(result: LoginResult) {
      if (!result.access_token || !result.refresh_token) {
        throw new Error('登录会话不完整')
      }
      this.applySession(result.access_token, result.refresh_token, result.user)
    },
    applySession(token: string, refreshToken: string, user: Principal) {
      this.token = token
      this.refreshToken = refreshToken
      localStorage.setItem('ops-token', token)
      localStorage.setItem('ops-refresh-token', refreshToken)
      this.applyUser(user)
    },
    applyUser(user: Principal) {
      this.userID = user.user_id
      this.username = user.username
      this.email = user.email
      this.roles = user.roles
      this.mfaEnabled = user.mfa_enabled
      this.mfaVerified = user.mfa_verified
      localStorage.setItem('ops-user-id', String(user.user_id))
      localStorage.setItem('ops-username', user.username)
      localStorage.setItem('ops-email', user.email)
      localStorage.setItem('ops-roles', JSON.stringify(user.roles))
      localStorage.setItem('ops-mfa-enabled', String(user.mfa_enabled))
      localStorage.setItem('ops-mfa-verified', String(user.mfa_verified))
    }
  }
})
