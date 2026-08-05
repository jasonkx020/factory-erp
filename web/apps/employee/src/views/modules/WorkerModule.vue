<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  useAuthStore,
  useNotifyStore,
  productionApi,
  payrollApi,
  fieldLedgerApi,
  canAccessEmployeeModule,
  portalHomeUrl,
  parsePayload,
} from '@erp/shared'
import { showToast } from 'vant'

const auth = useAuthStore()
const notify = useNotifyStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const tab = ref('scan')
const scan = reactive({
  badge_code: '',
  box_code: '',
  input_weight: 0,
  output_weight: 0,
  net_weight: 0,
  bag_qty: 0,
  scrap_type: '',
})
const preview = ref<Record<string, unknown> | null>(null)
const last = ref<Record<string, unknown> | null>(null)
const wages = ref<Record<string, unknown>[]>([])
const daily = ref<Record<string, unknown> | null>(null)
const notices = ref<Record<string, unknown>[]>([])
const myIssueLines = ref<Record<string, unknown>[]>([])
const myTools = ref<Record<string, unknown>[]>([])
const showScrap = ref(false)

const scrapOptions = [
  { text: '无次品', value: '' },
  { text: '切断次品', value: 'cut_defect' },
  { text: '去芯次品', value: 'core_defect' },
  { text: '切块次品', value: 'dice_defect' },
  { text: '过筛装袋次品', value: 'sieve_bag_defect' },
]

function onScrapPick({ selectedOptions }: { selectedOptions: { text: string; value: string }[] }) {
  scan.scrap_type = selectedOptions[0]?.value ?? ''
  showScrap.value = false
}

function applyNotifyHints() {
  const rows = notify.inbox.filter((r) => {
    const k = String(r.event_key || '')
    return k === 'production.report_confirmed' || k === 'payroll.labor_paid'
  })
  notices.value = rows
  for (const row of rows) {
    const p = parsePayload(row.payload ?? row.payload_json)
    const next = (p.next as Record<string, unknown>) || p
    const code = next.new_box_code || p.new_box_code || p.scan_code
    if (code && !scan.box_code) {
      scan.box_code = String(code)
      showToast(`推送箱码 ${code}`)
      break
    }
  }
}

async function boot() {
  if (!auth.isLoggedIn) {
    router.replace('/login')
    return
  }
  await auth.fetchMe()
  if (!canAccessEmployeeModule('worker', auth.permissions, auth.roles)) {
    showToast('无工人模块权限')
    router.replace('/')
    return
  }
  await notify.start()
  const pref = localStorage.getItem('erp.worker.badge') || ''
  if (pref && !scan.badge_code) scan.badge_code = pref
  const [w, d] = await Promise.all([
    payrollApi.wageRates(),
    productionApi.pieceworkMine(
      scan.badge_code ? `badge_code=${encodeURIComponent(scan.badge_code)}` : undefined,
    ),
  ])
  wages.value = ((w.data as { list?: Record<string, unknown>[] })?.list) || []
  daily.value = (d.data as Record<string, unknown>) || null
  applyNotifyHints()
  await loadMineExtras()
}

async function loadMineExtras() {
  const today = new Date().toISOString().slice(0, 10)
  const [sheets, tools] = await Promise.all([
    fieldLedgerApi.pieceIssueSheets(),
    fieldLedgerApi.toolIssues(),
  ])
  const sheetList = ((sheets.data as { list?: Record<string, unknown>[] })?.list) || []
  const lines: Record<string, unknown>[] = []
  const me = String(auth.user?.name || '')
  for (const sh of sheetList.slice(0, 5)) {
    if (String(sh.biz_date || '') && !String(sh.biz_date).startsWith(today)) continue
    const id = Number(sh.id)
    if (!id) continue
    const det = await fieldLedgerApi.getPieceIssueSheet(id)
    const arr = ((det.data as { lines?: Record<string, unknown>[] })?.lines) || []
    for (const ln of arr) {
      const name = String(ln.employee_name || '')
      if (!me || !name || name.includes(me)) lines.push({ ...ln, sheet_doc: sh.doc_no })
    }
  }
  myIssueLines.value = lines.slice(0, 30)
  myTools.value = (((tools.data as { list?: Record<string, unknown>[] })?.list) || []).filter(
    (t) => !me || String(t.employee_name || '').includes(me),
  )
}

