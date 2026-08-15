<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { Check, Delete, InfoFilled, Refresh, Tickets, VideoPlay, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  deletePod,
  followPodLogs,
  getDaemonSet,
  getDeployment,
  getCronJob,
  getJob,
  getIngress,
  getNamespace,
  getNode,
  getPV,
  getPVC,
  getPod,
  getReplicaSet,
  getSecret,
  getService,
  getStatefulSet,
  getPodLogs,
  getResourceYAML,
  listConfigMaps,
  listCronJobs,
  listDaemonSets,
  listDeployments,
  listIngresses,
  listJobs,
  listEvents,
  listNamespaces,
  listNodes,
  listPods,
  listPVCs,
  listPVs,
  listReplicaSets,
  listSecrets,
  listServices,
  listStatefulSets,
  listStorageClasses,
  restartDeployment,
  restartDaemonSet,
  resumeCronJob,
  restartPod as restartPodApi,
  restartStatefulSet,
  scaleDeployment,
  scaleStatefulSet,
  readSecretValue,
  suspendCronJob,
  updateResourceYAML,
  type ConfigMapSummary,
  type CronJobDetail,
  type CronJobSummary,
  type DaemonSetDetail,
  type DaemonSetSummary,
  type DeploymentDetail,
  type DeploymentSummary,
  type EventSummary,
  type IngressDetail,
  type IngressSummary,
  type JobDetail,
  type JobSummary,
  type LogResult,
  type NamespaceDetail,
  type NamespaceSummary,
  type NodeDetail,
  type NodeSummary,
  type PageResult,
  type PodDetail,
  type PodSummary,
  type PVCDetail,
  type PVCSummary,
  type PVDetail,
  type PVSummary,
  type ResourceKind,
  type ReplicaSetDetail,
  type SecretDetail,
  type SecretSummary,
  type SecretValueResponse,
  type ServiceDetail,
  type ServiceSummary,
  type StorageClassSummary,
  type StatefulSetDetail,
  type WorkloadSummary
} from '../api/resources'
import AppShell from '../components/AppShell.vue'
import { useAuthStore } from '../stores/auth'
import { formatDateTime } from '../utils/time'

type ResourceGroup = 'workloads' | 'pods' | 'network' | 'config' | 'storage' | 'events' | 'nodes'
type WorkloadKind = 'deployments' | 'statefulsets' | 'daemonsets' | 'replicasets' | 'jobs' | 'cronjobs'
type NetworkKind = 'services' | 'ingresses'
type ConfigKind = 'configmaps' | 'secrets'
type StorageKind = 'pvcs' | 'pvs' | 'storageclasses'

const pageSize = 100
const authStore = useAuthStore()

const namespaces = ref<NamespaceSummary[]>([])
const namespace = ref('')
const initialized = ref(false)
const activeGroup = ref<ResourceGroup>('workloads')
const activeWorkload = ref<WorkloadKind>('deployments')
const activeNetwork = ref<NetworkKind>('services')
const activeConfig = ref<ConfigKind>('configmaps')
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
const secrets = ref<PageResult<SecretSummary> | null>(null)
const pvcs = ref<PageResult<PVCSummary> | null>(null)
const pvs = ref<PageResult<PVSummary> | null>(null)
const storageClasses = ref<PageResult<StorageClassSummary> | null>(null)
const events = ref<EventSummary[]>([])
const eventFilters = reactive({ involved_kind: '', involved_name: '' })

const loading = ref(false)
const logsLoading = ref(false)
const logsVisible = ref(false)
const logMode = ref<'history' | 'realtime'>('history')
const realtimeConnected = ref(false)
const realtimeLines = ref<string[]>([])
let realtimeSocket: WebSocket | null = null
const podDetailVisible = ref(false)
const podDetailLoading = ref(false)
const podDetail = ref<PodDetail | null>(null)
const podEvents = ref<EventSummary[]>([])
const deploymentDetailVisible = ref(false)
const deploymentDetailLoading = ref(false)
const deploymentDetail = ref<DeploymentDetail | null>(null)
const replicaSetDetailVisible = ref(false)
const replicaSetDetailLoading = ref(false)
const replicaSetDetail = ref<ReplicaSetDetail | null>(null)
const jobDetailVisible = ref(false)
const jobDetailLoading = ref(false)
const jobDetail = ref<JobDetail | null>(null)
const cronJobDetailVisible = ref(false)
const cronJobDetailLoading = ref(false)
const cronJobDetail = ref<CronJobDetail | null>(null)
const statefulSetDetailVisible = ref(false)
const statefulSetDetailLoading = ref(false)
const statefulSetDetail = ref<StatefulSetDetail | null>(null)
const daemonSetDetailVisible = ref(false)
const daemonSetDetailLoading = ref(false)
const daemonSetDetail = ref<DaemonSetDetail | null>(null)
const serviceDetailVisible = ref(false)
const serviceDetailLoading = ref(false)
const serviceDetail = ref<ServiceDetail | null>(null)
const ingressDetailVisible = ref(false)
const ingressDetailLoading = ref(false)
const ingressDetail = ref<IngressDetail | null>(null)
const pvcDetailVisible = ref(false)
const pvcDetailLoading = ref(false)
const pvcDetail = ref<PVCDetail | null>(null)
const pvDetailVisible = ref(false)
const pvDetailLoading = ref(false)
const pvDetail = ref<PVDetail | null>(null)
const namespaceDetailVisible = ref(false)
const namespaceDetailLoading = ref(false)
const namespaceDetail = ref<NamespaceDetail | null>(null)
const nodeDetailVisible = ref(false)
const nodeDetailLoading = ref(false)
const nodeDetail = ref<NodeDetail | null>(null)
const secretDetailVisible = ref(false)
const secretDetailLoading = ref(false)
const secretDetail = ref<SecretDetail | null>(null)
const secretValueVisible = ref(false)
const secretValueLoading = ref(false)
const secretValue = ref<SecretValueResponse | null>(null)
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
  limit: 300,
  from: ''
})

const groupOptions = [
  { label: 'Workloads', name: 'workloads' },
  { label: 'Pods', name: 'pods' },
  { label: 'Network', name: 'network' },
  { label: 'Config', name: 'config' },
  { label: 'Storage', name: 'storage' },
  { label: 'Events', name: 'events' },
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
const canReadSecrets = computed(() => authStore.hasAnyRole(['configadmin']))
const canReadSecretValues = computed(() => authStore.roles.includes('admin'))
const configOptions = computed(() => {
  const options = [{ label: 'ConfigMaps', value: 'configmaps' }]
  if (canReadSecrets.value) {
    options.push({ label: 'Secrets', value: 'secrets' })
  }
  return options
})
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
const secretRows = computed(() => secrets.value?.items ?? [])
const pvcRows = computed(() => pvcs.value?.items ?? [])
const pvRows = computed(() => pvs.value?.items ?? [])
const storageClassRows = computed(() => storageClasses.value?.items ?? [])
const eventRows = computed(() => events.value)
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
  if (activeGroup.value === 'events') return events.value.length
  if (activeGroup.value === 'nodes') return nodes.value?.total ?? 0
  if (activeGroup.value === 'network') return activeNetwork.value === 'services' ? services.value?.total ?? 0 : ingresses.value?.total ?? 0
  if (activeGroup.value === 'config') {
    return activeConfig.value === 'secrets' ? secrets.value?.total ?? 0 : configMaps.value?.total ?? 0
  }
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

function volumeDataSourceText(detail: PVCDetail) {
  if (!detail.data_source) return '-'
  const group = detail.data_source.api_group ? `${detail.data_source.api_group}/` : ''
  const namespacePrefix = detail.data_source.namespace ? `${detail.data_source.namespace}/` : ''
  return `${group}${detail.data_source.kind} ${namespacePrefix}${detail.data_source.name}`
}

function mountTarget(row: { mount_path: string; device_path: string; sub_path: string }) {
  const target = row.mount_path || row.device_path || '-'
  return row.sub_path ? `${target} (subPath: ${row.sub_path})` : target
}

function resourceText(resource?: { cpu: string; memory: string; ephemeral_storage: string; pods: number } | null) {
  if (!resource) return '-'
  return `CPU ${resource.cpu || '0'} / Memory ${resource.memory || '0'} / Ephemeral ${resource.ephemeral_storage || '0'} / Pods ${resource.pods}`
}

function percentText(value: number) {
  return `${Number.isFinite(value) ? value.toFixed(2) : '0.00'}%`
}

function nodeAddressesText() {
  const addresses = nodeDetail.value?.addresses ?? []
  return addresses.length ? addresses.map((item) => `${item.type}: ${item.address}`).join(', ') : '-'
}

function openStorageWorkload(row: { kind: string; name: string }, rowNamespace: string) {
  const kind = row.kind.toLowerCase()
  if (kind === 'deployment') {
    void showDeploymentDetail({ namespace: rowNamespace, name: row.name })
  } else if (kind === 'statefulset') {
    void showStatefulSetDetail({ namespace: rowNamespace, name: row.name })
  } else if (kind === 'daemonset') {
    void showDaemonSetDetail({ namespace: rowNamespace, name: row.name })
  } else if (kind === 'replicaset') {
    void showReplicaSetDetail({ namespace: rowNamespace, name: row.name })
  } else if (kind === 'job') {
    void showJobDetail({ namespace: rowNamespace, name: row.name })
  } else if (kind === 'cronjob') {
    void showCronJobDetail({ namespace: rowNamespace, name: row.name })
  }
}

function controllerText() {
  const chain = podDetail.value?.controller_chain ?? []
  return chain.length ? chain.map((item) => `${item.kind}/${item.name}`).join(' → ') : '独立 Pod / 无可识别控制器'
}

function queryStartTime() {
  return logQuery.from ? new Date(logQuery.from).toISOString() : undefined
}

function statusType(status: string) {
  const normalized = status.toLowerCase()
  if (['ready', 'running', 'bound', 'active', 'succeeded', 'completed', 'complete'].includes(normalized)) return 'success'
  if (['pending', 'terminating', 'unknown'].includes(normalized)) return 'warning'
  if (['failed', 'notready'].includes(normalized)) return 'danger'
  return 'info'
}

function jobStatus(row: JobSummary) {
  if (row.active > 0) return '运行中'
  if (row.completions > 0 && row.succeeded >= row.completions) return '已完成'
  if (row.failed > 0) return '失败'
  return '等待中'
}

function jobStatusType(row: JobSummary) {
  const status = jobStatus(row)
  if (status === '已完成') return 'success'
  if (status === '运行中') return 'primary'
  if (status === '失败') return 'danger'
  return 'info'
}

function workloadUnavailable(row: DeploymentSummary | WorkloadSummary) {
  return Math.max(row.unavailable_replicas, 0)
}

function clearEventFilters() {
  eventFilters.involved_kind = ''
  eventFilters.involved_name = ''
  void loadCurrent()
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
    } else if (activeGroup.value === 'events') {
      events.value = await listEvents(ns, {
        involved_kind: eventFilters.involved_kind || undefined,
        involved_name: eventFilters.involved_name || undefined
      })
    } else if (activeGroup.value === 'nodes') {
      nodes.value = await listNodes(1, pageSize)
    } else if (activeGroup.value === 'network') {
      if (activeNetwork.value === 'services') {
        services.value = await listServices(ns, 1, pageSize)
      } else {
        ingresses.value = await listIngresses(ns, 1, pageSize)
      }
    } else if (activeGroup.value === 'config') {
      if (activeConfig.value === 'secrets' && canReadSecrets.value) {
        secrets.value = await listSecrets(ns, 1, pageSize)
      } else {
        configMaps.value = await listConfigMaps(ns, 1, pageSize)
      }
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
  stopRealtimeLogs()
  selectedPod.value = row
  logQuery.container = row.containers[0]?.name ?? ''
  logMode.value = 'history'
  logsVisible.value = true
  await loadLogs()
}

async function loadLogs() {
  if (!selectedPod.value) {
    return
  }
  logsLoading.value = true
  try {
    logResult.value = await getPodLogs(selectedPod.value.namespace, selectedPod.value.name, {
      ...logQuery,
      from: queryStartTime()
    })
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '日志加载失败')
  } finally {
    logsLoading.value = false
  }
}

