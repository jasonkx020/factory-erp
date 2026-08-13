<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { assetApi } from '@erp/shared'
import { DeptSelect, loadDepartments } from '../../components/select'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>
const departments = ref<Row[]>([])

function deptNameById(id: number | null | undefined): string {
  if (!id) return ''
  const row = departments.value.find((d) => Number(d.id) === Number(id))
  return row ? String(row.name ?? '') : ''
}

function onAssetDept(id: number | null) {
  assetForm.dept_id = id ?? 0
  assetForm.dept_name = deptNameById(id)
}

function onTransferDept(id: number | null) {
  transferForm.to_dept_id = id ?? 0
  transferForm.to_dept_name = deptNameById(id)
}

const categoryCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'parent_id', label: '上级ID' },
  { prop: 'remark', label: '备注' },
]
const assetCols: MobileCardColumn[] = [
  { prop: 'code', label: '编码', primary: true },
  { prop: 'name', label: '名称' },
  { prop: 'category_name', label: '类别' },
  { prop: 'dept_name', label: '部门' },
  { prop: 'location_text', label: '位置' },
  { prop: 'original_value', label: '原值' },
  { prop: 'net_value', label: '净值' },
  { prop: 'purchase_date', label: '购入日' },
  { prop: 'status', label: '状态' },
]
const transferCols: MobileCardColumn[] = [
  { prop: 'doc_no', label: '单号', primary: true },
  { prop: 'asset_code', label: '资产编码' },
  { prop: 'asset_name', label: '资产名称' },
  { prop: 'from_dept_name', label: '调出部门' },
  { prop: 'from_location', label: '调出位置' },
  { prop: 'to_dept_name', label: '调入部门' },
  { prop: 'to_location', label: '调入位置' },
  { prop: 'status', label: '状态' },
  { prop: 'transferred_at', label: '确认时间' },
]
const byCategoryCols: MobileCardColumn[] = [
  { prop: 'category_name', label: '类别', primary: true },
  { prop: 'category_code', label: '类别编码' },
  { prop: 'count', label: '数量' },
  { prop: 'original_value', label: '原值' },
  { prop: 'net_value', label: '净值' },
]
const byDeptCols: MobileCardColumn[] = [
  { prop: 'dept_name', label: '部门', primary: true },
  { prop: 'count', label: '数量' },
  { prop: 'original_value', label: '原值' },
  { prop: 'net_value', label: '净值' },
]
const byStatusCols: MobileCardColumn[] = [
  { prop: 'status', label: '状态', primary: true },
  { prop: 'count', label: '数量' },
  { prop: 'original_value', label: '原值' },
  { prop: 'net_value', label: '净值' },
]

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  categories: '固定资产类别',
  'fixed-assets': '固定资产项目',
  transfers: '固定资产内部转移',
  stats: '固定资产统计',
}

const active = computed(() => String(route.params.section || 'fixed-assets'))
const title = computed(() => TITLE_MAP[active.value] || '固定资产管理')

const loading = ref(false)
const list = ref<Row[]>([])
const categories = ref<Row[]>([])
const assets = ref<Row[]>([])
const stats = ref<Row | null>(null)
const byCategory = ref<Row[]>([])
const byDept = ref<Row[]>([])
const byStatus = ref<Row[]>([])
const keyword = ref('')

const catForm = reactive({
  code: '',
  name: '',
  parent_id: 0 as number,
  remark: '',
})

const assetForm = reactive({
  code: '',
  name: '',
  category_id: 0 as number,
  dept_id: 0 as number,
  dept_name: '',
  location_text: '',
  original_value: 0,
  net_value: 0,
  purchase_date: new Date().toISOString().slice(0, 10),
  useful_life_months: 60,
  residual_rate: 0.05,
  remark: '',
})

const transferForm = reactive({
  asset_id: 0 as number,
  to_dept_id: 0 as number,
  to_dept_name: '',
  to_location: '',
  remark: '',
})

