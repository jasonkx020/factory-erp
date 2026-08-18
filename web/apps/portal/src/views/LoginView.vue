<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore, portalHomeUrl } from '@erp/shared'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const form = reactive({ login_name: 'cust01', password: 'admin123' })
const loading = ref(false)
const portalUrl = portalHomeUrl()

async function onSubmit() {
  loading.value = true
  try {
    const ok = await auth.login(form.login_name, form.password, 'customer')
    if (!ok) return ElMessage.error(auth.error || '登录失败')
    const redirect = String(route.query.redirect || '/shop')
    router.replace(redirect.startsWith('/shop') ? redirect : '/shop')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="wrap">
    <el-card class="card">
      <a :href="portalUrl" class="back">← 返回入口</a>
      <h2>客户自助登录</h2>
      <p class="hint">账号由工厂开户，数据仅可见本客户单据。演示 <code>cust01</code> / <code>admin123</code></p>
      <el-form @submit.prevent="onSubmit">
        <el-form-item label="用户"><el-input v-model="form.login_name" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password autocomplete="current-password" /></el-form-item>
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
  padding: 16px;
  background:
    radial-gradient(ellipse 80% 50% at 20% -10%, rgba(13, 122, 111, 0.18), transparent),
    linear-gradient(180deg, #f4f7f8 0%, #e8eef1 100%);
}
.card { width: min(400px, 100%); }
.back { color: #0d7a6f; font-size: 13px; text-decoration: none; }
h2 { margin: 10px 0 6px; font-size: 22px; }
.hint { margin: 0 0 16px; color: #5c6b75; font-size: 13px; line-height: 1.5; }
code { background: rgba(0,0,0,.05); padding: 1px 5px; border-radius: 3px; }
</style>
