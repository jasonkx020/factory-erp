<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  crmApi,
  CUSTOMER_SOURCE_OPTIONS,
  SETTLE_METHOD_OPTIONS,
} from '@erp/shared'
import { CustomerSelect, UserSelect, EnumSelect } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const customerCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'contact_name', label: '联系人' },
  { prop: 'mobile', label: '手机' },
  { prop: 'level', label: '等级' },
  { prop: 'owner_user_id', label: '归属' },
  { prop: 'protect_until', label: '保护至' },
]
const oppCols: MobileCardColumn[] = [
  { prop: 'title', label: '标题', primary: true },
  { prop: 'customer_name', label: '客户' },
  { prop: 'stage', label: '阶段' },
  { prop: 'amount', label: '金额' },
  { prop: 'status', label: '状态' },
  { prop: 'converted_order_id', label: '订单ID' },
]
const followCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'follow_type', label: '方式' },
  { prop: 'follow_at', label: '时间' },
  { prop: 'content', label: '内容' },
  { prop: 'next_remind_at', label: '下次提醒' },
]
const assignCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'from_user_id', label: '原归属' },
  { prop: 'to_user_id', label: '新归属' },
  { prop: 'assigned_at', label: '时间' },
  { prop: 'lock_flag', label: '锁定' },
  { prop: 'remark', label: '备注' },
]
const protectCols: MobileCardColumn[] = [
  { prop: 'name', label: '名称', primary: true },
  { prop: 'protect_days', label: '天数' },
  { prop: 'status', label: '状态' },
  { prop: 'release_rule_json', label: '规则' },
]
const releaseCols: MobileCardColumn[] = [
  { prop: 'customer_name', label: '客户', primary: true },
  { prop: 'released_at', label: '时间' },
  { prop: 'from_user_id', label: '原归属' },
  { prop: 'reason', label: '原因' },
]
const inquiryCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '询价单号', primary: true },
  { prop: 'customer_name', label: '客户' },
  { prop: 'status', label: '状态' },
  { prop: 'source', label: '来源' },
  { prop: 'expire_at', label: '有效期' },
  { prop: 'created_at', label: '创建时间' },
]
const importCols: MobileCardColumn[] = [
  { prop: 'file_name', label: '文件', primary: true },
  { prop: 'imported_at', label: '时间' },
  { prop: 'success_count', label: '成功' },
  { prop: 'fail_count', label: '失败' },
  { prop: 'fail_detail_json', label: '失败明细' },
]
const lockCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'owner_user_id', label: '归属' },
  { prop: 'protect_until', label: '保护至' },
]
const hideCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'owner_user_id', label: '归属' },
]
const reminderCols: MobileCardColumn[] = [
  { prop: 'content', label: '内容', primary: true },
  { prop: 'remind_at', label: '提醒时间' },
  { prop: 'user_id', label: '用户' },
  { prop: 'ref_id', label: '关联ID' },
  { prop: 'status', label: '状态' },
]

const route = useRoute()

const MODULE_MAP: Record<string, string> = {
  CRM客户管理: 'customers',
  商机管理: 'opportunities',
  客户档案: 'profile',
  客户跟进: 'follow-ups',
  资源分配: 'assigns',
  保护机制: 'protect',
  释放机制: 'releases',
  询价管理: 'inquiries',
  导入客户: 'imports',
  线索锁定: 'locks',
  线索隐藏: 'hides',
  任务提醒: 'reminders',
}

const TITLE_MAP: Record<string, string> = {
  customers: 'CRM客户管理',
  opportunities: '商机管理',
  profile: '客户档案',
  'follow-ups': '客户跟进',
  assigns: '资源分配',
  protect: '保护机制',
  releases: '释放机制',
  inquiries: '询价管理',
  imports: '导入客户',
  locks: '线索锁定',
  hides: '线索隐藏',
  reminders: '任务提醒',
}

const active = computed(() => {
  const section = String(route.params.section || '')
  return section || 'customers'
})

