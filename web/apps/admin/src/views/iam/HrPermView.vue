<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { iamApi } from '@erp/shared'
import DesktopOnlyGate from '../../components/mobile/DesktopOnlyGate.vue'
import TableOrCards from '../../components/mobile/TableOrCards.vue'
import type { MobileCardColumn } from '../../components/mobile/MobileDataCards.vue'

type Row = Record<string, unknown>

const loading = ref(false)

const roles = ref<Row[]>([])
const permissions = ref<Row[]>([])

const selectedRoleId = ref<number | null>(null)
const roleForm = reactive({
  code: '', name: '', data_scope_type: 'self', remark: '', status: 'active', is_system: false,
})
const rolePermIds = ref<number[]>([])
const roleWhIds = ref<number[]>([])
const roleProcMap = reactive<Record<number, { can_report: boolean; can_dispatch: boolean; checked: boolean }>>({})
const roleBoundUsers = ref<Row[]>([])
const warehouses = ref<Row[]>([])
const processes = ref<Row[]>([])
const roleDlg = ref(false)
const newRole = reactive({ code: '', name: '', data_scope_type: 'self', remark: '' })
const permDomainFilter = ref('')
const permKeyword = ref('')
const permCollapse = ref<string[]>([])

type PermMod = { module: string; items: Row[] }
type PermDomainNode = { domain: string; modules: PermMod[]; ids: number[] }

const visiblePermissions = computed(() =>
  permissions.value.filter((p) => String(p.domain || '') !== '系统管理'),
)

const permDomains = computed(() => {
  const set = new Set<string>()
  visiblePermissions.value.forEach((p) => { if (p.domain) set.add(String(p.domain)) })
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})

const filteredPerms = computed(() => {
  const kw = permKeyword.value.trim().toLowerCase()
  return visiblePermissions.value.filter((p) => {
    if (permDomainFilter.value && String(p.domain) !== permDomainFilter.value) return false
    if (!kw) return true
    const hay = `${p.domain || ''} ${p.module || ''} ${p.action || ''} ${p.code || ''} ${p.name || ''}`.toLowerCase()
    return hay.includes(kw)
  })
})

const permTree = computed((): PermDomainNode[] => {
  const byDomain = new Map<string, Map<string, Row[]>>()
  for (const p of filteredPerms.value) {
    const d = String(p.domain || '未分类')
    const m = String(p.module || '未分类')
    if (!byDomain.has(d)) byDomain.set(d, new Map())
    const mods = byDomain.get(d)!
    if (!mods.has(m)) mods.set(m, [])
    mods.get(m)!.push(p)
  }
  const domains = [...byDomain.keys()].sort((a, b) => a.localeCompare(b, 'zh-CN'))
  return domains.map((domain) => {
    const mods = byDomain.get(domain)!
    const moduleNames = [...mods.keys()].sort((a, b) => a.localeCompare(b, 'zh-CN'))
    const modules = moduleNames.map((module) => {
      const items = [...mods.get(module)!].sort((a, b) =>
        String(a.action || '').localeCompare(String(b.action || ''), 'zh-CN'),
      )
      return { module, items }
    })
    const ids = modules.flatMap((m) => m.items.map((p) => Number(p.id)))
    return { domain, modules, ids }
  })
})

const rolePermIdSet = computed(() => new Set(rolePermIds.value))
const visiblePermIdSet = computed(() => new Set(visiblePermissions.value.map((p) => Number(p.id)).filter((id) => id > 0)))

const selectedPermCount = computed(() => rolePermIds.value.filter((id) => visiblePermIdSet.value.has(id)).length)

function permIdsOf(items: Row[]) {
  return items.map((p) => Number(p.id)).filter((id) => id > 0)
}

function countSelected(ids: number[]) {
  const set = rolePermIdSet.value
  return ids.reduce((n, id) => (set.has(id) ? n + 1 : n), 0)
}

function groupCheckState(ids: number[]): { checked: boolean; indeterminate: boolean } {
  if (!ids.length) return { checked: false, indeterminate: false }
  const n = countSelected(ids)
  return { checked: n === ids.length, indeterminate: n > 0 && n < ids.length }
}

/** 仅增删指定 id，保留过滤范围外的已选项 */
function setPermIdsChecked(ids: number[], checked: boolean) {
  if (!ids.length) return
  const set = new Set(rolePermIds.value)
  if (checked) ids.forEach((id) => set.add(id))
  else ids.forEach((id) => set.delete(id))
  rolePermIds.value = [...set]
}

