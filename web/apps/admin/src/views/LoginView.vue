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
    <div class="login-atmosphere" aria-hidden="true">
      <div class="leaf leaf-a" />
      <div class="leaf leaf-b" />
      <div class="leaf leaf-c" />
    </div>
    <div class="login-card">
      <a class="back" :href="portalUrl">← 返回入口</a>
      <div class="brand-row">
        <span class="mark">木薯</span>
        <div>
          <h1>木薯加工厂 ERP</h1>
          <p class="sub">从田间鲜薯到烘干成品 · 管理端登录</p>
        </div>
      </div>
      <ol class="chip-flow" aria-label="产线摘要">
        <li>清洗</li>
        <li>去皮</li>
        <li>切段</li>
        <li>去芯</li>
        <li>切片</li>
        <li class="end">烘干</li>
      </ol>
      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="用户名">
          <el-input v-model="form.login_name" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">登录</el-button>
      </el-form>
      <p class="demo-hint">演示账号 admin / admin123</p>
    </div>
  </div>
</template>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background:
    radial-gradient(ellipse 90% 70% at 10% 20%, rgba(61, 155, 106, 0.35), transparent 55%),
    radial-gradient(ellipse 70% 50% at 90% 80%, rgba(166, 124, 61, 0.22), transparent 50%),
    linear-gradient(145deg, #0f2a21 0%, #145c38 48%, #1a4535 100%);
}
.login-atmosphere {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.leaf {
  position: absolute;
  border-radius: 60% 40% 55% 45%;
  background: rgba(232, 245, 238, 0.08);
  filter: blur(2px);
  animation: drift 18s ease-in-out infinite;
}
.leaf-a { width: 180px; height: 90px; top: 12%; left: 8%; transform: rotate(-18deg); }
.leaf-b { width: 240px; height: 110px; bottom: 18%; right: 6%; transform: rotate(22deg); animation-delay: -6s; }
.leaf-c { width: 140px; height: 70px; top: 42%; right: 28%; transform: rotate(-8deg); animation-delay: -11s; opacity: 0.7; }
@keyframes drift {
  0%, 100% { translate: 0 0; }
  50% { translate: 12px -10px; }
}
.login-card {
  width: 420px;
  max-width: calc(100vw - 32px);
  background: rgba(255, 255, 255, 0.97);
  border-radius: 16px;
  padding: 28px 28px 22px;
  box-shadow: 0 20px 48px rgba(15, 42, 33, 0.35);
  position: relative;
  box-sizing: border-box;
  border: 1px solid rgba(232, 245, 238, 0.35);
}
.brand-row {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  margin-bottom: 14px;
}
.mark {
  flex-shrink: 0;
  margin-top: 2px;
  background: linear-gradient(135deg, var(--accent-leaf, #3d9b6a), var(--accent, #1f7a4d));
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  padding: 6px 8px;
  border-radius: 6px;
  letter-spacing: 0.04em;
}
.chip-flow {
  list-style: none;
  margin: 0 0 18px;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip-flow li {
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--accent-soft, #e8f5ee);
  color: var(--accent-dark, #145c38);
}
.chip-flow li.end {
  background: var(--accent, #1f7a4d);
  color: #fff;
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
    padding: 22px 18px;
  }
}
.back {
  display: inline-block;
  margin-bottom: 14px;
  font-size: 13px;
  color: var(--accent, #1f7a4d);
}
h1 { margin: 0 0 4px; color: var(--text, #1a2e24); font-size: 20px; }
.sub { margin: 0; color: var(--muted, #5a6e64); font-size: 13px; line-height: 1.4; }
.demo-hint {
  margin: 14px 0 0;
  text-align: center;
  font-size: 12px;
  color: var(--muted, #5a6e64);
}
</style>
