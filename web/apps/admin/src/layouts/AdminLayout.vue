<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  useAuthStore,
  usePermStore,
  portalHomeUrl,
  adminSpecialPath,
  adminModuleForPath,
  isAdminModuleActive,
  buildSidebarGroups,
  isDeliveryOnlineModule,
  OFFLINE_MENU_BADGE,
} from '@erp/shared'
import { ElMessage } from 'element-plus'
import NotifyBell from '../components/NotifyBell.vue'
import { useIsMobile } from '../composables/useMediaQuery'
import { displayModuleTitle, loadCarrierCodeUnit, useCarrierCodeLabel } from '../composables/useCarrierCodeLabel'

const auth = useAuthStore()
const perm = usePermStore()
const route = useRoute()
const router = useRouter()
const portalUrl = portalHomeUrl()
const menuFilter = ref('')
const activeDomain = ref('')
const isMobile = useIsMobile()
const sideDrawerOpen = ref(false)
const domainSheetOpen = ref(false)

/** How many domain icons to keep in the top bar on mobile; rest go to「更多」. */
const MOBILE_TOP_DOMAIN_COUNT = 5

onMounted(() => {
  void loadCarrierCodeUnit()
  if (auth.accessToken && (!auth.roles.length || !auth.permissions.length)) {
    void auth.fetchMe()
  }
})

const menus = computed(() => perm.visibleMenus)

const domainIcons: Record<string, string> = {
  销售管理: '🛒',
  客户管理: '👥',
  采购管理: '📦',
  生产管理: '🏭',
  库存管理: '📒',
  产品管理: '🏷️',
  固定资产管理: '🏗️',
  财务管理: '💰',
  工资管理: '💵',
  人事管理: '👤',
  统计报表: '📊',
  审批管理: '✅',
  系统管理: '⚙️',
}

const crumb = computed(() => {
  const d = route.params.domain as string
  const m = route.params.module as string
  if (d && m) return `${decodeURIComponent(d)} / ${displayModuleTitle(decodeURIComponent(m))}`
  const hit = adminModuleForPath(route.path)
  if (hit) return `${hit.domain} / ${displayModuleTitle(hit.module)}`
  if (route.path === '/' || route.path === '') return '工作台'
  return activeDomain.value || '工作台'
})

const currentDomain = computed(() => {
  if (activeDomain.value) return activeDomain.value
  const hit = adminModuleForPath(route.path)
  if (hit) return hit.domain
  const d = route.params.domain as string
  if (d) return decodeURIComponent(d)
  return ''
})

const sidebarGroups = computed(() => {
  const domain = currentDomain.value
  if (!domain) return []
  const entry = menus.value.find((x) => x.domain === domain)
  if (!entry) return []
  const q = menuFilter.value.trim()
  const mods = q
    ? entry.modules.filter((m) => m.includes(q))
    : entry.modules
  return buildSidebarGroups(domain, mods)
})

const showSidebar = computed(() => currentDomain.value !== '' && route.path !== '/' && route.path !== '')

const topDomains = computed(() => {
  if (!isMobile.value) return menus.value
  return menus.value.slice(0, MOBILE_TOP_DOMAIN_COUNT)
})

const moreDomains = computed(() => {
  if (!isMobile.value) return []
  return menus.value.slice(MOBILE_TOP_DOMAIN_COUNT)
})

watch(
  () => route.path,
  (path) => {
    sideDrawerOpen.value = false
    domainSheetOpen.value = false
    const hit = adminModuleForPath(path)
    if (hit) {
      activeDomain.value = hit.domain
      return
    }
    const d = route.params.domain as string
    if (d) activeDomain.value = decodeURIComponent(d)
    else if (path === '/' || path === '') activeDomain.value = ''
  },
  { immediate: true },
)

function selectDomain(domain: string) {
  activeDomain.value = domain
  menuFilter.value = ''
  domainSheetOpen.value = false
  const entry = menus.value.find((x) => x.domain === domain)
  const first = entry?.modules?.[0]
  if (first) goModule(domain, first)
  if (isMobile.value) sideDrawerOpen.value = true
}

