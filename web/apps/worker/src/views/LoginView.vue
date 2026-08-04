<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, portalHomeUrl } from '@erp/shared'
import { showToast } from 'vant'

const auth = useAuthStore()
const router = useRouter()
const form = reactive({ login_name: 'admin', password: 'admin123' })
const loading = ref(false)
const portalUrl = portalHomeUrl()

async function onSubmit() {
  loading.value = true
  try {
    const ok = await auth.login(form.login_name, form.password, 'mp_worker')
    if (!ok) return showToast(auth.error || '登录失败')
    router.replace('/')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;background:#eef1f4;padding:16px">
    <div style="width:100%;max-width:360px;background:#fff;border-radius:12px;padding:20px">
      <a :href="portalUrl" style="color:#0d7a6f;font-size:13px">← 返回入口</a>
      <h2 style="margin:8px 0 12px">工人端登录</h2>
      <van-field v-model="form.login_name" label="用户" />
      <van-field v-model="form.password" type="password" label="密码" />
      <van-button type="primary" block :loading="loading" style="margin-top:12px" @click="onSubmit">登录</van-button>
    </div>
  </div>
</template>
