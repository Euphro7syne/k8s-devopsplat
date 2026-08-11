<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Monitor, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const activeMenu = computed(() => String(route.name ?? 'dashboard'))

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
        <el-menu-item index="dashboard">集群概览</el-menu-item>
        <el-menu-item index="resources">资源管理</el-menu-item>
        <el-menu-item index="audit">操作审计</el-menu-item>
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
