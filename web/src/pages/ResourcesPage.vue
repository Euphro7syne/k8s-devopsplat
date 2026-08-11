<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Check, Delete, Refresh, Tickets, VideoPlay, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  deletePod,
  getPodLogs,
  getResourceYAML,
  listConfigMaps,
  listCronJobs,
  listDaemonSets,
  listDeployments,
  listIngresses,
  listJobs,
  listNamespaces,
  listNodes,
  listPods,
  listPVCs,
  listPVs,
  listReplicaSets,
  listServices,
  listStatefulSets,
  listStorageClasses,
  restartDeployment,
  scaleDeployment,
  updateResourceYAML,
  type ConfigMapSummary,
  type CronJobSummary,
  type DaemonSetSummary,
  type DeploymentSummary,
  type IngressSummary,
  type JobSummary,
  type LogResult,
  type NamespaceSummary,
  type NodeSummary,
  type PageResult,
  type PodSummary,
  type PVCSummary,
  type PVSummary,
  type ResourceKind,
  type ServiceSummary,
  type StorageClassSummary,
  type WorkloadSummary
} from '../api/resources'
import AppShell from '../components/AppShell.vue'
import { formatDateTime } from '../utils/time'

type ResourceGroup = 'workloads' | 'pods' | 'network' | 'config' | 'storage' | 'nodes'
type WorkloadKind = 'deployments' | 'statefulsets' | 'daemonsets' | 'replicasets' | 'jobs' | 'cronjobs'
type NetworkKind = 'services' | 'ingresses'
type StorageKind = 'pvcs' | 'pvs' | 'storageclasses'

const pageSize = 100

const namespaces = ref<NamespaceSummary[]>([])
const namespace = ref('')
const initialized = ref(false)
const activeGroup = ref<ResourceGroup>('workloads')
const activeWorkload = ref<WorkloadKind>('deployments')
const activeNetwork = ref<NetworkKind>('services')
const activeStorage = ref<StorageKind>('pvcs')

const nodes = ref<PageResult<NodeSummary> | null>(null)
const pods = ref<PageResult<PodSummary> | null>(null)
const deployments = ref<PageResult<DeploymentSummary> | null>(null)
const statefulSets = ref<PageResult<WorkloadSummary> | null>(null)
const daemonSets = ref<PageResult<DaemonSetSummary> | null>(null)
const replicaSets = ref<PageResult<WorkloadSummary> | null>(null)
const jobs = ref<PageResult<JobSummary> | null>(null)
const cronJobs = ref<PageResult<CronJobSummary> | null>(null)
const services = ref<PageResult<ServiceSummary> | null>(null)
const ingresses = ref<PageResult<IngressSummary> | null>(null)
const configMaps = ref<PageResult<ConfigMapSummary> | null>(null)
const pvcs = ref<PageResult<PVCSummary> | null>(null)
const pvs = ref<PageResult<PVSummary> | null>(null)
const storageClasses = ref<PageResult<StorageClassSummary> | null>(null)

const loading = ref(false)
const logsLoading = ref(false)
const logsVisible = ref(false)
const yamlVisible = ref(false)
const yamlText = ref('')
const yamlTitle = ref('YAML')
const yamlSaving = ref(false)
const yamlKind = ref<ResourceKind | ''>('')
const yamlName = ref('')
const yamlNamespace = ref('')
const selectedPod = ref<PodSummary | null>(null)
const logResult = ref<LogResult | null>(null)
const logQuery = reactive({
  container: '',
  keyword: '',
  level: '',
  previous: false,
  limit: 300
})

const groupOptions = [
  { label: 'Workloads', name: 'workloads' },
  { label: 'Pods', name: 'pods' },
  { label: 'Network', name: 'network' },
  { label: 'Config', name: 'config' },
  { label: 'Storage', name: 'storage' },
  { label: 'Nodes', name: 'nodes' }
]
const workloadOptions = [
  { label: 'Deployments', value: 'deployments' },
  { label: 'StatefulSets', value: 'statefulsets' },
  { label: 'DaemonSets', value: 'daemonsets' },
  { label: 'ReplicaSets', value: 'replicasets' },
  { label: 'Jobs', value: 'jobs' },
  { label: 'CronJobs', value: 'cronjobs' }
]
const networkOptions = [
  { label: 'Services', value: 'services' },
  { label: 'Ingresses', value: 'ingresses' }
]
const storageOptions = [
  { label: 'PVC', value: 'pvcs' },
  { label: 'PV', value: 'pvs' },
  { label: 'StorageClass', value: 'storageclasses' }
]