async function doResolve() {
  const res = await productionApi.scanResolve({
    badge_code: scan.badge_code,
    box_code: scan.box_code,
    input_weight: Number(scan.input_weight),
    output_weight: Number(scan.output_weight),
    net_weight: Number(scan.output_weight || scan.net_weight),
  })
  if (res.code !== 1) return showToast(res.msg)
  preview.value = (res.data as Record<string, unknown>) || null
  showToast('已解析工牌/箱码')
}

async function doScan() {
  if (scan.badge_code) localStorage.setItem('erp.worker.badge', scan.badge_code)
  const res = await productionApi.scan({
    badge_code: scan.badge_code,
    box_code: scan.box_code,
    input_weight: Number(scan.input_weight),
    output_weight: Number(scan.output_weight),
    net_weight: Number(scan.output_weight || scan.net_weight),
  })
  if (res.code !== 1) return showToast(res.msg)
  last.value = (res.data as Record<string, unknown>) || null
  if (last.value?.needs_confirm) {
    showToast('草稿已建，请核对后点「确认过账」')
    return
  }
  const amt = last.value?.wage_amount
  const next = (last.value?.next as Record<string, unknown>) || {}
  showToast(`报工成功 工钱¥${amt ?? 0} → ${next.next_step || (next.finished ? '完成' : '')}`)
  if (next.new_box_code) scan.box_code = String(next.new_box_code)
  await boot()
}

async function doConfirm() {
  const id = Number(last.value?.id)
  if (!id) return showToast('请先提交报工草稿')
  const res = await productionApi.confirmReportWork(id, {
    input_weight: Number(scan.input_weight),
    output_weight: Number(scan.output_weight),
    process_qc_result: 'pass',
    bag_qty: Number(scan.bag_qty) || 0,
    scrap_type: scan.scrap_type || undefined,
  })
  if (res.code !== 1) return showToast(res.msg)
  last.value = (res.data as Record<string, unknown>) || null
  const next = (last.value?.next as Record<string, unknown>) || {}
  showToast(`已确认过账 工钱¥${last.value?.wage_amount ?? 0}`)
  if (next.new_box_code) scan.box_code = String(next.new_box_code)
  await boot()
}

watch(tab, (t) => {
  if (t === 'wage') void boot()
  if (t === 'hr') applyNotifyHints()
  if (t === 'ledger') void loadMineExtras()
})

watch(
  () => notify.tick,
  () => applyNotifyHints(),
)

onMounted(boot)
</script>

