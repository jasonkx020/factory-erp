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
  <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;background:#0b1418">
    <el-card style="width:360px">
      <a :href="portalUrl" style="color:#0d7a6f;font-size:13px">← 返回入口</a>
      <h2>老板驾驶舱登录</h2>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="用户"><el-input v-model="form.login_name" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">登录</el-button>
      </el-form>
    </el-card>
  </div>
</template>
