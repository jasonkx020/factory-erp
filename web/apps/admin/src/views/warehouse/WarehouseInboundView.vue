<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getApiBase, notifyApi, purchaseApi, parsePayload } from '@erp/shared'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const loading = ref(false)
const detailLoading = ref(false)
const assignBusy = ref(false)
const tasks = ref<Row[]>([])
const warehouseUsers = ref<Row[]>([])

const detailVisible = ref(false)
const detail = ref<Row | null>(null)
const detailPhotos = ref<string[]>([])
const detailBoxes = ref<Row[]>([])
const activeTask = ref<Row | null>(null)

const assignVisible = ref(false)
const assignForm = reactive({ to_user_id: null as number | null, comment: '' })

function payloadOf(row: Row) {
  return parsePayload(row.payload ?? row.payload_json)
}

function mediaUrl(raw: unknown): string {
  const u = String(raw || '').trim()
  if (!u) return ''
  if (/^https?:\/\//i.test(u)) return u
  if (u.startsWith('/files') || u.startsWith('/uploads')) return u
  const base = getApiBase().replace(/\/api\/v1\/?$/, '')
  return `${base}${u.startsWith('/') ? u : `/${u}`}`
}

/** 待入厂 | 待入库 */
function phaseOf(row: Row): 'gate' | 'stockin' {
  const p = payloadOf(row)
  const status = String(p.status || row.status || '').toLowerCase()
  const kind = String(p.receive_kind || row.receive_kind || 'gate').toLowerCase()
  if (status === 'gate_accepted' || p.box_stockin_ready === true || kind === 'stockin') {
    return 'stockin'
  }
  return 'gate'
}

function phaseLabel(row: Row) {
  return phaseOf(row) === 'stockin' ? '待入库' : '待入厂'
}

function rowClassName({ row }: { row: Row }) {
  return phaseOf(row) === 'stockin' ? 'row-stockin' : 'row-gate'
}

function assigneeLabel(row: Row) {
  const name = String(row.assignee_name || '').trim()
  const id = Number(row.assignee_user_id || 0)
  if (name) return name
  if (id > 0) {
    const u = warehouseUsers.value.find((x) => Number(x.user_id ?? x.id) === id)
    if (u) return String(u.name || u.login_name || id)
    return `#${id}`
  }
  return '未指定'
}

const taskCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '过磅单号', primary: true },
  { prop: 'phase', label: '阶段' },
  { prop: 'trace_code', label: '溯源码' },
  { prop: 'assignee', label: '处理人' },
  { prop: 'weights', label: '入场/净重' },
  { prop: 'created_at', label: '推送时间' },
]

const tasksForCards = computed(() =>
  tasks.value.map((row) => {
    const p = payloadOf(row)
    return {
      ...row,
      phase: phaseLabel(row),
      plate_no: p.plate_no || '-',
      weights: `${p.gross_weight ?? '-'} / ${p.net_weight ?? '-'}`,
      assignee: assigneeLabel(row),
      _phase: phaseOf(row),
    }
  }),
)

function collectPhotos(m: Row): string[] {
  const out: string[] = []
  const add = (v: unknown) => {
    if (Array.isArray(v)) {
      v.forEach(add)
      return
    }
    if (v && typeof v === 'object') {
      const row = v as Row
      // 分箱复磅图单独展示，现场照片区跳过 box_reweigh
      if (String(row.evidence_type || '') === 'box_reweigh') return
      add(row.file_url ?? row.url)
      return
    }
    const url = mediaUrl(v)
    if (url && !out.includes(url)) out.push(url)
  }
  add(m.image_url)
  add(m.image_urls)
  add(m.verify_images)
  add(m.site_photos)
  add(m.evidences)
  return out
}