function goHome() {
  activeDomain.value = ''
  menuFilter.value = ''
  sideDrawerOpen.value = false
  router.push('/')
}

function goModule(domain: string, module: string) {
  activeDomain.value = domain
  if (!isDeliveryOnlineModule(domain, module)) {
    ElMessage.warning(`${module} 尚未纳入本版本交付范围（${OFFLINE_MENU_BADGE}）`)
  }
  const special = adminSpecialPath(domain, module)
  if (special) {
    router.push(special)
    return
  }
  router.push(`/m/${encodeURIComponent(domain)}/${encodeURIComponent(module)}`)
}

function moduleOffline(domain: string, module: string) {
  return !isDeliveryOnlineModule(domain, module)
}

function moduleActive(domain: string, module: string) {
  const rd = route.params.domain ? decodeURIComponent(String(route.params.domain)) : undefined
  const rm = route.params.module ? decodeURIComponent(String(route.params.module)) : undefined
  return isAdminModuleActive(domain, module, route.path, rd, rm)
}

function domainActive(domain: string) {
  return currentDomain.value === domain
}

function logout() {
  auth.logout()
  router.replace('/login')
}

function openSideDrawer() {
  if (!showSidebar.value) {
    ElMessage.info('请先选择业务域')
    return
  }
  sideDrawerOpen.value = true
}
</script>

