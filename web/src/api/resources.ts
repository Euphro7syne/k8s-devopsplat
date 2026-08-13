import { request } from './http'

export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface ClusterOverview {
  cluster: string
  node_count: number
  ready_node_count: number
  namespace_count: number
  pod_count: number
  abnormal_pod_count: number
  nodes: NodeSummary[]
}

export interface NodeSummary {
  name: string
  status: string
  cpu_allocatable: string
  memory_allocatable: string
  pods_allocatable: string
  created_at: string
}

export interface NamespaceSummary {
  name: string
  status: string
  created_at: string
}

export interface ResourceReference {
  kind: string
  namespace: string
  name: string
}

export interface DeclaredResources {
  cpu: string
  memory: string
  ephemeral_storage: string
  pods: number
}

export interface ResourceAllocation {
  requests: DeclaredResources
  limits: DeclaredResources
  cpu_request_percent: number
  memory_request_percent: number
  pod_percent: number
}

export interface NamespaceResourceCounts {
  pods: number
  ready_pods: number
  abnormal_pods: number
  deployments: number
  statefulsets: number
  daemonsets: number
  replicasets: number
  jobs: number
  cronjobs: number
  services: number
  ingresses: number
  persistent_volume_claims: number
  configmaps: number
}

export interface NamespaceDetail extends NamespaceSummary {
  labels: Record<string, string> | null
  finalizers: string[]
  conditions: PodCondition[]
  counts: NamespaceResourceCounts
  allocated: ResourceAllocation
  pods: PodSummary[]
  workloads: ResourceReference[]
  services: ServiceSummary[]
  ingresses: IngressSummary[]
  persistent_volume_claims: PVCSummary[]
  events: EventSummary[]
}

export interface NodeAddressDetail {
  type: string
  address: string
}

export interface NodeTaintDetail {
  key: string
  value: string
  effect: string
  time_added: string
}

export interface NodeConditionDetail {
  type: string
  status: string
  reason: string
  message: string
  last_heartbeat_time: string
  last_transition_time: string
}

export interface NodeSystemInfo {
  machine_id: string
  system_uuid: string
  boot_id: string
  kernel_version: string
  os_image: string
  container_runtime_version: string
  kubelet_version: string
  kube_proxy_version: string
  operating_system: string
  architecture: string
}

export interface NodeDetail extends NodeSummary {
  roles: string[]
  unschedulable: boolean
  pod_cidrs: string[]
  labels: Record<string, string> | null
  addresses: NodeAddressDetail[]
  taints: NodeTaintDetail[]
  conditions: NodeConditionDetail[]
  system_info: NodeSystemInfo
  capacity: DeclaredResources
  allocatable: DeclaredResources
  allocated: ResourceAllocation
  pods: PodSummary[]
  workloads: ResourceReference[]
  events: EventSummary[]
}

export interface ContainerSummary {
  name: string
  image: string
  ready: boolean
  restart_count: number
  state: string
  reason: string
  exit_code: number
  started_at: string
  finished_at: string
  last_state: string
  last_reason: string
  last_exit_code: number
  last_finished_at: string
}

export interface PodSummary {
  namespace: string
  name: string
  phase: string
  status: string
  node_name: string
  ready: boolean
  restart_count: number
  containers: ContainerSummary[]
  created_at: string
}

export interface OwnerReference {
  kind: string
  name: string
  controller: boolean
}

export interface PodCondition {
  type: string
  status: string
  reason: string
  message: string
}

export interface PodDetail extends PodSummary {
  pod_ip: string
  host_ip: string
  qos_class: string
  service_account: string
  restart_policy: string
  start_time: string
  labels: Record<string, string> | null
  annotations: Record<string, string> | null
  owner_refs: OwnerReference[]
  controller_chain: OwnerReference[]
  init_containers: ContainerSummary[]
  conditions: PodCondition[]
}

export interface EventSummary {
  type: string
  reason: string
  message: string
  count: number
  namespace: string
  involved_kind: string
  involved_name: string
  source: string
  first_timestamp: string
  last_timestamp: string
}

