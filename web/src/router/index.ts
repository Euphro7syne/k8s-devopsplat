import { createRouter, createWebHistory } from 'vue-router'

import DashboardPage from '../pages/DashboardPage.vue'
import AuditPage from '../pages/AuditPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import ResourcesPage from '../pages/ResourcesPage.vue'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { public: true }
    },
    {
      path: '/',
      name: 'dashboard',
      component: DashboardPage
    },
    {
      path: '/resources',
      name: 'resources',
      component: ResourcesPage
    },
    {
      path: '/audit',
      name: 'audit',
      component: AuditPage
    }
  ]
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  auth.bootstrap()
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