function boxPhotos(box: Row): string[] {
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

function kv(label: string, value: unknown) {
  if (value == null || value === '') return null
  return { label, value: String(value) }
}

const detailKvs = computed(() => {
  const m = detail.value
  if (!m) return [] as { label: string; value: string }[]
  const deduct = m.deduct_weight != null && `${m.deduct_weight}` !== ''
    ? `${m.deduct_weight} kg${m.deduct_rate != null && `${m.deduct_rate}` !== '' ? `（${m.deduct_rate}%）` : ''}`
    : null
  return [
    kv('单号', m.doc_no),
    kv('状态', m.status),
    kv('模式', String(m.receive_kind || '').toLowerCase() === 'stockin' ? '入库' : '入厂'),
    kv('溯源码', m.trace_code),
    kv('批号', m.batch_no),
    kv('农户', m.party_name || m.farmer_name),
    kv('品种', m.product_name || m.variety),
    kv('车牌', m.plate_no),
    kv('业务日', m.biz_date),
    kv('毛重', m.gross_weight != null ? `${m.gross_weight} kg` : null),
    kv('扣损', deduct),
    kv('净重', m.net_weight != null ? `${m.net_weight} kg` : null),
    kv('已分箱', m.boxed_qty != null ? `${m.boxed_qty} 箱 / ${m.boxed_weight ?? '-'} kg` : null),
    kv('运费', m.freight_fee),
    kv('装车费', m.loading_fee),
    kv('过磅费', m.weigh_fee),
  ].filter(Boolean) as { label: string; value: string }[]
})

async function loadWarehouseUsers() {
  const res = await purchaseApi.purchaseRoleUsers('warehouse')
  if (res.code !== 1) return
  const list = (res.data as { list?: Row[] })?.list || []
  warehouseUsers.value = list
}

async function refresh() {
  loading.value = true
  try {
    const res = await notifyApi.tasks('status=pending&page_num=1&page_size=50')
    if (res.code !== 1) return ElMessage.error(res.msg)
    const list = ((res.data as { list?: Row[] })?.list) || []
    tasks.value = list.filter(
      (t) => t.event_key === 'purchase.weigh_confirmed' || t.to_role === 'warehouse',
    )
  } finally {
    loading.value = false
  }
}

function bizIdOf(row: Row) {
  const p = payloadOf(row)
  return Number(p.weigh_ticket_id || row.biz_id || p.biz_id || 0)
}

async function openDetail(row: Row) {
  const bizId = bizIdOf(row)
  if (!bizId) return ElMessage.warning('未定位到过磅单')
  activeTask.value = row
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  detailPhotos.value = []
  detailBoxes.value = []
  try {
    const res = await purchaseApi.getWeighTicket(bizId)
    if (res.code !== 1) {
      ElMessage.error(res.msg)
      return
    }
    const m = (res.data as Row) || {}
    detail.value = m
    detailPhotos.value = collectPhotos(m)
    const boxes = m.boxes
    detailBoxes.value = Array.isArray(boxes) ? (boxes as Row[]) : []
  } finally {
    detailLoading.value = false
  }
}

function openAssign(row: Row) {
  activeTask.value = row
  assignForm.to_user_id = Number(row.assignee_user_id || 0) || null
  assignForm.comment = ''
  assignVisible.value = true
}

async function submitAssign() {
  const row = activeTask.value
  if (!row) return
  const tid = Number(row.id)
  if (!tid) return
  if (!assignForm.to_user_id) return ElMessage.warning('请选择仓管')
  assignBusy.value = true
  try {
    const res = await notifyApi.assignTask(tid, {
      to_user_id: assignForm.to_user_id,
      comment: assignForm.comment || undefined,
    })
    if (res.code !== 1) return ElMessage.error(res.msg)
    ElMessage.success('已指定仓管')
    assignVisible.value = false
    await refresh()
  } finally {
    assignBusy.value = false
  }
}

onMounted(async () => {
  await loadWarehouseUsers()
  await refresh()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <h2>仓管待办</h2>
    <p class="hint">
      本页仅查看单据与指定仓管；入厂接收、分箱入库请在 App 由指定仓管处理。
    </p>
    <div class="toolbar">
      <el-button type="primary" @click="refresh">刷新待办</el-button>
      <div class="legend">
        <span class="lg gate">待入厂</span>
        <span class="lg stockin">待入库</span>
      </div>
    </div>

    <TableOrCards :data="tasksForCards" :loading="loading" :columns="taskCols" style="margin-top:12px">
      <el-table :data="tasks" size="small" style="margin-top:12px" :row-class-name="rowClassName">
        <el-table-column prop="doc_no" label="过磅单号" width="160" />
        <el-table-column label="阶段" width="90">
          <template #default="{ row }">
            <el-tag :type="phaseOf(row) === 'stockin' ? 'success' : 'primary'" size="small">
              {{ phaseLabel(row) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="trace_code" label="溯源码" min-width="160" show-overflow-tooltip />
        <el-table-column label="处理人" width="110">
          <template #default="{ row }">{{ assigneeLabel(row) }}</template>
        </el-table-column>
        <el-table-column label="车牌" width="100">
          <template #default="{ row }">{{ payloadOf(row).plate_no || '-' }}</template>
        </el-table-column>
        <el-table-column label="入场/净重" width="120">
          <template #default="{ row }">
            {{ payloadOf(row).gross_weight ?? '-' }} / {{ payloadOf(row).net_weight ?? '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="推送时间" width="160" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">查看详情</el-button>
            <el-button link type="warning" @click="openAssign(row)">指定仓管</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #extra="{ row }">
        <el-tag :type="row._phase === 'stockin' ? 'success' : 'primary'" size="small">{{ row.phase }}</el-tag>
        <span class="muted">{{ row.assignee }}</span>
      </template>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openDetail(row)">查看详情</el-button>
        <el-button link type="warning" @click="openAssign(row)">指定仓管</el-button>
      </template>
    </TableOrCards>

    <el-drawer v-model="detailVisible" title="过磅单详情" size="480px" destroy-on-close>
      <div v-loading="detailLoading">
        <template v-if="detail">
          <div v-if="activeTask" class="phase-bar" :class="phaseOf(activeTask)">
            {{ phaseLabel(activeTask) }}
            <span class="muted">· 处理人 {{ assigneeLabel(activeTask) }}</span>
          </div>
          <div v-for="item in detailKvs" :key="item.label" class="kv">
            <span class="k">{{ item.label }}</span>
            <span class="v">{{ item.value }}</span>
          </div>
          <h4 class="sec">现场照片</h4>
          <div v-if="detailPhotos.length" class="photos">
            <el-image
              v-for="(url, i) in detailPhotos"
              :key="url + i"
              :src="url"
              :preview-src-list="detailPhotos"
              :initial-index="i"
              preview-teleported
              fit="cover"
              class="photo"
            />
          </div>
          <p v-else class="muted">暂无现场照片</p>

          <h4 class="sec">已分箱复磅</h4>
          <div v-if="detailBoxes.length" class="box-list">
            <div v-for="box in detailBoxes" :key="String(box.id || box.code)" class="box-item">
              <div class="box-meta">
                <strong>{{ box.code || '-' }}</strong>
                <span>{{ box.weight ?? '-' }} kg</span>
              </div>
              <div v-if="boxPhotos(box).length" class="photos">
                <el-image
                  v-for="(url, i) in boxPhotos(box)"
                  :key="url + i"
                  :src="url"
                  :preview-src-list="boxPhotos(box)"
                  :initial-index="i"
                  preview-teleported
                  fit="cover"
                  class="photo sm"
                />
              </div>
              <p v-else class="muted">无复磅图</p>
            </div>
          </div>
          <p v-else class="muted">暂无已分箱记录</p>

          <div class="drawer-actions">
            <el-button type="warning" @click="activeTask && openAssign(activeTask)">指定仓管</el-button>
          </div>
        </template>
      </div>
    </el-drawer>

    <el-dialog v-model="assignVisible" title="指定仓管" width="420px" align-center>
      <el-form label-width="88px" size="small">
        <el-form-item label="仓管">
          <el-select v-model="assignForm.to_user_id" filterable placeholder="选择仓管" style="width:100%">
            <el-option
              v-for="u in warehouseUsers"
              :key="String(u.user_id ?? u.id)"
              :label="String(u.name || u.login_name || u.user_id)"
              :value="Number(u.user_id ?? u.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="assignForm.comment" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignVisible = false">取消</el-button>
        <el-button type="primary" :loading="assignBusy" @click="submitAssign">确认指定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
.toolbar { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.legend { display: flex; gap: 8px; }
.lg {
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 4px;
}
.lg.gate { background: #eef2ff; color: #3730a3; }
.lg.stockin { background: #e6f7f2; color: #0f766e; }
.muted { color: #889; font-size: 12px; }
.kv {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 0;
  border-bottom: 1px solid #f0f2f5;
  font-size: 13px;
}
.k { color: #667; flex: 0 0 88px; }
.v { text-align: right; word-break: break-all; font-weight: 500; }
.sec { margin: 16px 0 8px; font-size: 14px; }
.photos { display: flex; flex-wrap: wrap; gap: 8px; }
.photo { width: 96px; height: 96px; border-radius: 8px; cursor: pointer; }
.photo.sm { width: 72px; height: 72px; }
.box-list { display: flex; flex-direction: column; gap: 10px; }
.box-item {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 10px;
  background: #fff;
}
.box-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
}
.drawer-actions { margin-top: 20px; }
.phase-bar {
  padding: 8px 12px;
  border-radius: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
}
.phase-bar.gate { background: #eef2ff; color: #3730a3; }
.phase-bar.stockin { background: #e6f7f2; color: #0f766e; }
</style>

<style>
/* 行背景：待入厂 / 待入库 */
.el-table .row-gate > td.el-table__cell { background: #eef2ff !important; }
.el-table .row-stockin > td.el-table__cell { background: #e6f7f2 !important; }
</style>
