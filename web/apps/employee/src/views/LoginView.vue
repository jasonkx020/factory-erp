<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore, portalHomeUrl } from '@erp/shared'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const form = reactive({ login_name: 'admin', password: 'admin123' })
const loading = ref(false)
const portalUrl = portalHomeUrl()

async function onSubmit() {
  loading.value = true
  try {
    const ok = await auth.login(form.login_name, form.password, 'employee')
    if (!ok) {
      ElMessage.error(auth.error || '登录失败')
      return
    }
    ElMessage.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.replace(redirect)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-card">
      <a class="back" :href="portalUrl">← 返回入口</a>
      <h1>员工端</h1>
      <p class="sub">车间 / 工人 / 销售 · 按权限开放 · 演示 admin / admin123</p>
      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="用户名">
          <el-input v-model="form.login_name" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">登录</el-button>
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.login-wrap { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: #0f1c22; }
.login-card { width: 360px; background: #fff; border-radius: 12px; padding: 24px; }
.back { color: #0d7a6f; font-size: 13px; }
h1 { margin: 12px 0 4px; font-size: 22px; }
.sub { color: #5c6b75; font-size: 13px; margin: 0 0 16px; }
</style>
