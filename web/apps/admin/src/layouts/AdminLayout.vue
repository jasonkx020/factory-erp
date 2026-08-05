<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore, usePermStore, portalHomeUrl } from '@erp/shared'
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
  if (d && m) return `${d} / ${m}`
  if (route.path.startsWith('/loops')) return '业务闭环'
  if (route.path.includes('/warehouse/inbound')) return '仓管待入库'
  if (route.path.includes('/sales/outbound-settle')) return '出厂结算'
  if (route.path.includes('/production/process-reports')) return '加工记录'
  if (route.path.includes('/production/piece-issue')) return '计件领料表'
  if (route.path.includes('/hr/tool-issues')) return '工具领还'
  if (route.path.includes('/inventory/ledger')) return '库存/地磅'
  if (route.path.includes('/automation/routing')) return '工艺路线'
  if (route.path.includes('/automation/trace')) return '全链追溯'
  if (route.path.includes('/automation/logs')) return '操作日志'
  if (route.path.includes('/automation/repairs')) return '数据修复'
  return '工作台'
})

function toggle(domain: string) {
  const i = openDomains.value.indexOf(domain)
  if (i >= 0) openDomains.value.splice(i, 1)
  else openDomains.value.push(domain)
}

function goModule(domain: string, module: string) {
  if (!openDomains.value.includes(domain)) openDomains.value.push(domain)
  router.push(`/m/${encodeURIComponent(domain)}/${encodeURIComponent(module)}`)
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
      <el-button class="home-btn ghost" @click="router.push('/loops')">业务闭环</el-button>
      <el-button class="home-btn ghost" @click="router.push('/warehouse/inbound')">仓管待入库</el-button>
      <el-button class="home-btn ghost" @click="router.push('/sales/outbound-settle')">出厂结算</el-button>
      <el-button class="home-btn ghost" @click="router.push('/production/process-reports')">加工记录</el-button>
      <el-button class="home-btn ghost" @click="router.push('/production/piece-issue')">计件领料表</el-button>
      <el-button class="home-btn ghost" @click="router.push('/hr/tool-issues')">工具领还</el-button>
      <el-button class="home-btn ghost" @click="router.push('/inventory/ledger')">库存/地磅</el-button>
      <el-button class="home-btn ghost" @click="router.push('/automation/routing')">工艺路线</el-button>
      <el-button class="home-btn ghost" @click="router.push('/automation/trace')">追溯</el-button>
      <el-button class="home-btn ghost" @click="router.push('/automation/logs')">审计日志</el-button>
      <el-button class="home-btn ghost" @click="router.push('/automation/repairs')">数据修复</el-button>
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
              :class="{ active: route.params.domain === d.domain && route.params.module === m, iam: perm.isIamModule(m) }"
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
  width: 260px;
  flex-shrink: 0;
  height: 100%;
  background: var(--sidebar, #1a2b34);
  color: #fff;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.brand {
  flex-shrink: 0;
  padding: 16px;
  font-weight: 600;
  border-bottom: 1px solid rgba(255,255,255,.08);
}
.brand small { display: block; font-weight: 400; opacity: .7; font-size: 11px; margin-top: 4px; }
.portal-link {
  display: inline-block;
  margin-top: 8px;
  font-size: 12px;
  font-weight: 400;
  color: #9fd4cb;
}
.home-btn {
  flex-shrink: 0;
  margin: 8px 12px 0;
  width: calc(100% - 24px);
  background: #243b47;
  border-color: #35525e;
  color: #fff;
}
.home-btn.ghost { background: transparent; }
.nav {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 0;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.22) transparent;
}
.nav::-webkit-scrollbar {
  width: 6px;
}
.nav::-webkit-scrollbar-track {
  background: transparent;
}
.nav::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.22);
  border-radius: 3px;
}
.nav::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.35);
}
.nav::-webkit-scrollbar-corner {
  background: transparent;
}
.dom-btn {
  width: 100%; display: flex; justify-content: space-between; align-items: center;
  background: transparent; border: 0; color: #c5d4dc; padding: 10px 14px; cursor: pointer; text-align: left;
}
.dom-btn:hover { background: #243b47; color: #fff; }
.mods { display: none; padding: 0 8px 8px; }
.nav-domain.open .mods { display: block; }
.mods a {
  display: block; padding: 6px 10px; color: #9fb0ba; border-radius: 4px; font-size: 13px;
}
.mods a:hover, .mods a.active { background: #0d7a6f; color: #fff; }
.mods a.iam { color: #9fd4cb; }
.main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.topbar {
  flex-shrink: 0;
  height: 52px;
  background: #fff;
  border-bottom: 1px solid var(--border, #d5dde3);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}
.crumb { font-weight: 500; }
.user { color: var(--muted, #5c6b75); font-size: 13px; display: flex; align-items: center; gap: 8px; }
.content {
  flex: 1;
  min-height: 0;
  padding: 16px 20px;
  overflow: auto;
}
</style>
