<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { iamApi, ERP_MENUS } from '@erp/shared'
import HrPermView from './HrPermView.vue'
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

const userCols: MobileCardColumn[] = [
  { prop: 'login_name', label: '登录名', primary: true },
  { prop: 'name', label: '姓名' },
  { prop: 'status', label: '状态' },
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

const COST_FIELDS = [
  { field_key: 'cost_price', field_name: '成本价' },
  { field_key: 'cost', field_name: '成本' },
  { field_key: 'gross_margin', field_name: '毛利率' },
  { field_key: 'gross_profit', field_name: '毛利' },
]

async function loadAll() {
  if (props.module === '权限分配') return
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

async function freeze(id: number, frozen: boolean) {
  const res = frozen ? await iamApi.unfreeze(id) : await iamApi.freeze(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(frozen ? '已解冻' : '已冻结')
  await loadAll()
}

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

const roleOptions = computed(() =>
  roles.value.map((r) => ({ label: `${r.code || ''} ${r.name || ''}`.trim(), value: Number(r.id) })),
)

const permissionPreview = computed(() => permissions.value.slice(0, 80))

onMounted(loadAll)
watch(() => props.module, loadAll)
</script>

<template>
  <HrPermView v-if="module === '权限分配'" />
  <div v-else v-loading="loading" class="iam">
    <h2 class="title">{{ module }}</h2>
    <p class="desc">身份与权限配置；角色裁剪菜单/字段后即时对对应用户生效。</p>

    <template v-if="module === '自定义权限' || module === '成本隐藏'">
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
      <h3 style="margin-top:16px">权限码字典（只读）</h3>
      <TableOrCards :data="permissionPreview" :columns="permCols">
        <el-table :data="permissionPreview" border height="280">
          <el-table-column prop="code" label="权限码" min-width="220" />
          <el-table-column prop="domain" label="域" />
          <el-table-column prop="module" label="模块" />
          <el-table-column prop="action" label="动作" />
        </el-table>
      </TableOrCards>
    </template>

    <template v-else-if="module === '自定义菜单'">
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
      <TableOrCards :data="users" :columns="userCols">
        <el-table :data="users" border>
          <el-table-column prop="login_name" label="登录名" />
          <el-table-column prop="name" label="姓名" />
          <el-table-column prop="status" label="状态" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button
                link
                :type="row.status === 'frozen' ? 'success' : 'danger'"
                @click="freeze(Number(row.id), row.status === 'frozen')"
              >{{ row.status === 'frozen' ? '解冻' : '冻结' }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <template #actions="{ row }">
          <el-button
            link
            :type="row.status === 'frozen' ? 'success' : 'danger'"
            @click="freeze(Number(row.id), row.status === 'frozen')"
          >{{ row.status === 'frozen' ? '解冻' : '冻结' }}</el-button>
        </template>
      </TableOrCards>
    </template>
  </div>
</template>

<style scoped>
.iam { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; align-items: center; flex-wrap: wrap; }
</style>
