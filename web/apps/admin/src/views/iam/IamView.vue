<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { iamApi, ERP_MENUS } from '@erp/shared'
import HrPermView from './HrPermView.vue'
import AccountFreezePanel from './AccountFreezePanel.vue'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

const props = defineProps<{ module: string }>()

type Row = Record<string, unknown>

const policyCols: MobileCardColumn[] = [
  { prop: 'field_key', label: '字段', primary: true },
  { prop: 'field_name', label: '名称' },
  { prop: 'visible', label: '可见' },
  { prop: 'editable', label: '可编辑' },
]

const permCols: MobileCardColumn[] = [
  { prop: 'code', label: '权限码', primary: true },
  { prop: 'domain', label: '域' },
  { prop: 'module', label: '模块' },
  { prop: 'action', label: '动作' },
]

const menuCols: MobileCardColumn[] = [
  { prop: 'module', label: '模块', primary: true },
  { prop: 'domain', label: '域' },
  { prop: 'menu_key', label: '菜单键' },
  { prop: 'visible', label: '可见' },
  { prop: 'sort_no', label: '排序' },
]

const users = ref<Row[]>([])
const roles = ref<Row[]>([])
const permissions = ref<Row[]>([])
const menus = ref<Row[]>([])
const fieldPolicies = ref<Row[]>([])
const loginPolicy = ref<Row>({})
const loading = ref(false)

const selectedRoleId = ref<number | null>(null)
const menuDraft = ref<Row[]>([])
const policyDraft = ref<Row[]>([])
const permKeyword = ref('')
const permDomainFilter = ref('')

const COST_FIELDS = [
  { field_key: 'cost_price', field_name: '成本价' },
  { field_key: 'cost', field_name: '成本' },
  { field_key: 'gross_margin', field_name: '毛利率' },
  { field_key: 'gross_profit', field_name: '毛利' },
]

async function loadAll() {
  if (props.module === '角色管理' || props.module === '账户冻结') return
  loading.value = true
  try {
    const [u, r, p, m, f, lp] = await Promise.all([
      iamApi.users(),
      iamApi.roles(),
      iamApi.permissions(),
      iamApi.menus(),
      iamApi.fieldPolicies(),
      iamApi.loginPolicy(),
    ])
    users.value = (u.data as { list?: Row[] })?.list || []
    roles.value = (r.data as { list?: Row[] })?.list || []
    permissions.value = (p.data as { list?: Row[] })?.list || []
    menus.value = (m.data as { list?: Row[] })?.list || []
    fieldPolicies.value = (f.data as { list?: Row[] })?.list || []
    loginPolicy.value = (lp.data as Row) || {}
    if (!selectedRoleId.value && roles.value.length) {
      selectedRoleId.value = Number(roles.value[0].id)
    }
    rebuildMenuDraft()
    rebuildPolicyDraft()
  } finally {
    loading.value = false
  }
}

function rebuildMenuDraft() {
  const rid = selectedRoleId.value
  if (!rid) {
    menuDraft.value = []
    return
  }
  const existing = menus.value.filter((x) => Number(x.role_id) === rid)
  const byKey = new Map(existing.map((x) => [`${x.domain}|${x.module}`, x]))
  const rows: Row[] = []
  let sort = 10
  for (const d of ERP_MENUS) {
    for (const mod of d.modules) {
      const key = `${d.domain}|${mod}`
      const old = byKey.get(key)
      rows.push({
        role_id: rid,
        domain: d.domain,
        module: mod,
        menu_key: old?.menu_key || `${d.domain}/${mod}`,
        visible: old ? Number(old.visible) !== 0 && old.visible !== false : true,
        sort_no: old?.sort_no ?? sort,
      })
      sort += 10
    }
  }
  menuDraft.value = rows
}

function rebuildPolicyDraft() {
  const rid = selectedRoleId.value
  if (!rid) {
    policyDraft.value = []
    return
  }
  const existing = fieldPolicies.value.filter((x) => Number(x.role_id) === rid)
  const byKey = new Map(existing.map((x) => [String(x.field_key), x]))
  policyDraft.value = COST_FIELDS.map((f) => {
    const old = byKey.get(f.field_key)
    return {
      role_id: rid,
      field_key: f.field_key,
      field_name: old?.field_name || f.field_name,
      visible: old ? Number(old.visible) !== 0 && old.visible !== false : false,
      editable: old ? Number(old.editable) !== 0 && old.editable !== false : false,
    }
  })
}

watch(selectedRoleId, () => {
  rebuildMenuDraft()
  rebuildPolicyDraft()
})

async function saveLoginPolicy() {
  const res = await iamApi.saveLoginPolicy({ ...loginPolicy.value })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('登录策略已保存')
}

