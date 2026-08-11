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

export interface ContainerSummary {
  name: string
  image: string
  ready: boolean
  restart_count: number
  state: string
  reason: string
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

export interface DaemonSetSummary {
  namespace: string
  name: string
  desired_number: number
  current_number: number
  ready_number: number
  available_number: number
  labels: Record<string, string> | null
  images: string[]
  created_at: string
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

export interface CronJobSummary {
  namespace: string
  name: string
  schedule: string
  suspend: boolean
  active: number
  last_schedule_time: string
  created_at: string
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

export interface IngressSummary {
  namespace: string
  name: string
  class_name: string
  hosts: string[]
  addresses: string[]
  tls: boolean
  created_at: string
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

export interface StorageClassSummary {
  name: string
  provisioner: string
  reclaim_policy: string
  volume_binding_mode: string
  allow_volume_expansion: boolean
  created_at: string
}

export type ResourceKind =
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

export interface OperationResult {
  kind: string
  namespace: string
  name: string
  operation: string
  updated_at: string
  deployment?: DeploymentSummary
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

export function listPods(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<PodSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/pods`,
    params: { page, page_size: pageSize }
  })
}

export function listNodes(page = 1, pageSize = 20) {
  return request<PageResult<NodeSummary>>({
    method: 'GET',
    url: '/nodes',
    params: { page, page_size: pageSize }
  })
}

export function listDeployments(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<DeploymentSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/deployments`,
    params: { page, page_size: pageSize }
  })
}

export function listStatefulSets(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<WorkloadSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/statefulsets`,
    params: { page, page_size: pageSize }
  })
}

export function listDaemonSets(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<DaemonSetSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/daemonsets`,
    params: { page, page_size: pageSize }
  })
}

export function listReplicaSets(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<WorkloadSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/replicasets`,
    params: { page, page_size: pageSize }
  })
}

export function listJobs(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<JobSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/jobs`,
    params: { page, page_size: pageSize }
  })
}

export function listCronJobs(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<CronJobSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/cronjobs`,
    params: { page, page_size: pageSize }
  })
}

export function listServices(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<ServiceSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/services`,
    params: { page, page_size: pageSize }
  })
}

export function listIngresses(namespace: string, page = 1, pageSize = 20) {
  return request<PageResult<IngressSummary>>({
    method: 'GET',
    url: `/namespaces/${namespace}/ingresses`,
    params: { page, page_size: pageSize }
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

export function listPVs(page = 1, pageSize = 20) {
  return request<PageResult<PVSummary>>({
    method: 'GET',
    url: '/persistentvolumes',
    params: { page, page_size: pageSize }
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
  params: { container?: string; keyword?: string; level?: string; previous?: boolean; limit?: number }
) {
  return request<LogResult>({
    method: 'GET',
    url: `/namespaces/${namespace}/pods/${pod}/logs`,
    params
  })
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
