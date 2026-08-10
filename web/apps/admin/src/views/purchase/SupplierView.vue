<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  purchaseApi,
  CURRENCY_OPTIONS,
  SETTLE_METHOD_OPTIONS,
  SUPPLIER_RATING_OPTIONS,
  LICENSE_TYPE_OPTIONS,
} from '@erp/shared'
import { EnumSelect, ProductSelect, WarehouseSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const supplierCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'code', label: '编码' },
  { prop: 'supplier_type', label: '类型' },
  { prop: 'status', label: '状态' },
  { prop: 'rating', label: '等级' },
  { prop: 'settle_method', label: '结算' },
]

const analyticsCols: MobileCardColumn[] = [
  { prop: 'supplier_name', label: '供应商', primary: true },
  { prop: 'product_id', label: '物料' },
  { prop: 'qty', label: '数量' },
  { prop: 'amount', label: '金额' },
  { prop: 'avg_price', label: '均价' },
]

const licenseCols: MobileCardColumn[] = [
  { prop: 'license_type', label: '类型', primary: true },
  { prop: 'license_no', label: '号码' },
  { prop: 'expire_date', label: '到期日' },
]

const supplyCols: MobileCardColumn[] = [
  { prop: 'product_id', label: '物料', primary: true },
  { prop: 'is_preferred', label: '首选' },
  { prop: 'moq', label: 'MOQ' },
  { prop: 'lead_time_days', label: '交期' },
  { prop: 'last_price', label: '最近价' },
]

const priceCols: MobileCardColumn[] = [
  { prop: 'product_id', label: '物料', primary: true },
  { prop: 'price', label: '价格' },
  { prop: 'biz_date', label: '日期' },
  { prop: 'source_doc_id', label: '来源入库' },
]

const loading = ref(false)
const list = ref<Row[]>([])
const total = ref(0)
const filter = reactive({ status: '', supplier_type: '', q: '' })
const alerts = ref<Row[]>([])
const dialog = ref(false)
const detailTab = ref('profile')
const currentId = ref<number | null>(null)
const form = reactive<Row>({
  code: '', name: '', short_name: '', supplier_type: 'raw', status: 'potential', rating: 'B',
  is_preferred: false, settle_method: 'monthly', payment_days: 30, currency: 'CNY',
  contact_json: [{ name: '', mobile: '', is_primary: true }],
  remark: '',
})
const licenses = ref<Row[]>([])
const supplyItems = ref<Row[]>([])
const prices = ref<Row[]>([])
const perf = ref<Row | null>(null)
const analytics = ref<Row[]>([])

const statusLabel: Record<string, string> = {
  potential: '潜在', qualified: '合格', frozen: '冻结', blacklist: '黑名单', eliminated: '淘汰',
}

const editing = computed(() => currentId.value != null)

