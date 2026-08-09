<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { jsPDF } from 'jspdf'
import QRCode from 'qrcode'
import { purchaseApi } from '@erp/shared'

const router = useRouter()

type Row = Record<string, unknown>

const loading = ref(false)
const exporting = ref(false)
const list = ref<Row[]>([])
/** code -> dataURL */
const qrMap = ref<Record<string, string>>({})
const viewMode = ref<'table' | 'labels'>('table')
const previewVisible = ref(false)
const previewCode = ref('')
const previewQr = ref('')

const form = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  lot_no: '01',
  qty: 20,
})
const filter = reactive({
  biz_date: new Date().toISOString().slice(0, 10),
  status: '',
  lot_no: '',
})

const codes = computed(() => list.value.map((x) => String(x.code || '')).filter(Boolean))

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
    /* ignore single fail */
  }
}

async function buildAllQr(force = false) {
  await Promise.all(codes.value.map((c) => buildQr(c, force)))
}

async function refresh() {
  loading.value = true
  try {
    const q = new URLSearchParams()
    q.set('biz_date', filter.biz_date)
    q.set('page_size', '200')
    if (filter.status) q.set('status', filter.status)
    if (filter.lot_no) q.set('lot_no', filter.lot_no)
    const res = await purchaseApi.traceBatchCodes(q.toString())
    list.value = ((res.data as { list?: Row[] })?.list) || []
    await buildAllQr()
  } finally {
    loading.value = false
  }
}