<template>
  <div class="admin-shell" :class="{ 'is-mobile': isMobile }">
    <header class="top-nav">
      <button
        v-if="isMobile"
        type="button"
        class="menu-toggle"
        title="菜单"
        aria-label="打开侧栏菜单"
        @click="openSideDrawer"
      >
        ☰
      </button>

      <button type="button" class="brand" @click="goHome" title="工作台">
        <span class="brand-mark">ERP</span>
        <span v-if="!isMobile" class="brand-text">加工厂</span>
      </button>

      <nav class="top-menus">
        <button
          type="button"
          class="top-item"
          :class="{ active: !currentDomain && (route.path === '/' || route.path === '') }"
          @click="goHome"
        >
          <span class="ico">🏠</span>
          <span class="lbl">工作台</span>
        </button>
        <button
          v-for="d in topDomains"
          :key="d.domain"
          type="button"
          class="top-item"
          :class="{ active: domainActive(d.domain) }"
          :title="d.domain"
          @click="selectDomain(d.domain)"
        >
          <span class="ico">{{ domainIcons[d.domain] || '📁' }}</span>
          <span class="lbl">{{ d.domain.replace(/管理$/, '') }}</span>
        </button>
        <button
          v-if="isMobile && moreDomains.length"
          type="button"
          class="top-item more-item"
          title="更多业务域"
          @click="domainSheetOpen = true"
        >
          <span class="ico">⋯</span>
          <span class="lbl">更多</span>
        </button>
      </nav>

      <div class="top-right">
        <a v-if="!isMobile" class="portal-link" :href="portalUrl" title="返回入口">入口</a>
        <NotifyBell />
        <span v-if="!isMobile" class="user-name">{{ auth.user?.name || auth.user?.login_name || '用户' }}</span>
        <el-button link type="danger" size="small" @click="logout">退出</el-button>
      </div>
    </header>

    <div class="body">
      <!-- Desktop fixed sidebar -->
      <aside v-if="showSidebar && !isMobile" class="side-nav">
        <div class="side-head">
          <div class="side-domain">{{ currentDomain }}</div>
          <el-input
            v-model="menuFilter"
            clearable
            size="small"
            placeholder="搜索菜单…"
            class="side-search"
          />
        </div>
        <nav class="side-groups">
          <div v-for="g in sidebarGroups" :key="g.title" class="side-group">
            <div class="group-title">{{ g.title }}</div>
            <a
              v-for="m in g.modules"
              :key="m"
              href="#"
              class="side-link"
              :class="{
                active: moduleActive(currentDomain, m),
                iam: perm.isIamModule(m),
                offline: moduleOffline(currentDomain, m),
              }"
              @click.prevent="goModule(currentDomain, m)"
            >
              <span class="side-link-text">{{ displayModuleTitle(m) }}</span>
              <span v-if="moduleOffline(currentDomain, m)" class="offline-badge">{{ OFFLINE_MENU_BADGE }}</span>
            </a>
          </div>
          <div v-if="!sidebarGroups.length" class="side-empty">无匹配菜单</div>
        </nav>
      </aside>

      <div class="main">
        <div class="crumb-bar">
          <span class="crumb">{{ crumb }}</span>
        </div>
        <main class="content" :class="{ 'full-width': !showSidebar || isMobile }">
          <router-view />
        </main>
      </div>
    </div>

    <!-- Mobile side menu drawer -->
    <el-drawer
      v-model="sideDrawerOpen"
      direction="ltr"
      size="280px"
      :with-header="true"
      title="功能菜单"
      class="side-drawer"
    >
      <div class="side-head drawer-head">
        <div class="side-domain">{{ currentDomain || '未选择业务域' }}</div>
        <el-input
          v-if="showSidebar"
          v-model="menuFilter"
          clearable
          size="small"
          placeholder="搜索菜单…"
          class="side-search"
        />
      </div>
      <nav v-if="showSidebar" class="side-groups drawer-groups">
        <div v-for="g in sidebarGroups" :key="g.title" class="side-group">
          <div class="group-title">{{ g.title }}</div>
          <a
            v-for="m in g.modules"
            :key="m"
            href="#"
            class="side-link"
            :class="{
              active: moduleActive(currentDomain, m),
              iam: perm.isIamModule(m),
              offline: moduleOffline(currentDomain, m),
            }"
            @click.prevent="goModule(currentDomain, m)"
          >
            <span class="side-link-text">{{ displayModuleTitle(m) }}</span>
            <span v-if="moduleOffline(currentDomain, m)" class="offline-badge">{{ OFFLINE_MENU_BADGE }}</span>
          </a>
        </div>
        <div v-if="!sidebarGroups.length" class="side-empty">无匹配菜单</div>
      </nav>
      <div v-else class="side-empty">请先从顶部选择业务域</div>
    </el-drawer>

    <!-- Mobile more domains sheet -->
    <el-drawer v-model="domainSheetOpen" direction="btt" size="55%" title="全部业务域">
      <div class="domain-sheet">
        <button
          v-for="d in menus"
          :key="d.domain"
          type="button"
          class="domain-sheet-item"
          :class="{ active: domainActive(d.domain) }"
          @click="selectDomain(d.domain)"
        >
          <span class="ico">{{ domainIcons[d.domain] || '📁' }}</span>
          <span>{{ d.domain }}</span>
        </button>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.admin-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: #e9ecef;
}