async function loadMeta() {
  const [c, a, d] = await Promise.all([assetApi.categories(), assetApi.list(), loadDepartments()])
  categories.value = ((c.data as { list?: Row[] })?.list) || []
  assets.value = ((a.data as { list?: Row[] })?.list) || []
  departments.value = d || []
  if (categories.value[0] && !assetForm.category_id) {
    assetForm.category_id = Number(categories.value[0].id)
  }
  if (assets.value[0] && !transferForm.asset_id) {
    transferForm.asset_id = Number(assets.value[0].id)
  }
  if (!assetForm.dept_id && departments.value[0]) {
    onAssetDept(Number(departments.value[0].id))
  }
}

async function refresh() {
  loading.value = true
  try {
    if (active.value === 'categories') {
      const res = await assetApi.categories()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
      categories.value = list.value
    } else if (active.value === 'fixed-assets') {
      const qs = keyword.value ? `keyword=${encodeURIComponent(keyword.value)}` : undefined
      const res = await assetApi.list(qs)
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
      assets.value = list.value
    } else if (active.value === 'transfers') {
      const res = await assetApi.transfers()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'stats') {
      const res = await assetApi.stats()
      if (res.code !== 1) return ElMessage.error(res.msg)
      const data = (res.data as Row) || {}
      stats.value = (data.summary as Row) || null
      byCategory.value = (data.by_category as Row[]) || []
      byDept.value = (data.by_dept as Row[]) || []
      byStatus.value = (data.by_status as Row[]) || []
      list.value = byCategory.value
    }
  } finally {
    loading.value = false
  }
}