const title = computed(() => TITLE_MAP[active.value] || MODULE_MAP[active.value] || '客户管理')

const loading = ref(false)
const customers = ref<Row[]>([])
const list = ref<Row[]>([])
const detail = ref<Row | null>(null)
const keyword = ref('')
const seaOnly = ref(false)

const customerForm = reactive({
  name: '',
  short_name: '',
  contact_name: '',
  mobile: '',
  address: '',
  level: 'B',
  source: 'phone',
  remark: '',
  is_public_sea: false,
})

const profileForm = reactive({
  customer_id: 0 as number,
  contact_name: '',
  mobile: '',
  address: '',
  settle_method: 'monthly',
  payment_days: 30,
  credit_limit: 50000,
  logistics_remark: '',
  level: 'B',
  source: '',
  remark: '',
})

const oppForm = reactive({
  customer_id: 1,
  title: '',
  stage: 'lead',
  amount: 10000,
  expected_date: '',
  remark: '',
})

const followForm = reactive({
  customer_id: 1,
  follow_type: 'visit',
  content: '',
  next_remind_at: '',
})

const assignForm = reactive({
  customer_id: 0 as number,
  to_user_id: 1,
  lock_flag: false,
  remark: '',
})

const protectForm = reactive({
  name: '默认保护规则',
  protect_days: 30,
  release_rule_json: '{"auto_release":true,"idle_days":30}',
})

const releaseForm = reactive({
  customer_id: 0 as number,
  reason: '保护期到期/手动回收',
})

const importText = ref(
  '[{"name":"北海水产加工厂","contact_name":"李厂长","mobile":"13900020001","address":"广西北海","level":"B","source":"导入"}]',
)

const reminderForm = reactive({
  user_id: 1,
  ref_type: 'customer',
  ref_id: 1,
  remind_at: '',
  content: '',
})

async function loadCustomersMeta() {
  const params = seaOnly.value ? 'is_public_sea=1&page_size=200' : 'page_size=200'
  const res = await crmApi.customers(params)
  customers.value = ((res.data as { list?: Row[] })?.list) || []
  if (customers.value[0]) {
    const id = Number(customers.value[0].id)
    if (!profileForm.customer_id) profileForm.customer_id = id
    if (!oppForm.customer_id) oppForm.customer_id = id
    if (!followForm.customer_id) followForm.customer_id = id
    if (!assignForm.customer_id) assignForm.customer_id = id
    if (!releaseForm.customer_id) releaseForm.customer_id = id
    if (!reminderForm.ref_id) reminderForm.ref_id = id
  }
}

async function refresh() {
  loading.value = true
  detail.value = null
  try {
    let res
    switch (active.value) {
      case 'customers':
      case 'locks':
      case 'hides': {
        const q: string[] = ['page_size=100']
        if (keyword.value) q.push(`keyword=${encodeURIComponent(keyword.value)}`)
        if (seaOnly.value) q.push('is_public_sea=1')
        if (active.value === 'hides') q.push('include_hidden=1')
        if (active.value === 'locks') q.push('is_locked=1')
        res = await crmApi.customers(q.join('&'))
        break
      }
      case 'profile':
        if (!profileForm.customer_id) await loadCustomersMeta()
        if (profileForm.customer_id) {
          res = await crmApi.getProfile(profileForm.customer_id)
          if (res.code !== 1) return ElMessage.error(res.msg)
          detail.value = (res.data as Row) || null
          const d = detail.value
          if (d) {
            profileForm.contact_name = String(d.contact_name || '')
            profileForm.mobile = String(d.mobile || '')
            profileForm.address = String(d.address || '')
            profileForm.settle_method = String(d.settle_method || 'monthly')
            profileForm.payment_days = Number(d.payment_days || 30)
            profileForm.credit_limit = Number(d.credit_limit || 0)
            profileForm.logistics_remark = String(d.logistics_remark || '')
            profileForm.level = String(d.level || 'B')
            profileForm.source = String(d.source || '')
            profileForm.remark = String(d.remark || '')
          }
          list.value = []
          return
        }
        list.value = []
        return
      case 'opportunities':
        res = await crmApi.opportunities()
        break
      case 'follow-ups':
        res = await crmApi.followUps()
        break
      case 'assigns':
        res = await crmApi.leadAssigns()
        break
      case 'protect':
        res = await crmApi.protectRules()
        break
      case 'releases':
        res = await crmApi.releases()
        break
      case 'inquiries':
        res = await crmApi.inquiries()
        break
      case 'imports':
        res = await crmApi.imports()
        break
      case 'reminders':
        res = await crmApi.taskReminders()
        break
      default:
        res = await crmApi.customers()
    }
    if (res && res.code !== 1) return ElMessage.error(res.msg)
    list.value = ((res?.data as { list?: Row[] })?.list) || []
  } finally {
    loading.value = false
  }
}

