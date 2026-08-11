import { defineStore } from 'pinia'

import { login as loginApi, profile, type Principal } from '../api/auth'

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
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    userID: 0,
    token: '',
    refreshToken: '',
    username: '',
    email: '',
    roles: []
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
      }
    },
    async login(form: LoginForm) {
      const result = await loginApi(form)
      this.applySession(result.access_token, result.refresh_token, result.user)
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
      localStorage.removeItem('ops-user-id')
      localStorage.removeItem('ops-token')
      localStorage.removeItem('ops-refresh-token')
      localStorage.removeItem('ops-username')
      localStorage.removeItem('ops-email')
      localStorage.removeItem('ops-roles')
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
      localStorage.setItem('ops-user-id', String(user.user_id))
      localStorage.setItem('ops-username', user.username)
      localStorage.setItem('ops-email', user.email)
      localStorage.setItem('ops-roles', JSON.stringify(user.roles))
    }
  }
})
