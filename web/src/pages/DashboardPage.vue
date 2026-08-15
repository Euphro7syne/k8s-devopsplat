<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { getHealthz, type HealthStatus } from '../api/health'
import { getOverview, type ClusterOverview } from '../api/resources'
import AppShell from '../components/AppShell.vue'
import { formatDateTime } from '../utils/time'

const health = ref<HealthStatus | null>(null)
const overview = ref<ClusterOverview | null>(null)
const loading = ref(false)

async function loadDashboard() {
  loading.value = true
  try {
    const [healthResult, overviewResult] = await Promise.all([getHealthz(), getOverview()])
    health.value = healthResult
    overview.value = overviewResult
  } catch (error) {
    const message = error instanceof Error ? error.message : '健康检查失败'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <AppShell>
    <div class="page-head">
      <div>
        <h1>集群概览</h1>
        <p>{{ overview?.cluster ?? 'in-cluster' }} / ops-platform</p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadDashboard">刷新</el-button>
    </div>

    <section class="status-grid">
      <el-card shadow="never">
        <template #header>API 状态</template>
        <div class="metric">
          <span class="metric-value">{{ health?.status ?? '-' }}</span>
          <el-tag :type="health?.status === 'ok' ? 'success' : 'warning'">
            {{ health?.status === 'ok' ? '正常' : '待检查' }}
          </el-tag>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>数据库</template>
        <div class="metric">
          <span class="metric-value">{{ health?.database ?? '-' }}</span>
          <el-tag :type="health?.database === 'ok' ? 'success' : 'warning'">
            {{ health?.database === 'ok' ? '连通' : '异常' }}
          </el-tag>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>Kubernetes API</template>
        <div class="metric">
          <span class="metric-value">{{ health?.kubernetes ?? '-' }}</span>
          <el-tag :type="health?.kubernetes === 'configured' ? 'success' : 'danger'">
            {{ health?.kubernetes === 'configured' ? '已连接' : '不可用' }}
          </el-tag>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>资源缓存</template>
        <div class="metric">
          <span class="metric-value">{{ health?.resource_cache ?? '-' }}</span>
          <el-tag :type="health?.resource_cache === 'ready' ? 'success' : 'warning'">
            {{ health?.resource_cache === 'ready' ? '已就绪' : 'API 回退' }}
          </el-tag>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>Namespace</template>
        <div class="metric">
          <span class="metric-value">{{ overview?.namespace_count ?? '-' }}</span>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>节点</template>
        <div class="metric">
          <span class="metric-value">{{ overview?.ready_node_count ?? '-' }}/{{ overview?.node_count ?? '-' }}</span>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>Pod</template>
        <div class="metric">
          <span class="metric-value">{{ overview?.pod_count ?? '-' }}</span>
          <el-tag :type="overview?.abnormal_pod_count === 0 ? 'success' : 'danger'">
            异常 {{ overview?.abnormal_pod_count ?? '-' }}
          </el-tag>
        </div>
      </el-card>
      <el-card shadow="never">
        <template #header>检查时间</template>
        <div class="metric">
          <span class="metric-value time">{{ formatDateTime(health?.time ?? '') || '-' }}</span>
        </div>
      </el-card>
    </section>

    <section class="table-section">
      <el-table :data="overview?.nodes ?? []" border>
        <el-table-column prop="name" label="节点" min-width="180" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="cpu_allocatable" label="CPU" width="120" />
        <el-table-column prop="memory_allocatable" label="Memory" width="140" />
        <el-table-column prop="pods_allocatable" label="Pods" width="120" />
      </el-table>
    </section>
  </AppShell>
</template>