export interface DeploymentSummary {
  namespace: string
  name: string
  replicas: number
  ready_replicas: number
  updated_replicas: number
  available_replicas: number
  unavailable_replicas: number
  labels: Record<string, string> | null
  images: string[]
  created_at: string
}

export interface WorkloadCondition {
  type: string
  status: string
  reason: string
  message: string
  last_update_time: string
  last_transition_time: string
}

export interface ReplicaSetDetail extends WorkloadSummary {
  revision: string
  current_replicas: number
  fully_labeled_replicas: number
  observed_generation: number
  min_ready_seconds: number
  selector: Record<string, string> | null
  owner: OwnerReference | null
  conditions: WorkloadCondition[]
  pods: PodSummary[]
  events: EventSummary[]
}

export interface DeploymentDetail extends DeploymentSummary {
  generation: number
  observed_generation: number
  paused: boolean
  strategy: string
  max_surge: string
  max_unavailable: string
  min_ready_seconds: number
  progress_deadline_seconds: number
  revision_history_limit: number
  selector: Record<string, string> | null
  conditions: WorkloadCondition[]
  replica_sets: ReplicaSetDetail[]
  pods: PodSummary[]
  events: EventSummary[]
}

export interface WorkloadSummary {
  kind: string
  namespace: string
  name: string
  replicas: number
  ready_replicas: number
  available_replicas: number
  unavailable_replicas: number
  labels: Record<string, string> | null
  images: string[]
  created_at: string
}

export interface StatefulSetVolumeClaim {
  name: string
  storage_class: string
  requested_storage: string
  access_modes: string[]
}

export interface StatefulSetDetail extends WorkloadSummary {
  service_name: string
  pod_management_policy: string
  update_strategy: string
  current_revision: string
  update_revision: string
  current_replicas: number
  updated_replicas: number
  selector: Record<string, string> | null
  volume_claims: StatefulSetVolumeClaim[]
}

export interface DaemonSetSummary {
  namespace: string
  name: string
  desired_number: number
  current_number: number
  ready_number: number
  updated_number: number
  available_number: number
  unavailable_number: number
  misscheduled_number: number
  labels: Record<string, string> | null
  images: string[]
  created_at: string
}

export interface DaemonSetDetail extends DaemonSetSummary {
  update_strategy: string
  selector: Record<string, string> | null
  node_selector: Record<string, string> | null
  tolerations: string[]
}

export interface JobSummary {
  namespace: string
  name: string
  completions: number
  succeeded: number
  failed: number
  active: number
  start_time: string
  completion_time: string
  created_at: string
}

export interface JobDetail extends JobSummary {
  parallelism: number
  backoff_limit: number
  active_deadline_seconds: number
  ttl_seconds_after_finished: number
  completion_mode: string
  suspend: boolean
  manual_selector: boolean
  selector: Record<string, string> | null
  owner: OwnerReference | null
  images: string[]
  conditions: WorkloadCondition[]
  pods: PodSummary[]
  events: EventSummary[]
}

export interface CronJobSummary {
  namespace: string
  name: string
  schedule: string
  suspend: boolean
  active: number
  last_schedule_time: string
  created_at: string
}

export interface JobTemplatePolicy {
  parallelism: number
  completions: number
  backoff_limit: number
  active_deadline_seconds: number
  ttl_seconds_after_finished: number
  completion_mode: string
  suspend: boolean
  restart_policy: string
  images: string[]
}

export interface CronJobDetail extends CronJobSummary {
  time_zone: string
  concurrency_policy: string
  starting_deadline_seconds: number
  successful_jobs_history_limit: number
  failed_jobs_history_limit: number
  last_successful_time: string
  job_template: JobTemplatePolicy
  jobs: JobDetail[]
  events: EventSummary[]
}

export interface ServiceSummary {
  namespace: string
  name: string
  type: string
  cluster_ip: string
  external_ip: string
  ports: string[]
  selector: Record<string, string> | null
  created_at: string
}