async function generate() {
  if (form.qty < 1) return ElMessage.warning('数量至少 1')
  const res = await purchaseApi.generateTraceBatchCodes({
    biz_date: form.biz_date,
    lot_no: form.lot_no || '01',
    qty: form.qty,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已生成 ${(res.data as Row)?.qty || form.qty} 条（含二维码）`)
  filter.biz_date = form.biz_date
  await refresh()
  viewMode.value = 'labels'
}

async function voidCode(row: Row) {
  const res = await purchaseApi.voidTraceBatchCode({ code: String(row.code || '') })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已作废')
  await refresh()
}

function copyCodes() {
  const text = codes.value.join('\n')
  if (!text) return ElMessage.warning('无数据')
  void navigator.clipboard.writeText(text).then(() => ElMessage.success('已复制全部批号'))
}

async function openPreview(row: Row) {
  const code = String(row.code || '')
  if (!code) return
  await buildQr(code)
  previewCode.value = code
  previewQr.value = qrMap.value[code] || ''
  previewVisible.value = true
}

function openInboundInfo(row: Row) {
  const code = String(row.code || '').trim()
  if (!code) {
    ElMessage.warning('无溯源批号')
    return
  }
  void router.push({ path: '/purchase/hub/trace', query: { code } })
}

function printLabels() {
  if (!codes.value.length) return ElMessage.warning('无批号可打印')
  const win = window.open('', '_blank', 'noopener,noreferrer')
  if (!win) return ElMessage.error('浏览器拦截了打印窗口，请允许弹窗')
  const cards = codes.value
    .map((code) => {
      const src = qrMap.value[code] || ''
      return `<div class="label"><img src="${src}" alt=""/><div class="code">${code}</div></div>`
    })
    .join('')
  win.document.write(`<!doctype html><html><head><meta charset="utf-8"/><title>溯源批号二维码</title>
<style>
  @page { margin: 10mm; }
  body { font-family: Arial, "Microsoft YaHei", sans-serif; margin: 0; }
  .sheet { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
  .label { border: 1px solid #ccc; border-radius: 6px; padding: 8px 6px; text-align: center; page-break-inside: avoid; break-inside: avoid; }
  .label img { width: 120px; height: 120px; display: block; margin: 0 auto 6px; }
  .code { font-size: 11px; font-weight: 600; letter-spacing: 0.02em; word-break: break-all; line-height: 1.3; }
  @media print {
    .label { border-color: #999; }
  }
</style></head><body>
<div class="sheet">${cards}</div>
<script>window.onload=function(){window.focus();window.print()}<\/script>
</body></html>`)
  win.document.close()
}

/** A4 批量 PDF：4 列 × 5 行 / 页，二维码下方印批号 */
async function exportPdf() {
  if (!codes.value.length) return ElMessage.warning('无批号可导出')
  exporting.value = true
  try {
    await buildAllQr(true)
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
    const title = `Trace Batch ${filter.biz_date}${filter.lot_no ? ` Lot${filter.lot_no}` : ''}  Total ${codes.value.length}`
    const fname = `trace-batch_${filter.biz_date}_${codes.value.length}.pdf`

    codes.value.forEach((code, i) => {
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
      if (img) {
        doc.addImage(img, 'PNG', imgX, imgY, qrSize, qrSize)
      }

      doc.setTextColor(0)
      doc.setFontSize(7.5)
      const textY = imgY + qrSize + 4
      const lines = doc.splitTextToSize(code, cellW - 4)
      doc.text(lines, x + cellW / 2, textY, { align: 'center', baseline: 'top' })
    })

    doc.save(fname)
    ElMessage.success(`已导出 PDF：${fname}（A4 · 每页 4×5）`)
  } catch (e) {
    ElMessage.error(`导出失败：${e instanceof Error ? e.message : e}`)
  } finally {
    exporting.value = false
  }
}

watch(viewMode, (m) => {
  if (m === 'labels') void buildAllQr()
})

onMounted(refresh)
</script>

<template>
  <div v-loading="loading || exporting" :element-loading-text="exporting ? '正在生成 PDF…' : ''">
    <el-card header="按天批量生成溯源批号" style="margin-bottom: 16px">
      <el-form inline size="small">
        <el-form-item label="业务日">
          <el-date-picker v-model="form.biz_date" type="date" value-format="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="批次号">
          <el-input v-model="form.lot_no" style="width: 80px" maxlength="2" placeholder="01" />
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="form.qty" :min="1" :max="500" />
        </el-form-item>
        <el-button type="primary" @click="generate">生成并出码</el-button>
      </el-form>
      <p class="hint">生成后自动生成二维码（码下显示溯源批号）。可用「批量导出 PDF」下载 A4 排版文件，直接打印。</p>
    </el-card>

    <el-card>
      <template #header>
        <div class="hdr">
          <span>批号池 / 二维码</span>
          <div class="hdr-actions">
            <el-date-picker v-model="filter.biz_date" type="date" value-format="YYYY-MM-DD" size="small" @change="refresh" />
            <el-select v-model="filter.status" clearable placeholder="状态" size="small" style="width: 120px" @change="refresh">
              <el-option value="available" label="可用" />
              <el-option value="used" label="已用" />
              <el-option value="void" label="作废" />
            </el-select>
            <el-radio-group v-model="viewMode" size="small">
              <el-radio-button value="table">列表</el-radio-button>
              <el-radio-button value="labels">标签预览</el-radio-button>
            </el-radio-group>
            <el-button size="small" @click="refresh">刷新</el-button>
            <el-button size="small" @click="copyCodes">复制批号</el-button>
            <el-button size="small" @click="printLabels">浏览器打印</el-button>
            <el-button type="primary" size="small" :loading="exporting" @click="exportPdf">批量导出 PDF</el-button>
          </div>
        </div>
      </template>

      <el-table v-if="viewMode === 'table'" :data="list" size="small" stripe>
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
        <el-table-column prop="code" label="溯源批号" min-width="200" />
        <el-table-column prop="seq_no" label="流水" width="80" />
        <el-table-column prop="lot_no" label="批次" width="70" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column prop="weigh_ticket_id" label="过磅单" width="90" />
        <el-table-column prop="created_at" label="生成时间" width="160" />
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button link type="primary" @click="openPreview(row)">查看</el-button>
            <el-button link type="primary" @click="openInboundInfo(row)">入库信息</el-button>
            <el-button v-if="row.status === 'available'" link type="danger" @click="voidCode(row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-else class="label-grid">
        <div v-for="row in list" :key="String(row.code)" class="label-card" @click="openPreview(row)">
          <img v-if="qrMap[String(row.code || '')]" :src="qrMap[String(row.code || '')]" alt="" />
          <div class="code">{{ row.code }}</div>
          <div class="meta">{{ row.status }} · #{{ row.seq_no }}</div>
        </div>
        <el-empty v-if="!list.length" description="当日无批号" />
      </div>
    </el-card>

    <el-dialog v-model="previewVisible" title="溯源批号二维码" width="360px" align-center>
      <div class="preview">
        <img v-if="previewQr" :src="previewQr" alt="qr" />
        <div class="preview-code">{{ previewCode }}</div>
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button @click="printLabels">浏览器打印</el-button>
        <el-button type="primary" :loading="exporting" @click="exportPdf">批量导出 PDF</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.hdr-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.hint { color: #888; font-size: 12px; margin: 8px 0 0; }
.muted { color: #bbb; }
.qr-thumb { width: 56px; height: 56px; cursor: pointer; vertical-align: middle; }
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
.label-card:hover { border-color: var(--el-color-primary); }
.label-card img { width: 120px; height: 120px; display: block; margin: 0 auto 8px; }
.label-card .code { font-size: 12px; font-weight: 600; word-break: break-all; line-height: 1.35; }
.label-card .meta { margin-top: 4px; font-size: 11px; color: #888; }
.preview { text-align: center; }
.preview img { width: 220px; height: 220px; }
.preview-code { margin-top: 12px; font-size: 14px; font-weight: 700; word-break: break-all; }
</style>