const needsNamespace = computed(() => {
  if (activeGroup.value === 'nodes') {
    return false
  }
  if (activeGroup.value === 'storage' && activeStorage.value !== 'pvcs') {
    return false
  }
  return true
})
const scopeLabel = computed(() => (needsNamespace.value ? namespace.value || '-' : 'cluster-scoped'))
const podRows = computed(() => pods.value?.items ?? [])
const deploymentRows = computed(() => deployments.value?.items ?? [])
const statefulSetRows = computed(() => statefulSets.value?.items ?? [])
const daemonSetRows = computed(() => daemonSets.value?.items ?? [])
const replicaSetRows = computed(() => replicaSets.value?.items ?? [])
const jobRows = computed(() => jobs.value?.items ?? [])
const cronJobRows = computed(() => cronJobs.value?.items ?? [])
const serviceRows = computed(() => services.value?.items ?? [])
const ingressRows = computed(() => ingresses.value?.items ?? [])
const configMapRows = computed(() => configMaps.value?.items ?? [])
const pvcRows = computed(() => pvcs.value?.items ?? [])
const pvRows = computed(() => pvs.value?.items ?? [])
const storageClassRows = computed(() => storageClasses.value?.items ?? [])
const nodeRows = computed(() => nodes.value?.items ?? [])
const editableYAMLKinds = new Set<ResourceKind>([
  'deployment',
  'statefulset',
  'daemonset',
  'job',
  'cronjob',
  'service',
  'ingress'
])
const yamlEditable = computed(() => yamlKind.value !== '' && editableYAMLKinds.has(yamlKind.value))
const activeTotal = computed(() => {
  if (activeGroup.value === 'pods') return pods.value?.total ?? 0
  if (activeGroup.value === 'nodes') return nodes.value?.total ?? 0
  if (activeGroup.value === 'network') return activeNetwork.value === 'services' ? services.value?.total ?? 0 : ingresses.value?.total ?? 0
  if (activeGroup.value === 'config') return configMaps.value?.total ?? 0
  if (activeGroup.value === 'storage') {
    if (activeStorage.value === 'pvcs') return pvcs.value?.total ?? 0
    if (activeStorage.value === 'pvs') return pvs.value?.total ?? 0
    return storageClasses.value?.total ?? 0
  }
  if (activeWorkload.value === 'deployments') return deployments.value?.total ?? 0
  if (activeWorkload.value === 'statefulsets') return statefulSets.value?.total ?? 0
  if (activeWorkload.value === 'daemonsets') return daemonSets.value?.total ?? 0
  if (activeWorkload.value === 'replicasets') return replicaSets.value?.total ?? 0
  if (activeWorkload.value === 'jobs') return jobs.value?.total ?? 0
  return cronJobs.value?.total ?? 0
})

function containerNames(row: PodSummary) {
  return listText(row.containers.map((item) => item.name))
}

function listText(items?: string[] | null) {
  return items?.length ? items.join(', ') : '-'
}

function mapText(value?: Record<string, string> | null) {
  if (!value || Object.keys(value).length === 0) {
    return '-'
  }
  return Object.entries(value)
    .slice(0, 6)
    .map(([key, item]) => `${key}=${item}`)
    .join(', ')
}

function statusType(status: string) {
  const normalized = status.toLowerCase()
  if (['ready', 'running', 'bound', 'active'].includes(normalized)) return 'success'
  if (['pending', 'terminating', 'unknown'].includes(normalized)) return 'warning'
  if (['failed', 'notready'].includes(normalized)) return 'danger'
  return 'info'
}

function workloadUnavailable(row: DeploymentSummary | WorkloadSummary) {
  return Math.max(row.unavailable_replicas, 0)
}