async function createCategory() {
  if (!catForm.name) return ElMessage.warning('填写类别名称')
  const res = await assetApi.createCategory({
    ...catForm,
    parent_id: catForm.parent_id || undefined,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`类别 ${(res.data as Row)?.name}`)
  catForm.code = ''
  catForm.name = ''
  catForm.remark = ''
  await loadMeta()
  await refresh()
}

async function removeCategory(id: number) {
  await ElMessageBox.confirm('确认删除该类别？', '提示', { type: 'warning' })
  const res = await assetApi.removeCategory(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已删除')
  await refresh()
}

async function createAsset() {
  if (!assetForm.name) return ElMessage.warning('填写资产名称')
  if (!assetForm.net_value && assetForm.original_value) {
    assetForm.net_value = assetForm.original_value
  }
  const res = await assetApi.create({ ...assetForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`资产卡片 ${(res.data as Row)?.code}`)
  assetForm.code = ''
  assetForm.name = ''
  assetForm.location_text = ''
  assetForm.original_value = 0
  assetForm.net_value = 0
  assetForm.remark = ''
  await loadMeta()
  await refresh()
}

async function removeAsset(id: number) {
  await ElMessageBox.confirm('确认报废/删除该资产？', '提示', { type: 'warning' })
  const res = await assetApi.remove(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已处理')
  await refresh()
}

async function createTransfer() {
  if (!transferForm.asset_id) return ElMessage.warning('选择资产')
  if (!transferForm.to_location && !transferForm.to_dept_id && !transferForm.to_dept_name) {
    return ElMessage.warning('填写调入位置或选择调入部门')
  }
  const res = await assetApi.createTransfer({ ...transferForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`转移单 ${(res.data as Row)?.doc_no}`)
  transferForm.to_location = ''
  transferForm.remark = ''
  await refresh()
}

async function confirmTransfer(id: number) {
  const res = await assetApi.confirmTransfer(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已确认转移，资产位置已更新')
  await loadMeta()
  await refresh()
}

onMounted(async () => {
  await loadMeta()
  await refresh()
})
watch(active, refresh)
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <el-button size="small" @click="refresh">刷新</el-button>
    </div>

    <!-- 类别 -->
    <template v-if="active === 'categories'">
      <el-card header="新建固定资产类别" class="mb">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="catForm.code" style="width:140px" placeholder="可空自动生成" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="catForm.name" style="width:160px" /></el-form-item>
          <el-form-item label="上级">
            <el-select v-model="catForm.parent_id" clearable style="width:160px" placeholder="无（顶级）">
              <el-option :value="0" label="无（顶级）" />
              <el-option
                v-for="c in categories"
                :key="String(c.id)"
                :label="String(c.name)"
                :value="Number(c.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="备注"><el-input v-model="catForm.remark" style="width:160px" /></el-form-item>
          <el-button type="primary" @click="createCategory">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="categoryCols">
        <el-table :data="list" size="small" row-key="id">
          <el-table-column prop="code" label="编码" width="140" />
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="parent_id" label="上级ID" width="90" />
          <el-table-column prop="remark" label="备注" min-width="140" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="removeCategory(Number(row.id))">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link type="danger" @click="removeCategory(Number(row.id))">删除</el-button>
        </template>
      </TableOrCards>
    </template>

    <!-- 资产项目 -->
    <template v-else-if="active === 'fixed-assets'">
      <el-card header="新建固定资产卡片" class="mb">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="assetForm.code" style="width:130px" placeholder="可空自动生成" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="assetForm.name" style="width:160px" /></el-form-item>
          <el-form-item label="类别">
            <el-select v-model="assetForm.category_id" filterable style="width:150px">
              <el-option
                v-for="c in categories"
                :key="String(c.id)"
                :label="String(c.name)"
                :value="Number(c.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="部门">
            <DeptSelect :model-value="assetForm.dept_id || null" style="width:180px" @update:model-value="onAssetDept" />
          </el-form-item>
          <el-form-item label="存放位置"><el-input v-model="assetForm.location_text" style="width:140px" /></el-form-item>
          <el-form-item label="原值"><el-input-number v-model="assetForm.original_value" :min="0" :step="100" /></el-form-item>
          <el-form-item label="净值"><el-input-number v-model="assetForm.net_value" :min="0" :step="100" /></el-form-item>
          <el-form-item label="购入日">
            <el-date-picker v-model="assetForm.purchase_date" type="date" value-format="YYYY-MM-DD" style="width:150px" />
          </el-form-item>
          <el-form-item label="年限(月)"><el-input-number v-model="assetForm.useful_life_months" :min="1" /></el-form-item>
          <el-button type="primary" @click="createAsset">新建</el-button>
        </el-form>
      </el-card>
      <div class="mb">
        <el-input v-model="keyword" size="small" clearable placeholder="搜索编码/名称/位置" style="width:240px" @keyup.enter="refresh" />
        <el-button size="small" style="margin-left:8px" @click="refresh">查询</el-button>
      </div>
      <TableOrCards :data="list" :loading="loading" :columns="assetCols">
        <el-table :data="list" size="small">
          <el-table-column prop="code" label="编码" width="130" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="category_name" label="类别" width="110" />
          <el-table-column prop="dept_name" label="部门" width="100" />
          <el-table-column prop="location_text" label="位置" width="120" />
          <el-table-column prop="original_value" label="原值" width="100" />
          <el-table-column prop="net_value" label="净值" width="100" />
          <el-table-column prop="purchase_date" label="购入日" width="110" />
          <el-table-column prop="status" label="状态" width="80" />
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button link type="danger" @click="removeAsset(Number(row.id))">报废</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button link type="danger" @click="removeAsset(Number(row.id))">报废</el-button>
        </template>
      </TableOrCards>
    </template>

    <!-- 内部转移 -->
    <template v-else-if="active === 'transfers'">
      <el-card header="新建内部转移（确认后回写资产部门/位置）" class="mb">
        <el-form inline size="small">
          <el-form-item label="资产">
            <el-select v-model="transferForm.asset_id" filterable style="width:220px">
              <el-option
                v-for="a in assets"
                :key="String(a.id)"
                :label="`${a.code} ${a.name}`"
                :value="Number(a.id)"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="调入部门">
            <DeptSelect :model-value="transferForm.to_dept_id || null" style="width:180px" @update:model-value="onTransferDept" />
          </el-form-item>
          <el-form-item label="调入位置"><el-input v-model="transferForm.to_location" style="width:140px" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="transferForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="createTransfer">新建</el-button>
        </el-form>
      </el-card>
      <TableOrCards :data="list" :loading="loading" :columns="transferCols">
        <el-table :data="list" size="small">
          <el-table-column prop="doc_no" label="单号" width="150" />
          <el-table-column prop="asset_code" label="资产编码" width="120" />
          <el-table-column prop="asset_name" label="资产名称" min-width="120" />
          <el-table-column prop="from_dept_name" label="调出部门" width="110" />
          <el-table-column prop="from_location" label="调出位置" width="110" />
          <el-table-column prop="to_dept_name" label="调入部门" width="110" />
          <el-table-column prop="to_location" label="调入位置" width="110" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="transferred_at" label="确认时间" width="160" />
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.status==='draft' || row.status==='submitted'"
                link
                type="primary"
                @click="confirmTransfer(Number(row.id))"
              >确认</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button
            v-if="row.status==='draft' || row.status==='submitted'"
            link
            type="primary"
            @click="confirmTransfer(Number(row.id))"
          >确认</el-button>
        </template>
      </TableOrCards>
    </template>

    <!-- 统计 -->
    <template v-else-if="active === 'stats'">
      <el-row :gutter="12" class="mb" v-if="stats">
        <el-col :span="6" :xs="24">
          <el-card shadow="never"><div class="stat-label">资产数量</div><div class="stat-value">{{ stats.asset_count }}</div></el-card>
        </el-col>
        <el-col :span="6" :xs="24">
          <el-card shadow="never"><div class="stat-label">原值合计</div><div class="stat-value">{{ Number(stats.original_value || 0).toFixed(2) }}</div></el-card>
        </el-col>
        <el-col :span="6" :xs="24">
          <el-card shadow="never"><div class="stat-label">净值合计</div><div class="stat-value">{{ Number(stats.net_value || 0).toFixed(2) }}</div></el-card>
        </el-col>
        <el-col :span="6" :xs="24">
          <el-card shadow="never"><div class="stat-label">累计折旧</div><div class="stat-value">{{ Number(stats.depreciation_value || 0).toFixed(2) }}</div></el-card>
        </el-col>
      </el-row>
      <el-row :gutter="12" class="mb" v-if="stats">
        <el-col :span="12" :xs="24">
          <el-card shadow="never"><div class="stat-label">待确认转移</div><div class="stat-value">{{ stats.transfer_draft }}</div></el-card>
        </el-col>
        <el-col :span="12" :xs="24">
          <el-card shadow="never"><div class="stat-label">已确认转移</div><div class="stat-value">{{ stats.transfer_confirmed }}</div></el-card>
        </el-col>
      </el-row>

      <h4>按类别</h4>
      <TableOrCards :data="byCategory" :loading="loading" :columns="byCategoryCols" class="mb">
        <el-table :data="byCategory" size="small" class="mb">
          <el-table-column prop="category_code" label="类别编码" width="120" />
          <el-table-column prop="category_name" label="类别" min-width="140" />
          <el-table-column prop="count" label="数量" width="90" />
          <el-table-column prop="original_value" label="原值" width="120" />
          <el-table-column prop="net_value" label="净值" width="120" />
        </el-table>
      </TableOrCards>

      <h4>按部门</h4>
      <TableOrCards :data="byDept" :loading="loading" :columns="byDeptCols" class="mb">
        <el-table :data="byDept" size="small" class="mb">
          <el-table-column prop="dept_name" label="部门" min-width="140" />
          <el-table-column prop="count" label="数量" width="90" />
          <el-table-column prop="original_value" label="原值" width="120" />
          <el-table-column prop="net_value" label="净值" width="120" />
        </el-table>
      </TableOrCards>

      <h4>按状态</h4>
      <TableOrCards :data="byStatus" :loading="loading" :columns="byStatusCols">
        <el-table :data="byStatus" size="small">
          <el-table-column prop="status" label="状态" width="120" />
          <el-table-column prop="count" label="数量" width="90" />
          <el-table-column prop="original_value" label="原值" width="120" />
          <el-table-column prop="net_value" label="净值" width="120" />
        </el-table>
      </TableOrCards>
    </template>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.head h2 { margin: 0; font-size: 18px; }
.mb { margin-bottom: 12px; }
.stat-label { color: #888; font-size: 13px; }
.stat-value { font-size: 22px; font-weight: 600; margin-top: 6px; }
h4 { margin: 8px 0 8px; font-size: 14px; }
</style>
