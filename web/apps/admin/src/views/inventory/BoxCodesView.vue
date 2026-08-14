<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TableInstance } from 'element-plus'
import { jsPDF } from 'jspdf'
import QRCode from 'qrcode'
import { inventoryApi, productApi } from '@erp/shared'
import { WarehouseSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const boxCols: MobileCardColumn[] = [
  { prop: 'code', label: '箱码', primary: true },
  { prop: 'status', label: '状态' },
  { prop: 'weight', label: '重量' },
  { prop: 'trace_code', label: '溯源码' },
  { prop: 'product_id', label: '物料' },
]

const loading = ref(false)
const exporting = ref(false)
const list = ref<Row[]>([])
const products = ref<Row[]>([])
const selected = ref<Row[]>([])
const tableRef = ref<TableInstance>()
/** code -> dataURL */
const qrMap = ref<Record<string, string>>({})
const viewMode = ref<'table' | 'labels'>('table')
const previewVisible = ref(false)
const previewCode = ref('')
const previewQr = ref('')
const previewRow = ref<Row | null>(null)
const boxCode = ref('')
const boxTrace = ref<Row | null>(null)
const total = ref(0)

const boxForm = reactive({
  code: '',
  product_id: 1,
  warehouse_id: 1,
  weight: 0,
  batch_no: '',
  trace_code: '',
})
const filter = reactive({
  status: '',
  q: '',
  trace_code: '',
})

const statusOptions = [
  { value: 'open', label: '在用(open)' },
  { value: 'active', label: '在用(active)' },
  { value: 'finished', label: '已完结' },
  { value: 'destroyed', label: '已销毁' },
]

const exportCodes = computed(() => {
  const rows = selected.value.length ? selected.value : list.value
  return rows.map((x) => String(x.code || '')).filter(Boolean)
})

function statusLabel(st: unknown) {
  const s = String(st || '').toLowerCase()
  if (s === 'open' || s === 'active') return '在用'
  if (s === 'finished') return '已完结'
  if (s === 'destroyed') return '已销毁'
  return s || '-'
}

function statusTagType(st: unknown): 'success' | 'warning' | 'info' | 'danger' {
  const s = String(st || '').toLowerCase()
  if (s === 'open' || s === 'active') return 'success'
  if (s === 'finished') return 'info'
  if (s === 'destroyed') return 'danger'
  return 'warning'
}

function productName(pid: unknown) {
  const id = Number(pid)
  const p = products.value.find((x) => Number(x.id) === id)
  return p ? String(p.name || p.code || id) : String(pid ?? '-')
}

async function buildQr(code: string, force = false) {
  if (!code || (qrMap.value[code] && !force)) return
  try {
    const url = await QRCode.toDataURL(code, {
      errorCorrectionLevel: 'M',
      margin: 1,
      width: 320,
      color: { dark: '#000000', light: '#ffffff' },
    })
    qrMap.value = { ...qrMap.value, [code]: url }
  } catch {
    /* ignore */
  }
}

async function buildAllQr(force = false) {
  const codes = list.value.map((x) => String(x.code || '')).filter(Boolean)
  await Promise.all(codes.map((c) => buildQr(c, force)))
}

async function loadProducts() {
  const res = await productApi.list()
  if (res.code !== 1) return
  products.value = ((res.data as { list?: Row[] })?.list) || []
  const pid = Number(products.value[0]?.id || 0)
  if (pid > 0) boxForm.product_id = pid
}

async function refresh() {
  loading.value = true
  try {
    const q = new URLSearchParams()
    q.set('page_size', '200')
    if (filter.status) q.set('status', filter.status)
    if (filter.q.trim()) q.set('q', filter.q.trim())
    if (filter.trace_code.trim()) q.set('trace_code', filter.trace_code.trim())
    const res = await inventoryApi.listBoxes(q.toString())
    if (res.code !== 1) {
      ElMessage.error(res.msg)
      return
    }
    const data = res.data as { list?: Row[]; total?: number }
    list.value = data?.list || []
    total.value = Number(data?.total || list.value.length)
    selected.value = []
    tableRef.value?.clearSelection()
    await buildAllQr()
  } finally {
    loading.value = false
  }
}

async function createBox() {
  if (!boxForm.code) return ElMessage.warning('填写箱码')
  if (!String(boxForm.trace_code || '').trim()) return ElMessage.warning('箱码须绑定溯源码')
  const res = await inventoryApi.createBox({ ...boxForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('箱码已建')
  boxForm.code = ''
  boxForm.trace_code = ''
  await refresh()
}

async function destroyBox(row: Row) {
  const id = Number(row.id)
  if (!id) return
  try {
    const { value } = await ElMessageBox.prompt('填写销毁原因（损耗等用不了的箱须标注销毁）', '销毁箱码', {
      confirmButtonText: '销毁',
      cancelButtonText: '取消',
      inputPattern: /\S+/,
      inputErrorMessage: '请填写原因',
    })
    const res = await inventoryApi.destroyBox(id, { reason: String(value || '').trim() })
    if (res.code !== 1) return ElMessage.error(res.msg)
    ElMessage.success('已销毁')
    await refresh()
  } catch {
    /* cancel */
  }
}

async function doBoxTrace() {
  if (!boxCode.value) return
  const res = await inventoryApi.boxTrace(boxCode.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  boxTrace.value = (res.data as Row) || null
}

function onSelectionChange(rows: Row[]) {
  selected.value = rows
}

async function openPreview(row: Row) {
  const code = String(row.code || '')
  if (!code) return
  await buildQr(code)
  previewCode.value = code
  previewQr.value = qrMap.value[code] || ''
  previewRow.value = row
  previewVisible.value = true
}

function copyCodes() {
  const text = exportCodes.value.join('\n')
  if (!text) return ElMessage.warning('无数据')
  void navigator.clipboard.writeText(text).then(() =>
    ElMessage.success(selected.value.length ? `已复制选中 ${exportCodes.value.length} 个箱码` : '已复制当前列表箱码'),
  )
}

function printLabels() {
  const codes = exportCodes.value
  if (!codes.length) return ElMessage.warning(selected.value.length ? '请先勾选箱码' : '无箱码可打印')
  void Promise.all(codes.map((c) => buildQr(c))).then(() => {
    const win = window.open('', '_blank', 'noopener,noreferrer')
    if (!win) return ElMessage.error('浏览器拦截了打印窗口，请允许弹窗')
    const cards = codes
      .map((code) => {
        const src = qrMap.value[code] || ''
        return `<div class="label"><img src="${src}" alt=""/><div class="code">${code}</div></div>`
      })
      .join('')
    win.document.write(`<!doctype html><html><head><meta charset="utf-8"/><title>箱码二维码</title>
<style>
  @page { margin: 10mm; }
  body { font-family: Arial, "Microsoft YaHei", sans-serif; margin: 0; }
  .sheet { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
  .label { border: 1px solid #ccc; border-radius: 6px; padding: 8px 6px; text-align: center; page-break-inside: avoid; break-inside: avoid; }
  .label img { width: 120px; height: 120px; display: block; margin: 0 auto 6px; }
  .code { font-size: 11px; font-weight: 600; letter-spacing: 0.02em; word-break: break-all; line-height: 1.3; }
</style></head><body>
<div class="sheet">${cards}</div>
<script>window.onload=function(){window.focus();window.print()}<\/script>
</body></html>`)
    win.document.close()
  })
}

/** A4 批量 PDF：4 列 × 5 行 / 页；优先导出勾选行，未勾选则导出当前列表 */
async function exportPdf() {
  const codes = exportCodes.value
  if (!codes.length) return ElMessage.warning('请先勾选要导出的箱码，或确保列表有数据')
  exporting.value = true
  try {
    await Promise.all(codes.map((c) => buildQr(c, true)))
    const doc = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4' })
    const pageW = doc.internal.pageSize.getWidth()
    const pageH = doc.internal.pageSize.getHeight()
    const marginX = 10
    const marginY = 12
    const cols = 4
    const rows = 5
    const gapX = 4
    const gapY = 4
    const cellW = (pageW - marginX * 2 - gapX * (cols - 1)) / cols
    const cellH = (pageH - marginY * 2 - gapY * (rows - 1) - 8) / rows
    const qrSize = Math.min(cellW - 6, cellH - 14)
    const perPage = cols * rows
    const scope = selected.value.length ? `Selected ${codes.length}` : `List ${codes.length}`
    const title = `Box Codes · ${scope}`
    const fname = `box-codes_${new Date().toISOString().slice(0, 10)}_${codes.length}.pdf`

    codes.forEach((code, i) => {
      const pageIdx = Math.floor(i / perPage)
      const onPage = i % perPage
      if (onPage === 0) {
        if (pageIdx > 0) doc.addPage()
        doc.setFontSize(10)
        doc.setTextColor(60)
        doc.text(title, marginX, 8)
        doc.setFontSize(8)
        doc.text(`Page ${pageIdx + 1}`, pageW - marginX, 8, { align: 'right' })
      }
      const col = onPage % cols
      const row = Math.floor(onPage / cols)
      const x = marginX + col * (cellW + gapX)
      const y = marginY + row * (cellH + gapY)

      doc.setDrawColor(180)
      doc.setLineWidth(0.2)
      doc.roundedRect(x, y, cellW, cellH, 1.5, 1.5, 'S')

      const img = qrMap.value[code]
      const imgX = x + (cellW - qrSize) / 2
      const imgY = y + 3
      if (img) doc.addImage(img, 'PNG', imgX, imgY, qrSize, qrSize)

      doc.setTextColor(0)
      doc.setFontSize(7.5)
      const textY = imgY + qrSize + 4
      const lines = doc.splitTextToSize(code, cellW - 4)
      doc.text(lines, x + cellW / 2, textY, { align: 'center', baseline: 'top' })
    })

    doc.save(fname)
    ElMessage.success(
      `已导出 PDF：${fname}（${selected.value.length ? `选中 ${codes.length}` : `列表 ${codes.length}`} 个 · A4 4×5）`,
    )
  } catch (e) {
    ElMessage.error(`导出失败：${e instanceof Error ? e.message : e}`)
  } finally {
    exporting.value = false
  }
}

watch(viewMode, (m) => {
  if (m === 'labels') void buildAllQr()
})

onMounted(async () => {
  await loadProducts()
  await refresh()
})
</script>

<template>
  <div v-loading="loading || exporting" :element-loading-text="exporting ? '正在生成 PDF…' : ''">
    <el-card header="新建箱码" class="mb">
      <el-form inline size="small">
        <el-form-item label="箱码"><el-input v-model="boxForm.code" style="width:160px" /></el-form-item>
        <el-form-item label="溯源码"><el-input v-model="boxForm.trace_code" style="width:160px" placeholder="必填" /></el-form-item>
        <el-form-item label="物料">
          <el-select v-model="boxForm.product_id" style="width:160px" filterable>
            <el-option v-for="p in products" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
          </el-select>
        </el-form-item>
        <el-form-item label="仓库"><WarehouseSelect v-model="boxForm.warehouse_id" /></el-form-item>
        <el-form-item label="重量"><el-input-number v-model="boxForm.weight" :min="0" /></el-form-item>
        <el-button type="primary" @click="createBox">新建</el-button>
      </el-form>
      <p class="hint">可勾选箱码批量导出 PDF（含二维码）便于打印；未勾选时导出当前筛选列表。</p>
    </el-card>

    <el-card header="箱码追溯" class="mb">
      <el-form inline size="small">
        <el-form-item label="箱码"><el-input v-model="boxCode" style="width:180px" /></el-form-item>
        <el-button type="primary" @click="doBoxTrace">追溯</el-button>
      </el-form>
      <pre v-if="boxTrace" class="trace">{{ boxTrace }}</pre>
    </el-card>

    <el-card>
      <template #header>
        <div class="hdr">
          <span>箱码 / 二维码 · 共 {{ total }}</span>
          <div class="hdr-actions">
            <el-input
              v-model="filter.q"
              clearable
              placeholder="箱码"
              size="small"
              style="width: 140px"
              @keyup.enter="refresh"
            />
            <el-input
              v-model="filter.trace_code"
              clearable
              placeholder="溯源码"
              size="small"
              style="width: 140px"
              @keyup.enter="refresh"
            />
            <el-select
              v-model="filter.status"
              clearable
              placeholder="使用状态"
              size="small"
              style="width: 130px"
              @change="refresh"
            >
              <el-option v-for="o in statusOptions" :key="o.value" :value="o.value" :label="o.label" />
            </el-select>
            <el-radio-group v-model="viewMode" size="small">
              <el-radio-button value="table">列表</el-radio-button>
              <el-radio-button value="labels">标签预览</el-radio-button>
            </el-radio-group>
            <el-button size="small" @click="refresh">刷新</el-button>
            <el-button size="small" @click="copyCodes">复制箱码</el-button>
            <el-button size="small" @click="printLabels">浏览器打印</el-button>
            <el-button type="primary" size="small" :loading="exporting" @click="exportPdf">
              {{ selected.length ? `导出选中 PDF(${selected.length})` : '导出列表 PDF' }}
            </el-button>
          </div>
        </div>
      </template>

      <TableOrCards v-if="viewMode === 'table'" :data="list" :loading="loading" :columns="boxCols">
        <el-table
          ref="tableRef"
          :data="list"
          size="small"
          stripe
          row-key="id"
          @selection-change="onSelectionChange"
        >
          <el-table-column type="selection" width="42" reserve-selection />
          <el-table-column label="二维码" width="88" align="center">
            <template #default="{ row }">
              <img
                v-if="qrMap[String(row.code || '')]"
                class="qr-thumb"
                :src="qrMap[String(row.code || '')]"
                alt=""
                @click="openPreview(row)"
              />
              <span v-else class="muted">…</span>
            </template>
          </el-table-column>
          <el-table-column prop="code" label="箱码" min-width="150" />
          <el-table-column label="使用状态" width="110">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small" :title="String(row.status || '')">
                {{ statusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="物料" min-width="120">
            <template #default="{ row }">{{ productName(row.product_id) }}</template>
          </el-table-column>
          <el-table-column prop="warehouse_id" label="仓" width="70" />
          <el-table-column prop="weight" label="重量" width="80" />
          <el-table-column prop="trace_code" label="溯源码" min-width="140" show-overflow-tooltip />
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openPreview(row)">查看</el-button>
              <el-button
                v-if="!['destroyed', 'finished'].includes(String(row.status || '').toLowerCase())"
                link
                type="danger"
                @click="destroyBox(row)"
              >销毁</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #extra="{ row }">
          <img
            v-if="qrMap[String(row.code || '')]"
            class="qr-thumb"
            :src="qrMap[String(row.code || '')]"
            alt=""
            @click="openPreview(row)"
          />
          <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
        <template #actions="{ row }">
          <el-button link type="primary" @click="openPreview(row)">查看</el-button>
          <el-button
            v-if="!['destroyed', 'finished'].includes(String(row.status || '').toLowerCase())"
            link
            type="danger"
            @click="destroyBox(row)"
          >销毁</el-button>
        </template>
      </TableOrCards>

      <div v-else class="label-grid">
        <div
          v-for="row in list"
          :key="String(row.id || row.code)"
          class="label-card"
          :class="{ selected: selected.some((s) => s.id === row.id) }"
          @click="openPreview(row)"
        >
          <img v-if="qrMap[String(row.code || '')]" :src="qrMap[String(row.code || '')]" alt="" />
          <div class="code">{{ row.code }}</div>
          <div class="meta">{{ statusLabel(row.status) }} · {{ productName(row.product_id) }}</div>
        </div>
        <el-empty v-if="!list.length" description="暂无箱码" />
      </div>
    </el-card>

    <el-dialog v-model="previewVisible" title="箱码二维码" width="380px" align-center>
      <div class="preview">
        <img v-if="previewQr" :src="previewQr" alt="qr" />
        <div class="preview-code">{{ previewCode }}</div>
        <div v-if="previewRow" class="preview-meta">
          <el-tag :type="statusTagType(previewRow.status)" size="small">{{ statusLabel(previewRow.status) }}</el-tag>
          <span>物料 {{ productName(previewRow.product_id) }}</span>
          <span v-if="previewRow.weight != null">重量 {{ previewRow.weight }}</span>
          <span v-if="previewRow.trace_code">溯源 {{ previewRow.trace_code }}</span>
        </div>
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button @click="printLabels">浏览器打印</el-button>
        <el-button type="primary" :loading="exporting" @click="exportPdf">导出 PDF</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.mb { margin-bottom: 12px; }
.hdr { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.hdr-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.hint { color: #888; font-size: 12px; margin: 8px 0 0; }
.muted { color: #bbb; }
.qr-thumb { width: 56px; height: 56px; cursor: pointer; vertical-align: middle; }
.trace {
  background: #f6f8fa;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  max-height: 240px;
  overflow: auto;
  margin-top: 8px;
}
.label-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}
.label-card {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 12px 8px;
  text-align: center;
  cursor: pointer;
  background: #fff;
}
.label-card:hover,
.label-card.selected { border-color: var(--el-color-primary); }
.label-card img { width: 120px; height: 120px; display: block; margin: 0 auto 8px; }
.label-card .code { font-size: 12px; font-weight: 600; word-break: break-all; line-height: 1.35; }
.label-card .meta { margin-top: 4px; font-size: 11px; color: #888; }
.preview { text-align: center; }
.preview img { width: 220px; height: 220px; }
.preview-code { margin-top: 12px; font-size: 14px; font-weight: 700; word-break: break-all; }
.preview-meta {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  font-size: 12px;
  color: #666;
}
</style>
