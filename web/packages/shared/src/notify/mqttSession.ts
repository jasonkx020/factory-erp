import mqtt, { type MqttClient } from 'mqtt'
import { notifyApi } from '../api/endpoints'

export type MqttConnectInfo = {
  enabled?: boolean
  broker_url?: string
  ws_url?: string
  client_id?: string
  username?: string
  password?: string
  keep_alive_seconds?: number
  qos_default?: number
  subscribe_topics?: string[]
}

export type MqttSession = {
  client: MqttClient | null
  info: MqttConnectInfo | null
  end: () => void
}

function normalizeWsUrl(raw: string): string {
  let u = raw.trim()
  if (!u) return u
  // NanoMQ listeners.ws bind is host:8083/mqtt
  try {
    const url = new URL(u)
    if (!url.pathname || url.pathname === '/') {
      url.pathname = '/mqtt'
    }
    return url.toString().replace(/\/$/, '')
  } catch {
    if (!u.includes('/mqtt')) {
      u = u.replace(/\/$/, '') + '/mqtt'
    }
    return u
  }
}

/**
 * Connect to NanoMQ over WebSocket using credentials from GET /notify/mqtt-connect.
 * Calls onMessage for every MQTT payload; returns session with end().
 */
export async function createMqttSession(opts: {
  onMessage?: (topic: string, payload: string) => void
  onStatus?: (status: 'connecting' | 'connected' | 'error' | 'closed', detail?: string) => void
}): Promise<MqttSession> {
  const empty: MqttSession = { client: null, info: null, end: () => undefined }
  try {
    const res = await notifyApi.mqttConnect()
    if (res.code !== 1 || !res.data) {
      opts.onStatus?.('error', res.msg || 'mqtt-connect failed')
      return empty
    }
    const mqttInfo = (res.data as { mqtt?: MqttConnectInfo })?.mqtt
    if (!mqttInfo?.enabled || !mqttInfo.ws_url || !mqttInfo.password) {
      opts.onStatus?.('closed', 'mqtt disabled')
      return { ...empty, info: mqttInfo || null }
    }
    opts.onStatus?.('connecting')
    const url = normalizeWsUrl(String(mqttInfo.ws_url))
    const client = mqtt.connect(url, {
      clientId: String(mqttInfo.client_id || `erp-web-${Date.now()}`),
      username: String(mqttInfo.username || ''),
      password: String(mqttInfo.password || ''),
      keepalive: Number(mqttInfo.keep_alive_seconds || 60),
      clean: true,
      reconnectPeriod: 5000,
      protocolVersion: 4,
    })
    const qos = (Number(mqttInfo.qos_default) === 0 ? 0 : 1) as 0 | 1 | 2
    client.on('connect', () => {
      opts.onStatus?.('connected')
      const topics = mqttInfo.subscribe_topics || []
      for (const t of topics) {
        if (t) client.subscribe(t, { qos })
      }
    })
    client.on('message', (topic, buf) => {
      opts.onMessage?.(topic, buf.toString('utf8'))
    })
    client.on('error', (err) => {
      opts.onStatus?.('error', err?.message || 'mqtt error')
    })
    client.on('close', () => {
      opts.onStatus?.('closed')
    })
    return {
      client,
      info: mqttInfo,
      end: () => {
        try {
          client.end(true)
        } catch {
          /* ignore */
        }
      },
    }
  } catch (e) {
    opts.onStatus?.('error', e instanceof Error ? e.message : String(e))
    return empty
  }
}