.top-nav {
  height: 56px;
  flex-shrink: 0;
  display: flex;
  align-items: stretch;
  background: #2c3e50;
  color: #ecf0f1;
  z-index: 20;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.25);
}
.menu-toggle {
  width: 44px;
  flex-shrink: 0;
  border: 0;
  background: #1a252f;
  color: #fff;
  font-size: 20px;
  cursor: pointer;
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border: 0;
  background: #1a252f;
  color: #fff;
  cursor: pointer;
  flex-shrink: 0;
}
.brand-mark {
  background: #714b67;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  padding: 3px 6px;
  border-radius: 3px;
  letter-spacing: 0.5px;
}
.brand-text {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}
.top-menus {
  flex: 1;
  display: flex;
  align-items: stretch;
  overflow-x: auto;
  overflow-y: hidden;
  min-width: 0;
}
.top-menus::-webkit-scrollbar {
  height: 0;
}
.top-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  min-width: 64px;
  padding: 4px 10px;
  border: 0;
  background: transparent;
  color: #bdc3c7;
  cursor: pointer;
  flex-shrink: 0;
  border-bottom: 3px solid transparent;
  transition: background 0.15s, color 0.15s;
}
.top-item:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #fff;
}
.top-item.active {
  background: rgba(113, 75, 103, 0.45);
  color: #fff;
  border-bottom-color: #c39bd3;
}
.top-item .ico {
  font-size: 16px;
  line-height: 1;
}
.top-item .lbl {
  font-size: 11px;
  line-height: 1.2;
  white-space: nowrap;
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.top-right {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 14px;
  flex-shrink: 0;
  border-left: 1px solid rgba(255, 255, 255, 0.08);
}
.portal-link {
  color: #aeb6bf;
  font-size: 12px;
  text-decoration: none;
}
.portal-link:hover {
  color: #fff;
}
.user-name {
  font-size: 12px;
  color: #d5dbdb;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.body {
  flex: 1;
  display: flex;
  min-height: 0;
}
.side-nav {
  width: 220px;
  flex-shrink: 0;
  background: #f8f9fa;
  border-right: 1px solid #dee2e6;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.side-head {
  padding: 12px 12px 8px;
  border-bottom: 1px solid #e9ecef;
  flex-shrink: 0;
}
.drawer-head {
  padding: 0 0 12px;
  border-bottom: 1px solid #e9ecef;
  margin-bottom: 8px;
}
.side-domain {
  font-size: 14px;
  font-weight: 700;
  color: #2c3e50;
  margin-bottom: 8px;
}
.side-search :deep(.el-input__wrapper) {
  background: #fff;
}
.side-groups {
  flex: 1;
  overflow: auto;
  padding: 8px 0 20px;
}
.drawer-groups {
  padding: 0;
}
.side-group {
  margin-bottom: 6px;
}
.group-title {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px 4px;
  font-size: 13px;
  font-weight: 700;
  color: #343a40;
}
.group-title::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #714b67;
  flex-shrink: 0;
}
.side-link {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding: 6px 14px 6px 26px;
  color: #5c6b75;
  text-decoration: none;
  font-size: 13px;
  border-radius: 0;
  line-height: 1.4;
}
.side-link.offline {
  color: #8a969e;
}
.side-link-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.offline-badge {
  flex-shrink: 0;
  font-size: 10px;
  line-height: 1;
  padding: 2px 4px;
  border-radius: 2px;
  background: #e8ebef;
  color: #6b7780;
}
.side-link.active .offline-badge {
  background: rgba(255, 255, 255, 0.25);
  color: #fff;
}
.side-link:hover {
  background: #eef1f4;
  color: #1a252f;
}
.side-link.active {
  background: #714b67;
  color: #fff;
  font-weight: 500;
}
.side-link.iam:not(.active) {
  color: #9a7b2f;
}
.side-empty {
  padding: 24px 14px;
  color: #98a2a8;
  font-size: 13px;
  text-align: center;
}

.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #f0f2f5;
}
.crumb-bar {
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 0 16px;
  background: #fff;
  border-bottom: 1px solid #e2e8ec;
}
.crumb {
  font-size: 13px;
  color: #5c6b75;
}
.content {
  flex: 1;
  overflow: auto;
  padding: 12px 16px;
}
.content.full-width {
  padding: 16px 20px;
}

.domain-sheet {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.domain-sheet-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border: 0;
  border-radius: 8px;
  background: #f8f9fa;
  color: #2c3e50;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}
.domain-sheet-item.active {
  background: #714b67;
  color: #fff;
}
.domain-sheet-item .ico {
  font-size: 18px;
}

@media (max-width: 768px) {
  .top-item .lbl {
    display: none;
  }
  .top-item {
    min-width: 44px;
  }
  .top-right {
    gap: 6px;
    padding: 0 8px;
  }
  .brand {
    padding: 0 10px;
  }
  .crumb-bar {
    height: 36px;
    padding: 0 10px;
  }
}
</style>
