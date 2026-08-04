<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, productionApi, payrollApi, portalHomeUrl } from '@erp/shared'
import { showToast } from 'vant'

const auth = useAuthStore()
const router = useRouter()
const portalUrl = portalHomeUrl()
const tab = ref('scan')
const scan = reactive({
  badge_code: 'EMP0301',
  box_code: 'BX-RAW-DEMO',
  input_weight: 110,
  output_weight: 100,
  net_weight: 100,
})
const preview = ref<Record<string, unknown> | null>(null)
const last = ref<Record<string, unknown> | null>(null)
const wages = ref<Record<string, unknown>[]>([])
const daily = ref<Record<string, unknown> | null>(null)

async function boot() {
  if (!auth.isLoggedIn) await auth.login('admin', 'admin123', 'mp_worker')
  const [w, d] = await Promise.all([
    payrollApi.wageRates(),
    productionApi.pieceworkMine(`badge_code=${encodeURIComponent(scan.badge_code)}`),
  ])
  wages.value = ((w.data as { list?: Record<string, unknown>[] })?.list) || []
  daily.value = (d.data as Record<string, unknown>) || null
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
  const res = await productionApi.scan({
    badge_code: scan.badge_code,
    box_code: scan.box_code,
    input_weight: Number(scan.input_weight),
    output_weight: Number(scan.output_weight),
    net_weight: Number(scan.output_weight || scan.net_weight),
  })
  if (res.code !== 1) return showToast(res.msg)
  last.value = (res.data as Record<string, unknown>) || null
  const amt = last.value?.wage_amount
  const next = (last.value?.next as Record<string, unknown>) || {}
  const util = last.value?.utilization
  showToast(`报工成功 工钱¥${amt ?? 0} 利用率${util ?? '-'} → ${next.next_step || (next.finished ? '完成' : '')}`)
  if (next.new_box_code) scan.box_code = String(next.new_box_code)
  await boot()
}

watch(tab, (t) => {
  if (t === 'wage') boot()
})

onMounted(boot)
</script>

<template>
  <div class="phone">
    <header>
      <a :href="portalUrl" style="color:#fff;font-size:12px;opacity:.9">← 入口</a>
      <h1>{{ { scan: '双扫报工', wage: '今日核对', hr: '考勤' }[tab] }}</h1>
      <button class="out" @click="auth.logout(); router.replace('/login')">退出</button>
    </header>
    <main>
      <section v-if="tab==='scan'" class="card">
        <p class="hint">工牌 + 箱码；录入投料重与完工重，系统自动算损耗与利用率，并累计当日计件。</p>
        <van-field v-model="scan.badge_code" label="工牌码" placeholder="EMP0301" />
        <van-field v-model="scan.box_code" label="箱码" placeholder="BX-RAW-DEMO" />
        <van-field v-model.number="scan.input_weight" type="number" label="投料重(kg)" />
        <van-field v-model.number="scan.output_weight" type="number" label="完工重(kg)" />
        <div style="display:flex;gap:8px;margin-top:8px">
          <van-button block @click="doResolve">预览</van-button>
          <van-button type="primary" block @click="doScan">确认报工</van-button>
        </div>
        <van-cell-group v-if="preview" inset title="预解析" style="margin-top:12px">
          <van-cell title="工人" :value="String(preview.worker_name||'')" />
          <van-cell title="工序" :value="String(preview.step_name||preview.process_id||'')" />
          <van-cell title="损耗" :value="String(preview.loss ?? '-')" />
          <van-cell title="利用率" :value="preview.utilization != null ? Number(preview.utilization).toFixed(2) : '-'" />
        </van-cell-group>
        <van-cell-group v-if="last" inset title="上次结果" style="margin-top:8px">
          <van-cell title="工钱" :value="`¥${last.wage_amount||0}`" />
          <van-cell title="损耗/利用率" :value="`${last.loss||0} / ${last.utilization != null ? Number(last.utilization).toFixed(2) : '-'}`" />
          <van-cell title="下一步" :value="String((last.next as any)?.next_step || ((last.next as any)?.finished ? '产线完成' : '-'))" />
          <van-cell title="新箱码" :value="String((last.next as any)?.new_box_code || '-')" />
        </van-cell-group>
      </section>
      <section v-else-if="tab==='wage'" class="card">
        <van-cell-group inset title="今日计件核对">
          <van-cell title="工人" :value="String(daily?.worker_name || scan.badge_code)" />
          <van-cell title="日期" :value="String(daily?.biz_date || '-')" />
          <van-cell title="总完工重" :value="String(daily?.total_output_weight ?? daily?.total_qty ?? 0)" />
          <van-cell title="总损耗" :value="String(daily?.total_loss ?? 0)" />
          <van-cell title="预计工钱" :value="`¥${daily?.total_amount ?? 0}`" />
        </van-cell-group>
        <van-cell-group inset title="分工序汇总" style="margin-top:8px">
          <van-cell
            v-for="s in ((daily?.summaries as Record<string, unknown>[]) || [])"
            :key="String(s.id)"
            :title="String(s.process_name || s.process_id)"
            :value="`¥${s.amount}`"
            :label="`完工 ${s.output_weight||s.qty} · 损耗 ${s.loss||0} · 单价 ¥${s.rate||0}`"
          />
          <van-empty v-if="!((daily?.summaries as unknown[])||[]).length" description="今日暂无计件" />
        </van-cell-group>
        <van-cell-group inset title="工序工价参考" style="margin-top:8px">
          <van-cell v-for="w in wages" :key="String(w.id)" :title="`工序 ${w.process_id}`" :value="`¥${w.rate}`" />
        </van-cell-group>
      </section>
      <section v-else class="card">
        <van-notice-bar text="计件以完工净重为准；收货卡点由固定工确认。下班请在「今日核对」确认总量与工钱。" />
      </section>
    </main>
    <nav class="tabbar">
      <button v-for="t in [['scan','双扫报工'],['wage','今日核对'],['hr','提醒']]" :key="t[0]"
        :class="{ active: tab===t[0] }" @click="tab=t[0]">{{ t[1] }}</button>
    </nav>
  </div>
</template>

<style scoped>
.phone { max-width: 420px; margin: 0 auto; min-height: 100vh; background: #f5f6f8; display: flex; flex-direction: column; }
header { display: flex; justify-content: space-between; align-items: center; padding: 12px 14px; background: #2f6b45; color: #fff; }
header h1 { margin: 0; font-size: 16px; }
.out { background: transparent; border: 0; color: #fff; }
main { flex: 1; padding: 12px; }
.card { background: #fff; border-radius: 10px; padding: 12px; }
.hint { font-size: 12px; color: #5c6b75; margin: 0 0 8px; }
.tabbar { display: grid; grid-template-columns: repeat(3,1fr); background: #fff; border-top: 1px solid #e5e5e5; }
.tabbar button { border: 0; background: transparent; padding: 10px 0; font-size: 12px; color: #666; }
.tabbar button.active { color: #2f6b45; font-weight: 600; }
</style>
