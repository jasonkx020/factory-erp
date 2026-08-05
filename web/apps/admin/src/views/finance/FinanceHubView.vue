<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { financeApi } from '@erp/shared'

type Row = Record<string, unknown>

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  subjects: '账目管理',
  ledger: '交易流水账',
  'income-expenses': '收入支出明细',
  orders: '订单管理',
  miniprogram: '小程序管理',
  vouchers: '凭证管理',
  invoices: '发票管理',
  writeoffs: '收款核单',
  fx: '外币结汇',
  'fx-query': '结汇查询',
  allocations: '分摊撤销',
  alerts: '收款预警',
  reconciles: '出纳对账',
  prepays: '预收预付管理',
  'cost-accountings': '成本核算',
  'contract-profits': '合同利润',
  recognitions: '销售认款',
  'return-finances': '销售退货退单',
  arap: '往来调整单',
  approvals: '财务审批',
  funds: '资金管理',
  statements: '财务报表',
  'cost-traces': '成本明细溯源表',
  'month-closes': '月度结转',
}

const active = computed(() => String(route.params.section || 'vouchers'))
const title = computed(() => TITLE_MAP[active.value] || '财务管理')
const loading = ref(false)
const list = ref<Row[]>([])
const funds = ref<Row[]>([])
const subjects = ref<Row[]>([])
const fundTab = ref<'accounts' | 'transfers'>('accounts')
const statementPreview = ref<Row | null>(null)

const subjectForm = reactive({ code: '', name: '', subject_type: 'asset' })
const ledgerForm = reactive({
  direction: 'in',
  amount: 100,
  account_id: 1,
  subject_id: 1,
  counterparty: '',
  category: '经营收支',
  remark: '',
})
const voucherForm = reactive({ summary: '', subject_id: 1, debit: 100, credit: 0 })
const invoiceForm = reactive({
  direction: 'out',
  invoice_no: '',
  counterparty_name: '',
  amount: 1000,
  tax: 130,
})
const writeoffForm = reactive({ customer_id: 1, amount: 1000, fund_account_id: 1, sales_order_id: 0 })
const recognitionForm = reactive({ customer_id: 1, amount: 1000, fund_account_id: 1, remark: '' })
const fxForm = reactive({ currency: 'USD', amount_fx: 1000, rate: 7.2, fund_account_id: 1 })
const allocForm = reactive({ source_amount: 1000 })
const alertForm = reactive({ customer_id: 1, overdue_days: 30, amount: 5000, due_date: '' })
const reconcileForm = reactive({ fund_account_id: 1, actual_balance: 0 })
const prepayForm = reactive({
  party_type: 'customer',
  party_id: 1,
  direction: 'in',
  amount: 1000,
})
const costForm = reactive({
  period: new Date().toISOString().slice(0, 7),
  product_id: 1,
  task_id: 0,
  material_cost: 0,
  labor_cost: 0,
  overhead: 0,
})
const returnForm = reactive({ order_id: 0, amount: 500 })
const arapForm = reactive({
  party_type: 'customer',
  party_id: 1,
  amount: 100,
  direction: 'increase',
  remark: '',
})
const fundAccForm = reactive({ code: '', name: '', currency: 'CNY', balance: 0 })
const fundTfForm = reactive({ from_account_id: 1, to_account_id: 2, amount: 100, remark: '' })
const monthForm = reactive({
  year: new Date().getFullYear(),
  month: new Date().getMonth() + 1,
})
const mpForm = reactive({ bill_no: '', channel: 'wechat', amount: 100, order_id: 0 })

async function loadMeta() {
  const [f, s] = await Promise.all([financeApi.fundAccounts(), financeApi.subjects()])
  funds.value = ((f.data as { list?: Row[] })?.list) || []
  subjects.value = ((s.data as { list?: Row[] })?.list) || []
  if (funds.value[0]) {
    const id = Number(funds.value[0].id)
    ledgerForm.account_id = id
    writeoffForm.fund_account_id = id
    recognitionForm.fund_account_id = id
    fxForm.fund_account_id = id
    reconcileForm.fund_account_id = id
    fundTfForm.from_account_id = id
    if (funds.value[1]) fundTfForm.to_account_id = Number(funds.value[1].id)
  }
  if (subjects.value[0]) {
    const sid = Number(subjects.value[0].id)
    ledgerForm.subject_id = sid
    voucherForm.subject_id = sid
  }
}