function startRealtimeLogs() {
  if (!selectedPod.value || realtimeConnected.value) {
    return
  }
  stopRealtimeLogs()
  realtimeLines.value = []
  try {
    const socket = followPodLogs(selectedPod.value.namespace, selectedPod.value.name, {
      container: logQuery.container,
      keyword: logQuery.keyword,
      level: logQuery.level,
      limit: logQuery.limit,
      from: queryStartTime()
    })
    realtimeSocket = socket
    socket.onmessage = (event) => {
      const message = JSON.parse(String(event.data)) as { type: string; line?: { raw: string }; message?: string }
      if (message.type === 'ready') {
        realtimeConnected.value = true
        return
      }
      if (message.type === 'line' && message.line) {
        realtimeLines.value.push(message.line.raw)
        if (realtimeLines.value.length > 5000) {
          realtimeLines.value.splice(0, realtimeLines.value.length - 5000)
        }
        return
      }
      if (message.type === 'error') {
        ElMessage.error(message.message || '实时日志流异常')
      }
    }
    socket.onerror = () => {
      ElMessage.error('实时日志连接失败')
    }
    socket.onclose = () => {
      if (realtimeSocket === socket) {
        realtimeSocket = null
        realtimeConnected.value = false
      }
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '实时日志连接失败')
  }
}

function stopRealtimeLogs() {
  const socket = realtimeSocket
  realtimeSocket = null
  realtimeConnected.value = false
  if (socket && socket.readyState < WebSocket.CLOSING) {
    socket.close(1000, 'client stopped')
  }
}

async function showPodDetail(row: PodSummary) {
  podDetailVisible.value = true
  podDetailLoading.value = true
  podDetail.value = null
  podEvents.value = []
  try {
    const [detail, events] = await Promise.all([
      getPod(row.namespace, row.name),
      listEvents(row.namespace, { involved_kind: 'Pod', involved_name: row.name })
    ])
    podDetail.value = detail
    podEvents.value = events
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Pod 详情加载失败')
  } finally {
    podDetailLoading.value = false
  }
}

