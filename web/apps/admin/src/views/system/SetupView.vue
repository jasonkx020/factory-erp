<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { systemApi } from '@erp/shared'

type CheckItem = {
  code: string
  label: string
  ok: boolean
  detail?: string
  required: boolean
}

const router = useRouter()
const loading = ref(false)
const ready = ref(false)
const profile = ref('core')
const pack = ref('cassava')
const items = ref<CheckItem[]>([])
const missing = ref<string[]>([])

const links: Record<string, string> = {
  organization: '/hr/departments',
  plant: '/hr/plants',
  warehouse_raw: '/inventory/hub/balances',
  warehouse_semi: '/inventory/hub/balances',
  warehouse_fg: '/inventory/hub/balances',
  product: '/product/hub/products',
  unit: '/product/hub/units',
  production_flow: '/production/hub/routings',
  weigh_variety: '/purchase/hub/varieties',
  purchase_gate_flow: '/purchase/hub/flow-graphs',
  shift: '/production/hub/shifts',
  field_user: '/hr/employees',
  wage_rate: '/payroll/wage-rates',
  customer: '/sales/hub/customers',
  finance_subject: '/finance/hub/subjects',
}

async function load() {
  loading.value = true
  try {
    const [r, p] = await Promise.all([systemApi.setupReadiness(), systemApi.productProfile()])
    const d = (r.data || {}) as {
      ready?: boolean
      items?: CheckItem[]
      missing?: string[]
      product_profile?: string
      industry_pack?: string
    }
    ready.value = !!d.ready
    items.value = d.items || []
    missing.value = d.missing || []
    const pd = (p.data || {}) as { profile?: string; industry_pack?: string }
    profile.value = pd.profile || d.product_profile || 'core'
    pack.value = pd.industry_pack || d.industry_pack || 'cassava'
  } catch (e) {
    ElMessage.error(String(e))
  } finally {
    loading.value = false
  }
}

function go(code: string) {
  const path = links[code]
  if (path) router.push(path)
}

onMounted(load)
</script>

<template>
  <div class="setup-page" v-loading="loading">
    <header class="head">
      <div>
        <h2>开厂配置</h2>
        <p class="sub">
          主数据须经后台配置后，现场过磅/过站/销售事务才会放行。
          当前 profile=<code>{{ profile }}</code>，industry_pack=<code>{{ pack }}</code>。
        </p>
      </div>
      <el-tag :type="ready ? 'success' : 'warning'" size="large">
        {{ ready ? '已就绪' : '未就绪' }}
      </el-tag>
    </header>

    <el-alert
      v-if="!ready"
      type="warning"
      :closable="false"
      show-icon
      title="配置未完成"
      :description="`缺项：${missing.join('、') || '见下方清单'}`"
      class="mb"
    />

    <el-table :data="items" border stripe>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.ok ? 'success' : row.required ? 'danger' : 'info'" size="small">
            {{ row.ok ? '通过' : row.required ? '必填' : '建议' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="label" label="检查项" min-width="180" />
      <el-table-column prop="code" label="编码" width="160" />
      <el-table-column prop="detail" label="说明" min-width="160" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button v-if="!row.ok && links[row.code]" link type="primary" @click="go(row.code)">去配置</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="actions">
      <el-button @click="load">刷新检查</el-button>
    </div>
  </div>
</template>

<style scoped>
.setup-page { padding: 4px; }
.head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; margin-bottom: 16px; }
h2 { margin: 0 0 6px; font-size: 18px; }
.sub { margin: 0; color: #64748b; font-size: 13px; line-height: 1.5; }
.mb { margin-bottom: 12px; }
.actions { margin-top: 16px; }
code { font-size: 12px; background: #f1f5f3; padding: 1px 6px; border-radius: 4px; }
</style>