async function saveMenus() {
  const items = menuDraft.value.map((x) => ({
    role_id: Number(x.role_id),
    domain: x.domain,
    module: x.module,
    menu_key: x.menu_key,
    visible: !!x.visible,
    sort_no: Number(x.sort_no) || 0,
  }))
  const res = await iamApi.saveMenus(items)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('菜单可见性已保存')
  await loadAll()
}

async function saveFieldPolicies() {
  const items = policyDraft.value.map((x) => ({
    role_id: Number(x.role_id),
    field_key: x.field_key,
    field_name: x.field_name,
    visible: !!x.visible,
    editable: !!x.editable,
  }))
  const res = await iamApi.saveFieldPolicies(items)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('字段策略已保存')
  await loadAll()
}

async function syncPermissions() {
  const res = await iamApi.syncPermissions()
  if (res.code !== 1) return ElMessage.error(res.msg || '同步失败')
  ElMessage.success(`权限码已同步，共 ${(res.data as { total?: number })?.total ?? 0} 条`)
  await loadAll()
}

const roleOptions = computed(() =>
  roles.value.map((r) => ({ label: `${r.code || ''} ${r.name || ''}`.trim(), value: Number(r.id) })),
)

const permDomains = computed(() => {
  const set = new Set<string>()
  permissions.value.forEach((p) => {
    const d = String(p.domain || '').trim()
    if (d) set.add(d)
  })
  return [...set]
})

const filteredPermissions = computed(() => {
  const kw = permKeyword.value.trim().toLowerCase()
  const domain = permDomainFilter.value
  return permissions.value.filter((p) => {
    if (domain && String(p.domain || '') !== domain) return false
    if (!kw) return true
    const hay = `${p.code || ''} ${p.domain || ''} ${p.module || ''} ${p.action || ''}`.toLowerCase()
    return hay.includes(kw)
  })
})

onMounted(loadAll)
watch(() => props.module, loadAll)
</script>