async function showDeploymentDetail(row: { namespace: string; name: string }) {
  deploymentDetailVisible.value = true
  deploymentDetailLoading.value = true
  deploymentDetail.value = null
  try {
    deploymentDetail.value = await getDeployment(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Deployment 详情加载失败')
  } finally {
    deploymentDetailLoading.value = false
  }
}

async function showReplicaSetDetail(row: { namespace: string; name: string }) {
  replicaSetDetailVisible.value = true
  replicaSetDetailLoading.value = true
  replicaSetDetail.value = null
  try {
    replicaSetDetail.value = await getReplicaSet(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'ReplicaSet 详情加载失败')
  } finally {
    replicaSetDetailLoading.value = false
  }
}

async function showJobDetail(row: { namespace: string; name: string }) {
  jobDetailVisible.value = true
  jobDetailLoading.value = true
  jobDetail.value = null
  try {
    jobDetail.value = await getJob(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Job 详情加载失败')
  } finally {
    jobDetailLoading.value = false
  }
}

async function showCronJobDetail(row: { namespace: string; name: string }) {
  cronJobDetailVisible.value = true
  cronJobDetailLoading.value = true
  cronJobDetail.value = null
  try {
    cronJobDetail.value = await getCronJob(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'CronJob 详情加载失败')
  } finally {
    cronJobDetailLoading.value = false
  }
}

async function showServiceDetail(row: { namespace: string; name: string }) {
  serviceDetailVisible.value = true
  serviceDetailLoading.value = true
  serviceDetail.value = null
  try {
    serviceDetail.value = await getService(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Service 详情加载失败')
  } finally {
    serviceDetailLoading.value = false
  }
}

async function showIngressDetail(row: { namespace: string; name: string }) {
  ingressDetailVisible.value = true
  ingressDetailLoading.value = true
  ingressDetail.value = null
  try {
    ingressDetail.value = await getIngress(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Ingress 详情加载失败')
  } finally {
    ingressDetailLoading.value = false
  }
}

async function showPVCDetail(row: { namespace: string; name: string }) {
  pvcDetailVisible.value = true
  pvcDetailLoading.value = true
  pvcDetail.value = null
  try {
    pvcDetail.value = await getPVC(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'PVC 详情加载失败')
  } finally {
    pvcDetailLoading.value = false
  }
}

async function showNamespaceDetail(name: string) {
  if (!name) return
  namespaceDetailVisible.value = true
  namespaceDetailLoading.value = true
  namespaceDetail.value = null
  try {
    namespaceDetail.value = await getNamespace(name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Namespace 详情加载失败')
  } finally {
    namespaceDetailLoading.value = false
  }
}

async function showNodeDetail(row: { name: string }) {
  nodeDetailVisible.value = true
  nodeDetailLoading.value = true
  nodeDetail.value = null
  try {
    nodeDetail.value = await getNode(row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Node 详情加载失败')
  } finally {
    nodeDetailLoading.value = false
  }
}

async function showPVDetail(row: { name: string }) {
  pvDetailVisible.value = true
  pvDetailLoading.value = true
  pvDetail.value = null
  try {
    pvDetail.value = await getPV(row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'PV 详情加载失败')
  } finally {
    pvDetailLoading.value = false
  }
}

async function setCronJobSuspended(row: CronJobSummary, suspend: boolean) {
  const action = suspend ? '暂停' : '恢复'
  const message = suspend
    ? `暂停 CronJob ${row.name}？这只会阻止未来调度，已经开始的 Job 不会被停止。`
    : `恢复 CronJob ${row.name}？Kubernetes 可能根据 StartingDeadlineSeconds 处理暂停期间错过的调度。`
  await ElMessageBox.confirm(message, '二次确认', {
    confirmButtonText: action,
    cancelButtonText: '取消',
    type: 'warning'
  })
  if (suspend) {
    await suspendCronJob(row.namespace, row.name)
  } else {
    await resumeCronJob(row.namespace, row.name)
  }
  ElMessage.success(`CronJob 已${action}`)
  await loadCurrent()
  if (cronJobDetailVisible.value && cronJobDetail.value?.name === row.name) {
    cronJobDetail.value = await getCronJob(row.namespace, row.name)
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
  const kind = yamlKind.value
  const targetNamespace = yamlNamespace.value
  const name = yamlName.value
  await ElMessageBox.confirm(`保存 ${yamlTitle.value}`, '二次确认', {
    confirmButtonText: '保存',
    cancelButtonText: '取消',
    type: 'warning'
  })
  yamlSaving.value = true
  try {
    const result = await updateResourceYAML(kind, name, targetNamespace, yamlText.value)
    yamlText.value = result.yaml
    ElMessage.success('YAML 已保存')
    await loadCurrent()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'YAML 保存失败')
    return
  } finally {
    yamlSaving.value = false
  }

  if (kind === 'deployment' || kind === 'statefulset' || kind === 'daemonset') {
    try {
      await offerRestartAfterYAMLSave(kind, targetNamespace, name)
    } catch (error) {
      ElMessage.error(error instanceof Error ? `YAML 已保存，但滚动重启提交失败：${error.message}` : 'YAML 已保存，但滚动重启提交失败')
    }
  }
}

async function offerRestartAfterYAMLSave(kind: 'deployment' | 'statefulset' | 'daemonset', targetNamespace: string, name: string) {
  try {
    await ElMessageBox.confirm(
      'YAML 已保存。Pod 模板变更会由 Kubernetes 自动滚动生效；如果调整的是外部 ConfigMap/Secret 等运行配置，现有 Pod 通常需要重启后重新加载。是否立即滚动重启？',
      '配置生效提示',
      {
        confirmButtonText: '立即重启',
        cancelButtonText: '稍后处理',
        type: 'info'
      }
    )
  } catch (error) {
    if (error === 'cancel' || error === 'close') {
      return
    }
    throw error
  }

  if (kind === 'deployment') {
    await restartDeployment(targetNamespace, name)
  } else if (kind === 'statefulset') {
    await restartStatefulSet(targetNamespace, name)
  } else {
    await restartDaemonSet(targetNamespace, name)
  }
  ElMessage.success('YAML 已保存，滚动重启已提交')
  await loadCurrent()
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

async function restartPod(row: PodSummary) {
  await ElMessageBox.confirm(
    `重启 Pod ${row.name}？平台会删除该 Pod，并由其工作负载控制器创建替代 Pod。`,
    '二次确认',
    {
      confirmButtonText: '重启',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
  await restartPodApi(row.namespace, row.name)
  ElMessage.success('Pod 重启请求已提交')
  await loadCurrent()
}

async function showStatefulSetDetail(row: { namespace: string; name: string }) {
  statefulSetDetailVisible.value = true
  statefulSetDetailLoading.value = true
  statefulSetDetail.value = null
  try {
    statefulSetDetail.value = await getStatefulSet(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'StatefulSet 详情加载失败')
  } finally {
    statefulSetDetailLoading.value = false
  }
}

async function showDaemonSetDetail(row: { namespace: string; name: string }) {
  daemonSetDetailVisible.value = true
  daemonSetDetailLoading.value = true
  daemonSetDetail.value = null
  try {
    daemonSetDetail.value = await getDaemonSet(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'DaemonSet 详情加载失败')
  } finally {
    daemonSetDetailLoading.value = false
  }
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

async function scaleStatefulSetRow(row: WorkloadSummary) {
  const result = await ElMessageBox.prompt(`StatefulSet ${row.name}`, '扩缩容', {
    inputValue: String(row.replicas),
    inputPattern: /^(?:[0-9]|[1-9][0-9]|100)$/,
    inputErrorMessage: '0-100'
  })
  await scaleStatefulSet(row.namespace, row.name, Number(result.value))
  ElMessage.success('StatefulSet 扩缩容已提交')
  await loadCurrent()
}

async function restartStatefulSetRow(row: WorkloadSummary) {
  await ElMessageBox.confirm(
    `滚动重启 StatefulSet ${row.name}？Pod 将按照 StatefulSet 更新策略依次重建。`,
    '二次确认',
    {
      confirmButtonText: '滚动重启',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
  await restartStatefulSet(row.namespace, row.name)
  ElMessage.success('StatefulSet 滚动重启已提交')
  await loadCurrent()
}

async function restartDaemonSetRow(row: DaemonSetSummary) {
  await ElMessageBox.confirm(
    `滚动重启 DaemonSet ${row.name}？每个目标节点上的 Pod 将按照 DaemonSet 更新策略依次重建。`,
    '二次确认',
    {
      confirmButtonText: '滚动重启',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
  await restartDaemonSet(row.namespace, row.name)
  ElMessage.success('DaemonSet 滚动重启已提交')
  await loadCurrent()
}

async function showSecretDetail(row: SecretSummary) {
  secretDetailVisible.value = true
  secretDetailLoading.value = true
  secretDetail.value = null
  clearSecretValue()
  try {
    secretDetail.value = await getSecret(row.namespace, row.name)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Secret 详情加载失败')
  } finally {
    secretDetailLoading.value = false
  }
}

async function showSecretValue(key: string) {
  const detail = secretDetail.value
  if (!detail || !canReadSecretValues.value) return
  await ElMessageBox.confirm(
    `确认查看 Secret ${detail.namespace}/${detail.name} 的 key ${key} 明文？该操作会写入审计日志，请避免截屏、复制到日志或发送给无权限人员。`,
    '高敏操作确认',
    {
      confirmButtonText: '确认查看',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
  secretValueLoading.value = true
  clearSecretValue()
  try {
    secretValue.value = await readSecretValue(detail.namespace, detail.name, key)
    secretValueVisible.value = true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Secret 明文读取失败')
  } finally {
    secretValueLoading.value = false
  }
}

function clearSecretValue() {
  secretValueVisible.value = false
  secretValue.value = null
}

watch([namespace, activeGroup, activeWorkload, activeNetwork, activeConfig, activeStorage], () => {
  if (initialized.value) {
    void loadCurrent()
  }
})

watch(canReadSecrets, (allowed) => {
  if (allowed) return
  activeConfig.value = 'configmaps'
  secrets.value = null
  secretDetailVisible.value = false
  secretDetail.value = null
  clearSecretValue()
})

watch(canReadSecretValues, (allowed) => {
  if (!allowed) clearSecretValue()
})

watch(logMode, (mode) => {
  if (mode === 'history') {
    stopRealtimeLogs()
  }
})

watch(logsVisible, (visible) => {
  if (!visible) {
    stopRealtimeLogs()
  }
})

onMounted(async () => {
  await loadNamespaces()
  initialized.value = true
  await loadCurrent()
})

onBeforeUnmount(stopRealtimeLogs)
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
        <el-button v-if="needsNamespace" :icon="InfoFilled" :disabled="!namespace" @click="showNamespaceDetail(namespace)">
          Namespace 详情
        </el-button>
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
        <el-table-column label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情 / ReplicaSet / Pod / Event">
              <el-button :icon="InfoFilled" circle @click="showDeploymentDetail(row)" />
            </el-tooltip>
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
        <el-table-column label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情">
              <el-button :icon="InfoFilled" circle @click="showStatefulSetDetail(row)" />
            </el-tooltip>
            <el-tooltip content="扩缩容">
              <el-button :icon="VideoPlay" circle @click="scaleStatefulSetRow(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('statefulset', row.name, row.namespace)" />
            </el-tooltip>
            <el-tooltip content="滚动重启">
              <el-button :icon="Refresh" circle type="warning" @click="restartStatefulSetRow(row)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeWorkload === 'daemonsets'" v-loading="loading" :data="daemonSetRows" border>
        <el-table-column prop="name" label="DaemonSet" min-width="220" show-overflow-tooltip />
        <el-table-column prop="desired_number" label="期望" width="90" />
        <el-table-column prop="ready_number" label="就绪" width="90" />
        <el-table-column prop="updated_number" label="已更新" width="90" />
        <el-table-column prop="unavailable_number" label="不可用" width="90" />
        <el-table-column prop="misscheduled_number" label="误调度" width="90" />
        <el-table-column label="镜像" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">{{ listText(row.images) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="168" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情">
              <el-button :icon="InfoFilled" circle @click="showDaemonSetDetail(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('daemonset', row.name, row.namespace)" />
            </el-tooltip>
            <el-tooltip content="滚动重启">
              <el-button :icon="Refresh" circle type="warning" @click="restartDaemonSetRow(row)" />
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情 / Deployment / Pod / Event">
              <el-button :icon="InfoFilled" circle @click="showReplicaSetDetail(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('replicaset', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <el-table v-else-if="activeWorkload === 'jobs'" v-loading="loading" :data="jobRows" border>
        <el-table-column prop="name" label="Job" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="jobStatusType(row)">{{ jobStatus(row) }}</el-tag>
          </template>
        </el-table-column>
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情 / Pod / Event / 日志">
              <el-button :icon="InfoFilled" circle @click="showJobDetail(row)" />
            </el-tooltip>
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
        <el-table-column label="操作" width="168" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情 / 历史 Job / Pod / Event">
              <el-button :icon="InfoFilled" circle @click="showCronJobDetail(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('cronjob', row.name, row.namespace)" />
            </el-tooltip>
            <el-tooltip :content="row.suspend ? '恢复未来调度' : '暂停未来调度'">
              <el-button
                :icon="row.suspend ? VideoPlay : Refresh"
                circle
                :type="row.suspend ? 'success' : 'warning'"
                @click="setCronJobSuspended(row, !row.suspend)"
              />
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
        <el-table-column label="操作" width="252" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="详情 / 事件 / 控制器">
              <el-button :icon="InfoFilled" circle @click="showPodDetail(row)" />
            </el-tooltip>
            <el-tooltip content="日志">
              <el-button :icon="Tickets" circle @click="openLogs(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('pod', row.name, row.namespace)" />
            </el-tooltip>
            <el-tooltip content="重启（删除后由控制器重建）">
              <el-button :icon="Refresh" circle type="warning" @click="restartPod(row)" />
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="Endpoint / Pod / Event 详情">
              <el-button :icon="InfoFilled" circle @click="showServiceDetail(row)" />
            </el-tooltip>
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="规则 / Service / Endpoint / Pod / Event 详情">
              <el-button :icon="InfoFilled" circle @click="showIngressDetail(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('ingress', row.name, row.namespace)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else-if="activeGroup === 'config'" class="resource-section">
      <div class="resource-subnav">
        <el-segmented v-model="activeConfig" :options="configOptions" />
        <span class="resource-count">page size {{ pageSize }}</span>
      </div>

      <el-table v-if="activeConfig === 'configmaps'" v-loading="loading" :data="configMapRows" border>
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

      <template v-else>
        <el-alert
          class="detail-scope-alert"
          type="warning"
          :closable="false"
          title="Secret 默认只展示类型、key 名和大小，不提供 YAML 导出。仅 Admin 可在二次确认后查看单个 key 明文，且操作会进入审计。"
        />
        <el-table v-loading="loading" :data="secretRows" border>
          <el-table-column prop="name" label="Secret" min-width="220" show-overflow-tooltip />
          <el-table-column prop="type" label="类型" min-width="190" show-overflow-tooltip />
          <el-table-column prop="key_count" label="Keys" width="90" />
          <el-table-column label="Key 名称" min-width="320" show-overflow-tooltip>
            <template #default="{ row }">{{ listText(row.keys) }}</template>
          </el-table-column>
          <el-table-column label="创建时间" width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="showSecretDetail(row)">安全详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="存储关联详情">
              <el-button :icon="InfoFilled" circle @click="showPVCDetail(row)" />
            </el-tooltip>
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="存储关联详情">
              <el-button :icon="InfoFilled" circle @click="showPVDetail(row)" />
            </el-tooltip>
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

    <section v-else-if="activeGroup === 'events'" class="resource-section">
      <div class="resource-subnav">
        <el-select v-model="eventFilters.involved_kind" clearable placeholder="资源类型" style="width: 180px">
          <el-option label="Pod" value="Pod" />
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
          <el-option label="ReplicaSet" value="ReplicaSet" />
          <el-option label="Job" value="Job" />
          <el-option label="CronJob" value="CronJob" />
          <el-option label="Service" value="Service" />
          <el-option label="Ingress" value="Ingress" />
          <el-option label="PersistentVolumeClaim" value="PersistentVolumeClaim" />
        </el-select>
        <el-input v-model="eventFilters.involved_name" clearable placeholder="资源名称" style="width: 240px" />
        <el-button type="primary" @click="loadCurrent">查询</el-button>
        <el-button @click="clearEventFilters">重置</el-button>
        <span class="resource-count">按最后发生时间倒序</span>
      </div>
      <el-alert
        class="detail-scope-alert"
        type="info"
        :closable="false"
        title="这里展示当前 Namespace 的 Kubernetes Event；事件聚合、去重、告警和 webhook 属于 P1。"
      />
      <el-table v-loading="loading" :data="eventRows" border empty-text="暂无 Event">
        <el-table-column prop="type" label="类型" width="96">
          <template #default="{ row }"><el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="reason" label="Reason" min-width="160" show-overflow-tooltip />
        <el-table-column label="资源" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">{{ `${row.involved_kind || '-'}/${row.involved_name || '-'}` }}</template>
        </el-table-column>
        <el-table-column prop="message" label="消息" min-width="360" show-overflow-tooltip />
        <el-table-column prop="count" label="次数" width="80" />
        <el-table-column prop="source" label="来源" min-width="150" show-overflow-tooltip />
        <el-table-column label="最后发生" width="180">
          <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
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
        <el-table-column label="操作" width="126" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="只读诊断详情">
              <el-button :icon="InfoFilled" circle @click="showNodeDetail(row)" />
            </el-tooltip>
            <el-tooltip content="YAML">
              <el-button :icon="View" circle @click="showYAML('node', row.name)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-drawer
      v-model="namespaceDetailVisible"
      size="84%"
      :title="namespaceDetail ? `Namespace ${namespaceDetail.name}` : 'Namespace 详情'"
    >
      <div v-loading="namespaceDetailLoading">
        <el-descriptions v-if="namespaceDetail" :column="3" border>
          <el-descriptions-item label="状态"><el-tag :type="statusType(namespaceDetail.status)">{{ namespaceDetail.status || '-' }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(namespaceDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="Finalizers">{{ listText(namespaceDetail.finalizers) }}</el-descriptions-item>
          <el-descriptions-item label="Labels" :span="3">{{ mapText(namespaceDetail.labels) }}</el-descriptions-item>
          <el-descriptions-item label="Pod">
            {{ namespaceDetail.counts.pods }}（Ready {{ namespaceDetail.counts.ready_pods }} / 异常 {{ namespaceDetail.counts.abnormal_pods }}）
          </el-descriptions-item>
          <el-descriptions-item label="Workloads">
            Deployment {{ namespaceDetail.counts.deployments }} / StatefulSet {{ namespaceDetail.counts.statefulsets }} / DaemonSet {{ namespaceDetail.counts.daemonsets }} / ReplicaSet {{ namespaceDetail.counts.replicasets }} / Job {{ namespaceDetail.counts.jobs }} / CronJob {{ namespaceDetail.counts.cronjobs }}
          </el-descriptions-item>
          <el-descriptions-item label="关联资源">
            Service {{ namespaceDetail.counts.services }} / Ingress {{ namespaceDetail.counts.ingresses }} / PVC {{ namespaceDetail.counts.persistent_volume_claims }} / ConfigMap {{ namespaceDetail.counts.configmaps }}
          </el-descriptions-item>
          <el-descriptions-item label="Active Pod Requests" :span="3">{{ resourceText(namespaceDetail.allocated.requests) }}</el-descriptions-item>
          <el-descriptions-item label="Active Pod Limits" :span="3">{{ resourceText(namespaceDetail.allocated.limits) }}</el-descriptions-item>
        </el-descriptions>

        <template v-if="namespaceDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="Namespace 详情只提供诊断和下钻，不开放删除。资源数据来自 Pod 声明的 requests/limits，不代表实时 CPU/Memory 使用率；实时指标将在 P1 接入 metrics-server。"
          />

          <h3>Pod</h3>
          <el-table :data="namespaceDetail.pods" border empty-text="Namespace 内没有 Pod">
            <el-table-column prop="name" label="Pod" min-width="220" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="150"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ row.status || row.phase || '-' }}</el-tag></template></el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
            <el-table-column prop="restart_count" label="重启" width="80" />
            <el-table-column label="操作" width="126" fixed="right"><template #default="{ row }"><el-tooltip content="Pod 详情"><el-button :icon="InfoFilled" circle @click="showPodDetail(row)" /></el-tooltip><el-tooltip content="Pod 历史 / 实时日志"><el-button :icon="Tickets" circle @click="openLogs(row)" /></el-tooltip></template></el-table-column>
          </el-table>

          <h3>顶层 Workload</h3>
          <el-table :data="namespaceDetail.workloads" border empty-text="没有可识别的顶层 Workload">
            <el-table-column prop="kind" label="类型" min-width="160" />
            <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
            <el-table-column label="操作" width="90"><template #default="{ row }"><el-button link type="primary" @click="openStorageWorkload(row, row.namespace)">查看详情</el-button></template></el-table-column>
          </el-table>

          <h3>Service</h3>
          <el-table :data="namespaceDetail.services" border empty-text="没有 Service">
            <el-table-column prop="name" label="Service" min-width="220" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column prop="cluster_ip" label="Cluster IP" min-width="150" />
            <el-table-column label="操作" width="90"><template #default="{ row }"><el-button link type="primary" @click="showServiceDetail(row)">查看详情</el-button></template></el-table-column>
          </el-table>

          <h3>Ingress</h3>
          <el-table :data="namespaceDetail.ingresses" border empty-text="没有 Ingress">
            <el-table-column prop="name" label="Ingress" min-width="220" show-overflow-tooltip />
            <el-table-column prop="class_name" label="Class" min-width="140" />
            <el-table-column label="Hosts" min-width="240"><template #default="{ row }">{{ listText(row.hosts) }}</template></el-table-column>
            <el-table-column label="操作" width="90"><template #default="{ row }"><el-button link type="primary" @click="showIngressDetail(row)">查看详情</el-button></template></el-table-column>
          </el-table>

          <h3>PVC</h3>
          <el-table :data="namespaceDetail.persistent_volume_claims" border empty-text="没有 PVC">
            <el-table-column prop="name" label="PVC" min-width="220" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="120" />
            <el-table-column prop="storage_class" label="StorageClass" min-width="160" />
            <el-table-column label="容量" width="140"><template #default="{ row }">{{ row.capacity || row.requested || '-' }}</template></el-table-column>
            <el-table-column label="操作" width="90"><template #default="{ row }"><el-button link type="primary" @click="showPVCDetail(row)">查看详情</el-button></template></el-table-column>
          </el-table>

          <h3>Namespace Conditions</h3>
          <el-table :data="namespaceDetail.conditions" border empty-text="无 Namespace Condition">
            <el-table-column prop="type" label="类型" min-width="220" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
          </el-table>

          <h3>Namespace 自身事件</h3>
          <el-table :data="namespaceDetail.events" border empty-text="暂无 Namespace Event">
            <el-table-column label="级别" width="90"><template #default="{ row }"><el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag></template></el-table-column>
            <el-table-column prop="reason" label="原因" min-width="180" />
            <el-table-column prop="message" label="消息" min-width="360" show-overflow-tooltip />
            <el-table-column label="最后发生" width="180"><template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template></el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="secretDetailVisible"
      size="64%"
      :title="secretDetail ? `Secret ${secretDetail.namespace}/${secretDetail.name}` : 'Secret 安全详情'"
      @close="clearSecretValue"
    >
      <div v-loading="secretDetailLoading || secretValueLoading">
        <el-alert
          class="detail-scope-alert"
          type="warning"
          :closable="false"
          title="本详情不包含 Secret 值。明文读取仅限 Admin，必须逐 key 二次确认，并通过独立 POST 接口记录审计。"
        />
        <el-descriptions v-if="secretDetail" :column="2" border>
          <el-descriptions-item label="Namespace">{{ secretDetail.namespace }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ secretDetail.type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Immutable">{{ secretDetail.immutable ? '是' : '否' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(secretDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="Labels" :span="2">{{ mapText(secretDetail.labels) }}</el-descriptions-item>
        </el-descriptions>

        <h3 v-if="secretDetail">Keys</h3>
        <el-table v-if="secretDetail" :data="secretDetail.key_details" border empty-text="Secret 无数据 key">
          <el-table-column prop="name" label="Key" min-width="260" show-overflow-tooltip />
          <el-table-column prop="size_bytes" label="大小（bytes）" width="140" />
          <el-table-column label="值" min-width="180">
            <template #default>
              <span class="masked-secret-value">••••••••</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="110">
            <template #default="{ row }">
              <el-button
                v-if="canReadSecretValues"
                link
                type="danger"
                :loading="secretValueLoading"
                @click="showSecretValue(row.name)"
              >
                查看明文
              </el-button>
              <span v-else>仅 Admin</span>
            </template>
          </el-table-column>
        </el-table>

        <el-alert
          v-if="secretValueVisible && secretValue"
          class="secret-value-alert"
          type="error"
          :closable="false"
          title="高敏明文已显示；关闭详情后会立即从页面状态清除。"
        />
        <el-descriptions v-if="secretValueVisible && secretValue" :column="1" border>
          <el-descriptions-item label="Key">{{ secretValue.key }}</el-descriptions-item>
          <el-descriptions-item label="编码">{{ secretValue.encoding }}</el-descriptions-item>
          <el-descriptions-item label="值">
            <pre class="secret-value-view">{{ secretValue.value }}</pre>
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>

    <el-drawer v-model="nodeDetailVisible" size="84%" :title="nodeDetail ? `Node ${nodeDetail.name}` : 'Node 详情'">
      <div v-loading="nodeDetailLoading">
        <el-descriptions v-if="nodeDetail" :column="3" border>
          <el-descriptions-item label="状态"><el-tag :type="statusType(nodeDetail.status)">{{ nodeDetail.status || '-' }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="Roles">{{ listText(nodeDetail.roles) }}</el-descriptions-item>
          <el-descriptions-item label="禁止调度"><el-tag :type="nodeDetail.unschedulable ? 'warning' : 'success'">{{ nodeDetail.unschedulable ? '是' : '否' }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="Addresses" :span="3">{{ nodeAddressesText() }}</el-descriptions-item>
          <el-descriptions-item label="Pod CIDRs" :span="3">{{ listText(nodeDetail.pod_cidrs) }}</el-descriptions-item>
          <el-descriptions-item label="Capacity" :span="3">{{ resourceText(nodeDetail.capacity) }}</el-descriptions-item>
          <el-descriptions-item label="Allocatable" :span="3">{{ resourceText(nodeDetail.allocatable) }}</el-descriptions-item>
          <el-descriptions-item label="Pod Requests" :span="3">{{ resourceText(nodeDetail.allocated.requests) }}</el-descriptions-item>
          <el-descriptions-item label="Pod Limits" :span="3">{{ resourceText(nodeDetail.allocated.limits) }}</el-descriptions-item>
          <el-descriptions-item label="Requests 占比">CPU {{ percentText(nodeDetail.allocated.cpu_request_percent) }} / Memory {{ percentText(nodeDetail.allocated.memory_request_percent) }}</el-descriptions-item>
          <el-descriptions-item label="Pod 占比">{{ percentText(nodeDetail.allocated.pod_percent) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(nodeDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="系统" :span="3">{{ nodeDetail.system_info.os_image || '-' }} / {{ nodeDetail.system_info.kernel_version || '-' }} / {{ nodeDetail.system_info.architecture || '-' }}</el-descriptions-item>
          <el-descriptions-item label="运行时">{{ nodeDetail.system_info.container_runtime_version || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Kubelet">{{ nodeDetail.system_info.kubelet_version || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Kube Proxy">{{ nodeDetail.system_info.kube_proxy_version || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Labels" :span="3">{{ mapText(nodeDetail.labels) }}</el-descriptions-item>
        </el-descriptions>

        <template v-if="nodeDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="Node 详情只提供只读诊断，不开放 cordon/drain、污点修改或指定 Pod 节点调度。Capacity、Allocatable、Requests/Limits 都是声明值，不是实时使用率。"
          />

          <h3>Conditions</h3>
          <el-table :data="nodeDetail.conditions" border empty-text="无 Node Condition">
            <el-table-column prop="type" label="类型" min-width="180" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="200" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="340" show-overflow-tooltip />
            <el-table-column label="最后转换" width="180"><template #default="{ row }">{{ formatDateTime(row.last_transition_time) }}</template></el-table-column>
          </el-table>

          <h3>Taints</h3>
          <el-table :data="nodeDetail.taints" border empty-text="无 Taint">
            <el-table-column prop="key" label="Key" min-width="220" />
            <el-table-column prop="value" label="Value" min-width="160" />
            <el-table-column prop="effect" label="Effect" min-width="160" />
            <el-table-column label="添加时间" width="180"><template #default="{ row }">{{ formatDateTime(row.time_added) }}</template></el-table-column>
          </el-table>

          <h3>调度到该 Node 的 Pod</h3>
          <el-table :data="nodeDetail.pods" border empty-text="该 Node 没有 Pod">
            <el-table-column prop="namespace" label="Namespace" min-width="150" />
            <el-table-column prop="name" label="Pod" min-width="220" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="150"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ row.status || row.phase || '-' }}</el-tag></template></el-table-column>
            <el-table-column prop="restart_count" label="重启" width="80" />
            <el-table-column label="操作" width="126" fixed="right"><template #default="{ row }"><el-tooltip content="Pod 详情"><el-button :icon="InfoFilled" circle @click="showPodDetail(row)" /></el-tooltip><el-tooltip content="Pod 历史 / 实时日志"><el-button :icon="Tickets" circle @click="openLogs(row)" /></el-tooltip></template></el-table-column>
          </el-table>

          <h3>顶层 Workload</h3>
          <el-table :data="nodeDetail.workloads" border empty-text="没有可识别的顶层 Workload">
            <el-table-column prop="namespace" label="Namespace" min-width="160" />
            <el-table-column prop="kind" label="类型" min-width="160" />
            <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
            <el-table-column label="操作" width="90"><template #default="{ row }"><el-button link type="primary" @click="openStorageWorkload(row, row.namespace)">查看详情</el-button></template></el-table-column>
          </el-table>

          <h3>Node 自身事件</h3>
          <el-table :data="nodeDetail.events" border empty-text="暂无 Node Event">
            <el-table-column label="级别" width="90"><template #default="{ row }"><el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag></template></el-table-column>
            <el-table-column prop="reason" label="原因" min-width="180" />
            <el-table-column prop="message" label="消息" min-width="360" show-overflow-tooltip />
            <el-table-column label="最后发生" width="180"><template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template></el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="pvcDetailVisible"
      size="82%"
      :title="pvcDetail ? `PVC ${pvcDetail.namespace}/${pvcDetail.name}` : 'PVC 存储关联详情'"
    >
      <div v-loading="pvcDetailLoading">
        <el-descriptions v-if="pvcDetail" :column="3" border>
          <el-descriptions-item label="状态">
            <el-tag :type="statusType(pvcDetail.status)">{{ pvcDetail.status || '-' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="容量">
            {{ pvcDetail.capacity || pvcDetail.requested || '-' }} / 请求 {{ pvcDetail.requested || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="VolumeMode">{{ pvcDetail.volume_mode || '-' }}</el-descriptions-item>
          <el-descriptions-item label="StorageClass">{{ pvcDetail.storage_class || '-' }}</el-descriptions-item>
          <el-descriptions-item label="访问模式">{{ listText(pvcDetail.access_modes) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(pvcDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="绑定 PV" :span="3">
            <el-button v-if="pvcDetail.pv" link type="primary" @click="showPVDetail(pvcDetail.pv)">
              {{ pvcDetail.pv.name }}
            </el-button>
            <span v-else>{{ pvcDetail.volume_name ? `${pvcDetail.volume_name}（绑定校验未通过或 PV 不存在）` : '未绑定' }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="Selector" :span="3">
            {{ mapText(pvcDetail.selector) }}
            <template v-if="pvcDetail.selector_expressions?.length">
             ；{{ listText(pvcDetail.selector_expressions) }}
            </template>
          </el-descriptions-item>
          <el-descriptions-item label="DataSource" :span="3">{{ volumeDataSourceText(pvcDetail) }}</el-descriptions-item>
        </el-descriptions>

        <template v-if="pvcDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="该详情为只读诊断链路。平台不会在这里删除、扩容或修改 PVC/PV/StorageClass；卷源只返回安全摘要，不展示 CSI attributes、volume handle 或 SecretRef。"
          />

          <h3>使用该 PVC 的 Pod</h3>
          <el-table :data="pvcDetail.pods" border empty-text="当前没有 Pod 引用该 PVC">
            <el-table-column prop="name" label="Pod" min-width="220" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="150">
              <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ row.status || row.phase || '-' }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="150" show-overflow-tooltip />
            <el-table-column label="容器" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ containerNames(row) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Pod 详情"><el-button :icon="InfoFilled" circle @click="showPodDetail(row)" /></el-tooltip>
                <el-tooltip content="Pod 历史 / 实时日志"><el-button :icon="Tickets" circle @click="openLogs(row)" /></el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>容器挂载 / 块设备</h3>
          <el-table :data="pvcDetail.mounts" border empty-text="Pod 已引用 Claim，但未发现容器挂载或块设备路径">
            <el-table-column prop="pod_name" label="Pod" min-width="200" show-overflow-tooltip />
            <el-table-column prop="volume_name" label="Volume" min-width="140" show-overflow-tooltip />
            <el-table-column prop="container_type" label="容器类型" width="150" />
            <el-table-column prop="container_name" label="容器" min-width="150" show-overflow-tooltip />
            <el-table-column label="挂载 / 设备路径" min-width="260" show-overflow-tooltip>
              <template #default="{ row }">{{ mountTarget(row) }}</template>
            </el-table-column>
            <el-table-column label="只读" width="80">
              <template #default="{ row }"><el-tag :type="row.read_only ? 'warning' : 'info'">{{ row.read_only ? '是' : '否' }}</el-tag></template>
            </el-table-column>
          </el-table>

          <h3>顶层 Workload</h3>
          <el-table :data="pvcDetail.workloads" border empty-text="独立 Pod 或无可识别控制器">
            <el-table-column prop="kind" label="类型" min-width="150" />
            <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="primary" @click="openStorageWorkload(row, pvcDetail.namespace)">查看详情</el-button>
              </template>
            </el-table-column>
          </el-table>

          <h3>PVC Conditions</h3>
          <el-table :data="pvcDetail.conditions" border empty-text="无 PVC Condition">
            <el-table-column prop="type" label="类型" min-width="180" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="pvcDetail.events" border empty-text="暂无 PVC / PV / Pod / Workload Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }"><el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag></template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180"><template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template></el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="pvDetailVisible"
      size="82%"
      :title="pvDetail ? `PV ${pvDetail.name}` : 'PV 存储关联详情'"
    >
      <div v-loading="pvDetailLoading">
        <el-descriptions v-if="pvDetail" :column="3" border>
          <el-descriptions-item label="状态"><el-tag :type="statusType(pvDetail.status)">{{ pvDetail.status || '-' }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="容量">{{ pvDetail.capacity || '-' }}</el-descriptions-item>
          <el-descriptions-item label="VolumeMode">{{ pvDetail.volume_mode || '-' }}</el-descriptions-item>
          <el-descriptions-item label="StorageClass">{{ pvDetail.storage_class || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回收策略">{{ pvDetail.reclaim_policy || '-' }}</el-descriptions-item>
          <el-descriptions-item label="访问模式">{{ listText(pvDetail.access_modes) }}</el-descriptions-item>
          <el-descriptions-item label="绑定 PVC" :span="3">
            <el-button v-if="pvDetail.pvc" link type="primary" @click="showPVCDetail(pvDetail.pvc)">
              {{ pvDetail.pvc.namespace }}/{{ pvDetail.pvc.name }}
            </el-button>
            <span v-else>{{ pvDetail.claim_name ? `${pvDetail.claim_namespace}/${pvDetail.claim_name}（绑定校验未通过或 PVC 不存在）` : '未绑定' }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="卷源类型">{{ pvDetail.volume_source_type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="卷源安全摘要" :span="2">{{ mapText(pvDetail.volume_source_info) }}</el-descriptions-item>
          <el-descriptions-item label="Mount Options" :span="3">{{ listText(pvDetail.mount_options) }}</el-descriptions-item>
          <el-descriptions-item label="Node Affinity" :span="3">{{ listText(pvDetail.node_affinity) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(pvDetail.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <template v-if="pvDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="PV 仅在 claimRef 与 PVC 的 namespace/name/UID、PVC volumeName 一致时建立反向关联。CSI attributes、volume handle 和所有 SecretRef 不会返回到页面。"
          />

          <h3>使用该 PV 的 Pod</h3>
          <el-table :data="pvDetail.pods" border empty-text="当前没有经有效 PVC 绑定引用该 PV 的 Pod">
            <el-table-column prop="name" label="Pod" min-width="220" show-overflow-tooltip />
            <el-table-column prop="namespace" label="Namespace" min-width="150" />
            <el-table-column prop="status" label="状态" width="150">
              <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ row.status || row.phase || '-' }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="150" show-overflow-tooltip />
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Pod 详情"><el-button :icon="InfoFilled" circle @click="showPodDetail(row)" /></el-tooltip>
                <el-tooltip content="Pod 历史 / 实时日志"><el-button :icon="Tickets" circle @click="openLogs(row)" /></el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>容器挂载 / 块设备</h3>
          <el-table :data="pvDetail.mounts" border empty-text="没有容器挂载或块设备路径">
            <el-table-column prop="pod_name" label="Pod" min-width="200" show-overflow-tooltip />
            <el-table-column prop="volume_name" label="Volume" min-width="140" show-overflow-tooltip />
            <el-table-column prop="container_type" label="容器类型" width="150" />
            <el-table-column prop="container_name" label="容器" min-width="150" show-overflow-tooltip />
            <el-table-column label="挂载 / 设备路径" min-width="260" show-overflow-tooltip><template #default="{ row }">{{ mountTarget(row) }}</template></el-table-column>
            <el-table-column label="只读" width="80"><template #default="{ row }"><el-tag :type="row.read_only ? 'warning' : 'info'">{{ row.read_only ? '是' : '否' }}</el-tag></template></el-table-column>
          </el-table>

          <h3>顶层 Workload</h3>
          <el-table :data="pvDetail.workloads" border empty-text="独立 Pod、无有效 PVC 或无可识别控制器">
            <el-table-column prop="kind" label="类型" min-width="150" />
            <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button v-if="pvDetail.pvc" link type="primary" @click="openStorageWorkload(row, pvDetail.pvc.namespace)">查看详情</el-button>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="pvDetail.events" border empty-text="暂无 PV / PVC / Pod / Workload Event">
            <el-table-column label="级别" width="90"><template #default="{ row }"><el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag></template></el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip><template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template></el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180"><template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template></el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="serviceDetailVisible"
      size="78%"
      :title="serviceDetail ? `Service ${serviceDetail.namespace}/${serviceDetail.name}` : 'Service 详情'"
    >
      <div v-loading="serviceDetailLoading">
        <el-descriptions v-if="serviceDetail" :column="3" border>
          <el-descriptions-item label="类型">{{ serviceDetail.type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="ClusterIPs">{{ listText(serviceDetail.cluster_ips) }}</el-descriptions-item>
          <el-descriptions-item label="External IP">{{ serviceDetail.external_ip || '-' }}</el-descriptions-item>
          <el-descriptions-item label="ExternalName">{{ serviceDetail.external_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="IP Families">{{ listText(serviceDetail.ip_families) }}</el-descriptions-item>
          <el-descriptions-item label="IP Family Policy">{{ serviceDetail.ip_family_policy || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Session Affinity">{{ serviceDetail.session_affinity || '-' }}</el-descriptions-item>
          <el-descriptions-item label="External Traffic Policy">
            {{ serviceDetail.external_traffic_policy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Internal Traffic Policy">
            {{ serviceDetail.internal_traffic_policy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="发布未就绪地址">
            <el-tag :type="serviceDetail.publish_not_ready_addresses ? 'warning' : 'info'">
              {{ serviceDetail.publish_not_ready_addresses ? '是' : '否' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="端点来源">
            {{ serviceDetail.endpoint_source || '无 EndpointSlice / Endpoints' }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(serviceDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="Selector" :span="3">{{ mapText(serviceDetail.selector) }}</el-descriptions-item>
          <el-descriptions-item label="LB Source Ranges" :span="3">
            {{ listText(serviceDetail.load_balancer_source_ranges) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="serviceDetail">
          <el-alert
            v-if="!serviceDetail.selector || Object.keys(serviceDetail.selector).length === 0"
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="该 Service 没有 selector。关联 Pod 只依据 EndpointSlice/Endpoints 的 Pod targetRef；ExternalName 或手工端点也可能没有可关联 Pod。"
          />

          <h3>Service Ports</h3>
          <el-table :data="serviceDetail.port_details" border empty-text="无端口">
            <el-table-column prop="name" label="名称" min-width="140">
              <template #default="{ row }">{{ row.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="protocol" label="协议" width="100" />
            <el-table-column prop="port" label="Service Port" width="120" />
            <el-table-column prop="target_port" label="TargetPort" min-width="140" />
            <el-table-column prop="node_port" label="NodePort" width="110">
              <template #default="{ row }">{{ row.node_port || '-' }}</template>
            </el-table-column>
            <el-table-column prop="app_protocol" label="AppProtocol" min-width="140">
              <template #default="{ row }">{{ row.app_protocol || '-' }}</template>
            </el-table-column>
          </el-table>

          <h3>Endpoints</h3>
          <el-table :data="serviceDetail.endpoints" border empty-text="暂无 EndpointSlice / Endpoints 地址">
            <el-table-column label="来源" min-width="190" show-overflow-tooltip>
              <template #default="{ row }">{{ row.source }}/{{ row.source_name }}</template>
            </el-table-column>
            <el-table-column label="地址" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ listText(row.addresses) }}</template>
            </el-table-column>
            <el-table-column label="端口" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">{{ listText(row.ports) }}</template>
            </el-table-column>
            <el-table-column label="Ready" width="88">
              <template #default="{ row }">
                <el-tag :type="row.ready ? 'success' : 'danger'">{{ row.ready ? 'yes' : 'no' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Serving" width="96">
              <template #default="{ row }">
                <el-tag :type="row.serving ? 'success' : 'info'">{{ row.serving ? 'yes' : 'no' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Terminating" width="116">
              <template #default="{ row }">
                <el-tag :type="row.terminating ? 'warning' : 'info'">{{ row.terminating ? 'yes' : 'no' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Node / Zone" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.node_name || '-' }} / {{ row.zone || '-' }}</template>
            </el-table-column>
            <el-table-column label="TargetRef" min-width="190" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.target_kind && row.target_name ? `${row.target_kind}/${row.target_name}` : '-' }}
              </template>
            </el-table-column>
          </el-table>

          <h3>关联 Pods</h3>
          <el-table :data="serviceDetail.pods" border empty-text="无 selector 命中或 targetRef 关联的 Pod">
            <el-table-column prop="name" label="Pod" min-width="230" show-overflow-tooltip />
            <el-table-column label="状态" width="150">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
            <el-table-column prop="restart_count" label="容器重启" width="100" />
            <el-table-column label="容器" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ containerNames(row) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Pod 详情">
                  <el-button :icon="InfoFilled" circle @click="showPodDetail(row)" />
                </el-tooltip>
                <el-tooltip content="Pod 历史 / 实时日志">
                  <el-button :icon="Tickets" circle @click="openLogs(row)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="serviceDetail.events" border empty-text="暂无 Service / Endpoint / Pod Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="ingressDetailVisible"
      size="82%"
      :title="ingressDetail ? `Ingress ${ingressDetail.namespace}/${ingressDetail.name}` : 'Ingress 详情'"
    >
      <div v-loading="ingressDetailLoading">
        <el-descriptions v-if="ingressDetail" :column="3" border>
          <el-descriptions-item label="IngressClass">{{ ingressDetail.class_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Addresses">{{ listText(ingressDetail.addresses) }}</el-descriptions-item>
          <el-descriptions-item label="TLS">
            <el-tag :type="ingressDetail.tls ? 'success' : 'info'">{{ ingressDetail.tls ? 'on' : 'off' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Hosts" :span="2">{{ listText(ingressDetail.hosts) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(ingressDetail.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <template v-if="ingressDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="Ingress 详情只展示当前规则和后端关联。公网 IP :80 能否无 Host 访问由实际 Ingress 规则决定，Host-only 规则不会自动匹配 IP；无 Host/IP 入口仍是后续独立部署任务。"
          />

          <h3>Rules / Backends</h3>
          <el-table :data="ingressDetail.backends" border empty-text="无默认后端或 HTTP 路径规则">
            <el-table-column label="类型" width="100">
              <template #default="{ row }">
                <el-tag :type="row.is_default ? 'warning' : 'info'">{{ row.is_default ? 'Default' : 'Rule' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="host" label="Host" min-width="210">
              <template #default="{ row }">{{ row.host || '*' }}</template>
            </el-table-column>
            <el-table-column prop="path" label="Path" min-width="150">
              <template #default="{ row }">{{ row.path || '-' }}</template>
            </el-table-column>
            <el-table-column prop="path_type" label="PathType" width="120">
              <template #default="{ row }">{{ row.path_type || '-' }}</template>
            </el-table-column>
            <el-table-column label="Backend" min-width="230" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.backend_kind && row.backend_name ? `${row.backend_api_group ? `${row.backend_api_group}/` : ''}${row.backend_kind}/${row.backend_name}${row.backend_kind === 'Service' ? `:${row.backend_port || '-'}` : ''}` : '-' }}
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag
                  v-if="row.backend_kind === 'Service'"
                  :type="row.service_found && row.service_port_found ? 'success' : 'danger'"
                >
                  {{ !row.service_found ? 'Service 缺失' : row.service_port_found ? 'Service/端口正常' : 'Service 端口缺失' }}
                </el-tag>
                <el-tag v-else type="info">Resource</el-tag>
              </template>
            </el-table-column>
          </el-table>

          <h3>TLS</h3>
          <el-table :data="ingressDetail.tls_details" border empty-text="未配置 TLS">
            <el-table-column label="Hosts" min-width="280">
              <template #default="{ row }">{{ listText(row.hosts) }}</template>
            </el-table-column>
            <el-table-column prop="secret_name" label="Secret" min-width="220">
              <template #default="{ row }">{{ row.secret_name || 'Ingress controller 默认证书' }}</template>
            </el-table-column>
          </el-table>

          <h3>后端 Services</h3>
          <el-table :data="ingressDetail.services" border empty-text="无可解析的后端 Service">
            <el-table-column prop="name" label="Service" min-width="210" show-overflow-tooltip />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column prop="cluster_ip" label="Cluster IP" min-width="150" />
            <el-table-column label="端口" min-width="190" show-overflow-tooltip>
              <template #default="{ row }">{{ listText(row.ports) }}</template>
            </el-table-column>
            <el-table-column prop="endpoint_source" label="端点来源" width="130">
              <template #default="{ row }">{{ row.endpoint_source || '-' }}</template>
            </el-table-column>
            <el-table-column label="Endpoint / Pod" width="140">
              <template #default="{ row }">{{ row.endpoints.length }} / {{ row.pods.length }}</template>
            </el-table-column>
            <el-table-column label="操作" width="78" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="打开完整 Service / Endpoint / Pod 详情">
                  <el-button :icon="InfoFilled" circle @click="showServiceDetail(row)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <template v-for="backendService in ingressDetail.services" :key="backendService.name">
            <h3>{{ backendService.name }} Endpoints</h3>
            <el-table :data="backendService.endpoints" border empty-text="无 EndpointSlice / Endpoints 地址">
              <el-table-column label="来源" min-width="190">
                <template #default="{ row }">{{ row.source }}/{{ row.source_name }}</template>
              </el-table-column>
              <el-table-column label="地址" min-width="180">
                <template #default="{ row }">{{ listText(row.addresses) }}</template>
              </el-table-column>
              <el-table-column label="端口" min-width="160">
                <template #default="{ row }">{{ listText(row.ports) }}</template>
              </el-table-column>
              <el-table-column label="Ready / Serving / Terminating" min-width="220">
                <template #default="{ row }">
                  {{ row.ready ? 'yes' : 'no' }} / {{ row.serving ? 'yes' : 'no' }} / {{ row.terminating ? 'yes' : 'no' }}
                </template>
              </el-table-column>
              <el-table-column label="TargetRef" min-width="190">
                <template #default="{ row }">
                  {{ row.target_kind && row.target_name ? `${row.target_kind}/${row.target_name}` : '-' }}
                </template>
              </el-table-column>
            </el-table>

            <h3>{{ backendService.name }} Pods</h3>
            <el-table :data="backendService.pods" border empty-text="无 selector/targetRef 关联 Pod">
              <el-table-column prop="name" label="Pod" min-width="230" show-overflow-tooltip />
              <el-table-column label="状态" width="150">
                <template #default="{ row }">
                  <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
              <el-table-column prop="restart_count" label="容器重启" width="100" />
              <el-table-column label="操作" width="126" fixed="right">
                <template #default="{ row }">
                  <el-tooltip content="Pod 详情">
                    <el-button :icon="InfoFilled" circle @click="showPodDetail(row)" />
                  </el-tooltip>
                  <el-tooltip content="Pod 历史 / 实时日志">
                    <el-button :icon="Tickets" circle @click="openLogs(row)" />
                  </el-tooltip>
                </template>
              </el-table-column>
            </el-table>
          </template>

          <h3>关联事件</h3>
          <el-table :data="ingressDetail.events" border empty-text="暂无 Ingress / Service / Endpoint / Pod Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="podDetailVisible"
      size="68%"
      :title="podDetail ? `Pod ${podDetail.namespace}/${podDetail.name}` : 'Pod 详情'"
    >
      <div v-loading="podDetailLoading">
        <el-descriptions v-if="podDetail" :column="2" border>
          <el-descriptions-item label="状态">
            <el-tag :type="podDetail.ready ? 'success' : 'danger'">{{ podDetail.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Phase">{{ podDetail.phase || '-' }}</el-descriptions-item>
          <el-descriptions-item label="节点">{{ podDetail.node_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Pod IP / Host IP">
            {{ podDetail.pod_ip || '-' }} / {{ podDetail.host_ip || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="ServiceAccount">{{ podDetail.service_account || '-' }}</el-descriptions-item>
          <el-descriptions-item label="QoS / 重启策略">
            {{ podDetail.qos_class || '-' }} / {{ podDetail.restart_policy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="所属控制器" :span="2">{{ controllerText() }}</el-descriptions-item>
          <el-descriptions-item label="Labels" :span="2">{{ mapText(podDetail.labels) }}</el-descriptions-item>
          <el-descriptions-item label="启动时间">{{ formatDateTime(podDetail.start_time) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(podDetail.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <template v-if="podDetail">
          <h3>容器状态</h3>
          <el-table :data="podDetail.containers" border empty-text="无容器状态">
            <el-table-column prop="name" label="容器" min-width="140" />
            <el-table-column prop="image" label="镜像" min-width="220" show-overflow-tooltip />
            <el-table-column label="当前状态" min-width="150">
              <template #default="{ row }">{{ row.state }}{{ row.reason ? ` / ${row.reason}` : '' }}</template>
            </el-table-column>
            <el-table-column prop="restart_count" label="重启" width="80" />
            <el-table-column label="上次退出" min-width="190">
              <template #default="{ row }">
                {{ row.last_state ? `${row.last_reason || row.last_state} (exit ${row.last_exit_code})` : '-' }}
              </template>
            </el-table-column>
          </el-table>

          <h3>Pod Conditions</h3>
          <el-table :data="podDetail.conditions" border empty-text="无 Condition">
            <el-table-column prop="type" label="类型" min-width="160" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="260" show-overflow-tooltip />
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="podEvents" border empty-text="暂无关联 Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="150" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="deploymentDetailVisible"
      size="76%"
      :title="deploymentDetail ? `Deployment ${deploymentDetail.namespace}/${deploymentDetail.name}` : 'Deployment 详情'"
    >
      <div v-loading="deploymentDetailLoading">
        <el-descriptions v-if="deploymentDetail" :column="3" border>
          <el-descriptions-item label="副本">
            {{ deploymentDetail.ready_replicas }}/{{ deploymentDetail.replicas }} Ready
          </el-descriptions-item>
          <el-descriptions-item label="已更新/可用">
            {{ deploymentDetail.updated_replicas }}/{{ deploymentDetail.available_replicas }}
          </el-descriptions-item>
          <el-descriptions-item label="不可用">
            {{ deploymentDetail.unavailable_replicas }}
          </el-descriptions-item>
          <el-descriptions-item label="发布策略">
            {{ deploymentDetail.strategy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="MaxSurge / MaxUnavailable">
            {{ deploymentDetail.max_surge || '-' }} / {{ deploymentDetail.max_unavailable || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="暂停">
            <el-tag :type="deploymentDetail.paused ? 'warning' : 'success'">
              {{ deploymentDetail.paused ? '是' : '否' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Generation">
            {{ deploymentDetail.observed_generation }}/{{ deploymentDetail.generation }} observed
          </el-descriptions-item>
          <el-descriptions-item label="MinReady / Progress Deadline">
            {{ deploymentDetail.min_ready_seconds }}s /
            {{ deploymentDetail.progress_deadline_seconds ? `${deploymentDetail.progress_deadline_seconds}s` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Revision History">
            {{ deploymentDetail.revision_history_limit }}
          </el-descriptions-item>
          <el-descriptions-item label="Selector" :span="3">
            {{ mapText(deploymentDetail.selector) }}
          </el-descriptions-item>
          <el-descriptions-item label="镜像" :span="3">
            {{ listText(deploymentDetail.images) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="deploymentDetail">
          <h3>Deployment Conditions</h3>
          <el-table :data="deploymentDetail.conditions" border empty-text="无 Condition">
            <el-table-column prop="type" label="类型" min-width="150" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="300" show-overflow-tooltip />
            <el-table-column label="最后更新" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_update_time) }}</template>
            </el-table-column>
          </el-table>

          <h3>直属 ReplicaSets</h3>
          <el-table :data="deploymentDetail.replica_sets" border empty-text="无直属 ReplicaSet">
            <el-table-column prop="name" label="ReplicaSet" min-width="230" show-overflow-tooltip />
            <el-table-column prop="revision" label="Revision" width="100">
              <template #default="{ row }">{{ row.revision || '-' }}</template>
            </el-table-column>
            <el-table-column label="期望/当前" width="120">
              <template #default="{ row }">{{ row.replicas }}/{{ row.current_replicas }}</template>
            </el-table-column>
            <el-table-column label="就绪/可用/已标记" width="150">
              <template #default="{ row }">
                {{ row.ready_replicas }}/{{ row.available_replicas }}/{{ row.fully_labeled_replicas }}
              </template>
            </el-table-column>
            <el-table-column label="镜像" min-width="250" show-overflow-tooltip>
              <template #default="{ row }">{{ listText(row.images) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="ReplicaSet 详情">
                  <el-button :icon="InfoFilled" circle @click="showReplicaSetDetail(row)" />
                </el-tooltip>
                <el-tooltip content="ReplicaSet YAML">
                  <el-button :icon="View" circle @click="showYAML('replicaset', row.name, row.namespace)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联 Pods</h3>
          <el-table :data="deploymentDetail.pods" border empty-text="无关联 Pod">
            <el-table-column prop="name" label="Pod" min-width="230" show-overflow-tooltip />
            <el-table-column label="状态" width="150">
              <template #default="{ row }">
                <el-tag :type="row.ready ? 'success' : 'danger'">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
            <el-table-column prop="restart_count" label="重启" width="80" />
            <el-table-column label="容器" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ containerNames(row) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Pod 详情">
                  <el-button :icon="InfoFilled" circle @click="showPodDetail(row)" />
                </el-tooltip>
                <el-tooltip content="Pod 日志">
                  <el-button :icon="Tickets" circle @click="openLogs(row)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="deploymentDetail.events" border empty-text="暂无 Deployment / ReplicaSet / Pod Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="replicaSetDetailVisible"
      size="72%"
      :title="replicaSetDetail ? `ReplicaSet ${replicaSetDetail.namespace}/${replicaSetDetail.name}` : 'ReplicaSet 详情'"
    >
      <div v-loading="replicaSetDetailLoading">
        <el-descriptions v-if="replicaSetDetail" :column="3" border>
          <el-descriptions-item label="所属控制器">
            <template v-if="replicaSetDetail.owner">
              {{ replicaSetDetail.owner.kind }}/{{ replicaSetDetail.owner.name }}
              <el-button
                v-if="replicaSetDetail.owner.kind === 'Deployment'"
                link
                type="primary"
                @click="showDeploymentDetail({ namespace: replicaSetDetail.namespace, name: replicaSetDetail.owner.name })"
              >
                打开 Deployment
              </el-button>
            </template>
            <template v-else>独立 ReplicaSet</template>
          </el-descriptions-item>
          <el-descriptions-item label="Revision">
            {{ replicaSetDetail.revision || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Observed Generation">
            {{ replicaSetDetail.observed_generation }}
          </el-descriptions-item>
          <el-descriptions-item label="期望/当前">
            {{ replicaSetDetail.replicas }}/{{ replicaSetDetail.current_replicas }}
          </el-descriptions-item>
          <el-descriptions-item label="就绪/可用">
            {{ replicaSetDetail.ready_replicas }}/{{ replicaSetDetail.available_replicas }}
          </el-descriptions-item>
          <el-descriptions-item label="完全匹配/不可用">
            {{ replicaSetDetail.fully_labeled_replicas }}/{{ replicaSetDetail.unavailable_replicas }}
          </el-descriptions-item>
          <el-descriptions-item label="MinReadySeconds">
            {{ replicaSetDetail.min_ready_seconds }}s
          </el-descriptions-item>
          <el-descriptions-item label="Selector" :span="2">
            {{ mapText(replicaSetDetail.selector) }}
          </el-descriptions-item>
          <el-descriptions-item label="镜像" :span="3">
            {{ listText(replicaSetDetail.images) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="replicaSetDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="ReplicaSet 通常由 Deployment 管理；平台仅提供诊断详情，不开放直接扩缩容或删除，避免与 Deployment 控制器冲突。"
          />

          <h3>ReplicaSet Conditions</h3>
          <el-table :data="replicaSetDetail.conditions" border empty-text="无 Condition">
            <el-table-column prop="type" label="类型" min-width="160" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column label="最后变化" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_transition_time) }}</template>
            </el-table-column>
          </el-table>

          <h3>直属 Pods</h3>
          <el-table :data="replicaSetDetail.pods" border empty-text="无直属 Pod">
            <el-table-column prop="name" label="Pod" min-width="230" show-overflow-tooltip />
            <el-table-column label="状态" width="150">
              <template #default="{ row }">
                <el-tag :type="row.ready ? 'success' : 'danger'">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
            <el-table-column prop="restart_count" label="重启" width="80" />
            <el-table-column label="容器" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ containerNames(row) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Pod 详情">
                  <el-button :icon="InfoFilled" circle @click="showPodDetail(row)" />
                </el-tooltip>
                <el-tooltip content="Pod 日志">
                  <el-button :icon="Tickets" circle @click="openLogs(row)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="replicaSetDetail.events" border empty-text="暂无 ReplicaSet / Pod Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="jobDetailVisible"
      size="72%"
      :title="jobDetail ? `Job ${jobDetail.namespace}/${jobDetail.name}` : 'Job 详情'"
    >
      <div v-loading="jobDetailLoading">
        <el-descriptions v-if="jobDetail" :column="3" border>
          <el-descriptions-item label="状态">
            <el-tag :type="jobStatusType(jobDetail)">{{ jobStatus(jobDetail) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="所属控制器">
            <template v-if="jobDetail.owner">
              {{ jobDetail.owner.kind }}/{{ jobDetail.owner.name }}
              <el-button
                v-if="jobDetail.owner.kind === 'CronJob'"
                link
                type="primary"
                @click="showCronJobDetail({ namespace: jobDetail.namespace, name: jobDetail.owner.name })"
              >
                打开 CronJob
              </el-button>
            </template>
            <template v-else>手动创建 / 无 CronJob owner</template>
          </el-descriptions-item>
          <el-descriptions-item label="CompletionMode">
            {{ jobDetail.completion_mode || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="成功/失败/运行中">
            {{ jobDetail.succeeded }}/{{ jobDetail.failed }}/{{ jobDetail.active }}
          </el-descriptions-item>
          <el-descriptions-item label="Parallelism / Completions">
            {{ jobDetail.parallelism }}/{{ jobDetail.completions }}
          </el-descriptions-item>
          <el-descriptions-item label="BackoffLimit">
            {{ jobDetail.backoff_limit }}
          </el-descriptions-item>
          <el-descriptions-item label="ActiveDeadlineSeconds">
            {{ jobDetail.active_deadline_seconds || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="完成后 TTL">
            {{ jobDetail.ttl_seconds_after_finished ? `${jobDetail.ttl_seconds_after_finished}s` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Suspend / ManualSelector">
            {{ jobDetail.suspend ? '是' : '否' }} / {{ jobDetail.manual_selector ? '是' : '否' }}
          </el-descriptions-item>
          <el-descriptions-item label="开始时间">
            {{ formatDateTime(jobDetail.start_time) }}
          </el-descriptions-item>
          <el-descriptions-item label="完成时间">
            {{ formatDateTime(jobDetail.completion_time) }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ formatDateTime(jobDetail.created_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="Selector" :span="3">
            {{ mapText(jobDetail.selector) }}
          </el-descriptions-item>
          <el-descriptions-item label="镜像" :span="3">
            {{ listText(jobDetail.images) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="jobDetail">
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="平台当前不开放 Job 删除或重新运行。Job Pod 也不提供“重启”：删除运行中 Pod 会影响失败计数，删除已完成 Pod 不会重新执行 Job；显式 Pod 删除仍是独立高风险操作。"
          />

          <h3>Job Conditions</h3>
          <el-table :data="jobDetail.conditions" border empty-text="无 Condition">
            <el-table-column prop="type" label="类型" min-width="160" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column label="最后变化" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_transition_time) }}</template>
            </el-table-column>
          </el-table>

          <h3>直属 Pods</h3>
          <el-table :data="jobDetail.pods" border empty-text="无直属 Pod">
            <el-table-column prop="name" label="Pod" min-width="230" show-overflow-tooltip />
            <el-table-column label="状态" width="150">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="node_name" label="节点" min-width="160" show-overflow-tooltip />
            <el-table-column prop="restart_count" label="容器重启" width="100" />
            <el-table-column label="容器" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ containerNames(row) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Pod 详情">
                  <el-button :icon="InfoFilled" circle @click="showPodDetail(row)" />
                </el-tooltip>
                <el-tooltip content="Pod 历史 / 实时日志">
                  <el-button :icon="Tickets" circle @click="openLogs(row)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="jobDetail.events" border empty-text="暂无 Job / Pod Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="cronJobDetailVisible"
      size="76%"
      :title="cronJobDetail ? `CronJob ${cronJobDetail.namespace}/${cronJobDetail.name}` : 'CronJob 详情'"
    >
      <div v-loading="cronJobDetailLoading">
        <el-descriptions v-if="cronJobDetail" :column="3" border>
          <el-descriptions-item label="Schedule">{{ cronJobDetail.schedule || '-' }}</el-descriptions-item>
          <el-descriptions-item label="TimeZone">{{ cronJobDetail.time_zone || '控制器默认时区' }}</el-descriptions-item>
          <el-descriptions-item label="ConcurrencyPolicy">
            {{ cronJobDetail.concurrency_policy || 'Allow' }}
          </el-descriptions-item>
          <el-descriptions-item label="暂停">
            <el-tag :type="cronJobDetail.suspend ? 'warning' : 'success'">
              {{ cronJobDetail.suspend ? '是' : '否' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="运行中 Job">{{ cronJobDetail.active }}</el-descriptions-item>
          <el-descriptions-item label="StartingDeadlineSeconds">
            {{ cronJobDetail.starting_deadline_seconds || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="成功/失败历史保留">
            {{ cronJobDetail.successful_jobs_history_limit }}/{{ cronJobDetail.failed_jobs_history_limit }}
          </el-descriptions-item>
          <el-descriptions-item label="上次调度">
            {{ formatDateTime(cronJobDetail.last_schedule_time) }}
          </el-descriptions-item>
          <el-descriptions-item label="上次成功">
            {{ formatDateTime(cronJobDetail.last_successful_time) }}
          </el-descriptions-item>
          <el-descriptions-item label="Job Parallelism / Completions">
            {{ cronJobDetail.job_template.parallelism }}/{{ cronJobDetail.job_template.completions }}
          </el-descriptions-item>
          <el-descriptions-item label="Job BackoffLimit">
            {{ cronJobDetail.job_template.backoff_limit }}
          </el-descriptions-item>
          <el-descriptions-item label="CompletionMode">
            {{ cronJobDetail.job_template.completion_mode || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="ActiveDeadlineSeconds">
            {{ cronJobDetail.job_template.active_deadline_seconds || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="完成后 TTL">
            {{ cronJobDetail.job_template.ttl_seconds_after_finished ? `${cronJobDetail.job_template.ttl_seconds_after_finished}s` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Pod RestartPolicy">
            {{ cronJobDetail.job_template.restart_policy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Job 镜像" :span="3">
            {{ listText(cronJobDetail.job_template.images) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="cronJobDetail">
          <div class="drawer-toolbar detail-action-toolbar">
            <el-button
              :type="cronJobDetail.suspend ? 'success' : 'warning'"
              @click="setCronJobSuspended(cronJobDetail, !cronJobDetail.suspend)"
            >
              {{ cronJobDetail.suspend ? '恢复未来调度' : '暂停未来调度' }}
            </el-button>
          </div>
          <el-alert
            class="detail-scope-alert"
            type="info"
            :closable="false"
            title="暂停只影响未来调度，不会停止已经开始的 Job。平台当前不开放立即运行或删除 CronJob；立即运行需要创建唯一名称的新 Job，并单独设计创建权限、确认和审计。"
          />

          <h3>历史 Jobs</h3>
          <el-table :data="cronJobDetail.jobs" border empty-text="暂无保留的直属 Job">
            <el-table-column prop="name" label="Job" min-width="230" show-overflow-tooltip />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="jobStatusType(row)">{{ jobStatus(row) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="成功/失败/运行" width="130">
              <template #default="{ row }">{{ row.succeeded }}/{{ row.failed }}/{{ row.active }}</template>
            </el-table-column>
            <el-table-column label="Pod" width="80">
              <template #default="{ row }">{{ row.pods.length }}</template>
            </el-table-column>
            <el-table-column label="开始时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.start_time) }}</template>
            </el-table-column>
            <el-table-column label="完成时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.completion_time) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="126" fixed="right">
              <template #default="{ row }">
                <el-tooltip content="Job / Pod / 日志 / Event 详情">
                  <el-button :icon="InfoFilled" circle @click="showJobDetail(row)" />
                </el-tooltip>
                <el-tooltip content="Job YAML">
                  <el-button :icon="View" circle @click="showYAML('job', row.name, row.namespace)" />
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>

          <h3>关联事件</h3>
          <el-table :data="cronJobDetail.events" border empty-text="暂无 CronJob / Job / Pod Event">
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="对象" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">{{ row.involved_kind }}/{{ row.involved_name }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip />
            <el-table-column prop="message" label="消息" min-width="320" show-overflow-tooltip />
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最后发生" width="180">
              <template #default="{ row }">{{ formatDateTime(row.last_timestamp) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="statefulSetDetailVisible"
      size="58%"
      :title="statefulSetDetail ? `StatefulSet ${statefulSetDetail.namespace}/${statefulSetDetail.name}` : 'StatefulSet 详情'"
    >
      <div v-loading="statefulSetDetailLoading">
        <el-descriptions v-if="statefulSetDetail" :column="2" border>
          <el-descriptions-item label="副本">
            {{ statefulSetDetail.ready_replicas }}/{{ statefulSetDetail.replicas }} Ready
          </el-descriptions-item>
          <el-descriptions-item label="当前/已更新">
            {{ statefulSetDetail.current_replicas }}/{{ statefulSetDetail.updated_replicas }}
          </el-descriptions-item>
          <el-descriptions-item label="Headless Service">
            {{ statefulSetDetail.service_name || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Pod 管理策略">
            {{ statefulSetDetail.pod_management_policy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="更新策略">
            {{ statefulSetDetail.update_strategy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="镜像">
            {{ listText(statefulSetDetail.images) }}
          </el-descriptions-item>
          <el-descriptions-item label="当前 Revision">
            {{ statefulSetDetail.current_revision || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="目标 Revision">
            {{ statefulSetDetail.update_revision || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Selector" :span="2">
            {{ mapText(statefulSetDetail.selector) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="statefulSetDetail">
          <h3>VolumeClaimTemplates</h3>
          <el-table :data="statefulSetDetail.volume_claims" border empty-text="无 VolumeClaimTemplate">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="storage_class" label="StorageClass" min-width="160">
              <template #default="{ row }">{{ row.storage_class || '-' }}</template>
            </el-table-column>
            <el-table-column prop="requested_storage" label="申请容量" width="130">
              <template #default="{ row }">{{ row.requested_storage || '-' }}</template>
            </el-table-column>
            <el-table-column label="访问模式" min-width="160">
              <template #default="{ row }">{{ listText(row.access_modes) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-drawer>

    <el-drawer
      v-model="daemonSetDetailVisible"
      size="58%"
      :title="daemonSetDetail ? `DaemonSet ${daemonSetDetail.namespace}/${daemonSetDetail.name}` : 'DaemonSet 详情'"
    >
      <div v-loading="daemonSetDetailLoading">
        <el-descriptions v-if="daemonSetDetail" :column="2" border>
          <el-descriptions-item label="期望/当前">
            {{ daemonSetDetail.desired_number }}/{{ daemonSetDetail.current_number }}
          </el-descriptions-item>
          <el-descriptions-item label="就绪/可用">
            {{ daemonSetDetail.ready_number }}/{{ daemonSetDetail.available_number }}
          </el-descriptions-item>
          <el-descriptions-item label="已更新/不可用">
            {{ daemonSetDetail.updated_number }}/{{ daemonSetDetail.unavailable_number }}
          </el-descriptions-item>
          <el-descriptions-item label="误调度">
            {{ daemonSetDetail.misscheduled_number }}
          </el-descriptions-item>
          <el-descriptions-item label="更新策略">
            {{ daemonSetDetail.update_strategy || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="镜像">
            {{ listText(daemonSetDetail.images) }}
          </el-descriptions-item>
          <el-descriptions-item label="Selector" :span="2">
            {{ mapText(daemonSetDetail.selector) }}
          </el-descriptions-item>
          <el-descriptions-item label="NodeSelector" :span="2">
            {{ mapText(daemonSetDetail.node_selector) }}
          </el-descriptions-item>
          <el-descriptions-item label="Tolerations" :span="2">
            {{ listText(daemonSetDetail.tolerations) }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>

    <el-drawer v-model="logsVisible" size="72%" title="Pod 日志">
      <el-tabs v-model="logMode" class="log-mode-tabs">
        <el-tab-pane label="历史查看" name="history" />
        <el-tab-pane label="实时跟随" name="realtime" />
      </el-tabs>
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
        <el-date-picker
          v-model="logQuery.from"
          type="datetime"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
          placeholder="起始时间"
          clearable
        />
        <el-switch v-if="logMode === 'history'" v-model="logQuery.previous" active-text="上一次容器" />
        <el-button v-if="logMode === 'history'" :loading="logsLoading" @click="loadLogs">查询</el-button>
        <template v-else>
          <el-tag :type="realtimeConnected ? 'success' : 'info'">
            {{ realtimeConnected ? '跟随中' : '已停止' }}
          </el-tag>
          <el-button v-if="!realtimeConnected" type="primary" @click="startRealtimeLogs">开始跟随</el-button>
          <el-button v-else type="danger" @click="stopRealtimeLogs">停止</el-button>
        </template>
      </div>
      <el-alert
        v-if="logMode === 'history'"
        class="log-scope-alert"
        type="info"
        :closable="false"
        title="这里查看节点仍保留的当前或最近一次容器日志；跨 Pod、跨轮转的长期历史将在 P1 由 Loki 提供。"
      />
      <pre v-if="logMode === 'history'" class="log-view">{{ logResult?.lines.map((line) => line.raw).join('\n') }}</pre>
      <pre v-else class="log-view">{{ realtimeLines.join('\n') }}</pre>
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