async function refresh() {
  loading.value = true
  statementPreview.value = null
  try {
    let res
    switch (active.value) {
      case 'subjects':
        res = await financeApi.subjects()
        break
      case 'ledger':
        res = await financeApi.ledger()
        break
      case 'income-expenses':
        res = await financeApi.incomeExpenses()
        break
      case 'orders':
        res = await financeApi.orders()
        break
      case 'miniprogram':
        res = await financeApi.miniprogramBills()
        break
      case 'vouchers':
        res = await financeApi.vouchers()
        break
      case 'invoices':
        res = await financeApi.invoices()
        break
      case 'writeoffs':
        res = await financeApi.writeoffs()
        break
      case 'fx':
        res = await financeApi.fxSettlements()
        break
      case 'fx-query':
        res = await financeApi.fxQuery()
        break
      case 'allocations':
        res = await financeApi.allocations()
        break
      case 'alerts':
        res = await financeApi.alerts()
        break
      case 'reconciles':
        res = await financeApi.reconciles()
        break
      case 'prepays':
        res = await financeApi.prepays()
        break
      case 'cost-accountings':
        res = await financeApi.costAccountings()
        break
      case 'contract-profits':
        res = await financeApi.contractProfits()
        break
      case 'recognitions':
        res = await financeApi.recognitions()
        break
      case 'return-finances':
        res = await financeApi.returnFinances()
        break
      case 'arap':
        res = await financeApi.arapAdjusts()
        break
      case 'approvals':
        res = await financeApi.approvals()
        break
      case 'funds':
        res = fundTab.value === 'accounts' ? await financeApi.fundAccounts() : await financeApi.fundTransfers()
        break
      case 'statements':
        res = await financeApi.statements()
        break
      case 'cost-traces':
        res = await financeApi.costTraces()
        break
      case 'month-closes':
        res = await financeApi.monthCloses()
        break
      default:
        res = await financeApi.vouchers()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
    if (active.value === 'funds' && fundTab.value === 'accounts') funds.value = list.value
  } finally {
    loading.value = false
  }
}

async function run(fn: () => Promise<{ code: number; msg: string }>, ok = '成功') {
  const res = await fn()
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(ok)
  await refresh()
  await loadMeta()
}

async function generateStatements() {
  const r = await financeApi.generateStatements({})
  if (r.code !== 1) return ElMessage.error(r.msg)
  statementPreview.value = r.data as Row
  ElMessage.success('已生成')
  await refresh()
}

onMounted(async () => {
  await loadMeta()
  await refresh()
})
watch([active, fundTab], refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <el-button size="small" @click="refresh">刷新</el-button>
    </div>

    <!-- 账目 -->
    <template v-if="active === 'subjects'">
      <el-card class="mb" header="新建会计科目">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="subjectForm.code" style="width:120px" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="subjectForm.name" style="width:160px" /></el-form-item>
          <el-form-item label="类型">
            <el-select v-model="subjectForm.subject_type" style="width:110px">
              <el-option label="资产" value="asset" /><el-option label="负债" value="liability" />
              <el-option label="收入" value="income" /><el-option label="费用" value="expense" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createSubject({ ...subjectForm }), '已建科目')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="code" label="编码" width="100" /><el-table-column prop="name" label="名称" />
        <el-table-column prop="subject_type" label="类型" width="100" /><el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <!-- 流水 -->
    <template v-else-if="active === 'ledger'">
      <el-card class="mb" header="登记流水（同步更新资金账户余额）">
        <el-form inline size="small">
          <el-form-item label="方向">
            <el-select v-model="ledgerForm.direction" style="width:90px"><el-option label="收入" value="in" /><el-option label="支出" value="out" /></el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="ledgerForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="ledgerForm.account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="对方"><el-input v-model="ledgerForm.counterparty" style="width:120px" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createLedger({ ...ledgerForm }), '流水已登记')">登记</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="direction" label="方向" width="80" />
        <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="biz_date" label="日期" width="110" />
        <el-table-column prop="counterparty" label="对方" /><el-table-column prop="remark" label="备注" />
      </el-table>
    </template>

    <template v-else-if="active === 'income-expenses'">
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="流水号" width="150" /><el-table-column prop="category" label="类别" width="120" />
        <el-table-column prop="direction" label="方向" width="80" /><el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="biz_date" label="日期" width="110" /><el-table-column prop="counterparty" label="对方" />
      </el-table>
    </template>

    <template v-else-if="active === 'orders'">
      <el-alert class="mb" type="info" :closable="false" title="财务视角销售订单（只读）；业务办理请到销售管理" />
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="160" /><el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="created_at" label="创建时间" />
      </el-table>
    </template>

    <template v-else-if="active === 'miniprogram'">
      <el-card class="mb" header="小程序账单">
        <el-form inline size="small">
          <el-form-item label="单号"><el-input v-model="mpForm.bill_no" style="width:140px" placeholder="可空" /></el-form-item>
          <el-form-item label="渠道"><el-input v-model="mpForm.channel" style="width:100px" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="mpForm.amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createMiniprogramBill({ ...mpForm }), '已建账单')">新建</el-button>
          <el-button @click="run(() => financeApi.reconcileMiniprogram({}), '已对账未付款账单')">一键对账</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="bill_no" label="账单号" width="150" /><el-table-column prop="channel" label="渠道" width="100" />
        <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="paid_at" label="支付/对账时间" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='unpaid'" link type="primary" @click="run(() => financeApi.reconcileMiniprogram({ id: Number(row.id) }), '已对账')">对账</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'vouchers'">
      <el-card class="mb" header="新建凭证（借贷需平衡后可审批）">
        <el-form inline size="small">
          <el-form-item label="摘要"><el-input v-model="voucherForm.summary" style="width:180px" /></el-form-item>
          <el-form-item label="科目">
            <el-select v-model="voucherForm.subject_id" style="width:140px">
              <el-option v-for="s in subjects" :key="String(s.id)" :label="`${s.code} ${s.name}`" :value="Number(s.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="借方"><el-input-number v-model="voucherForm.debit" :min="0" /></el-form-item>
          <el-form-item label="贷方"><el-input-number v-model="voucherForm.credit" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createVoucher({ ...voucherForm, lines: [
            { subject_id: voucherForm.subject_id, debit: voucherForm.debit, credit: voucherForm.credit },
            { subject_id: voucherForm.subject_id, debit: voucherForm.credit, credit: voucherForm.debit },
          ] }), '凭证已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="凭证号" width="150" /><el-table-column prop="period" label="期间" width="100" />
        <el-table-column prop="biz_date" label="日期" width="110" /><el-table-column prop="summary" label="摘要" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="success" @click="run(() => financeApi.approveVoucher(Number(row.id)), '已审批')">审批</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'invoices'">
      <el-card class="mb" header="新建发票">
        <el-form inline size="small">
          <el-form-item label="方向">
            <el-select v-model="invoiceForm.direction" style="width:100px"><el-option label="销项" value="out" /><el-option label="进项" value="in" /></el-select>
          </el-form-item>
          <el-form-item label="票号"><el-input v-model="invoiceForm.invoice_no" style="width:140px" /></el-form-item>
          <el-form-item label="对方"><el-input v-model="invoiceForm.counterparty_name" style="width:140px" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="invoiceForm.amount" :min="0" /></el-form-item>
          <el-form-item label="税额"><el-input-number v-model="invoiceForm.tax" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createInvoice({ ...invoiceForm }), '发票已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="invoice_no" label="票号" width="150" /><el-table-column prop="direction" label="方向" width="80" />
        <el-table-column prop="counterparty_name" label="对方" /><el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="tax" label="税额" width="90" /><el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <template v-else-if="active === 'writeoffs'">
      <el-card class="mb" header="收款核单（确认后入资金账户）">
        <el-form inline size="small">
          <el-form-item label="客户ID"><el-input-number v-model="writeoffForm.customer_id" :min="1" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="writeoffForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="writeoffForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="订单ID"><el-input-number v-model="writeoffForm.sales_order_id" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createWriteoff({ ...writeoffForm }), '核单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="customer_id" label="客户" width="90" />
        <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmWriteoff(Number(row.id)), '已确认入账')">确认</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'recognitions'">
      <el-card class="mb" header="销售认款">
        <el-form inline size="small">
          <el-form-item label="客户ID"><el-input-number v-model="recognitionForm.customer_id" :min="1" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="recognitionForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="资金账户">
            <el-select v-model="recognitionForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createRecognition({ ...recognitionForm }), '认款已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="customer_id" label="客户" width="90" />
        <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmRecognition(Number(row.id)), '已认款')">确认</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'fx' || active === 'fx-query'">
      <el-card v-if="active==='fx'" class="mb" header="外币结汇">
        <el-form inline size="small">
          <el-form-item label="币种"><el-input v-model="fxForm.currency" style="width:90px" /></el-form-item>
          <el-form-item label="外币金额"><el-input-number v-model="fxForm.amount_fx" :min="0.01" /></el-form-item>
          <el-form-item label="汇率"><el-input-number v-model="fxForm.rate" :min="0.0001" :step="0.01" /></el-form-item>
          <el-form-item label="入账账户">
            <el-select v-model="fxForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createFx({ ...fxForm }), '结汇单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="currency" label="币种" width="80" />
        <el-table-column prop="amount_fx" label="外币" width="100" /><el-table-column prop="rate" label="汇率" width="90" />
        <el-table-column prop="amount_local" label="本币" width="100" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column v-if="active==='fx'" label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmFx(Number(row.id)), '结汇已确认')">确认</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'allocations'">
      <el-card class="mb" header="费用分摊">
        <el-form inline size="small">
          <el-form-item label="源金额"><el-input-number v-model="allocForm.source_amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createAllocation({ ...allocForm }), '分摊已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="source_amount" label="金额" width="120" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status!=='revoked'" link type="danger" @click="run(() => financeApi.revokeAllocation(Number(row.id)), '已撤销')">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'alerts'">
      <el-card class="mb" header="收款预警">
        <el-form inline size="small">
          <el-form-item label="客户ID"><el-input-number v-model="alertForm.customer_id" :min="1" /></el-form-item>
          <el-form-item label="逾期天"><el-input-number v-model="alertForm.overdue_days" :min="0" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="alertForm.amount" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createAlert({ ...alertForm }), '预警已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="customer_id" label="客户" width="90" /><el-table-column prop="overdue_days" label="逾期天" width="90" />
        <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='open'" link type="primary" @click="run(() => financeApi.handleAlert(Number(row.id), { remark: '已跟进' }), '已处理')">处理</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'reconciles'">
      <el-card class="mb" header="出纳对账（账面余额自动带出）">
        <el-form inline size="small">
          <el-form-item label="资金账户">
            <el-select v-model="reconcileForm.fund_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="String(f.id)" :label="`${f.name}(${f.balance})`" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="实盘余额"><el-input-number v-model="reconcileForm.actual_balance" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createReconcile({ ...reconcileForm }), '对账单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="book_balance" label="账面" width="100" />
        <el-table-column prop="actual_balance" label="实盘" width="100" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmReconcile(Number(row.id)), '已确认')">确认</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'prepays'">
      <el-card class="mb" header="预收/预付">
        <el-form inline size="small">
          <el-form-item label="类型">
            <el-select v-model="prepayForm.party_type" style="width:110px"><el-option label="客户" value="customer" /><el-option label="供应商" value="supplier" /></el-select>
          </el-form-item>
          <el-form-item label="对方ID"><el-input-number v-model="prepayForm.party_id" :min="1" /></el-form-item>
          <el-form-item label="方向">
            <el-select v-model="prepayForm.direction" style="width:90px"><el-option label="预收" value="in" /><el-option label="预付" value="out" /></el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="prepayForm.amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createPrepay({ ...prepayForm }), '已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="party_type" label="类型" width="90" />
        <el-table-column prop="party_id" label="对方" width="90" /><el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="balance" label="余额" width="100" /><el-table-column prop="status" label="状态" width="90" />
      </el-table>
    </template>

    <template v-else-if="active === 'cost-accountings'">
      <el-card class="mb" header="成本核算">
        <el-form inline size="small">
          <el-form-item label="期间"><el-input v-model="costForm.period" style="width:110px" /></el-form-item>
          <el-form-item label="物料成本"><el-input-number v-model="costForm.material_cost" :min="0" /></el-form-item>
          <el-form-item label="人工"><el-input-number v-model="costForm.labor_cost" :min="0" /></el-form-item>
          <el-form-item label="制造费用"><el-input-number v-model="costForm.overhead" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createCostAccounting({ ...costForm }), '成本单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="period" label="期间" width="100" />
        <el-table-column prop="material_cost" label="物料" width="90" /><el-table-column prop="labor_cost" label="人工" width="90" />
        <el-table-column prop="overhead" label="制造" width="90" /><el-table-column prop="total_cost" label="合计" width="100" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.calcCost(Number(row.id)), '已核算')">核算</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'contract-profits'">
      <div class="mb"><el-button type="primary" size="small" @click="run(() => financeApi.recalcContractProfit({}), '利润已重算')">重算合同利润</el-button></div>
      <el-table :data="list" size="small">
        <el-table-column prop="contract_id" label="合同" width="90" /><el-table-column prop="revenue" label="收入" width="120" />
        <el-table-column prop="cost" label="成本" width="120" /><el-table-column prop="profit" label="利润" width="120" />
        <el-table-column prop="period" label="期间" width="100" />
      </el-table>
    </template>

    <template v-else-if="active === 'return-finances'">
      <el-card class="mb" header="销售退货退单（财务）">
        <el-form inline size="small">
          <el-form-item label="订单ID"><el-input-number v-model="returnForm.order_id" :min="0" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="returnForm.amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createReturnFinance({ ...returnForm }), '退单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="order_id" label="订单" width="90" />
        <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.confirmReturnFinance(Number(row.id)), '已确认')">确认</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'arap'">
      <el-card class="mb" header="往来调整">
        <el-form inline size="small">
          <el-form-item label="类型">
            <el-select v-model="arapForm.party_type" style="width:110px"><el-option label="客户" value="customer" /><el-option label="供应商" value="supplier" /></el-select>
          </el-form-item>
          <el-form-item label="对方ID"><el-input-number v-model="arapForm.party_id" :min="1" /></el-form-item>
          <el-form-item label="金额"><el-input-number v-model="arapForm.amount" :min="0.01" /></el-form-item>
          <el-form-item label="方向">
            <el-select v-model="arapForm.direction" style="width:110px"><el-option label="调增" value="increase" /><el-option label="调减" value="decrease" /></el-select>
          </el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createArap({ ...arapForm }), '调整单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="单号" width="150" /><el-table-column prop="party_type" label="类型" width="90" />
        <el-table-column prop="party_id" label="对方" width="90" /><el-table-column prop="amount" label="金额" width="100" />
        <el-table-column prop="direction" label="方向" width="90" /><el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.postArap(Number(row.id)), '已过账')">过账</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'approvals'">
      <el-table :data="list" size="small">
        <el-table-column prop="biz_type" label="类型" width="100" /><el-table-column prop="doc_no" label="单号" width="150" />
        <el-table-column prop="title" label="摘要" /><el-table-column prop="status" label="状态" width="100" />
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button v-if="row.status==='draft' || row.status==='pending' || row.status==='submitted'" link type="success" @click="run(() => financeApi.approveFinance(Number(row.id)), '已批准')">批准</el-button>
            <el-button v-if="row.status==='draft' || row.status==='pending' || row.status==='submitted'" link type="danger" @click="run(() => financeApi.rejectFinance(Number(row.id)), '已驳回')">驳回</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <template v-else-if="active === 'funds'">
      <el-radio-group v-model="fundTab" size="small" class="mb">
        <el-radio-button value="accounts">资金账户</el-radio-button>
        <el-radio-button value="transfers">资金调拨</el-radio-button>
      </el-radio-group>
      <el-card v-if="fundTab==='accounts'" class="mb" header="新建资金账户">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="fundAccForm.code" style="width:120px" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="fundAccForm.name" style="width:140px" /></el-form-item>
          <el-form-item label="币种"><el-input v-model="fundAccForm.currency" style="width:80px" /></el-form-item>
          <el-form-item label="期初"><el-input-number v-model="fundAccForm.balance" :min="0" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createFundAccount({ ...fundAccForm }), '账户已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-card v-else class="mb" header="资金调拨（过账双边余额）">
        <el-form inline size="small">
          <el-form-item label="转出">
            <el-select v-model="fundTfForm.from_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="'f'+f.id" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="转入">
            <el-select v-model="fundTfForm.to_account_id" style="width:140px">
              <el-option v-for="f in funds" :key="'t'+f.id" :label="String(f.name)" :value="Number(f.id)" />
            </el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="fundTfForm.amount" :min="0.01" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.createFundTransfer({ ...fundTfForm }), '调拨单已建')">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <template v-if="fundTab==='accounts'">
          <el-table-column prop="code" label="编码" width="120" /><el-table-column prop="name" label="名称" />
          <el-table-column prop="currency" label="币种" width="80" /><el-table-column prop="balance" label="余额" width="120" />
          <el-table-column prop="status" label="状态" width="90" />
        </template>
        <template v-else>
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="from_account_id" label="转出" width="90" /><el-table-column prop="to_account_id" label="转入" width="90" />
          <el-table-column prop="amount" label="金额" width="100" /><el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status==='draft'" link type="primary" @click="run(() => financeApi.postFundTransfer(Number(row.id)), '已过账')">过账</el-button>
            </template>
          </el-table-column>
        </template>
      </el-table>
    </template>

    <template v-else-if="active === 'statements'">
      <div class="mb">
        <el-button type="primary" size="small" @click="generateStatements">生成三大表</el-button>
      </div>
      <pre v-if="statementPreview" class="preview">{{ statementPreview }}</pre>
      <el-table :data="list" size="small">
        <el-table-column prop="code" label="报表" width="120" /><el-table-column prop="period" label="期间" width="110" />
        <el-table-column prop="title" label="标题" /><el-table-column prop="generated_at" label="生成时间" width="170" />
      </el-table>
    </template>

    <template v-else-if="active === 'cost-traces'">
      <el-table :data="list" size="small">
        <el-table-column prop="doc_no" label="成本单" width="150" /><el-table-column prop="period" label="期间" width="100" />
        <el-table-column prop="source_type" label="来源类型" width="120" /><el-table-column prop="source_id" label="来源ID" width="90" />
        <el-table-column prop="amount" label="金额" width="120" />
      </el-table>
    </template>

    <template v-else-if="active === 'month-closes'">
      <el-card class="mb" header="月度结转">
        <el-form inline size="small">
          <el-form-item label="年"><el-input-number v-model="monthForm.year" :min="2020" /></el-form-item>
          <el-form-item label="月"><el-input-number v-model="monthForm.month" :min="1" :max="12" /></el-form-item>
          <el-button type="primary" @click="run(() => financeApi.closeMonth({ ...monthForm }), '期间已结转')">结转</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="year" label="年" width="90" /><el-table-column prop="month" label="月" width="80" />
        <el-table-column prop="status" label="状态" width="100" /><el-table-column prop="closed_at" label="结转时间" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button v-if="row.status==='closed'" link type="warning" @click="run(() => financeApi.reopenMonth(Number(row.id)), '已反结转')">反结转</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.head h2 { margin: 0; font-size: 18px; }
.mb { margin-bottom: 12px; }
.preview {
  background: #f6f8fa; padding: 12px; border-radius: 6px; font-size: 12px;
  white-space: pre-wrap; word-break: break-all; margin-bottom: 12px;
}
</style>