<template>
  <div class="phone">
    <header>
      <a :href="portalUrl" style="color:#fff;font-size:12px;opacity:.9">← 入口</a>
      <button type="button" class="mod" @click="router.push('/')">模块</button>
      <h1>{{ { scan: '双扫报工', wage: '今日核对', ledger: '领料/工具', hr: '提醒' }[tab] }}</h1>
      <button class="out" @click="auth.logout(); router.replace('/login')">退出</button>
    </header>
    <main>
      <section v-if="tab==='scan'" class="card">
        <p class="hint">工牌 + 箱码；系统带出投料，核对完工/损耗后确认过账。</p>
        <van-field v-model="scan.badge_code" label="工牌码" placeholder="扫工牌或手输" />
        <van-field v-model="scan.box_code" label="箱码" placeholder="扫箱码或手输" />
        <van-field v-model.number="scan.input_weight" type="number" label="投料重(kg)" />
        <van-field v-model.number="scan.output_weight" type="number" label="完工重(kg)" />
        <van-field v-model.number="scan.bag_qty" type="number" label="袋数" />
        <van-field v-model="scan.scrap_type" is-link readonly label="次品类型" :placeholder="scrapOptions.find(o=>o.value===scan.scrap_type)?.text || '无次品'" @click="showScrap = true" />
        <van-popup v-model:show="showScrap" position="bottom">
          <van-picker :columns="scrapOptions" @confirm="onScrapPick" @cancel="showScrap = false" />
        </van-popup>
        <div style="display:flex;gap:8px;margin-top:8px">
          <van-button block @click="doResolve">预览</van-button>
          <van-button type="primary" block @click="doScan">提交草稿</van-button>
        </div>
        <van-button
          v-if="last?.needs_confirm || last?.status==='confirm_pending'"
          type="success"
          block
          style="margin-top:8px"
          @click="doConfirm"
        >确认过账（定损/QC）</van-button>
        <van-cell-group v-if="preview" inset title="预解析" style="margin-top:12px">
          <van-cell title="工人" :value="String(preview.worker_name||'')" />
          <van-cell title="工序" :value="String(preview.step_name||preview.process_id||'')" />
          <van-cell title="投料" :value="String(preview.input_weight||'')" />
          <van-cell title="损耗" :value="String(preview.loss||'')" />
        </van-cell-group>
        <van-cell-group v-if="last" inset title="上次结果" style="margin-top:8px">
          <van-cell title="状态" :value="String(last.status||'')" />
          <van-cell title="工钱" :value="`¥${last.wage_amount||0}`" />
          <van-cell title="新箱码" :value="String((last.next as any)?.new_box_code || '-')" />
        </van-cell-group>
      </section>
      <section v-else-if="tab==='wage'" class="card">
        <van-cell-group inset title="今日计件核对">
          <van-cell title="工人" :value="String(daily?.worker_name || scan.badge_code || '-')" />
          <van-cell title="总完工重" :value="String(daily?.total_output_weight ?? daily?.total_qty ?? 0)" />
          <van-cell title="预计工钱" :value="`¥${daily?.total_amount ?? 0}`" />
        </van-cell-group>
        <van-cell-group inset title="工序工价参考" style="margin-top:8px">
          <van-cell v-for="w in wages" :key="String(w.id)" :title="`工序 ${w.process_id}`" :value="`¥${w.rate}`" />
        </van-cell-group>
      </section>
      <section v-else-if="tab==='ledger'" class="card">
        <p class="hint">本人当日领料行 / 工具领还（只读）</p>
        <van-cell-group inset title="计件领料">
          <van-empty v-if="!myIssueLines.length" description="暂无" />
          <van-cell
            v-for="(ln, i) in myIssueLines"
            :key="i"
            :title="String(ln.process_name || ln.process_kind)"
            :label="`${ln.employee_name || ''} · 数量${ln.qty_total ?? ln.qty} · ¥${ln.amount ?? 0}`"
            :value="String(ln.sheet_doc || '')"
          />
        </van-cell-group>
        <van-cell-group inset title="工具" style="margin-top:8px">
          <van-empty v-if="!myTools.length" description="暂无" />
          <van-cell
            v-for="t in myTools"
            :key="String(t.id)"
            :title="String(t.tool_name)"
            :label="`领${t.issue_qty} / 还${t.return_qty}`"
            :value="String(t.status)"
          />
        </van-cell-group>
      </section>
      <section v-else class="card">
        <p class="hint">推送知会（报工确认 / 劳动支付）。MQTT {{ notify.mqttStatus }}</p>
        <van-empty v-if="!notices.length" description="暂无提醒" />
        <van-cell-group v-else inset>
          <van-cell
            v-for="n in notices"
            :key="String(n.id)"
            :title="String(n.title||n.event_key)"
            :label="String(n.body||'')"
            :value="String(n.created_at||'')"
            @click="notify.markRead(Number(n.id))"
          />
        </van-cell-group>
      </section>
    </main>
    <nav class="tabbar">
      <button
        v-for="t in [['scan','双扫报工'],['wage','今日核对'],['ledger','领料/工具'],['hr','提醒']]"
        :key="t[0]"
        :class="{ active: tab===t[0] }"
        @click="tab=t[0]"
      >{{ t[1] }}</button>
    </nav>
  </div>
</template>

<style scoped>
.phone { max-width: 420px; margin: 0 auto; min-height: 100vh; background: #f5f6f8; display: flex; flex-direction: column; }
header { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 12px 14px; background: #2f6b45; color: #fff; }
header h1 { margin: 0; font-size: 16px; flex: 1; }
.mod, .out { background: transparent; border: 0; color: #fff; cursor: pointer; }
main { flex: 1; padding: 12px; }
.card { background: #fff; border-radius: 10px; padding: 12px; }
.hint { font-size: 12px; color: #5c6b75; margin: 0 0 8px; }
.tabbar { display: grid; grid-template-columns: repeat(4,1fr); background: #fff; border-top: 1px solid #e5e5e5; }
.tabbar button { border: 0; background: transparent; padding: 10px 0; font-size: 12px; color: #666; }
.tabbar button.active { color: #2f6b45; font-weight: 600; }
</style>