async function loadNamespaces() {
  const result = await listNamespaces()
  namespaces.value = result
  if (!namespace.value || !result.some((item) => item.name === namespace.value)) {
    namespace.value = result[0]?.name ?? ''
  }
}

async function loadCurrent() {
  if (needsNamespace.value && !namespace.value) {
    return
  }
  loading.value = true
  try {
    const ns = namespace.value
    if (activeGroup.value === 'pods') {
      pods.value = await listPods(ns, 1, pageSize)
    } else if (activeGroup.value === 'nodes') {
      nodes.value = await listNodes(1, pageSize)
    } else if (activeGroup.value === 'network') {
      if (activeNetwork.value === 'services') {
        services.value = await listServices(ns, 1, pageSize)
      } else {
        ingresses.value = await listIngresses(ns, 1, pageSize)
      }
    } else if (activeGroup.value === 'config') {
      configMaps.value = await listConfigMaps(ns, 1, pageSize)
    } else if (activeGroup.value === 'storage') {
      if (activeStorage.value === 'pvcs') {
        pvcs.value = await listPVCs(ns, 1, pageSize)
      } else if (activeStorage.value === 'pvs') {
        pvs.value = await listPVs(1, pageSize)
      } else {
        storageClasses.value = await listStorageClasses(1, pageSize)
      }
    } else if (activeWorkload.value === 'deployments') {
      deployments.value = await listDeployments(ns, 1, pageSize)
    } else if (activeWorkload.value === 'statefulsets') {
      statefulSets.value = await listStatefulSets(ns, 1, pageSize)
    } else if (activeWorkload.value === 'daemonsets') {
      daemonSets.value = await listDaemonSets(ns, 1, pageSize)
    } else if (activeWorkload.value === 'replicasets') {
      replicaSets.value = await listReplicaSets(ns, 1, pageSize)
    } else if (activeWorkload.value === 'jobs') {
      jobs.value = await listJobs(ns, 1, pageSize)
    } else {
      cronJobs.value = await listCronJobs(ns, 1, pageSize)
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源加载失败')
  } finally {
    loading.value = false
  }
}

async function refreshAll() {
  const previousNamespace = namespace.value
  try {
    await loadNamespaces()
    if (!initialized.value || previousNamespace === namespace.value) {
      await loadCurrent()
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源加载失败')
  }
}

async function openLogs(row: PodSummary) {
  selectedPod.value = row
  logQuery.container = row.containers[0]?.name ?? ''
  logsVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  if (!selectedPod.value) {
    return
  }
  logsLoading.value = true
  try {
    logResult.value = await getPodLogs(selectedPod.value.namespace, selectedPod.value.name, logQuery)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '日志加载失败')
  } finally {
    logsLoading.value = false
  }
}

async function showYAML(kind: ResourceKind, name: string, rowNamespace?: string) {
  try {
    const result = await getResourceYAML(kind, name, rowNamespace)
    yamlTitle.value = rowNamespace ? `${kind}/${rowNamespace}/${name}` : `${kind}/${name}`
    yamlKind.value = kind
    yamlName.value = name
    yamlNamespace.value = rowNamespace ?? ''
    yamlText.value = result.yaml
    yamlVisible.value = true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'YAML 加载失败')
  }
}

async function reloadYAML() {
  if (!yamlKind.value || !yamlName.value) {
    return
  }
  await showYAML(yamlKind.value, yamlName.value, yamlNamespace.value || undefined)
}

async function saveYAML() {
  if (!yamlEditable.value || !yamlKind.value || !yamlName.value || !yamlNamespace.value) {
    return
  }
  await ElMessageBox.confirm(`保存 ${yamlTitle.value}`, '二次确认', {
    confirmButtonText: '保存',
    cancelButtonText: '取消',
    type: 'warning'
  })
  yamlSaving.value = true
  try {
    const result = await updateResourceYAML(yamlKind.value, yamlName.value, yamlNamespace.value, yamlText.value)
    yamlText.value = result.yaml
    ElMessage.success('YAML 已保存')
    await loadCurrent()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'YAML 保存失败')
  } finally {
    yamlSaving.value = false
  }
}

