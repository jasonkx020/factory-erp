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
    const ok = await auth.login(form.login_name, form.password, 'admin')
    if (!ok) {
      ElMessage.error(auth.error || '登录失败')
      return
    }
    ElMessage.success('登录成功')
    router.replace('/')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-card">
      <a class="back" :href="portalUrl">← 返回入口</a>
      <h1>加工厂 ERP</h1>
      <p class="sub">管理端登录 · 演示账号 admin / admin123</p>
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
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a2b34, #0d7a6f);
}
.login-card {
  width: 380px;
  max-width: calc(100vw - 32px);
  background: #fff;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 12px 40px rgba(0,0,0,.25);
  position: relative;
  box-sizing: border-box;
}
@media (max-width: 768px) {
  .login-wrap {
    padding: 16px;
    padding-bottom: max(16px, env(safe-area-inset-bottom));
    align-items: flex-start;
    padding-top: max(48px, env(safe-area-inset-top));
  }
  .login-card {
    width: 100%;
    padding: 24px 20px;
  }
}
.back {
  display: inline-block;
  margin-bottom: 12px;
  font-size: 13px;
  color: #0d7a6f;
}
h1 { margin: 0 0 4px; color: #1a2b34; font-size: 22px; }
.sub { margin: 0 0 24px; color: #5c6b75; font-size: 13px; }
</style>
