<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Monitor, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const activeMenu = computed(() => String(route.name ?? 'dashboard'))
const menuItems = computed(() =>
  [
    { index: 'dashboard', label: '集群概览', roles: [] },
    { index: 'resources', label: '资源管理', roles: [] },
    { index: 'audit', label: '操作审计', roles: ['auditor', 'admin'] },
    { index: 'users', label: '用户管理', roles: ['admin'] }
  ].filter((item) => auth.hasAnyRole(item.roles))
)

function selectMenu(index: string) {
  if (index === 'dashboard') {
    router.push({ name: 'dashboard' })
    return
  }
  router.push({ name: index })
}

function logout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <el-container class="app-shell">
    <el-aside class="app-sidebar" width="232px">
      <div class="brand">
        <el-icon :size="22"><Monitor /></el-icon>
        <span>ops-platform</span>
      </div>
      <el-menu :default-active="activeMenu" class="nav-menu" @select="selectMenu">
        <el-menu-item v-for="item in menuItems" :key="item.index" :index="item.index">
          {{ item.label }}
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="app-header">
        <div class="cluster-select">
          <span class="label">集群</span>
          <el-select model-value="in-cluster" size="small" disabled>
            <el-option label="in-cluster" value="in-cluster" />
          </el-select>
        </div>
        <div class="header-user">
          <span>{{ auth.username || 'anonymous' }}</span>
          <el-button :icon="SwitchButton" circle @click="logout" />
        </div>
      </el-header>
      <el-main class="app-main">
        <slot />
      </el-main>
    </el-container>
  </el-container>
</template>