async function removePod(row: PodSummary) {
  await ElMessageBox.confirm(`删除 Pod ${row.name}`, '二次确认', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  })
  await deletePod(row.namespace, row.name)
  ElMessage.success('删除请求已提交')
  await loadCurrent()
}

async function scale(row: DeploymentSummary) {
  const result = await ElMessageBox.prompt(`Deployment ${row.name}`, '扩缩容', {
    inputValue: String(row.replicas),
    inputPattern: /^(?:[0-9]|[1-9][0-9]|100)$/,
    inputErrorMessage: '0-100'
  })
  await scaleDeployment(row.namespace, row.name, Number(result.value))
  ElMessage.success('扩缩容已提交')
  await loadCurrent()
}

async function restart(row: DeploymentSummary) {
  await ElMessageBox.confirm(`重启 Deployment ${row.name}`, '二次确认', {
    confirmButtonText: '重启',
    cancelButtonText: '取消',
    type: 'warning'
  })
  await restartDeployment(row.namespace, row.name)
  ElMessage.success('重启已提交')
  await loadCurrent()
}

watch([namespace, activeGroup, activeWorkload, activeNetwork, activeStorage], () => {
  if (initialized.value) {
    void loadCurrent()
  }
})

onMounted(async () => {
  await loadNamespaces()
  initialized.value = true
  await loadCurrent()
})
</script>

