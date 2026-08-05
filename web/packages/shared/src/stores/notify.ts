import { defineStore } from 'pinia'
import { ref } from 'vue'
import { notifyApi } from '../api/endpoints'
import { createMqttSession, type MqttSession } from '../notify/mqttSession'
import { parsePayload } from '../notify/routes'

export type NotifyInboxRow = Record<string, unknown>

export const useNotifyStore = defineStore('notify', () => {
  const inbox = ref<NotifyInboxRow[]>([])
  const unread = ref(0)
  const loading = ref(false)
  const mqttStatus = ref<'idle' | 'connecting' | 'connected' | 'error' | 'closed'>('idle')
  const lastPayload = ref<Record<string, unknown> | null>(null)
  const tick = ref(0)

  let session: MqttSession | null = null
  let pollTimer: ReturnType<typeof setInterval> | undefined
  let started = false
  let lastSig = ''

  function inboxSig(list: NotifyInboxRow[], unreadCount: number) {
    const ids = list.map((r) => `${r.id}:${r.read_at || ''}`).join(',')
    return `${unreadCount}|${ids}`
  }

  async function refresh() {
    loading.value = true
    try {
      const res = await notifyApi.inbox('page_num=1&page_size=30')
      if (res.code !== 1) return
      const data = res.data as { list?: NotifyInboxRow[]; unread?: number }
      const list = data?.list || []
      const nextUnread = Number(data?.unread || 0)
      inbox.value = list
      unread.value = nextUnread
      const sig = inboxSig(list, nextUnread)
      // 内容未变不 bump tick，避免各模块 watch(tick) 无意义连刷
      if (sig !== lastSig) {
        lastSig = sig
        tick.value += 1
      }
    } finally {
      loading.value = false
    }
  }

  async function markRead(id: number) {
    await notifyApi.readInbox(id)
    await refresh()
  }

  function handleMqttMessage(_topic: string, raw: string) {
    try {
      const msg = JSON.parse(raw)
      if (msg && typeof msg === 'object') {
        lastPayload.value = parsePayload(msg.payload ?? msg)
      }
    } catch {
      /* ignore non-json */
    }
    void refresh()
  }

  async function start() {
    if (started) {
      // 已启动：不强制 refresh，避免与 watch(tick) 形成自激
      return
    }
    started = true
    await refresh()
    session = await createMqttSession({
      onMessage: handleMqttMessage,
      onStatus: (s) => {
        mqttStatus.value = s
      },
    })
    // Always poll as fallback (broker down / mqtt disabled)
    if (pollTimer) clearInterval(pollTimer)
    pollTimer = setInterval(() => {
      void refresh()
    }, 15000)
  }

  function stop() {
    started = false
    lastSig = ''
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = undefined
    }
    session?.end()
    session = null
    mqttStatus.value = 'idle'
    inbox.value = []
    unread.value = 0
    lastPayload.value = null
  }

  return {
    inbox,
    unread,
    loading,
    mqttStatus,
    lastPayload,
    tick,
    refresh,
    markRead,
    start,
    stop,
  }
})
