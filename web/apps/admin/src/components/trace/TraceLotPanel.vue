<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getApiBase, purchaseApi } from '@erp/shared'
import { useCarrierCodeLabel } from '../../composables/useCarrierCodeLabel'

type Row = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    /** 外部指定要倒查的码（点选列表时传入） */
    code?: string
    /** 是否显示搜索栏 */
    showSearch?: boolean
    /** 自动查询外部 code */
    autoLoad?: boolean
  }>(),
  { code: '', showSearch: true, autoLoad: true },
)

const emit = defineEmits<{
  'update:code': [string]
  loaded: [Row | null]
}>()

const { codeLabel, short, ensureLoaded } = useCarrierCodeLabel()

const inputCode = ref('')
const loading = ref(false)
const result = ref<Row | null>(null)

function mediaUrl(raw: unknown): string {
  const u = String(raw || '').trim()
  if (!u) return ''
  if (/^https?:\/\//i.test(u)) return u
  if (u.startsWith('/files') || u.startsWith('/uploads')) return u
  const base = getApiBase().replace(/\/api\/v1\/?$/, '')
  return `${base}${u.startsWith('/') ? u : `/${u}`}`
}

const STEP_META = computed(() => ({
  trace_lot: { label: '溯源批次', color: '#0d7a6f' },
  arrival: { label: '到货登记', color: '#409eff' },
  weigh: { label: '过磅收货', color: '#e6a23c' },
  box: { label: '分板入库', color: '#67c23a' },
  boxes: { label: '已分板明细', color: '#67c23a' },
  box_family: { label: `关联${codeLabel.value}`, color: '#909399' },
  farmer_settlement: { label: '农户结算', color: '#f56c6c' },
  audit: { label: '审计纠错', color: '#909399' },
}))

const STATUS_MAP: Record<string, string> = {
  draft: '草稿',
  qc_pending: '待质检',
  qc_pass: '质检合格',
  weighed: '待入厂',
  gate_accepted: '待入库',
  stocked: '已入库',
  settle_pending: '待结算',
  settle_paid: '已支付',
  active: '有效',
}

function statusLabel(v: unknown) {
  const s = String(v || '')
  return STATUS_MAP[s] || s || '-'
}

function fmtKg(v: unknown) {
  if (v == null || v === '') return ''
  const n = Number(v)
  if (Number.isNaN(n)) return String(v)
  return `${n} kg`
}

function fmtMoney(v: unknown) {
  if (v == null || v === '') return ''
  const n = Number(v)
  if (Number.isNaN(n)) return String(v)
  return `¥ ${n.toFixed(2)}`
}

function stepLabel(step: unknown) {
  const s = String(step || '')
  return STEP_META.value[s as keyof typeof STEP_META.value]?.label || s || '未知步骤'
}

function stepColor(step: unknown) {
  const s = String(step || '')
  return STEP_META.value[s as keyof typeof STEP_META.value]?.color || '#909399'
}

function evidenceUrls(ev: Row): string[] {
  const out: string[] = []
  const add = (v: unknown) => {
    if (Array.isArray(v)) {
      v.forEach(add)
      return
    }
    if (v && typeof v === 'object') {
      const r = v as Row
      add(r.file_url ?? r.url ?? r.image_url)
      return
    }
    const url = mediaUrl(v)
    if (url && !out.includes(url)) out.push(url)
  }
  add(ev.evidences)
  const data = (ev.data as Row) || {}
  add(data.image_url)
  add(data.image_urls)
  return out
}

type Field = { label: string; value: string }

function pick(data: Row | null | undefined, keys: { key: string; label: string; fmt?: (v: unknown) => string }[]): Field[] {
  if (!data) return []
  const out: Field[] = []
  for (const k of keys) {
    let v = data[k.key]
    if (v == null || v === '') continue
    if (k.key === 'status') v = statusLabel(v)
    const text = k.fmt ? k.fmt(v) : String(v)
    if (!text) continue
    out.push({ label: k.label, value: text })
  }
  return out
}

function eventFields(ev: Row): Field[] {
  const step = String(ev.step || '')
  const data = ((ev.data as Row) || ev) as Row

  if (step === 'trace_lot') {
    return pick(data, [
      { key: 'trace_code', label: '溯源码' },
      { key: 'batch_no', label: '批号' },
      { key: 'biz_date', label: '业务日' },
      { key: 'net_weight', label: '净重', fmt: fmtKg },
      { key: 'ticket_count', label: '过磅车数', fmt: (v) => `${v} 车` },
      { key: 'boxed_qty', label: '板数', fmt: (v) => `${v} 板` },
      { key: 'grade', label: '等级' },
      { key: 'status', label: '状态' },
      { key: 'farmer_id', label: '农户ID' },
    ])
  }
  if (step === 'arrival') {
    return pick(data, [
      { key: 'doc_no', label: '到货单号' },
      { key: 'farmer_name', label: '农户' },
      { key: 'estimate_weight', label: '估重', fmt: fmtKg },
      { key: 'plate_no', label: '车牌' },
      { key: 'origin', label: '产地' },
      { key: 'qc_result', label: '质检' },
      { key: 'grade', label: '等级' },
      { key: 'status', label: '状态' },
    ])
  }
  if (step === 'weigh') {
    return pick(data, [
      { key: 'doc_no', label: '过磅单号' },
      { key: 'trace_code', label: '溯源码' },
      { key: 'batch_no', label: '批号' },
      { key: 'party_name', label: '姓名' },
      { key: 'farmer_name', label: '农户' },
      { key: 'plate_no', label: '车牌' },
      { key: 'variety', label: '品种' },
      { key: 'product_name', label: '物料' },
      { key: 'gross_weight', label: '毛重', fmt: fmtKg },
      { key: 'net_weight', label: '净重', fmt: fmtKg },
      { key: 'settle_amount', label: '结算额', fmt: fmtMoney },
      { key: 'status', label: '状态' },
      { key: 'biz_date', label: '业务日' },
    ])
  }
  if (step === 'box') {
    return [
      ...(ev.box_code ? [{ label: codeLabel.value, value: String(ev.box_code) }] : []),
      ...(ev.box_id ? [{ label: `${short.value}ID`, value: String(ev.box_id) }] : []),
    ]
  }
  if (step === 'box_family') {
    const related = ev.related_boxes
    const text = Array.isArray(related) ? related.map(String).join('、') : String(related || '')
    return text ? [{ label: `关联${short.value}`, value: text }] : []
  }
  if (step === 'boxes') {
    const boxes = (ev.boxes as Row[]) || (data.boxes as Row[]) || []
    const qty = ev.boxed_qty != null ? Number(ev.boxed_qty) : boxes.length
    if (!boxes.length) return [{ label: `${short.value}数`, value: String(qty || 0) }]
    const totalW = boxes.reduce((s, b) => s + (Number(b.weight) || 0), 0)
    return [
      { label: `${short.value}数`, value: String(qty) },
      { label: '合计重量', value: fmtKg(totalW) },
    ]
  }
  if (step === 'farmer_settlement') {
    return pick(data, [
      { key: 'doc_no', label: '结算单' },
      { key: 'amount', label: '金额', fmt: fmtMoney },
      { key: 'status', label: '状态' },
      { key: 'transfer_no', label: '转账单号' },
      { key: 'paid_at', label: '支付时间' },
    ])
  }
  if (step === 'audit') {
    return [
      ...(ev.action ? [{ label: '动作', value: String(ev.action) }] : []),
      ...(ev.reason ? [{ label: '原因', value: String(ev.reason) }] : []),
      ...(ev.actor_user_id ? [{ label: '操作人', value: String(ev.actor_user_id) }] : []),
      ...(ev.at ? [{ label: '时间', value: String(ev.at) }] : []),
    ]
  }
  // fallback：只展示少量安全字段
  return pick(data, [
    { key: 'doc_no', label: '单号' },
    { key: 'code', label: '编码' },
    { key: 'status', label: '状态' },
    { key: 'at', label: '时间' },
  ])
}

function eventBoxes(ev: Row): Row[] {
  if (String(ev.step) !== 'boxes' && String(ev.step) !== 'weigh') return []
  const data = (ev.data as Row) || {}
  const boxes = (ev.boxes as Row[]) || (data.boxes as Row[]) || []
  return Array.isArray(boxes) ? boxes : []
}

function boxPhotoList(box: Row): string[] {
  const out: string[] = []
  const add = (v: unknown) => {
    if (Array.isArray(v)) {
      v.forEach(add)
      return
    }
    const url = mediaUrl(v)
    if (url && !out.includes(url)) out.push(url)
  }
  add(box.image_url)
  add(box.image_urls)
  return out
}

const timeline = computed(() => {
  const list = result.value?.timeline
  return Array.isArray(list) ? (list as Row[]) : []
})

const lot = computed(() => (result.value?.lot as Row) || null)

const summaryFields = computed(() => {
  const lotRow = lot.value
  const weighEv = timeline.value.find((e) => String(e.step) === 'weigh')
  const weigh = ((weighEv?.data as Row) || weighEv || {}) as Row
  const src = lotRow || weigh
  if (!src || !Object.keys(src).length) {
    if (result.value?.trace_code) {
      return [{ label: '查询码', value: String(result.value.trace_code) }]
    }
    return [] as Field[]
  }
  return pick(
    {
      ...weigh,
      ...src,
      trace_code: src.trace_code || result.value?.trace_code || weigh.trace_code,
      farmer_name: weigh.farmer_name || weigh.party_name || src.farmer_name,
      boxed_qty: src.boxed_qty ?? result.value?.boxed_qty ?? (result.value?.box_completion as Row)?.boxed_qty,
      boxed_weight: src.boxed_weight ?? result.value?.boxed_weight ?? (result.value?.box_completion as Row)?.boxed_weight,
      ticket_count: src.ticket_count ?? result.value?.ticket_count,
      product_name: (result.value?.box_completion as Row)?.product_name ?? weigh.product_name,
      product_category: (result.value?.box_completion as Row)?.product_category,
      remaining_weight: (result.value?.box_completion as Row)?.remaining_weight,
    },
    [
      { key: 'trace_code', label: '溯源码' },
      { key: 'batch_no', label: '批号' },
      { key: 'biz_date', label: '业务日' },
      { key: 'farmer_name', label: '农户' },
      { key: 'party_name', label: '姓名' },
      { key: 'product_name', label: '物料' },
      { key: 'product_category', label: '品类' },
      { key: 'net_weight', label: '净重', fmt: fmtKg },
      { key: 'ticket_count', label: '过磅车数', fmt: (v) => `${v} 车` },
      { key: 'boxed_qty', label: '板数', fmt: (v) => `${v} 板` },
      { key: 'remaining_weight', label: '待分 kg', fmt: fmtKg },
      { key: 'doc_no', label: '过磅单' },
      { key: 'status', label: '状态' },
    ],
  )
})

const signatureValid = computed(() => {
  const v = lot.value?.signature_valid
  if (v === true) return true
  if (v === false) return false
  return null
})

async function load(code?: string) {
  const c = String(code ?? inputCode.value ?? props.code ?? '').trim()
  if (!c) {
    ElMessage.warning(`请输入溯源批号 / 溯源码 / ${codeLabel.value} / 过磅单号`)
    return
  }
  inputCode.value = c
  emit('update:code', c)
  loading.value = true
  try {
    const res = await purchaseApi.traceLot(c)
    if (res.code !== 1) {
      ElMessage.error(res.msg)
      result.value = null
      emit('loaded', null)
      return
    }
    result.value = (res.data as Row) || null
    emit('loaded', result.value)
  } finally {
    loading.value = false
  }
}

function clear() {
  result.value = null
  emit('loaded', null)
}

watch(
  () => props.code,
  (c) => {
    const next = String(c || '').trim()
    if (next && next !== inputCode.value) {
      inputCode.value = next
      if (props.autoLoad) void load(next)
    }
  },
  { immediate: true },
)

defineExpose({ load, clear, result })

onMounted(() => {
  void ensureLoaded()
})
</script>

<template>
  <div class="trace-panel" v-loading="loading">
    <div v-if="showSearch" class="search-row">
      <el-input
        v-model="inputCode"
        clearable
        :placeholder="`溯源批号 / T1- / ${codeLabel} / 过磅单号`"
        @keyup.enter="load()"
      />
      <el-button type="primary" @click="load()">倒查</el-button>
    </div>

    <template v-if="result">
      <el-alert
        v-if="signatureValid === false"
        type="warning"
        title="HMAC 验签未通过，溯源签名可能被篡改或密钥不匹配"
        show-icon
        :closable="false"
        class="mb"
      />
      <el-alert
        v-else-if="signatureValid === true"
        type="success"
        title="溯源签名校验通过"
        show-icon
        :closable="false"
        class="mb"
      />
      <el-alert
        v-if="(result?.box_completion as Row)?.complete === true"
        type="success"
        title="该溯源分板已全部完成"
        show-icon
        :closable="false"
        class="mb"
      />
      <el-alert
        v-else-if="(result?.box_completion as Row)?.trace_code"
        type="info"
        :title="`分板进行中：已 ${(result?.box_completion as Row)?.boxed_qty ?? 0} 板，剩余 ${Number((result?.box_completion as Row)?.remaining_weight || 0).toFixed(2)} kg`"
        show-icon
        :closable="false"
        class="mb"
      />

      <div v-if="summaryFields.length" class="summary">
        <h4 class="sec">摘要</h4>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item v-for="f in summaryFields" :key="f.label" :label="f.label">
            {{ f.value }}
          </el-descriptions-item>
          <el-descriptions-item v-if="signatureValid !== null" label="验签">
            <el-tag :type="signatureValid ? 'success' : 'danger'" size="small">
              {{ signatureValid ? '通过' : '未通过' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <h4 class="sec">流转时间轴</h4>
      <el-timeline v-if="timeline.length">
        <el-timeline-item
          v-for="(ev, i) in timeline"
          :key="i"
          :timestamp="String(ev.at || (ev.data as Row)?.biz_date || (ev.data as Row)?.created_at || '')"
          :color="stepColor(ev.step)"
          placement="top"
        >
          <div class="tl-card">
            <div class="tl-title">{{ stepLabel(ev.step) }}</div>
            <el-descriptions v-if="eventFields(ev).length" :column="2" size="small" class="tl-desc">
              <el-descriptions-item v-for="f in eventFields(ev)" :key="f.label" :label="f.label">
                {{ f.value }}
              </el-descriptions-item>
            </el-descriptions>

            <div v-if="evidenceUrls(ev).length" class="photos">
              <el-image
                v-for="(url, pi) in evidenceUrls(ev)"
                :key="url + pi"
                :src="url"
                :preview-src-list="evidenceUrls(ev)"
                :initial-index="pi"
                preview-teleported
                fit="cover"
                class="photo"
              />
            </div>

            <div v-if="eventBoxes(ev).length" class="box-list">
              <div v-for="box in eventBoxes(ev)" :key="String(box.id || box.code)" class="box-item">
                <div class="box-meta">
                  <strong>{{ box.code || '-' }}</strong>
                  <span>{{ box.weight != null ? `${box.weight} kg` : '-' }}</span>
                </div>
                <div v-if="boxPhotoList(box).length" class="photos">
                  <el-image
                    v-for="(url, pi) in boxPhotoList(box)"
                    :key="url + pi"
                    :src="url"
                    :preview-src-list="boxPhotoList(box)"
                    :initial-index="pi"
                    preview-teleported
                    fit="cover"
                    class="photo sm"
                  />
                </div>
              </div>
            </div>
          </div>
        </el-timeline-item>
      </el-timeline>
      <p v-else class="muted">暂无流转记录</p>

      <el-collapse class="raw-collapse">
        <el-collapse-item title="原始数据（排查用）" name="raw">
          <pre class="raw">{{ JSON.stringify(result, null, 2) }}</pre>
        </el-collapse-item>
      </el-collapse>
    </template>
    <p v-else-if="!loading" class="muted">输入或点选左侧单据后点击倒查</p>
  </div>
</template>

<style scoped>
.search-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.mb { margin-bottom: 12px; }
.sec {
  margin: 12px 0 8px;
  font-size: 14px;
  font-weight: 600;
}
.summary { margin-bottom: 4px; }
.tl-card {
  background: #f8fafb;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
}
.tl-title { font-weight: 600; margin-bottom: 6px; }
.tl-desc { margin-top: 4px; }
.photos { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.photo { width: 72px; height: 72px; border-radius: 6px; cursor: pointer; }
.photo.sm { width: 56px; height: 56px; }
.box-list { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; }
.box-item {
  background: #fff;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 8px;
}
.box-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  margin-bottom: 4px;
}
.muted { color: #889; font-size: 13px; }
.raw-collapse { margin-top: 12px; }
.raw {
  background: #f6f8fa;
  padding: 10px;
  border-radius: 6px;
  font-size: 11px;
  max-height: 280px;
  overflow: auto;
  margin: 0;
}
</style>
