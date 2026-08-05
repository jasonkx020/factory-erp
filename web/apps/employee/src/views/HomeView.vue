<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  useAuthStore,
  useNotifyStore,
  portalHomeUrl,
  visibleEmployeeModules,
  employeeRouteForEvent,
  type EmployeeModuleKey,
} from '@erp/shared'

const auth = useAuthStore()
const notify = useNotifyStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const drawer = ref(false)

const modules = computed(() => visibleEmployeeModules(auth.permissions, auth.roles))

async function boot() {
  if (!auth.isLoggedIn) {
    router.replace('/login')
    return
  }
  await auth.fetchMe()
  await notify.start()
  if (modules.value.length === 1) {
    router.replace(`/m/${modules.value[0].key}`)
  }
}

function open(key: EmployeeModuleKey) {
  router.push(`/m/${key}`)
}

function logout() {
  auth.logout()
  router.replace('/login')
}

async function openInbox() {
  drawer.value = true
  await notify.refresh()
}

async function goInbox(row: Record<string, unknown>) {
  const id = Number(row.id)
  if (id) await notify.markRead(id)
  drawer.value = false
  const path = employeeRouteForEvent(String(row.event_key || ''))
  if (path) router.push(path)
}

onMounted(boot)
</script>

<template>
  <div class="home">
    <header>
      <a :href="portalUrl">← 入口</a>
      <h1>员工工作台</h1>
      <div class="user">
        <button type="button" class="bell" @click="openInbox">
          通知
          <span v-if="notify.unread" class="badge">{{ notify.unread > 99 ? '99+' : notify.unread }}</span>
        </button>
        <span>{{ auth.user?.name || auth.user?.login_name || '用户' }}</span>
        <button type="button" class="out" @click="logout">退出</button>
      </div>
    </header>
    <p class="lead">按账号权限显示可用模块；通知经 MQTT/收件箱驱动下一岗确认。</p>
    <div v-if="!modules.length" class="empty">当前账号无可访问员工模块，请联系管理员授权。</div>
    <div class="grid">
      <button
        v-for="m in modules"
        :key="m.key"
        type="button"
        class="card"
        @click="open(m.key)"
      >
        <h2>{{ m.title }}</h2>
        <p>{{ m.desc }}</p>
        <span>进入 →</span>
      </button>
    </div>

    <div v-if="drawer" class="drawer-mask" @click.self="drawer = false">
      <aside class="drawer">
        <header>
          <h2>收件箱</h2>
          <button type="button" @click="drawer = false">关闭</button>
        </header>
        <p class="meta">MQTT: {{ notify.mqttStatus }}</p>
        <div v-if="!notify.inbox.length" class="empty">暂无通知</div>
        <button
          v-for="row in notify.inbox"
          :key="String(row.id)"
          type="button"
          class="inbox-item"
          :class="{ unread: !row.read_at }"
          @click="goInbox(row)"
        >
          <div class="t">{{ row.title }}</div>
          <div class="b">{{ row.body }}</div>
          <div class="m">{{ row.created_at }} · {{ row.event_key }}</div>
        </button>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.home { min-height: 100vh; background: #f3f5f7; padding: 16px; }
header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
header a { color: #0d7a6f; font-size: 13px; }
h1 { margin: 0; flex: 1; font-size: 18px; }
.user { display: flex; gap: 10px; align-items: center; font-size: 13px; color: #445; }
.user .out { border: 0; background: transparent; color: #c0392b; cursor: pointer; }
.bell {
  position: relative; border: 1px solid #0d7a6f; background: #fff; color: #0d7a6f;
  border-radius: 6px; padding: 4px 10px; cursor: pointer;
}
.badge {
  position: absolute; top: -6px; right: -8px; background: #c0392b; color: #fff;
  font-size: 10px; min-width: 16px; height: 16px; line-height: 16px; border-radius: 8px; padding: 0 4px;
}
.lead { color: #5c6b75; font-size: 13px; margin: 0 0 16px; }
.empty { background: #fff; border-radius: 10px; padding: 24px; color: #888; text-align: center; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.card {
  text-align: left; border: 0; border-radius: 12px; padding: 18px; background: #fff;
  box-shadow: 0 1px 3px rgba(0,0,0,.06); cursor: pointer;
}
.card h2 { margin: 0 0 6px; font-size: 16px; color: #0d7a6f; }
.card p { margin: 0 0 12px; font-size: 13px; color: #5c6b75; }
.card span { font-size: 13px; color: #0d7a6f; }
.drawer-mask {
  position: fixed; inset: 0; background: rgba(0,0,0,.35); display: flex; justify-content: flex-end; z-index: 40;
}
.drawer { width: min(360px, 92vw); background: #fff; height: 100%; padding: 12px; overflow: auto; }
.drawer header { display: flex; align-items: center; }
.drawer h2 { margin: 0; flex: 1; font-size: 16px; }
.drawer header button { border: 0; background: transparent; color: #0d7a6f; cursor: pointer; }
.meta { font-size: 11px; color: #999; }
.inbox-item {
  display: block; width: 100%; text-align: left; border: 0; border-bottom: 1px solid #eee;
  background: transparent; padding: 10px 0; cursor: pointer;
}
.inbox-item.unread .t { font-weight: 600; color: #0d7a6f; }
.t { font-size: 14px; }
.b { font-size: 12px; color: #556; margin-top: 4px; }
.m { font-size: 11px; color: #999; margin-top: 4px; }
</style>