function toggleDomain(domain: string, checked: boolean) {
  const node = permTree.value.find((d) => d.domain === domain)
  if (!node) return
  setPermIdsChecked(node.ids, checked)
}

function toggleModule(domain: string, module: string, checked: boolean) {
  const node = permTree.value.find((d) => d.domain === domain)
  const mod = node?.modules.find((m) => m.module === module)
  if (!mod) return
  setPermIdsChecked(permIdsOf(mod.items), checked)
}

function toggleOnePerm(id: number, checked: boolean) {
  setPermIdsChecked([id], checked)
}

watch(permTree, (tree) => {
  if (!tree.length) {
    permCollapse.value = []
    return
  }
  // 搜索或筛选后默认展开匹配域；无筛选时展开前 3 个域
  const keys = tree.map((d) => d.domain)
  if (permKeyword.value.trim() || permDomainFilter.value) {
    permCollapse.value = keys
  } else if (!permCollapse.value.length) {
    permCollapse.value = keys.slice(0, 3)
  }
}, { immediate: true })

function listOf(env: { code?: number; data?: unknown } | undefined): Row[] {
  if (!env || env.code !== 1) return []
  const d = env.data
  if (Array.isArray(d)) return d as Row[]
  if (d && typeof d === 'object' && Array.isArray((d as { list?: unknown }).list)) {
    return (d as { list: Row[] }).list
  }
  return []
}

async function loadCore() {
  loading.value = true
  try {
    const results = await Promise.allSettled([
      iamApi.roles(),
      iamApi.permissions(),
    ])
    const val = <T,>(i: number): T | undefined => {
      const r = results[i]
      return r.status === 'fulfilled' ? (r.value as T) : undefined
    }
    roles.value = listOf(val(0))
    permissions.value = listOf(val(1))
    const permRes = val<{ code?: number; msg?: string }>(1)
    if (permRes && permRes.code !== 1) {
      ElMessage.warning(`权限码加载失败：${permRes.msg || '未知错误'}`)
    }
    if (!selectedRoleId.value && roles.value.length) {
      selectedRoleId.value = Number(roles.value[0].id)
    }
  } finally {
    loading.value = false
  }
}

watch(selectedRoleId, (rid) => {
  if (rid) void loadRoleDetail(rid)
})

async function loadRoleDetail(rid: number) {
  const res = await iamApi.getRole(rid)
  if (res.code !== 1) return ElMessage.error(res.msg)
  const data = res.data as Row
  const role = (data.role as Row) || {}
  Object.assign(roleForm, {
    code: String(role.code || ''),
    name: String(role.name || ''),
    data_scope_type: String(role.data_scope_type || 'self'),
    remark: String(role.remark || ''),
    status: String(role.status || 'active'),
    is_system: !!role.is_system,
  })
  rolePermIds.value = ((data.permission_ids as number[]) || []).map(Number)
  roleWhIds.value = ((data.warehouse_ids as number[]) || []).map(Number)
  roleBoundUsers.value = (data.bound_users as Row[]) || []
  warehouses.value = (data.warehouses as Row[]) || []
  processes.value = (data.processes as Row[]) || []
  const scopes = (data.process_scopes as Row[]) || []
  const map: Record<number, { can_report: boolean; can_dispatch: boolean; checked: boolean }> = {}
  processes.value.forEach((p) => {
    const id = Number(p.id)
    const hit = scopes.find((x) => Number(x.process_id) === id)
    map[id] = {
      checked: !!hit,
      can_report: hit ? !!hit.can_report : true,
      can_dispatch: hit ? !!hit.can_dispatch : false,
    }
  })
  Object.keys(roleProcMap).forEach((k) => delete roleProcMap[Number(k)])
  Object.assign(roleProcMap, map)
}

function openCreateRole() {
  Object.assign(newRole, { code: '', name: '', data_scope_type: 'self', remark: '' })
  roleDlg.value = true
}

async function submitCreateRole() {
  if (!newRole.code || !newRole.name) return ElMessage.warning('请填写编码与名称')
  const res = await iamApi.createRole({ ...newRole })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('角色已创建')
  roleDlg.value = false
  await loadCore()
  selectedRoleId.value = Number((res.data as Row)?.id)
}

