<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, CopyDocument, Key, Lock, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import QrcodeVue from 'qrcode.vue'

import { setupMFA, type MFASetupResult } from '../api/auth'
import { useAuthStore } from '../stores/auth'

type LoginStep = 'credentials' | 'verify' | 'setup'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const step = ref<LoginStep>('credentials')
const mfaToken = ref('')
const enrollment = ref<MFASetupResult | null>(null)
const form = reactive({
  email: 'admin@example.com',
  password: 'change-me-admin-password',
  code: ''
})

const subtitle = computed(() => {
  if (step.value === 'setup') return '首次登录需要绑定身份验证器'
  if (step.value === 'verify') return '输入身份验证器中的 6 位动态码'
  return 'k3s 运维控制台'
})

async function submitCredentials() {
  loading.value = true
  try {
    const result = await auth.login({ email: form.email, password: form.password })
    if (!result.mfa_required) {
      await router.push({ name: 'dashboard' })
      return
    }
    if (!result.mfa_token) {
      throw new Error('MFA 挑战信息不完整')
    }
    mfaToken.value = result.mfa_token
    form.code = ''
    if (result.mfa_setup_required) {
      enrollment.value = await setupMFA(result.mfa_token)
      mfaToken.value = enrollment.value.mfa_token
      step.value = 'setup'
      return
    }
    enrollment.value = null
    step.value = 'verify'
  } catch (error) {
    showError(error, '登录失败')
  } finally {
    loading.value = false
  }
}

async function submitMFA() {
  loading.value = true
  try {
    await auth.verifyMFA(mfaToken.value, form.code)
    ElMessage.success(step.value === 'setup' ? 'MFA 已绑定' : '验证成功')
    await router.push({ name: 'dashboard' })
  } catch (error) {
    showError(error, '动态码验证失败')
  } finally {
    loading.value = false
  }
}

function resetLogin() {
  step.value = 'credentials'
  mfaToken.value = ''
  enrollment.value = null
  form.code = ''
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

function showError(error: unknown, fallback: string) {
  ElMessage.error(error instanceof Error ? error.message : fallback)
}
</script>

<template>
  <main class="login-page">
    <section class="login-panel">
      <div class="login-heading">
        <h1>ops-platform</h1>
        <p>{{ subtitle }}</p>
      </div>

      <el-form
        v-if="step === 'credentials'"
        :model="form"
        label-position="top"
        @submit.prevent="submitCredentials"
      >
        <el-form-item label="邮箱">
          <el-input v-model="form.email" autocomplete="username" :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="form.password"
            autocomplete="current-password"
            type="password"
            show-password
            :prefix-icon="Lock"
          />
        </el-form-item>
        <el-button class="login-button" type="primary" :loading="loading" @click="submitCredentials">
          登录
        </el-button>
      </el-form>

      <div v-else class="mfa-login-step">
        <template v-if="step === 'setup' && enrollment">
          <el-alert
            title="请先完成绑定"
            description="使用 Microsoft Authenticator、Google Authenticator 或其他兼容 TOTP 的应用扫描二维码。"
            type="info"
            :closable="false"
            show-icon
          />
          <div class="mfa-qr">
            <QrcodeVue :value="enrollment.provisioning_uri" :size="184" level="M" />
          </div>
          <div class="mfa-secret-row">
            <code>{{ enrollment.secret }}</code>
            <el-button :icon="CopyDocument" circle title="复制密钥" @click="copySecret" />
          </div>
        </template>

        <el-form :model="form" label-position="top" @submit.prevent="submitMFA">
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
          <el-button class="login-button" type="primary" :loading="loading" @click="submitMFA">
            {{ step === 'setup' ? '绑定并登录' : '验证并登录' }}
          </el-button>
          <el-button class="login-button secondary-login-button" :icon="ArrowLeft" @click="resetLogin">
            返回账号登录
          </el-button>
        </el-form>
      </div>
    </section>
  </main>
</template>
