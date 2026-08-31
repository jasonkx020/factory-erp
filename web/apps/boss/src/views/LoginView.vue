<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, portalHomeUrl } from '@erp/shared'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const router = useRouter()
const form = reactive({ login_name: 'admin', password: 'admin123' })
const loading = ref(false)
const portalUrl = portalHomeUrl()

async function onSubmit() {
  loading.value = true
  try {
    const ok = await auth.login(form.login_name, form.password, 'boss')
    if (!ok) return ElMessage.error(auth.error || '登录失败')
    router.replace('/')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="wrap">
    <el-card class="card">
      <a :href="portalUrl" class="back">← 返回入口</a>
      <h2>老板驾驶舱登录</h2>
      <p class="sub">木薯产线经营看板 · 只读</p>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="用户"><el-input v-model="form.login_name" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">登录</el-button>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(ellipse 70% 50% at 30% 20%, rgba(31, 122, 77, 0.28), transparent),
    #0c1a14;
  padding: 16px;
}
.card { width: min(360px, 100%); }
.back { color: var(--accent, #1f7a4d); font-size: 13px; text-decoration: none; }
h2 { margin: 10px 0 4px; }
.sub { margin: 0 0 14px; color: var(--muted, #5a6e64); font-size: 13px; }
</style>