<template>
  <HrPermView v-if="module === '角色管理'" />
  <div v-else v-loading="loading" class="iam" :class="{ fill: module === '自定义权限' }">
    <template v-if="module === '自定义权限'">
      <div class="page-head">
        <div>
          <h2 class="title">自定义权限</h2>
          <p class="desc">字段策略按角色生效；权限码字典只读，同步后可在「角色管理」中勾选授权。</p>
        </div>
        <div class="row" style="margin:0">
          <el-button type="primary" @click="syncPermissions">同步权限码</el-button>
          <el-button @click="loadAll">刷新</el-button>
        </div>
      </div>

      <div class="field-panel">
        <div class="field-panel-head">
          <h3 class="sub">字段权限控制</h3>
          <div class="row" style="margin:0">
            <span>角色</span>
            <el-select v-model="selectedRoleId" style="width:220px" placeholder="选择角色">
              <el-option v-for="o in roleOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <el-button type="primary" @click="saveFieldPolicies">保存字段策略</el-button>
          </div>
        </div>
        <TableOrCards :data="policyDraft" :columns="policyCols">
          <el-table :data="policyDraft" border>
            <el-table-column prop="field_key" label="字段" min-width="140" />
            <el-table-column prop="field_name" label="名称" min-width="120" />
            <el-table-column label="可见" width="100">
              <template #default="{ row }"><el-switch v-model="row.visible" /></template>
            </el-table-column>
            <el-table-column label="可编辑" width="100">
              <template #default="{ row }"><el-switch v-model="row.editable" /></template>
            </el-table-column>
          </el-table>
          <template #field-visible="{ row }"><el-switch v-model="row.visible" /></template>
          <template #field-editable="{ row }"><el-switch v-model="row.editable" /></template>
        </TableOrCards>
      </div>

      <div class="dict-toolbar">
        <h3 class="sub">权限码字典</h3>
        <span class="dict-count">共 {{ filteredPermissions.length }} / {{ permissions.length }} 条</span>
        <el-input
          v-model="permKeyword"
          clearable
          placeholder="搜索域 / 模块 / 动作 / 权限码"
          style="width:260px"
        />
        <el-select v-model="permDomainFilter" clearable placeholder="按域筛选" style="width:160px">
          <el-option v-for="d in permDomains" :key="d" :label="d" :value="d" />
        </el-select>
      </div>
      <div class="dict-body">
        <TableOrCards :data="filteredPermissions" :columns="permCols" empty-text="暂无权限码，请先同步">
          <el-table :data="filteredPermissions" border height="100%" stripe>
            <el-table-column prop="code" label="权限码" min-width="260" show-overflow-tooltip />
            <el-table-column prop="domain" label="域" width="140" />
            <el-table-column prop="module" label="模块" min-width="160" show-overflow-tooltip />
            <el-table-column prop="action" label="动作" width="100" />
          </el-table>
        </TableOrCards>
      </div>
    </template>

    <template v-else-if="module === '成本隐藏'">
      <h2 class="title">{{ module }}</h2>
      <p class="desc">按角色隐藏成本相关字段；保存后即时对对应用户生效。</p>
      <div class="row">
        <span>角色</span>
        <el-select v-model="selectedRoleId" style="width:260px" placeholder="选择角色">
          <el-option v-for="o in roleOptions" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
        <el-button type="primary" @click="saveFieldPolicies">保存字段策略</el-button>
        <el-button @click="loadAll">刷新</el-button>
      </div>
      <h3>成本字段策略（当前角色）</h3>
      <TableOrCards :data="policyDraft" :columns="policyCols">
        <el-table :data="policyDraft" border>
          <el-table-column prop="field_key" label="字段" min-width="140" />
          <el-table-column prop="field_name" label="名称" min-width="120" />
          <el-table-column label="可见" width="100">
            <template #default="{ row }"><el-switch v-model="row.visible" /></template>
          </el-table-column>
          <el-table-column label="可编辑" width="100">
            <template #default="{ row }"><el-switch v-model="row.editable" /></template>
          </el-table-column>
        </el-table>
        <template #field-visible="{ row }"><el-switch v-model="row.visible" /></template>
        <template #field-editable="{ row }"><el-switch v-model="row.editable" /></template>
      </TableOrCards>
    </template>

    <template v-else-if="module === '自定义菜单'">
      <h2 class="title">{{ module }}</h2>
      <p class="desc">按角色裁剪菜单可见性；保存后即时对对应用户生效。</p>
      <div class="row">
        <span>角色</span>
        <el-select v-model="selectedRoleId" style="width:260px" placeholder="选择角色">
          <el-option v-for="o in roleOptions" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
        <el-button type="primary" @click="saveMenus">保存菜单可见性</el-button>
        <el-button @click="loadAll">刷新</el-button>
      </div>
      <TableOrCards :data="menuDraft" :columns="menuCols">
        <el-table :data="menuDraft" border height="520">
          <el-table-column prop="domain" label="域" width="120" />
          <el-table-column prop="module" label="模块" min-width="160" />
          <el-table-column prop="menu_key" label="菜单键" min-width="180" />
          <el-table-column label="可见" width="90">
            <template #default="{ row }"><el-switch v-model="row.visible" /></template>
          </el-table-column>
          <el-table-column label="排序" width="120">
            <template #default="{ row }"><el-input-number v-model="row.sort_no" :min="0" :step="10" controls-position="right" /></template>
          </el-table-column>
        </el-table>
        <template #field-visible="{ row }"><el-switch v-model="row.visible" /></template>
        <template #field-sort_no="{ row }">
          <el-input-number v-model="row.sort_no" :min="0" :step="10" controls-position="right" />
        </template>
      </TableOrCards>
    </template>

    <template v-else-if="module === '登录控制'">
      <h2 class="title">{{ module }}</h2>
      <p class="desc">登录失败锁定、会话时长与密码策略。</p>
      <el-form label-width="140px" style="max-width:480px">
        <el-form-item label="最大失败次数">
          <el-input-number v-model="loginPolicy.max_fail_count" :min="1" />
        </el-form-item>
        <el-form-item label="锁定分钟">
          <el-input-number v-model="loginPolicy.lock_minutes" :min="1" />
        </el-form-item>
        <el-form-item label="会话 TTL(分)">
          <el-input-number v-model="loginPolicy.session_ttl_min" :min="10" />
        </el-form-item>
        <el-form-item label="密码最短长度">
          <el-input-number v-model="loginPolicy.password_min_len" :min="6" />
        </el-form-item>
        <el-button type="primary" @click="saveLoginPolicy">保存策略</el-button>
      </el-form>
    </template>

    <template v-else-if="module === '账户冻结'">
      <AccountFreezePanel />
    </template>
  </div>
</template>

<style scoped>
.iam { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.iam.fill {
  height: calc(100vh - 120px);
  min-height: 360px;
  display: flex;
  flex-direction: column;
  padding: 12px 16px 8px;
  box-sizing: border-box;
}
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  flex-shrink: 0;
  margin-bottom: 8px;
}
.title { margin: 0 0 4px; font-size: 18px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.page-head .desc { margin: 0; }
.sub { margin: 0; font-size: 14px; font-weight: 600; }
.row { display: flex; gap: 8px; margin-bottom: 12px; align-items: center; flex-wrap: wrap; }
.field-panel {
  flex-shrink: 0;
  margin-bottom: 10px;
}
.field-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}
.dict-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  flex-shrink: 0;
  margin-bottom: 8px;
}
.dict-count { color: #5c6b75; font-size: 13px; margin-right: 4px; }
.dict-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.dict-body :deep(.table-or-cards),
.dict-body :deep(.desktop-table) {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.dict-body :deep(.el-table) {
  flex: 1;
}
@media (max-width: 768px) {
  .iam.fill { height: auto; min-height: 0; }
  .dict-body { min-height: 320px; }
}
</style>