export interface ServicePortDetail {
  name: string
  protocol: string
  port: number
  target_port: string
  node_port: number
  app_protocol: string
}

export interface ServiceEndpoint {
  source: 'EndpointSlice' | 'Endpoints'
  source_name: string
  addresses: string[]
  ready: boolean
  serving: boolean
  terminating: boolean
  hostname: string
  node_name: string
  zone: string
  target_kind: string
  target_name: string
  ports: string[]
}

export interface ServiceDetail extends ServiceSummary {
  cluster_ips: string[]
  external_name: string
  ip_families: string[]
  ip_family_policy: string
  session_affinity: string
  external_traffic_policy: string
  internal_traffic_policy: string
  publish_not_ready_addresses: boolean
  load_balancer_source_ranges: string[]
  port_details: ServicePortDetail[]
  endpoint_source: string
  endpoints: ServiceEndpoint[]
  pods: PodSummary[]
  events: EventSummary[]
}

export interface IngressSummary {
  namespace: string
  name: string
  class_name: string
  hosts: string[]
  addresses: string[]
  tls: boolean
  created_at: string
}

export interface IngressBackendDetail {
  host: string
  path: string
  path_type: string
  is_default: boolean
  backend_kind: string
  backend_api_group: string
  backend_name: string
  backend_port: string
  service_found: boolean
  service_port_found: boolean
}

export interface IngressTLSDetail {
  hosts: string[]
  secret_name: string
}

export interface IngressDetail extends IngressSummary {
  backends: IngressBackendDetail[]
  tls_details: IngressTLSDetail[]
  services: ServiceDetail[]
  events: EventSummary[]
}

export interface ConfigMapSummary {
  namespace: string
  name: string
  key_count: number
  binary_data_count: number
  keys: string[]
  created_at: string
}

export interface PVCSummary {
  namespace: string
  name: string
  status: string
  storage_class: string
  volume_name: string
  requested: string
  capacity: string
  access_modes: string[]
  created_at: string
}

export interface PVSummary {
  name: string
  status: string
  storage_class: string
  capacity: string
  claim_namespace: string
  claim_name: string
  reclaim_policy: string
  access_modes: string[]
  created_at: string
}

export interface VolumeDataSource {
  api_group: string
  kind: string
  name: string
  namespace: string
}

export interface VolumeMountDetail {
  pod_namespace: string
  pod_name: string
  volume_name: string
  container_type: 'Container' | 'InitContainer' | 'EphemeralContainer'
  container_name: string
  mount_path: string
  device_path: string
  sub_path: string
  read_only: boolean
}

export interface PVCDetail extends PVCSummary {
  volume_mode: string
  selector: Record<string, string> | null
  selector_expressions: string[]
  data_source: VolumeDataSource | null
  conditions: PodCondition[]
  pv: PVSummary | null
  pods: PodSummary[]
  workloads: OwnerReference[]
  mounts: VolumeMountDetail[]
  events: EventSummary[]
}

export interface PVDetail extends PVSummary {
  volume_mode: string
  mount_options: string[]
  node_affinity: string[]
  volume_source_type: string
  volume_source_info: Record<string, string> | null
  pvc: PVCSummary | null
  pods: PodSummary[]
  workloads: OwnerReference[]
  mounts: VolumeMountDetail[]
  events: EventSummary[]
}

export interface StorageClassSummary {
  name: string
  provisioner: string
  reclaim_policy: string
  volume_binding_mode: string
  allow_volume_expansion: boolean
  created_at: string
}

export type ResourceKind =
  | 'namespace'
  | 'node'
  | 'pod'
  | 'deployment'
  | 'statefulset'
  | 'daemonset'
  | 'replicaset'
  | 'job'
  | 'cronjob'
  | 'service'
  | 'ingress'
  | 'configmap'
  | 'pvc'
  | 'pv'
  | 'storageclass'

export interface LogLine {
  raw: string
}

export interface LogResult {
  source: string
  namespace: string
  pod: string
  container?: string
  lines: LogLine[]
  total: number
}

