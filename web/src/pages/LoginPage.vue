<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Lock, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({
  email: 'admin@example.com',
  password: 'change-me-admin-password'
})

async function submit() {
  loading.value = true
  try {
    await auth.login(form)
    await router.push({ name: 'dashboard' })
  } catch (error) {
    const message = error instanceof Error ? error.message : '登录失败'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <section class="login-panel">
      <div class="login-heading">
        <h1>ops-platform</h1>
        <p>k3s 运维控制台</p>
      </div>
      <el-form :model="form" label-position="top" @submit.prevent="submit">
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
        <el-button class="login-button" type="primary" :loading="loading" @click="submit">
          登录
        </el-button>
      </el-form>
    </section>
  </main>
</template>