async function loadList() {
  loading.value = true
  try {
    const qs = new URLSearchParams()
    qs.set('page_num', '1')
    qs.set('page_size', '50')
    if (filter.status) qs.set('status', filter.status)
    if (filter.supplier_type) qs.set('supplier_type', filter.supplier_type)
    if (filter.q) qs.set('q', filter.q)
    const res = await purchaseApi.suppliers(qs.toString())
    if (res.code !== 1) return ElMessage.error(res.msg)
    const data = res.data as { list?: Row[]; total?: number }
    list.value = data?.list || []
    total.value = data?.total || list.value.length
    const al = await purchaseApi.certificateAlerts(60)
    if (al.code === 1) alerts.value = ((al.data as { list?: Row[] })?.list) || []
    const vp = await purchaseApi.volumePrice()
    if (vp.code === 1) analytics.value = ((vp.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, {
    code: `SUP${Date.now().toString().slice(-6)}`, name: '', short_name: '', supplier_type: 'raw',
    status: 'potential', rating: 'B', is_preferred: false, settle_method: 'monthly', payment_days: 30,
    currency: 'CNY', uscc: '', legal_person: '', register_address: '', invoice_title: '', tax_no: '',
    bank_name: '', bank_account: '', lead_time_days: 3, moq: 0, default_warehouse_id: 1, remark: '',
    contact_json: [{ name: '', mobile: '', is_primary: true }],
  })
  licenses.value = []
  supplyItems.value = []
  prices.value = []
  perf.value = null
  detailTab.value = 'profile'
}

function openCreate() {
  currentId.value = null
  resetForm()
  dialog.value = true
}

async function openEdit(row: Row) {
  currentId.value = Number(row.id)
  resetForm()
  const res = await purchaseApi.getSupplier(currentId.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  Object.assign(form, res.data || {})
  if (!Array.isArray(form.contact_json)) form.contact_json = [{ name: '', mobile: '', is_primary: true }]
  await loadDetailTabs()
  dialog.value = true
}

async function loadDetailTabs() {
  if (!currentId.value) return
  const id = currentId.value
  const [lic, items, ph, pf] = await Promise.all([
    purchaseApi.licenses(id),
    purchaseApi.supplyItems(id),
    purchaseApi.priceHistories(`supplier_id=${id}&page_size=50`),
    purchaseApi.performance(id),
  ])
  licenses.value = ((lic.data as { list?: Row[] })?.list) || []
  supplyItems.value = ((items.data as { list?: Row[] })?.list) || []
  prices.value = ((ph.data as { list?: Row[] })?.list) || []
  perf.value = (pf.data as Row) || null
}

async function save() {
  const body = { ...form }
  let res
  if (editing.value && currentId.value) {
    res = await purchaseApi.updateSupplier(currentId.value, body)
  } else {
    res = await purchaseApi.createSupplier(body)
  }
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已保存')
  currentId.value = Number((res.data as Row)?.id || currentId.value)
  dialog.value = false
  await loadList()
}

async function doAction(row: Row, act: 'qualify' | 'freeze' | 'blacklist' | 'activate') {
  const id = Number(row.id)
  const map = {
    qualify: purchaseApi.qualify,
    freeze: purchaseApi.freeze,
    blacklist: purchaseApi.blacklist,
    activate: purchaseApi.activate,
  }
  const res = await map[act](id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('状态已更新')
  await loadList()
}

async function removeRow(row: Row) {
  await ElMessageBox.confirm('确认淘汰/删除该供应商？', '提示')
  const res = await purchaseApi.removeSupplier(Number(row.id))
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已删除')
  await loadList()
}

function addContact() {
  const arr = Array.isArray(form.contact_json) ? [...(form.contact_json as Row[])] : []
  arr.push({ name: '', mobile: '', is_primary: false })
  form.contact_json = arr
}

function addLicense() {
  licenses.value.push({ license_type: 'business', license_no: '', expire_date: '' })
}

async function saveLicenses() {
  if (!currentId.value) return ElMessage.warning('请先保存供应商档案')
  const res = await purchaseApi.saveLicenses(currentId.value, licenses.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  licenses.value = ((res.data as { list?: Row[] })?.list) || licenses.value
  ElMessage.success('证照已保存')
}

function addSupply() {
  supplyItems.value.push({ product_id: 1, is_preferred: false, moq: 0, lead_time_days: 3, last_price: 0 })
}

async function saveSupply() {
  if (!currentId.value) return ElMessage.warning('请先保存供应商档案')
  const res = await purchaseApi.saveSupplyItems(currentId.value, supplyItems.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  supplyItems.value = ((res.data as { list?: Row[] })?.list) || supplyItems.value
  ElMessage.success('可供物料已保存')
}

onMounted(loadList)
</script>

<template>
  <div class="panel" v-loading="loading">
    <h2 class="title">供应商管理</h2>
    <p class="desc">主数据 · 证照 · 可供物料 · 历史价 · 绩效（P0–P2）</p>

    <el-alert v-if="alerts.length" type="warning" show-icon :closable="false" style="margin-bottom:12px"
      :title="`证照临期/过期 ${alerts.length} 条（60天内）`" />

    <div class="toolbar">
      <el-select v-model="filter.status" clearable placeholder="状态" style="width:120px" @change="loadList">
        <el-option v-for="(lab,k) in statusLabel" :key="k" :label="lab" :value="k" />
      </el-select>
      <el-select v-model="filter.supplier_type" clearable placeholder="类型" style="width:120px" @change="loadList">
        <el-option label="原料" value="raw" />
        <el-option label="辅料" value="aux" />
        <el-option label="包材" value="pack" />
        <el-option label="物流" value="logistics" />
        <el-option label="委外" value="outsource" />
        <el-option label="服务" value="service" />
      </el-select>
      <el-input v-model="filter.q" placeholder="编码/名称" clearable style="width:180px" @keyup.enter="loadList" />
      <el-button type="primary" @click="openCreate">新建供应商</el-button>
      <el-button @click="loadList">刷新</el-button>
      <span class="muted">共 {{ total }} 家</span>
    </div>

    <TableOrCards :data="list" :loading="loading" :columns="supplierCols">
      <el-table :data="list" border stripe>
        <el-table-column prop="code" label="编码" width="120" />
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="supplier_type" label="类型" width="90" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">{{ statusLabel[String(row.status)] || row.status }}</template>
        </el-table-column>
        <el-table-column prop="rating" label="等级" width="70" />
        <el-table-column prop="settle_method" label="结算" width="90" />
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">详情</el-button>
            <el-button v-if="row.status==='potential'" link type="success" @click="doAction(row,'qualify')">准入</el-button>
            <el-button v-if="row.status==='qualified'" link type="warning" @click="doAction(row,'freeze')">冻结</el-button>
            <el-button v-if="row.status==='frozen'" link type="success" @click="doAction(row,'activate')">解冻</el-button>
            <el-button v-if="row.status!=='blacklist'" link type="danger" @click="doAction(row,'blacklist')">拉黑</el-button>
            <el-button link type="danger" @click="removeRow(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #field-status="{ row }">{{ statusLabel[String(row.status)] || row.status }}</template>
      <template #actions="{ row }">
        <el-button link type="primary" @click="openEdit(row)">详情</el-button>
        <el-button v-if="row.status==='potential'" link type="success" @click="doAction(row,'qualify')">准入</el-button>
        <el-button v-if="row.status==='qualified'" link type="warning" @click="doAction(row,'freeze')">冻结</el-button>
        <el-button v-if="row.status==='frozen'" link type="success" @click="doAction(row,'activate')">解冻</el-button>
        <el-button v-if="row.status!=='blacklist'" link type="danger" @click="doAction(row,'blacklist')">拉黑</el-button>
        <el-button link type="danger" @click="removeRow(row)">删除</el-button>
      </template>
    </TableOrCards>

    <el-card shadow="never" style="margin-top:16px">
      <template #header>量价分析（已过账入库）</template>
      <TableOrCards :data="analytics" :columns="analyticsCols">
        <el-table :data="analytics" size="small" border>
          <el-table-column prop="supplier_name" label="供应商" />
          <el-table-column prop="product_id" label="物料" width="80" />
          <el-table-column prop="qty" label="数量" width="100" />
          <el-table-column prop="amount" label="金额" width="100" />
          <el-table-column prop="avg_price" label="均价" width="100" />
        </el-table>
      </TableOrCards>
    </el-card>

    <el-dialog v-model="dialog" :title="editing ? '供应商详情' : '新建供应商'" width="860px" destroy-on-close>
      <el-tabs v-model="detailTab">
        <el-tab-pane label="档案" name="profile">
          <el-form label-width="110px">
            <el-row :gutter="12">
              <el-col :span="12"><el-form-item label="编码"><el-input v-model="form.code" :disabled="editing" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="名称"><el-input v-model="form.name" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="简称"><el-input v-model="form.short_name" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="类型">
                <el-select v-model="form.supplier_type" style="width:100%">
                  <el-option label="原料" value="raw" /><el-option label="辅料" value="aux" />
                  <el-option label="包材" value="pack" /><el-option label="物流" value="logistics" />
                  <el-option label="委外" value="outsource" /><el-option label="服务" value="service" />
                </el-select>
              </el-form-item></el-col>
              <el-col :span="12"><el-form-item label="等级"><EnumSelect v-model="form.rating" :options="SUPPLIER_RATING_OPTIONS" :clearable="false" style="width:100%" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="首选"><el-switch v-model="form.is_preferred" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="信用代码"><el-input v-model="form.uscc" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="法人"><el-input v-model="form.legal_person" /></el-form-item></el-col>
              <el-col :span="24"><el-form-item label="注册地址"><el-input v-model="form.register_address" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="结算方式"><EnumSelect v-model="form.settle_method" :options="SETTLE_METHOD_OPTIONS" :clearable="false" style="width:100%" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="账期天"><el-input-number v-model="form.payment_days" :min="0" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="币种"><EnumSelect v-model="form.currency" :options="CURRENCY_OPTIONS" :clearable="false" style="width:100%" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="默认仓库"><WarehouseSelect v-model="form.default_warehouse_id" style="width:100%" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="开户行"><el-input v-model="form.bank_name" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="账号"><el-input v-model="form.bank_account" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="税号"><el-input v-model="form.tax_no" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="交期天"><el-input-number v-model="form.lead_time_days" :min="0" /></el-form-item></el-col>
              <el-col :span="24"><el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item></el-col>
            </el-row>
            <div class="sub">联系人</div>
            <div v-for="(c,i) in (form.contact_json as Row[])" :key="i" class="contact-row">
              <el-input v-model="c.name" placeholder="姓名" />
              <el-input v-model="c.mobile" placeholder="手机" />
              <el-switch v-model="c.is_primary" active-text="主" />
            </div>
            <el-button size="small" @click="addContact">加联系人</el-button>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="证照" name="licenses" :disabled="!editing">
          <el-button size="small" @click="addLicense">新增</el-button>
          <el-button size="small" type="primary" @click="saveLicenses">保存证照</el-button>
          <TableOrCards :data="licenses" :columns="licenseCols" style="margin-top:8px">
            <el-table :data="licenses" size="small" border style="margin-top:8px">
              <el-table-column label="类型"><template #default="{row}"><EnumSelect v-model="row.license_type" :options="LICENSE_TYPE_OPTIONS" style="width:100%" /></template></el-table-column>
              <el-table-column label="号码"><template #default="{row}"><el-input v-model="row.license_no" /></template></el-table-column>
              <el-table-column label="到期日"><template #default="{row}"><el-date-picker v-model="row.expire_date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></template></el-table-column>
            </el-table>
            <template #title="{ row }">
              <EnumSelect v-model="row.license_type" :options="LICENSE_TYPE_OPTIONS" style="width:100%" />
            </template>
            <template #field-license_no="{ row }"><el-input v-model="row.license_no" /></template>
            <template #field-expire_date="{ row }">
              <el-date-picker v-model="row.expire_date" type="date" value-format="YYYY-MM-DD" style="width:100%" />
            </template>
          </TableOrCards>
        </el-tab-pane>
        <el-tab-pane label="可供物料" name="supply" :disabled="!editing">
          <el-button size="small" @click="addSupply">新增</el-button>
          <el-button size="small" type="primary" @click="saveSupply">保存</el-button>
          <TableOrCards :data="supplyItems" :columns="supplyCols" style="margin-top:8px">
            <el-table :data="supplyItems" size="small" border style="margin-top:8px">
              <el-table-column label="物料" width="180"><template #default="{row}"><ProductSelect v-model="row.product_id" :clearable="false" style="width:100%" /></template></el-table-column>
              <el-table-column label="首选" width="80"><template #default="{row}"><el-switch v-model="row.is_preferred" /></template></el-table-column>
              <el-table-column label="MOQ"><template #default="{row}"><el-input-number v-model="row.moq" :min="0" /></template></el-table-column>
              <el-table-column label="交期"><template #default="{row}"><el-input-number v-model="row.lead_time_days" :min="0" /></template></el-table-column>
              <el-table-column label="最近价"><template #default="{row}"><el-input-number v-model="row.last_price" :min="0" :step="0.01" /></template></el-table-column>
            </el-table>
            <template #title="{ row }">
              <ProductSelect v-model="row.product_id" :clearable="false" style="width:100%" />
            </template>
            <template #field-is_preferred="{ row }"><el-switch v-model="row.is_preferred" /></template>
            <template #field-moq="{ row }"><el-input-number v-model="row.moq" :min="0" /></template>
            <template #field-lead_time_days="{ row }"><el-input-number v-model="row.lead_time_days" :min="0" /></template>
            <template #field-last_price="{ row }"><el-input-number v-model="row.last_price" :min="0" :step="0.01" /></template>
          </TableOrCards>
        </el-tab-pane>
        <el-tab-pane label="价格历史" name="prices" :disabled="!editing">
          <TableOrCards :data="prices" :columns="priceCols">
            <el-table :data="prices" size="small" border>
              <el-table-column prop="product_id" label="物料" />
              <el-table-column prop="price" label="价格" />
              <el-table-column prop="biz_date" label="日期" />
              <el-table-column prop="source_doc_id" label="来源入库" />
            </el-table>
          </TableOrCards>
        </el-tab-pane>
        <el-tab-pane label="绩效" name="perf" :disabled="!editing">
          <el-descriptions v-if="perf" :column="2" border size="small">
            <el-descriptions-item label="采购额">{{ perf.purchase_amount }}</el-descriptions-item>
            <el-descriptions-item label="采购量">{{ perf.purchase_qty }}</el-descriptions-item>
            <el-descriptions-item label="合格率">{{ Number(perf.pass_rate||0).toFixed(2) }}</el-descriptions-item>
            <el-descriptions-item label="退货率">{{ Number(perf.return_rate||0).toFixed(2) }}</el-descriptions-item>
            <el-descriptions-item label="入库单数">{{ perf.inbound_count }}</el-descriptions-item>
            <el-descriptions-item label="最近采购">{{ perf.last_purchase_date || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="dialog=false">关闭</el-button>
        <el-button type="primary" @click="save">保存档案</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.panel { background: #fff; border-radius: 8px; padding: 16px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; font-size: 18px; }
.desc { margin: 0 0 12px; color: #5c6b75; font-size: 13px; }
.toolbar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 12px; }
.muted { color: #5c6b75; font-size: 13px; margin-left: auto; }
.sub { font-weight: 600; margin: 8px 0; }
.contact-row { display: grid; grid-template-columns: 1fr 1fr auto; gap: 8px; margin-bottom: 8px; }
</style>
