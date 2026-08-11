<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { listAuditLogs, type AuditLog } from '../api/audit'
import AppShell from '../components/AppShell.vue'
import { formatDateTime } from '../utils/time'

const loading = ref(false)
const logs = ref<AuditLog[]>([])
const filters = reactive({
  namespace: '',
  action: '',
  resource: ''
})

async function loadLogs() {
  loading.value = true
  try {
    const result = await listAuditLogs({ ...filters, limit: 100 })
    logs.value = result.items
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '审计日志加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadLogs)
</script>

<template>
  <AppShell>
    <div class="page-head">
      <div>
        <h1>操作审计</h1>
        <p>最近 100 条</p>
      </div>
      <div class="toolbar">
        <el-input v-model="filters.namespace" placeholder="Namespace" clearable />
        <el-select v-model="filters.action" placeholder="Action" clearable>
          <el-option label="POST" value="POST" />
          <el-option label="PUT" value="PUT" />
          <el-option label="PATCH" value="PATCH" />
          <el-option label="DELETE" value="DELETE" />
        </el-select>
        <el-input v-model="filters.resource" placeholder="Resource" clearable />
        <el-button :icon="Refresh" :loading="loading" @click="loadLogs">刷新</el-button>
      </div>
    </div>

    <el-table v-loading="loading" :data="logs" border>
      <el-table-column prop="action" label="动作" width="100" />
      <el-table-column prop="namespace" label="Namespace" width="150" show-overflow-tooltip />
      <el-table-column prop="resource_name" label="资源" min-width="260" show-overflow-tooltip />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column label="时间" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column prop="request_body" label="请求体" min-width="260" show-overflow-tooltip />
    </el-table>
  </AppShell>
</template>
