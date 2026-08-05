<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  useAuthStore,
  usePermStore,
  portalHomeUrl,
  adminSpecialPath,
  adminModuleForPath,
  isAdminModuleActive,
} from '@erp/shared'
import NotifyBell from '../components/NotifyBell.vue'

const auth = useAuthStore()
const perm = usePermStore()
const route = useRoute()
const router = useRouter()
const openDomains = ref<string[]>([])
const portalUrl = portalHomeUrl()

const menus = computed(() => perm.visibleMenus)
const crumb = computed(() => {
  const d = route.params.domain as string
  const m = route.params.module as string
  if (d && m) return `${decodeURIComponent(d)} / ${decodeURIComponent(m)}`
  const hit = adminModuleForPath(route.path)
  if (hit) return `${hit.domain} / ${hit.module}`
  if (route.path === '/' || route.path === '') return '工作台'
  return '工作台'
})

watch(
  () => route.path,
  (path) => {
    const hit = adminModuleForPath(path)
    if (hit && !openDomains.value.includes(hit.domain)) openDomains.value.push(hit.domain)
    const d = route.params.domain as string
    if (d) {
      const domain = decodeURIComponent(d)
      if (!openDomains.value.includes(domain)) openDomains.value.push(domain)
    }
  },
  { immediate: true },
)

function toggle(domain: string) {
  const i = openDomains.value.indexOf(domain)
  if (i >= 0) openDomains.value.splice(i, 1)
  else openDomains.value.push(domain)
}

function goModule(domain: string, module: string) {
  if (!openDomains.value.includes(domain)) openDomains.value.push(domain)
  const special = adminSpecialPath(domain, module)
  if (special) {
    router.push(special)
    return
  }
  router.push(`/m/${encodeURIComponent(domain)}/${encodeURIComponent(module)}`)
}

function moduleActive(domain: string, module: string) {
  const rd = route.params.domain ? decodeURIComponent(String(route.params.domain)) : undefined
  const rm = route.params.module ? decodeURIComponent(String(route.params.module)) : undefined
  return isAdminModuleActive(domain, module, route.path, rd, rm)
}

function logout() {
  auth.logout()
  router.replace('/login')
}
</script>

<template>
  <div class="admin-shell">
    <aside class="sidebar">
      <div class="brand">
        加工厂 ERP 管理端
        <small>13 域 · 契约对齐</small>
        <a class="portal-link" :href="portalUrl">← 返回入口</a>
      </div>
      <el-button class="home-btn" @click="router.push('/')">工作台</el-button>
      <nav class="nav">
        <div v-for="d in menus" :key="d.domain" class="nav-domain" :class="{ open: openDomains.includes(d.domain) }">
          <button type="button" class="dom-btn" @click="toggle(d.domain)">
            <span>{{ d.domain }}</span>
            <span>▾</span>
          </button>
          <div class="mods">
            <a
              v-for="m in d.modules"
              :key="m"
              href="#"
              :class="{ active: moduleActive(d.domain, m), iam: perm.isIamModule(m) }"
              @click.prevent="goModule(d.domain, m)"
            >{{ m }}</a>
          </div>
        </div>
      </nav>
    </aside>
    <div class="main">
      <header class="topbar">
        <span class="crumb">{{ crumb }}</span>
        <span class="user">
          <NotifyBell />
          {{ auth.user?.name || auth.user?.login_name || '用户' }}
          <el-button link type="danger" @click="logout">退出</el-button>
        </span>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.admin-shell {
  display: flex;
  height: 100vh;
  overflow: hidden;
}
.sidebar {
  width: 240px;
  flex-shrink: 0;
  background: #0f1c22;
  color: #c5d0d6;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.brand {
  padding: 16px 14px 10px;
  font-weight: 700;
  color: #9fd4cb;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.brand small {
  display: block;
  font-weight: 400;
  font-size: 11px;
  color: #6a7d86;
  margin-top: 4px;
}
.portal-link {
  display: inline-block;
  margin-top: 8px;
  font-size: 12px;
  color: #7eb8ae;
  text-decoration: none;
}
.home-btn {
  margin: 10px 12px 6px;
  width: calc(100% - 24px);
}
.nav {
  flex: 1;
  overflow: auto;
  padding: 4px 0 16px;
}
.nav-domain .dom-btn {
  width: 100%;
  display: flex;
  justify-content: space-between;
  background: transparent;
  border: 0;
  color: #9fb0ba;
  padding: 10px 14px;
  cursor: pointer;
  font-size: 13px;
}
.nav-domain.open .dom-btn {
  color: #e8eef1;
}
.mods {
  display: none;
  padding: 0 8px 8px;
}
.nav-domain.open .mods {
  display: block;
}
.mods a {
  display: block;
  padding: 6px 10px;
  border-radius: 4px;
  color: #8a9ba4;
  text-decoration: none;
  font-size: 12px;
}
.mods a:hover {
  background: #1a2b34;
  color: #e8eef1;
}
.mods a.active {
  background: #0d7a6f;
  color: #fff;
}
.mods a.iam {
  color: #c9a86c;
}
.mods a.iam.active {
  color: #fff;
}
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #eef2f4;
}
.topbar {
  height: 48px;
  background: #fff;
  border-bottom: 1px solid #d5dde3;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
}
.crumb {
  font-size: 13px;
  color: #44555e;
}
.user {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.content {
  flex: 1;
  overflow: auto;
  padding: 12px 16px;
}
</style>
