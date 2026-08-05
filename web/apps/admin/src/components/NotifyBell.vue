<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useNotifyStore, adminRouteForEvent } from '@erp/shared'
import { ElMessage } from 'element-plus'

const router = useRouter()
const notify = useNotifyStore()
const drawer = ref(false)

async function openDrawer() {
  drawer.value = true
  await notify.refresh()
}

async function goTask(row: Record<string, unknown>) {
  const id = Number(row.id)
  if (id) await notify.markRead(id)
  const path = adminRouteForEvent(String(row.event_key || ''))
  if (path) router.push(path)
}

onMounted(() => {
  void notify.start()
})

onUnmounted(() => {
  // keep store alive for admin session; only stop on logout via auth
})
</script>

<template>
  <span class="bell-wrap">
    <el-badge :value="notify.unread" :hidden="!notify.unread" :max="99">
      <el-button link @click="openDrawer">通知</el-button>
    </el-badge>
    <el-drawer v-model="drawer" title="实时通知 / 待办" size="360px">
      <div class="meta">MQTT: {{ notify.mqttStatus }}</div>
      <div v-if="!notify.inbox.length" class="empty">暂无通知</div>
      <div
        v-for="row in notify.inbox"
        :key="String(row.id)"
        class="item"
        :class="{ unread: !row.read_at }"
        @click="goTask(row)"
      >
        <div class="t">{{ row.title }}</div>
        <div class="b">{{ row.body }}</div>
        <div class="m">{{ row.created_at }} · {{ row.event_key }}</div>
      </div>
      <el-button style="margin-top:12px" @click="router.push('/warehouse/inbound')">仓管待入库</el-button>
      <el-button
        link
        type="primary"
        @click="() => { notify.refresh(); ElMessage.success('已刷新') }"
      >刷新</el-button>
    </el-drawer>
  </span>
</template>

<style scoped>
.bell-wrap { margin-right: 12px; }
.meta { font-size: 11px; color: #999; margin-bottom: 8px; }
.item { padding: 10px 0; border-bottom: 1px solid #eee; cursor: pointer; }
.item.unread .t { font-weight: 600; color: #0d7a6f; }
.t { font-size: 14px; }
.b { font-size: 12px; color: #556; margin-top: 4px; }
.m { font-size: 11px; color: #999; margin-top: 4px; }
.empty { color: #999; padding: 24px 0; text-align: center; }
</style>