export interface LogStreamMessage {
  type: 'ready' | 'line' | 'error' | 'complete'
  line?: LogLine
  code?: number
  message?: string
}

export interface PodLogQuery {
  container?: string
  keyword?: string
  level?: string
  previous?: boolean
  limit?: number
  from?: string
}

export interface OperationResult {
  kind: string
  namespace: string
  name: string
  operation: string
  updated_at: string
  deployment?: DeploymentSummary
  statefulset?: WorkloadSummary
  daemonset?: DaemonSetSummary
  cronjob?: CronJobSummary
}

export interface ResourceYAMLUpdateResult {
  kind: string
  namespace: string
  name: string
  operation: string
  updated_at: string
  yaml: string
}

export function getOverview() {
  return request<ClusterOverview>({
    method: 'GET',
    url: '/overview'
  })
}

export function listNamespaces() {
  return request<NamespaceSummary[]>({
    method: 'GET',
    url: '/namespaces'
  })
}

export function getNamespace(name: string) {
  return request<NamespaceDetail>({
    method: 'GET',
    url: `/namespaces/${name}`
  })
}

export function listPods(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<PodSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/pods`,
    params: { page, page_size: pageSize }
  })
}

export function getPod(namespace: string, pod: string) {
  return request<PodDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/pods/${pod}`
  })
}

export function listEvents(namespace: string, params?: { involved_kind?: string; involved_name?: string }) {
  return request<EventSummary[]>({
    method: 'GET',
    url: `/namespaces/${namespace}/events`,
    params
  })
}

export function listNodes(page = 1, pageSize = 20) {
  return request<PageResult<NodeSummary>>({
    method: 'GET',
    url: '/nodes',
    params: { page, page_size: pageSize }
  })
}

export function getNode(name: string) {
  return request<NodeDetail>({
    method: 'GET',
    url: `/nodes/${name}`
  })
}

export function listDeployments(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<DeploymentSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/deployments`,
    params: { page, page_size: pageSize }
  })
}

export function getDeployment(namespace: string, name: string) {
  return request<DeploymentDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/deployments/${name}`
  })
}

export function listStatefulSets(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<WorkloadSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/statefulsets`,
    params: { page, page_size: pageSize }
  })
}

export function getStatefulSet(namespace: string, name: string) {
  return request<StatefulSetDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/statefulsets/${name}`
  })
}

export function listDaemonSets(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<DaemonSetSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/daemonsets`,
    params: { page, page_size: pageSize }
  })
}

export function getDaemonSet(namespace: string, name: string) {
  return request<DaemonSetDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/daemonsets/${name}`
  })
}

export function listReplicaSets(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<WorkloadSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/replicasets`,
    params: { page, page_size: pageSize }
  })
}

export function getReplicaSet(namespace: string, name: string) {
  return request<ReplicaSetDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/replicasets/${name}`
  })
}

export function listJobs(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<JobSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/jobs`,
    params: { page, page_size: pageSize }
  })
}

export function getJob(namespace: string, name: string) {
  return request<JobDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/jobs/${name}`
  })
}

export function listCronJobs(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<CronJobSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/cronjobs`,
    params: { page, page_size: pageSize }
  })
}

export function getCronJob(namespace: string, name: string) {
  return request<CronJobDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/cronjobs/${name}`
  })
}

export function listServices(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<ServiceSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/services`,
    params: { page, page_size: pageSize }
  })
}

export function getService(namespace: string, name: string) {
  return request<ServiceDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/services/${name}`
  })
}

export function listIngresses(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<IngressSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/ingresses`,
    params: { page, page_size: pageSize }
  })
}

export function getIngress(namespace: string, name: string) {
  return request<IngressDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/ingresses/${name}`
  })
}

export function listConfigMaps(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<ConfigMapSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/configmaps`,
    params: { page, page_size: pageSize }
  })
}

export function listPVCs(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<PVCSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/persistentvolumeclaims`,
    params: { page, page_size: pageSize }
  })
}

export function getPVC(namespace: string, name: string) {
  return request<PVCDetail>({
    method: 'GET',
    url: `/namespaces/${namespace}/persistentvolumeclaims/${name}`
  })
}

