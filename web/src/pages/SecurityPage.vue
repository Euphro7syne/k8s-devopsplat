<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { CopyDocument, Key, Lock } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import QrcodeVue from 'qrcode.vue'

import {
  disableMFA as disableMFAApi,
  getMFAStatus,
  startMFAEnrollment,
  type MFASetupResult,
  type MFAStatus
} from '../api/auth'
import AppShell from '../components/AppShell.vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const loading = ref(false)
const submitting = ref(false)
const status = ref<MFAStatus>({ enabled: auth.mfaEnabled, required: false })
const enrollment = ref<MFASetupResult | null>(null)
const form = reactive({
  code: '',
  password: ''
})

onMounted(loadStatus)

async function loadStatus() {
  loading.value = true
  try {
    status.value = await getMFAStatus()
  } catch (error) {
    showError(error, '读取 MFA 状态失败')
  } finally {
    loading.value = false
  }
}

async function beginEnrollment() {
  loading.value = true
  try {
    enrollment.value = await startMFAEnrollment()
    form.code = ''
  } catch (error) {
    showError(error, '创建 MFA 绑定信息失败')
  } finally {
    loading.value = false
  }
}

async function confirmEnrollment() {
  if (!enrollment.value) return
  submitting.value = true
  try {
    await auth.enableMFA(enrollment.value.mfa_token, form.code)
    enrollment.value = null
    form.code = ''
    status.value.enabled = true
    ElMessage.success('MFA 已启用')
  } catch (error) {
    showError(error, '启用 MFA 失败')
  } finally {
    submitting.value = false
  }
}

async function disableMFA() {
  try {
    await ElMessageBox.confirm(
      '关闭后，账号只使用密码登录。确认继续吗？',
      '关闭 MFA',
      { type: 'warning', confirmButtonText: '确认关闭', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  submitting.value = true
  try {
    const result = await disableMFAApi(form.password, form.code)
    auth.applyLoginResult(result)
    form.password = ''
    form.code = ''
    status.value.enabled = false
    ElMessage.success('MFA 已关闭')
  } catch (error) {
    showError(error, '关闭 MFA 失败')
  } finally {
    submitting.value = false
  }
}

async function copySecret() {
  if (!enrollment.value) return
  try {
    await navigator.clipboard.writeText(enrollment.value.secret)
    ElMessage.success('密钥已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制密钥')
  }
}

function cancelEnrollment() {
  enrollment.value = null
  form.code = ''
}

function showError(error: unknown, fallback: string) {
  ElMessage.error(error instanceof Error ? error.message : fallback)
}
</script>

<template>
  <AppShell>
    <div class="page-head">
      <div>
        <h1>安全设置</h1>
        <p>管理当前账号的 TOTP 多因素认证。</p>
      </div>
    </div>

    <el-card v-loading="loading" class="security-card" shadow="never">
      <template #header>
        <div class="security-card-head">
          <span>身份验证器</span>
          <el-tag :type="status.enabled ? 'success' : 'info'">
            {{ status.enabled ? '已启用' : '未启用' }}
          </el-tag>
        </div>
      </template>

      <el-alert
        v-if="status.required"
        title="平台已强制 MFA"
        description="所有账号必须绑定身份验证器，当前账号无法关闭 MFA。"
        type="warning"
        :closable="false"
        show-icon
      />

      <template v-if="!status.enabled">
        <div v-if="!enrollment" class="security-action">
          <p>启用后，每次密码校验成功还需要输入 6 位动态码。</p>
          <el-button type="primary" @click="beginEnrollment">绑定身份验证器</el-button>
        </div>

        <div v-else class="mfa-enrollment">
          <p>扫描二维码，或在身份验证器中手动输入密钥。</p>
          <div class="mfa-qr">
            <QrcodeVue :value="enrollment.provisioning_uri" :size="200" level="M" />
          </div>
          <div class="mfa-secret-row security-secret-row">
            <code>{{ enrollment.secret }}</code>
            <el-button :icon="CopyDocument" circle title="复制密钥" @click="copySecret" />
          </div>
          <el-form class="security-form" :model="form" label-position="top" @submit.prevent="confirmEnrollment">
            <el-form-item label="身份验证器中的 6 位动态码">
              <el-input
                v-model="form.code"
                autocomplete="one-time-code"
                inputmode="numeric"
                maxlength="6"
                placeholder="000000"
                :prefix-icon="Key"
              />
            </el-form-item>
            <div class="security-buttons">
              <el-button @click="cancelEnrollment">取消</el-button>
              <el-button type="primary" :loading="submitting" @click="confirmEnrollment">
                验证并启用
              </el-button>
            </div>
          </el-form>
        </div>
      </template>

      <div v-else-if="!status.required" class="security-action">
        <p>关闭 MFA 需要同时验证当前密码和身份验证器动态码。</p>
        <el-form class="security-form" :model="form" label-position="top" @submit.prevent="disableMFA">
          <el-form-item label="当前密码">
            <el-input
              v-model="form.password"
              autocomplete="current-password"
              type="password"
              show-password
              :prefix-icon="Lock"
            />
          </el-form-item>
          <el-form-item label="6 位动态码">
            <el-input
              v-model="form.code"
              autocomplete="one-time-code"
              inputmode="numeric"
              maxlength="6"
              placeholder="000000"
              :prefix-icon="Key"
            />
          </el-form-item>
          <el-button type="danger" :loading="submitting" @click="disableMFA">关闭 MFA</el-button>
        </el-form>
      </div>
    </el-card>
  </AppShell>
</template>
