<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { CircleCheck, CircleClose, Key, Plus, Refresh, Select } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  createUser,
  listRoles,
  listUsers,
  resetUserMFA,
  updateUserRoles,
  updateUserStatus,
  type CreateUserPayload,
  type RoleRecord,
  type UserRecord
} from '../api/users'
import AppShell from '../components/AppShell.vue'
import { useAuthStore } from '../stores/auth'
import { formatDateTime } from '../utils/time'

const auth = useAuthStore()
const loading = ref(false)
const createLoading = ref(false)
const createVisible = ref(false)
const roleSaving = ref<number | null>(null)
const statusSaving = ref<number | null>(null)
const mfaResetting = ref<number | null>(null)
const users = ref<UserRecord[]>([])
const roles = ref<RoleRecord[]>([])
const roleDrafts = reactive<Record<number, string[]>>({})
const createForm = reactive<CreateUserPayload>({
  username: '',
  email: '',
  password: '',
  roles: ['viewer']
})

const roleOptions = computed(() => roles.value.map((role) => ({ label: role.name, value: role.name })))

function isSelf(row: UserRecord) {
  return row.id === auth.userID
}

function statusType(status: UserRecord['status']) {
  return status === 'active' ? 'success' : 'info'
}

function seedRoleDrafts() {
  for (const row of users.value) {
    roleDrafts[row.id] = [...row.roles]
  }
}

async function loadAll() {
  loading.value = true
  try {
    const [roleResult, userResult] = await Promise.all([listRoles(), listUsers()])
    roles.value = roleResult
    users.value = userResult
    seedRoleDrafts()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '用户加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.username = ''
  createForm.email = ''
  createForm.password = ''
  createForm.roles = ['viewer']
  createVisible.value = true
}

async function submitCreate() {
  if (!createForm.username || !createForm.email || !createForm.password) {
    ElMessage.error('用户名、邮箱和密码必填')
    return
  }
  if (createForm.roles.length === 0) {
    ElMessage.error('至少选择一个角色')
    return
  }
  createLoading.value = true
  try {
    await createUser({ ...createForm, roles: [...createForm.roles] })
    ElMessage.success('用户已创建')
    createVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '用户创建失败')
  } finally {
    createLoading.value = false
  }
}

async function saveRoles(row: UserRecord) {
  if (isSelf(row)) {
    ElMessage.error('不能修改当前用户角色')
    return
  }
  const nextRoles = roleDrafts[row.id] ?? []
  if (nextRoles.length === 0) {
    ElMessage.error('至少选择一个角色')
    return
  }
  roleSaving.value = row.id
  try {
    await updateUserRoles(row.id, nextRoles)
    ElMessage.success('角色已保存')
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '角色保存失败')
  } finally {
    roleSaving.value = null
  }
}

async function toggleStatus(row: UserRecord) {
  const nextStatus: UserRecord['status'] = row.status === 'active' ? 'disabled' : 'active'
  const action = nextStatus === 'disabled' ? '禁用' : '启用'
  await ElMessageBox.confirm(`${action}用户 ${row.username}`, '二次确认', {
    confirmButtonText: action,
    cancelButtonText: '取消',
    type: nextStatus === 'disabled' ? 'warning' : 'info'
  })
  statusSaving.value = row.id
  try {
    await updateUserStatus(row.id, nextStatus)
    ElMessage.success(`用户已${action}`)
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '状态更新失败')
  } finally {
    statusSaving.value = null
  }
}

async function resetMFA(row: UserRecord) {
  if (isSelf(row) || !row.mfa_enabled) return
  try {
    await ElMessageBox.confirm(
      `重置用户 ${row.username} 的 MFA？其现有动态码会立即失效，下次登录需要重新绑定。`,
      '二次确认',
      { confirmButtonText: '确认重置', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  mfaResetting.value = row.id
  try {
    await resetUserMFA(row.id)
    ElMessage.success('MFA 已重置')
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'MFA 重置失败')
  } finally {
    mfaResetting.value = null
  }
}

onMounted(loadAll)
</script>

<template>
  <AppShell>
    <div class="page-head">
      <div>
        <h1>用户管理</h1>
        <p>{{ users.length }} 个用户 · {{ roles.length }} 个角色</p>
      </div>
      <div class="toolbar">
        <el-button :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
        <el-button :icon="Plus" type="primary" @click="openCreate">新建用户</el-button>
      </div>
    </div>

    <section class="resource-section">
      <el-table v-loading="loading" :data="users" border>
        <el-table-column prop="username" label="用户名" min-width="160" show-overflow-tooltip />
        <el-table-column prop="email" label="邮箱" min-width="220" show-overflow-tooltip />
        <el-table-column prop="provider" label="来源" width="100" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="MFA" width="100">
          <template #default="{ row }">
            <el-tag :type="row.mfa_enabled ? 'success' : 'info'">
              {{ row.mfa_enabled ? '已启用' : '未启用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="角色" min-width="280">
          <template #default="{ row }">
            <el-select
              v-model="roleDrafts[row.id]"
              class="role-select"
              multiple
              collapse-tags
              collapse-tags-tooltip
              :disabled="isSelf(row)"
            >
              <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="224" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="保存角色">
              <el-button :icon="Select" circle :loading="roleSaving === row.id" :disabled="isSelf(row)" @click="saveRoles(row)" />
            </el-tooltip>
            <el-tooltip :content="row.status === 'active' ? '禁用' : '启用'">
              <el-button
                :icon="row.status === 'active' ? CircleClose : CircleCheck"
                circle
                :type="row.status === 'active' ? 'warning' : 'success'"
                :loading="statusSaving === row.id"
                :disabled="isSelf(row) && row.status === 'active'"
                @click="toggleStatus(row)"
              />
            </el-tooltip>
            <el-tooltip content="重置 MFA">
              <el-button
                :icon="Key"
                circle
                type="danger"
                :loading="mfaResetting === row.id"
                :disabled="isSelf(row) || !row.mfa_enabled"
                @click="resetMFA(row)"
              />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="createVisible" title="新建用户" width="520px">
      <el-form label-width="92px">
        <el-form-item label="用户名">
          <el-input v-model="createForm.username" autocomplete="off" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" autocomplete="off" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="createForm.password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.roles" multiple class="form-select">
            <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createVisible = false">取消</el-button>
          <el-button type="primary" :loading="createLoading" @click="submitCreate">创建</el-button>
        </div>
      </template>
    </el-dialog>
  </AppShell>
</template>