async function saveRoleBasic() {
  if (!selectedRoleId.value) return
  if (roleForm.is_system) {
    return ElMessage.warning('系统预置角色只读，请新建自定义角色后再调整。')
  }
  const res = await iamApi.updateRole(selectedRoleId.value, {
    name: roleForm.name,
    code: roleForm.is_system ? undefined : roleForm.code,
    data_scope_type: roleForm.data_scope_type,
    remark: roleForm.remark,
    status: roleForm.status,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('角色信息已保存')
  await loadCore()
}

async function saveRolePerms() {
  if (!selectedRoleId.value) return
  if (roleForm.is_system) {
    return ElMessage.warning('系统预置角色只读，不可调整权限码。')
  }
  const permissionIds = rolePermIds.value.filter((id) => visiblePermIdSet.value.has(id))
  const res = await iamApi.setPermissions(selectedRoleId.value, { permission_ids: permissionIds })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('权限码已保存')
  rolePermIds.value = permissionIds
}

async function saveRoleScopes() {
  if (!selectedRoleId.value) return
  if (roleForm.is_system) {
    return ElMessage.warning('系统预置角色只读，不可调整仓/工序范围。')
  }
  const wh = await iamApi.setWarehouseScope(selectedRoleId.value, roleWhIds.value)
  if (wh.code !== 1) return ElMessage.error(wh.msg)
  const items = Object.entries(roleProcMap)
    .filter(([, v]) => v.checked)
    .map(([pid, v]) => ({
      process_id: Number(pid),
      can_report: v.can_report,
      can_dispatch: v.can_dispatch,
    }))
  const ps = await iamApi.setProcessScope(selectedRoleId.value, items)
  if (ps.code !== 1) return ElMessage.error(ps.msg)
  ElMessage.success('仓/工序范围已保存')
}

const roleUserCardCols: MobileCardColumn[] = [
  { prop: 'login_name', label: '登录名', primary: true },
  { prop: 'name', label: '姓名' },
  { prop: 'status', label: '状态' },
]

onMounted(loadCore)
</script>

<template>
  <div v-loading="loading" class="hr-perm">
    <h2 class="title">角色管理</h2>
    <p class="desc">维护角色模板（权限码、仓/工序、数据范围）；给人赋权请到员工档案 / 公司架构。系统管理域权限请到系统管理处理。</p>

    <div class="role-layout">
          <aside class="role-side">
            <div class="row">
              <strong>角色列表</strong>
              <el-button type="primary" size="small" @click="openCreateRole">新建</el-button>
            </div>
            <div
              v-for="r in roles"
              :key="String(r.id)"
              class="role-item"
              :class="{ active: selectedRoleId === Number(r.id) }"
              @click="selectedRoleId = Number(r.id)"
            >
              <div>{{ r.name }}</div>
              <small>{{ r.code }} · {{ r.data_scope_type }}</small>
            </div>
          </aside>
          <section v-if="selectedRoleId" class="role-main">
            <div class="row">
              <strong>{{ roleForm.name }}</strong>
              <el-tag v-if="roleForm.is_system" size="small">系统预置</el-tag>
              <el-tag size="small" type="info">{{ roleForm.code }}</el-tag>
              <span class="spacer" />
              <el-button type="primary" :disabled="roleForm.is_system" @click="saveRoleBasic">保存基本信息</el-button>
            </div>
            <el-alert
              v-if="roleForm.is_system"
              type="info"
              :closable="false"
              show-icon
              title="系统预置角色仅供查看；权限码、基础属性、仓范围与工序范围均不可在此修改。"
              style="margin-bottom:12px"
            />
            <el-form label-width="100px" inline class="role-basic">
              <el-form-item label="编码">
                <el-input v-model="roleForm.code" :disabled="roleForm.is_system" style="width:140px" />
              </el-form-item>
              <el-form-item label="名称">
                <el-input v-model="roleForm.name" :disabled="roleForm.is_system" style="width:140px" />
              </el-form-item>
              <el-form-item label="数据范围">
                <el-select v-model="roleForm.data_scope_type" :disabled="roleForm.is_system" style="width:140px">
                  <el-option label="本人 self" value="self" />
                  <el-option label="班组 team" value="team" />
                  <el-option label="车间 dept_workshop" value="dept_workshop" />
                  <el-option label="仓库 warehouse" value="warehouse" />
                  <el-option label="全部 all" value="all" />
                </el-select>
              </el-form-item>
              <el-form-item label="状态">
                <el-select v-model="roleForm.status" :disabled="roleForm.is_system" style="width:110px">
                  <el-option label="启用" value="active" />
                  <el-option label="停用" value="inactive" />
                </el-select>
              </el-form-item>
              <el-form-item label="备注">
                <el-input v-model="roleForm.remark" :disabled="roleForm.is_system" style="width:200px" />
              </el-form-item>
            </el-form>

            <DesktopOnlyGate message="仓/工序范围与权限码矩阵需在桌面浏览器操作。">
            <el-row :gutter="16">
              <el-col :span="12" :xs="24">
                <h3 class="sub">仓范围</h3>
                <el-checkbox-group v-model="roleWhIds">
                  <el-checkbox v-for="w in warehouses" :key="String(w.id)" :value="Number(w.id)">
                    {{ w.name }} ({{ w.code }})
                  </el-checkbox>
                </el-checkbox-group>
              </el-col>
              <el-col :span="12" :xs="24">
                <h3 class="sub">工序范围</h3>
                <div v-for="p in processes" :key="String(p.id)" class="proc-row">
                  <template v-if="roleProcMap[Number(p.id)]">
                    <el-checkbox v-model="roleProcMap[Number(p.id)].checked" :disabled="roleForm.is_system">{{ p.name }}</el-checkbox>
                    <el-checkbox v-model="roleProcMap[Number(p.id)].can_report" :disabled="roleForm.is_system || !roleProcMap[Number(p.id)].checked">可报工</el-checkbox>
                    <el-checkbox v-model="roleProcMap[Number(p.id)].can_dispatch" :disabled="roleForm.is_system || !roleProcMap[Number(p.id)].checked">可派工</el-checkbox>
                  </template>
                </div>
              </el-col>
            </el-row>
            <el-button type="primary" :disabled="roleForm.is_system" style="margin-top:10px" @click="saveRoleScopes">保存仓/工序范围</el-button>

            <section class="perm-panel">
              <div class="perm-panel-head">
                <h3 class="sub" style="margin:0">
                  权限码授权
                  <span class="muted">
                    （已选 {{ selectedPermCount }} · 展示 {{ filteredPerms.length }} / 全部 {{ permissions.length }}）
                  </span>
                </h3>
                <div class="row" style="margin:0;gap:8px;flex-wrap:wrap">
                  <el-input
                    v-model="permKeyword"
                    clearable
                    placeholder="搜索域/模块/动作/权限码"
                    style="width:220px"
                  />
                  <el-select v-model="permDomainFilter" clearable placeholder="按域筛选" style="width:160px">
                    <el-option v-for="d in permDomains" :key="d" :label="d" :value="d" />
                  </el-select>
                  <el-button type="primary" :disabled="!filteredPerms.length || roleForm.is_system" @click="saveRolePerms">保存权限码</el-button>
                </div>
              </div>

              <el-empty v-if="!filteredPerms.length" description="暂无可分配权限码，请调整筛选或到系统管理同步权限字典" :image-size="64" />
              <div v-else class="perm-tree-wrap">
                <el-collapse v-model="permCollapse">
                  <el-collapse-item v-for="dom in permTree" :key="dom.domain" :name="dom.domain">
                    <template #title>
                      <div class="perm-domain-title" @click.stop>
                        <el-checkbox
                          :model-value="groupCheckState(dom.ids).checked"
                          :indeterminate="groupCheckState(dom.ids).indeterminate"
                          :disabled="roleForm.is_system"
                          @change="(v: boolean | string | number) => toggleDomain(dom.domain, !!v)"
                        />
                        <span class="perm-domain-name">{{ dom.domain }}</span>
                        <span class="muted">{{ countSelected(dom.ids) }}/{{ dom.ids.length }}</span>
                      </div>
                    </template>
                    <div
                      v-for="mod in dom.modules"
                      :key="`${dom.domain}-${mod.module}`"
                      class="perm-mod-row"
                    >
                      <div class="perm-mod-label">
                        <el-checkbox
                          :model-value="groupCheckState(permIdsOf(mod.items)).checked"
                          :indeterminate="groupCheckState(permIdsOf(mod.items)).indeterminate"
                          :disabled="roleForm.is_system"
                          @change="(v: boolean | string | number) => toggleModule(dom.domain, mod.module, !!v)"
                        />
                        <span :title="mod.module">{{ mod.module }}</span>
                        <span class="muted">{{ countSelected(permIdsOf(mod.items)) }}/{{ mod.items.length }}</span>
                      </div>
                      <div class="perm-actions">
                        <el-checkbox
                          v-for="p in mod.items"
                          :key="String(p.id)"
                          :model-value="rolePermIdSet.has(Number(p.id))"
                          :title="String(p.code || '')"
                          :disabled="roleForm.is_system"
                          @change="(v: boolean | string | number) => toggleOnePerm(Number(p.id), !!v)"
                        >
                          {{ p.action || p.name || '权限' }}
                        </el-checkbox>
                      </div>
                    </div>
                  </el-collapse-item>
                </el-collapse>
              </div>
            </section>
            </DesktopOnlyGate>

            <h3 class="sub">已绑定用户</h3>
            <TableOrCards :data="roleBoundUsers" :columns="roleUserCardCols">
            <el-table :data="roleBoundUsers" border size="small" max-height="180">
              <el-table-column prop="login_name" label="登录名" />
              <el-table-column prop="name" label="姓名" />
              <el-table-column prop="status" label="状态" width="90" />
            </el-table>
            <template #extra="{ row }">
              <span class="muted">{{ row.status }}</span>
            </template>
            </TableOrCards>
          </section>
          <el-empty v-else description="请选择角色" />
        </div>

    <el-dialog v-model="roleDlg" title="新建角色" width="440px">
      <el-form label-width="100px">
        <el-form-item label="编码" required>
          <el-input v-model="newRole.code" placeholder="如 line_worker" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="newRole.name" />
        </el-form-item>
        <el-form-item label="数据范围">
          <el-select v-model="newRole.data_scope_type" style="width:100%">
            <el-option label="本人 self" value="self" />
            <el-option label="班组 team" value="team" />
            <el-option label="车间 dept_workshop" value="dept_workshop" />
            <el-option label="仓库 warehouse" value="warehouse" />
            <el-option label="全部 all" value="all" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="newRole.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleDlg = false">取消</el-button>
        <el-button type="primary" @click="submitCreateRole">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hr-perm { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 12px; }
.spacer { flex: 1; }
.sub { margin: 16px 0 8px; font-size: 15px; }
.muted { color: #5c6b75; font-size: 12px; }
.role-layout { display: grid; grid-template-columns: 220px 1fr; gap: 16px; min-height: 420px; }
.role-side { border: 1px solid #e2e8ec; border-radius: 8px; padding: 10px; overflow: auto; max-height: 640px; }
.role-item {
  padding: 8px 10px; border-radius: 6px; cursor: pointer; margin-bottom: 4px;
}
.role-item:hover { background: #f4f7f9; }
.role-item.active { background: #e8f3f0; border: 1px solid #3d8b7a; }
.role-item small { display: block; color: #5c6b75; font-size: 12px; margin-top: 2px; }
.role-main { border: 1px solid #e2e8ec; border-radius: 8px; padding: 12px 16px; }
.proc-row { display: flex; gap: 12px; align-items: center; margin-bottom: 4px; flex-wrap: wrap; }
.perm-panel {
  margin-top: 16px;
  border: 1px solid #e2e8ec;
  border-radius: 8px;
  padding: 12px 14px;
  background: #fafcfc;
}
.perm-panel-head {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.perm-tree-wrap {
  max-height: 460px;
  overflow: auto;
  background: #fff;
  border: 1px solid #e8eef1;
  border-radius: 6px;
  padding: 4px 8px 10px;
}
.perm-domain-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}
.perm-domain-name { color: #1a2b33; }
.perm-mod-row {
  display: grid;
  grid-template-columns: minmax(140px, 220px) 1fr;
  gap: 8px 16px;
  align-items: start;
  padding: 8px 4px;
  border-bottom: 1px dashed #e8eef1;
}
.perm-mod-row:last-child { border-bottom: none; }
.perm-mod-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #2c3e48;
  min-width: 0;
}
.perm-mod-label > span:nth-child(2) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.perm-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  align-items: center;
}
.perm-actions :deep(.el-checkbox) { margin-right: 0; height: 28px; }
.perm-actions :deep(.el-checkbox__label) { font-size: 13px; padding-left: 6px; }
.perm-tree-wrap :deep(.el-collapse-item__header) { height: auto; min-height: 44px; line-height: 1.4; }
.perm-tree-wrap :deep(.el-collapse-item__content) { padding-bottom: 8px; }
@media (max-width: 900px) {
  .role-layout { grid-template-columns: 1fr; }
  .perm-mod-row { grid-template-columns: 1fr; }
}
@media (max-width: 768px) {
  .hr-perm { padding: 12px; }
}
</style>