export function listPVs(page = 1, pageSize = 20) {
  return request<PageResult<PVSummary>>({
    method: 'GET',
    url: '/persistentvolumes',
    params: { page, page_size: pageSize }
  })
}

export function getPV(name: string) {
  return request<PVDetail>({
    method: 'GET',
    url: `/persistentvolumes/${name}`
  })
}

export function listStorageClasses(page = 1, pageSize = 20) {
  return request<PageResult<StorageClassSummary>>({
    method: 'GET',
    url: '/storageclasses',
    params: { page, page_size: pageSize }
  })
}

export function getPodLogs(
  namespace: string,
  pod: string,
  params: PodLogQuery
) {
  return request<LogResult>({
    method: 'GET',
    url: `/namespaces/${namespace}/pods/${pod}/logs`,
    params
  })
}

export function followPodLogs(namespace: string, pod: string, params: PodLogQuery) {
  const token = localStorage.getItem('ops-token') ?? ''
  if (!token) {
    throw new Error('登录会话已失效')
  }
  const query = new URLSearchParams()
  if (params.container) query.set('container', params.container)
  if (params.keyword) query.set('keyword', params.keyword)
  if (params.level) query.set('level', params.level)
  if (params.limit) query.set('limit', String(params.limit))
  if (params.from) query.set('from', params.from)

  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const encodedQuery = query.toString()
  const suffix = encodedQuery ? `?${encodedQuery}` : ''
  const url = `${scheme}//${window.location.host}/ws/v1/namespaces/${encodeURIComponent(namespace)}/pods/${encodeURIComponent(pod)}/logs/follow${suffix}`
  return new WebSocket(url, ['ops-platform.logs.v1', `bearer.${token}`])
}

export function getResourceYAML(kind: ResourceKind, name: string, namespace?: string) {
  return request<{ yaml: string }>({
    method: 'GET',
    url: '/resources/yaml',
    params: { kind, namespace, name }
  })
}

export function updateResourceYAML(kind: ResourceKind, name: string, namespace: string, resourceYAML: string) {
  return request<ResourceYAMLUpdateResult>({
    method: 'PUT',
    url: '/resources/yaml',
    data: {
      kind,
      namespace,
      name,
      yaml: resourceYAML
    }
  })
}

export function getPodYAML(namespace: string, pod: string) {
  return request<{ yaml: string }>({
    method: 'GET',
    url: `/namespaces/${namespace}/pods/${pod}/yaml`
  })
}

export function getDeploymentYAML(namespace: string, name: string) {
  return request<{ yaml: string }>({
    method: 'GET',
    url: `/namespaces/${namespace}/deployments/${name}/yaml`
  })
}

export function deletePod(namespace: string, pod: string) {
  return request<OperationResult>({
    method: 'DELETE',
    url: `/namespaces/${namespace}/pods/${pod}`,
    params: { confirm: true }
  })
}

export function restartPod(namespace: string, pod: string) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/pods/${pod}/restart`,
    params: { confirm: true }
  })
}

export function scaleDeployment(namespace: string, name: string, replicas: number) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/deployments/${name}/scale`,
    data: { replicas }
  })
}

export function restartDeployment(namespace: string, name: string) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/deployments/${name}/restart`
  })
}

export function scaleStatefulSet(namespace: string, name: string, replicas: number) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/statefulsets/${name}/scale`,
    data: { replicas }
  })
}

export function restartStatefulSet(namespace: string, name: string) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/statefulsets/${name}/restart`
  })
}

export function restartDaemonSet(namespace: string, name: string) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/daemonsets/${name}/restart`
  })
}

export function suspendCronJob(namespace: string, name: string) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/cronjobs/${name}/suspend`,
    params: { confirm: true }
  })
}

export function resumeCronJob(namespace: string, name: string) {
  return request<OperationResult>({
    method: 'POST',
    url: `/namespaces/${namespace}/cronjobs/${name}/resume`,
    params: { confirm: true }
  })
}