async function createCustomer() {
  if (!customerForm.name) return ElMessage.warning('请填写客户名称')
  const res = await crmApi.createCustomer({ ...customerForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已创建 ${(res.data as Row)?.code}`)
  customerForm.name = ''
  customerForm.contact_name = ''
  customerForm.mobile = ''
  await loadCustomersMeta()
  await refresh()
}

async function saveProfile() {
  if (!profileForm.customer_id) return
  const res = await crmApi.updateProfile(profileForm.customer_id, { ...profileForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('档案已保存')
  await refresh()
}

async function createOpp() {
  const res = await crmApi.createOpportunity({ ...oppForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('商机已创建')
  await refresh()
}

async function convertOpp(id: number) {
  const res = await crmApi.convertOpportunity(id, { product_id: 3, qty: 100 })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const order = (res.data as Row)?.order as Row | undefined
  ElMessage.success(`已转化订单 ${order?.doc_no || ''}`)
  await refresh()
}

async function createFollow() {
  const res = await crmApi.createFollowUp({ ...followForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('跟进已记录')
  await refresh()
}

async function assignLead() {
  if (!assignForm.customer_id) return ElMessage.warning('请选择客户')
  const res = await crmApi.assignLead({ ...assignForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('分配成功')
  await loadCustomersMeta()
  await refresh()
}

async function createProtect() {
  const res = await crmApi.createProtectRule({ ...protectForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('保护规则已保存')
  await refresh()
}

async function releaseLead() {
  if (!releaseForm.customer_id) return ElMessage.warning('请选择客户')
  await ElMessageBox.confirm('确认释放到公海？归属业务员将被清空。')
  const res = await crmApi.releaseLead({ ...releaseForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已释放公海')
  await loadCustomersMeta()
  await refresh()
}

async function autoRelease() {
  const res = await crmApi.releaseLead({ auto: true })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`自动释放 ${(res.data as Row)?.released_count || 0} 条`)
  await loadCustomersMeta()
  await refresh()
}

async function doImport() {
  let rows: Row[]
  try {
    rows = JSON.parse(importText.value)
    if (!Array.isArray(rows)) throw new Error('not array')
  } catch {
    return ElMessage.error('导入内容须为 JSON 数组')
  }
  const res = await crmApi.importCustomers({ file_name: 'admin-import.json', rows })
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = res.data as Row
  ElMessage.success(`成功 ${d.success_count} / 失败 ${d.fail_count}`)
  await loadCustomersMeta()
  await refresh()
}

async function createReminder() {
  if (!reminderForm.remind_at) return ElMessage.warning('请填写提醒时间')
  const res = await crmApi.createTaskReminder({ ...reminderForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('提醒已创建')
  await refresh()
}

async function doneReminder(id: number) {
  const res = await crmApi.updateTaskReminder(id, { status: 'done' })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已完成')
  await refresh()
}

async function lockRow(id: number, locked: boolean) {
  const res = locked ? await crmApi.unlockCustomer(id) : await crmApi.lockCustomer(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(locked ? '已解锁' : '已锁定')
  await refresh()
}

async function hideRow(id: number, hidden: boolean) {
  const res = hidden ? await crmApi.unhideCustomer(id) : await crmApi.hideCustomer(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(hidden ? '已取消隐藏' : '已隐藏')
  await refresh()
}

async function openProfile(id: number) {
  profileForm.customer_id = id
  const res = await crmApi.getProfile(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  detail.value = (res.data as Row) || null
}

watch(active, async () => {
  await loadCustomersMeta()
  await refresh()
})

onMounted(async () => {
  if (!reminderForm.remind_at) {
    const d = new Date()
    d.setDate(d.getDate() + 1)
    reminderForm.remind_at = d.toISOString().slice(0, 19).replace('T', ' ')
  }
  await loadCustomersMeta()
  await refresh()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <p class="hint">工厂客户交付：客户主档 → 跟进/商机 → 分配保护公海 → 询价转销售订单。与销售 Hub 共用客户主数据。</p>
    </div>

    <!-- 客户主档 -->
    <template v-if="active === 'customers'">
      <el-card header="新建客户" class="mb">
        <el-form inline size="small">
          <el-form-item label="名称"><el-input v-model="customerForm.name" placeholder="必填" /></el-form-item>
          <el-form-item label="简称"><el-input v-model="customerForm.short_name" /></el-form-item>
          <el-form-item label="联系人"><el-input v-model="customerForm.contact_name" /></el-form-item>
          <el-form-item label="手机"><el-input v-model="customerForm.mobile" /></el-form-item>
          <el-form-item label="等级">
            <el-select v-model="customerForm.level" style="width:80px">
              <el-option label="A" value="A" /><el-option label="B" value="B" /><el-option label="C" value="C" />
            </el-select>
          </el-form-item>
          <el-form-item label="来源"><EnumSelect v-model="customerForm.source" :options="CUSTOMER_SOURCE_OPTIONS" :clearable="false" style="width:120px" /></el-form-item>
          <el-form-item label="公海"><el-switch v-model="customerForm.is_public_sea" /></el-form-item>
          <el-button type="primary" @click="createCustomer">新建</el-button>
        </el-form>
      </el-card>
      <el-card class="mb">
        <el-form inline size="small">
          <el-form-item label="搜索"><el-input v-model="keyword" clearable placeholder="名称/编码/手机" @keyup.enter="refresh" /></el-form-item>
          <el-form-item label="仅公海"><el-switch v-model="seaOnly" @change="refresh" /></el-form-item>
          <el-button @click="refresh">查询</el-button>
        </el-form>
        <TableOrCards :data="list" :loading="loading" :columns="customerCols">
          <el-table :data="list" size="small">
            <el-table-column prop="code" label="编码" width="120" />
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="contact_name" label="联系人" width="100" />
            <el-table-column prop="mobile" label="手机" width="120" />
            <el-table-column prop="level" label="等级" width="70" />
            <el-table-column prop="owner_user_id" label="归属" width="80" />
            <el-table-column prop="protect_until" label="保护至" width="110" />
            <el-table-column label="状态" width="140">
              <template #default="{ row }">
                <el-tag v-if="row.is_public_sea" size="small" type="warning">公海</el-tag>
                <el-tag v-if="row.is_locked" size="small" type="danger">锁定</el-tag>
                <el-tag v-if="row.is_hidden" size="small" type="info">隐藏</el-tag>
                <span v-if="!row.is_public_sea && !row.is_locked && !row.is_hidden">{{ row.status }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openProfile(Number(row.id))">档案</el-button>
                <el-button link @click="lockRow(Number(row.id), !!row.is_locked)">{{ row.is_locked ? '解锁' : '锁定' }}</el-button>
                <el-button link @click="hideRow(Number(row.id), !!row.is_hidden)">{{ row.is_hidden ? '取消隐藏' : '隐藏' }}</el-button>
              </template>
            </el-table-column>
          </el-table>
          <template #actions="{ row }">
            <el-button link type="primary" @click="openProfile(Number(row.id))">档案</el-button>
            <el-button link @click="lockRow(Number(row.id), !!row.is_locked)">{{ row.is_locked ? '解锁' : '锁定' }}</el-button>
            <el-button link @click="hideRow(Number(row.id), !!row.is_hidden)">{{ row.is_hidden ? '取消隐藏' : '隐藏' }}</el-button>
          </template>
        </TableOrCards>
      </el-card>
    </template>

    <!-- 客户档案 360 -->
    <template v-else-if="active === 'profile'">
      <el-card header="客户档案" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户">
            <CustomerSelect v-model="profileForm.customer_id" :clearable="false" style="width:220px" @update:model-value="refresh" />
          </el-form-item>
          <el-form-item label="联系人"><el-input v-model="profileForm.contact_name" /></el-form-item>
          <el-form-item label="手机"><el-input v-model="profileForm.mobile" /></el-form-item>
          <el-form-item label="地址"><el-input v-model="profileForm.address" style="width:200px" /></el-form-item>
          <el-form-item label="结算"><EnumSelect v-model="profileForm.settle_method" :options="SETTLE_METHOD_OPTIONS" :clearable="false" style="width:120px" /></el-form-item>
          <el-form-item label="账期天"><el-input-number v-model="profileForm.payment_days" :min="0" /></el-form-item>
          <el-form-item label="信用额"><el-input-number v-model="profileForm.credit_limit" :min="0" :step="1000" /></el-form-item>
          <el-form-item label="来源"><EnumSelect v-model="profileForm.source" :options="CUSTOMER_SOURCE_OPTIONS" style="width:120px" /></el-form-item>
          <el-form-item label="物流备注"><el-input v-model="profileForm.logistics_remark" style="width:160px" /></el-form-item>
          <el-button type="primary" @click="saveProfile">保存档案</el-button>
        </el-form>
      </el-card>
    </template>

    <!-- 商机 -->
    <template v-else-if="active === 'opportunities'">
      <el-card header="新建商机" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="oppForm.customer_id" :clearable="false" /></el-form-item>
          <el-form-item label="标题"><el-input v-model="oppForm.title" placeholder="年供/询价意向" /></el-form-item>
          <el-form-item label="阶段">
            <el-select v-model="oppForm.stage" style="width:120px">
              <el-option label="线索" value="lead" />
              <el-option label="意向" value="interest" />
              <el-option label="谈判" value="negotiation" />
              <el-option label="成交" value="won" />
              <el-option label="丢单" value="lost" />
            </el-select>
          </el-form-item>
          <el-form-item label="金额"><el-input-number v-model="oppForm.amount" :min="0" :step="1000" /></el-form-item>
          <el-form-item label="预计成交"><el-date-picker v-model="oppForm.expected_date" type="date" value-format="YYYY-MM-DD" style="width:150px" /></el-form-item>
          <el-button type="primary" @click="createOpp">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="oppCols">
        <el-table :data="list" size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="customer_name" label="客户" width="140" />
          <el-table-column prop="title" label="标题" min-width="160" />
          <el-table-column prop="stage" label="阶段" width="100" />
          <el-table-column prop="amount" label="金额" width="100" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="converted_order_id" label="订单ID" width="90" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button v-if="row.status !== 'won'" link type="primary" @click="convertOpp(Number(row.id))">转订单</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status !== 'won'" link type="primary" @click="convertOpp(Number(row.id))">转订单</el-button>
        </template>
      </TableOrCards>
    </template>

    <!-- 跟进 -->
    <template v-else-if="active === 'follow-ups'">
      <el-card header="登记跟进" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="followForm.customer_id" :clearable="false" /></el-form-item>
          <el-form-item label="方式">
            <el-select v-model="followForm.follow_type" style="width:100px">
              <el-option label="拜访" value="visit" />
              <el-option label="电话" value="call" />
              <el-option label="微信" value="wechat" />
            </el-select>
          </el-form-item>
          <el-form-item label="内容"><el-input v-model="followForm.content" style="width:260px" /></el-form-item>
          <el-form-item label="下次提醒"><el-date-picker v-model="followForm.next_remind_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:190px" /></el-form-item>
          <el-button type="primary" @click="createFollow">保存</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="followCols">
        <el-table :data="list" size="small">
          <el-table-column prop="customer_name" label="客户" width="140" />
          <el-table-column prop="follow_type" label="方式" width="80" />
          <el-table-column prop="follow_at" label="时间" width="160" />
          <el-table-column prop="content" label="内容" min-width="200" />
          <el-table-column prop="next_remind_at" label="下次提醒" width="160" />
        </el-table>
      </TableOrCards>
    </template>

    <!-- 资源分配 -->
    <template v-else-if="active === 'assigns'">
      <el-card header="分配/认领客户" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="assignForm.customer_id" :clearable="false" style="width:220px" /></el-form-item>
          <el-form-item label="分配给"><UserSelect v-model="assignForm.to_user_id" :clearable="false" /></el-form-item>
          <el-form-item label="同时锁定"><el-switch v-model="assignForm.lock_flag" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="assignForm.remark" /></el-form-item>
          <el-button type="primary" @click="assignLead">执行分配</el-button>
          <el-button @click="seaOnly = true; loadCustomersMeta()">刷新公海客户</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="assignCols">
        <el-table :data="list" size="small">
          <el-table-column prop="customer_name" label="客户" width="160" />
          <el-table-column prop="from_user_id" label="原归属" width="90" />
          <el-table-column prop="to_user_id" label="新归属" width="90" />
          <el-table-column prop="assigned_at" label="时间" width="160" />
          <el-table-column prop="lock_flag" label="锁定" width="80" />
          <el-table-column prop="remark" label="备注" />
        </el-table>
      </TableOrCards>
    </template>

    <!-- 保护 -->
    <template v-else-if="active === 'protect'">
      <el-card header="保护规则" class="mb">
        <el-form inline size="small">
          <el-form-item label="名称"><el-input v-model="protectForm.name" /></el-form-item>
          <el-form-item label="保护天数"><el-input-number v-model="protectForm.protect_days" :min="1" /></el-form-item>
          <el-form-item label="释放规则JSON"><el-input v-model="protectForm.release_rule_json" style="width:280px" /></el-form-item>
          <el-button type="primary" @click="createProtect">新建规则</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="protectCols">
        <el-table :data="list" size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="protect_days" label="天数" width="90" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="release_rule_json" label="规则" min-width="200" />
        </el-table>
      </TableOrCards>
    </template>

    <!-- 释放 -->
    <template v-else-if="active === 'releases'">
      <el-card header="释放公海" class="mb">
        <el-form inline size="small">
          <el-form-item label="客户"><CustomerSelect v-model="releaseForm.customer_id" :clearable="false" style="width:220px" /></el-form-item>
          <el-form-item label="原因"><el-input v-model="releaseForm.reason" style="width:220px" /></el-form-item>
          <el-button type="warning" @click="releaseLead">释放到公海</el-button>
          <el-button @click="autoRelease">到期自动释放</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="releaseCols">
        <el-table :data="list" size="small">
          <el-table-column prop="customer_name" label="客户" width="160" />
          <el-table-column prop="released_at" label="时间" width="160" />
          <el-table-column prop="from_user_id" label="原归属" width="90" />
          <el-table-column prop="reason" label="原因" />
        </el-table>
      </TableOrCards>
    </template>

    <!-- 询价关联 -->
    <template v-else-if="active === 'inquiries'">
      <TableOrCards :data="list" :loading="loading" :columns="inquiryCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="询价单号" width="150" />
          <el-table-column prop="customer_name" label="客户" width="160" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="source" label="来源" width="90" />
          <el-table-column prop="expire_at" label="有效期" width="160" />
          <el-table-column prop="created_at" label="创建时间" />
        </el-table>
      </TableOrCards>
      <p class="hint">询价新建/审批在「销售管理」完成；此处按客户查看关联询价。</p>
    </template>

    <!-- 导入 -->
    <template v-else-if="active === 'imports'">
      <el-card header="批量导入（JSON 数组）" class="mb">
        <el-input v-model="importText" type="textarea" :rows="6" />
        <div style="margin-top:8px">
          <el-button type="primary" @click="doImport">执行导入</el-button>
          <span class="hint" style="margin-left:8px">重复编码/名称自动跳过；导入客户默认进入公海待分配。</span>
        </div>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="importCols">
        <el-table :data="list" size="small">
          <el-table-column prop="id" label="批次" width="80" />
          <el-table-column prop="file_name" label="文件" width="160" />
          <el-table-column prop="imported_at" label="时间" width="160" />
          <el-table-column prop="success_count" label="成功" width="80" />
          <el-table-column prop="fail_count" label="失败" width="80" />
          <el-table-column prop="fail_detail_json" label="失败明细" min-width="200" show-overflow-tooltip />
        </el-table>
      </TableOrCards>
    </template>

    <!-- 锁定列表 -->
    <template v-else-if="active === 'locks'">
      <TableOrCards :data="list" :loading="loading" :columns="lockCols">
        <el-table :data="list" size="small">
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="owner_user_id" label="归属" width="90" />
          <el-table-column prop="protect_until" label="保护至" width="120" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button link type="primary" @click="lockRow(Number(row.id), true)">解锁</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link type="primary" @click="lockRow(Number(row.id), true)">解锁</el-button>
        </template>
      </TableOrCards>
      <p class="hint">无锁定客户时列表为空。可在客户管理中执行锁定。</p>
    </template>

    <!-- 隐藏列表 -->
    <template v-else-if="active === 'hides'">
      <TableOrCards :data="list.filter((r) => r.is_hidden)" :loading="loading" :columns="hideCols">
        <el-table :data="list.filter((r) => r.is_hidden)" size="small">
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="owner_user_id" label="归属" width="90" />
          <el-table-column label="操作" width="140">
            <template #default="{ row }">
              <el-button link type="primary" @click="hideRow(Number(row.id), true)">取消隐藏</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link type="primary" @click="hideRow(Number(row.id), true)">取消隐藏</el-button>
        </template>
      </TableOrCards>
    </template>

    <!-- 任务提醒 -->
    <template v-else-if="active === 'reminders'">
      <el-card header="新建提醒" class="mb">
        <el-form inline size="small">
          <el-form-item label="用户"><UserSelect v-model="reminderForm.user_id" :clearable="false" /></el-form-item>
          <el-form-item label="关联客户"><CustomerSelect v-model="reminderForm.ref_id" :clearable="false" /></el-form-item>
          <el-form-item label="时间"><el-date-picker v-model="reminderForm.remind_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" style="width:190px" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="reminderForm.content" style="width:220px" /></el-form-item>
          <el-button type="primary" @click="createReminder">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="reminderCols">
        <el-table :data="list" size="small">
          <el-table-column prop="remind_at" label="提醒时间" width="160" />
          <el-table-column prop="content" label="内容" min-width="200" />
          <el-table-column prop="user_id" label="用户" width="80" />
          <el-table-column prop="ref_id" label="关联ID" width="90" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status === 'pending'" link type="primary" @click="doneReminder(Number(row.id))">完成</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button v-if="row.status === 'pending'" link type="primary" @click="doneReminder(Number(row.id))">完成</el-button>
        </template>
      </TableOrCards>
    </template>

    <el-card v-if="detail" header="明细 / 客户360" style="margin-top:16px">
      <pre class="detail">{{ JSON.stringify(detail, null, 2) }}</pre>
    </el-card>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px; }
.head h2 { margin: 0 0 4px; }
.hint { color: #667; font-size: 13px; margin: 0 0 12px; }
.mb { margin-bottom: 12px; }
.detail { background: #f6f8fa; padding: 12px; border-radius: 8px; max-height: 420px; overflow: auto; font-size: 12px; }
</style>