<template>
  <AppShell>
    <div class="page-head">
      <div>
        <h1>资源管理</h1>
        <p>{{ scopeLabel }} · {{ activeTotal }} 条</p>
      </div>
      <div class="toolbar">
        <el-select v-model="namespace" class="namespace-select" :disabled="!needsNamespace" placeholder="Namespace">
          <el-option v-for="item in namespaces" :key="item.name" :label="item.name" :value="item.name" />
        </el-select>
        <el-button :icon="Refresh" :loading="loading" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeGroup" class="resource-tabs">
      <el-tab-pane v-for="item in groupOptions" :key="item.name" :label="item.label" :name="item.name" />
    </el-tabs>

    <section v-if="activeGroup === 'workloads'" class="resource-section">
      <div class="resource-subnav">
        <el-segmented v-model="activeWorkload" :options="workloadOptions" />
        <span class="resource-count">page size {{ pageSize }}</span>
      </div>

      <el-table v-if="activeWorkload === 'deployments'" v-loading="loading" :data="deploymentRows" border>
        <el-table-column prop="name" label="Deployment" min-width="220" show-overflow-tooltip />
        <el-table-column label="副本" width="110">
          <template #default="{ row }">{{ row.ready_replicas }}/{{ row.replicas }}</template>
        </el-table-column>
        <el-table-column label="不可用" width="90">
          <template #default="{ row }">{{ workloadUnavailable(row) }}</template>
        </el-table-column>
        <el-table-column label="镜像" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.images) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="168" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="扩缩容">
              <el-button :icon="VideoPlay" circle @click="scale(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('deployment', row.name, row.namespace)" />
            </el-tooltip>
            <el-tooltip content="重启">
              <el-button :icon="Refresh" circle type="warning" @click="restart(row)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeWorkload === 'statefulsets'" v-loading="loading" :data="statefulSetRows" border>
        <el-table-column prop="name" label="StatefulSet" min-width="220" show-overflow-tooltip />
        <el-table-column label="副本" width="110">
          <template #default="{ row }">{{ row.ready_replicas }}/{{ row.replicas }}</template>
        </el-table-column>
        <el-table-column prop="available_replicas" label="可用" width="90" />
        <el-table-column label="镜像" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.images) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('statefulset', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeWorkload === 'daemonsets'" v-loading="loading" :data="daemonSetRows" border>
        <el-table-column prop="name" label="DaemonSet" min-width="220" show-overflow-tooltip />
        <el-table-column prop="desired_number" label="期望" width="90" />
        <el-table-column prop="current_number" label="当前" width="90" />
        <el-table-column prop="ready_number" label="就绪" width="90" />
        <el-table-column prop="available_number" label="可用" width="90" />
        <el-table-column label="镜像" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.images) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('daemonset', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeWorkload === 'replicasets'" v-loading="loading" :data="replicaSetRows" border>
        <el-table-column prop="name" label="ReplicaSet" min-width="240" show-overflow-tooltip />
        <el-table-column label="副本" width="110">
          <template #default="{ row }">{{ row.ready_replicas }}/{{ row.replicas }}</template>
        </el-table-column>
        <el-table-column prop="available_replicas" label="可用" width="90" />
        <el-table-column label="镜像" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.images) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('replicaset', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeWorkload === 'jobs'" v-loading="loading" :data="jobRows" border>
        <el-table-column prop="name" label="Job" min-width="220" show-overflow-tooltip />
        <el-table-column prop="completions" label="完成数" width="92" />
        <el-table-column prop="succeeded" label="成功" width="82" />
        <el-table-column prop="failed" label="失败" width="82" />
        <el-table-column prop="active" label="运行中" width="92" />
        <el-table-column label="开始时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.start_time) }}</template>
        </el-table-column>
        <el-table-column label="完成时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.completion_time) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('job', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else v-loading="loading" :data="cronJobRows" border>
        <el-table-column prop="name" label="CronJob" min-width="220" show-overflow-tooltip />
        <el-table-column prop="schedule" label="计划" min-width="160" show-overflow-tooltip />
        <el-table-column label="暂停" width="90">
          <template #default="{ row }">
            <el-tag :type="row.suspend ? 'warning' : 'success'">{{ row.suspend ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="active" label="运行中" width="92" />
        <el-table-column label="上次调度" width="180">
          <template #default="{ row }">{{ formatDateTime(row.last_schedule_time) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('cronjob', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else-if="activeGroup === 'pods'" class="resource-section">
      <el-table v-loading="loading" :data="podRows" border>
        <el-table-column prop="name" label="Pod" min-width="240" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="150">
          <template #default="{ row }">
            <el-tag :type="row.ready ? 'success' : 'danger'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
        <el-table-column prop="restart_count" label="重启" width="90" />
        <el-table-column label="容器" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ containerNames(row) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="168" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="日志">
              <el-button :icon="Tickets" circle @click="openLogs(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('pod', row.name, row.namespace)" />
            </el-tooltip>
            <el-tooltip content="删除">
              <el-button :icon="Delete" circle type="danger" @click="removePod(row)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else-if="activeGroup === 'network'" class="resource-section">
      <div class="resource-subnav">
        <el-segmented v-model="activeNetwork" :options="networkOptions" />
        <span class="resource-count">page size {{ pageSize }}</span>
      </div>

      <el-table v-if="activeNetwork === 'services'" v-loading="loading" :data="serviceRows" border>
        <el-table-column prop="name" label="Service" min-width="220" show-overflow-tooltip />
        <el-table-column prop="type" label="类型" width="120" />
        <el-table-column prop="cluster_ip" label="Cluster IP" width="140" show-overflow-tooltip />
        <el-table-column prop="external_ip" label="External IP" min-width="140" show-overflow-tooltip />
        <el-table-column label="端口" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.ports) }}</template>
        </el-table-column>
        <el-table-column label="Selector" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ mapText(row.selector) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('service', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else v-loading="loading" :data="ingressRows" border>
        <el-table-column prop="name" label="Ingress" min-width="220" show-overflow-tooltip />
        <el-table-column prop="class_name" label="Class" width="140" show-overflow-tooltip />
        <el-table-column label="Hosts" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.hosts) }}</template>
        </el-table-column>
        <el-table-column label="Addresses" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.addresses) }}</template>
        </el-table-column>
        <el-table-column label="TLS" width="82">
          <template #default="{ row }">
            <el-tag :type="row.tls ? 'success' : 'info'">{{ row.tls ? 'on' : 'off' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('ingress', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else-if="activeGroup === 'config'" class="resource-section">
      <el-table v-loading="loading" :data="configMapRows" border>
        <el-table-column prop="name" label="ConfigMap" min-width="220" show-overflow-tooltip />
        <el-table-column prop="key_count" label="Data" width="90" />
        <el-table-column prop="binary_data_count" label="Binary" width="90" />
        <el-table-column label="Keys" min-width="320" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.keys) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('configmap', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else-if="activeGroup === 'storage'" class="resource-section">
      <div class="resource-subnav">
        <el-segmented v-model="activeStorage" :options="storageOptions" />
        <span class="resource-count">page size {{ pageSize }}</span>
      </div>

      <el-table v-if="activeStorage === 'pvcs'" v-loading="loading" :data="pvcRows" border>
        <el-table-column prop="name" label="PVC" min-width="220" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ row.status || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="storage_class" label="StorageClass" min-width="150" show-overflow-tooltip />
        <el-table-column prop="volume_name" label="PV" min-width="180" show-overflow-tooltip />
        <el-table-column label="容量" width="130">
          <template #default="{ row }">{{ row.capacity || row.requested || '-' }}</template>
        </el-table-column>
        <el-table-column label="访问模式" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.access_modes) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('pvc', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeStorage === 'pvs'" v-loading="loading" :data="pvRows" border>
        <el-table-column prop="name" label="PV" min-width="220" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ row.status || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="storage_class" label="StorageClass" min-width="150" show-overflow-tooltip />
        <el-table-column prop="capacity" label="容量" width="120" />
        <el-table-column label="Claim" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.claim_namespace && row.claim_name ? `${row.claim_namespace}/${row.claim_name}` : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="reclaim_policy" label="回收策略" width="120" />
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('pv', row.name)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else v-loading="loading" :data="storageClassRows" border>
        <el-table-column prop="name" label="StorageClass" min-width="220" show-overflow-tooltip />
        <el-table-column prop="provisioner" label="Provisioner" min-width="260" show-overflow-tooltip />
        <el-table-column prop="reclaim_policy" label="回收策略" width="120" />
        <el-table-column prop="volume_binding_mode" label="绑定模式" min-width="160" show-overflow-tooltip />
        <el-table-column label="扩容" width="90">
          <template #default="{ row }">
            <el-tag :type="row.allow_volume_expansion ? 'success' : 'info'">
              {{ row.allow_volume_expansion ? 'on' : 'off' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('storageclass', row.name)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else class="resource-section">
      <el-table v-loading="loading" :data="nodeRows" border>
        <el-table-column prop="name" label="Node" min-width="220" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cpu_allocatable" label="CPU" width="120" />
        <el-table-column prop="memory_allocatable" label="Memory" width="140" />
        <el-table-column prop="pods_allocatable" label="Pods" width="120" />
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="78" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('node', row.name)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer v-model="logsVisible" size="68%" title="Pod 日志">
      <div class="drawer-toolbar">
        <el-select v-model="logQuery.container" placeholder="Container" clearable>
          <el-option
            v-for="item in selectedPod?.containers ?? []"
            :key="item.name"
            :label="item.name"
            :value="item.name"
          />
        </el-select>
        <el-input v-model="logQuery.keyword" placeholder="关键字" clearable />
        <el-select v-model="logQuery.level" placeholder="级别" clearable>
          <el-option label="error" value="error" />
          <el-option label="warn" value="warn" />
          <el-option label="info" value="info" />
          <el-option label="debug" value="debug" />
        </el-select>
        <el-switch v-model="logQuery.previous" active-text="Previous" />
        <el-button :loading="logsLoading" @click="loadLogs">查询</el-button>
      </div>
      <pre class="log-view">{{ logResult?.lines.map((line) => line.raw).join('\n') }}</pre>
    </el-drawer>

    <el-drawer v-model="yamlVisible" size="68%" :title="yamlTitle">
      <div class="drawer-toolbar yaml-toolbar">
        <el-button :icon="Refresh" @click="reloadYAML">重载</el-button>
        <el-button v-if="yamlEditable" :icon="Check" type="primary" :loading="yamlSaving" @click="saveYAML">
          保存
        </el-button>
      </div>
      <el-input
        v-if="yamlEditable"
        v-model="yamlText"
        class="yaml-editor"
        type="textarea"
        :autosize="{ minRows: 24 }"
        spellcheck="false"
      />
      <pre v-else class="yaml-view">{{ yamlText }}</pre>
    </el-drawer>
  </AppShell>
</template>
