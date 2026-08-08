<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { empTypeLabel, hrApi, iamApi } from '@erp/shared'
import { WorkshopSelect } from '../../components/select'

type Row = Record<string, unknown>

const tab = ref('overview')
const loading = ref(false)

const overview = ref<Row>({})
const users = ref<Row[]>([])
const roles = ref<Row[]>([])
const employees = ref<Row[]>([])
const sessions = ref<Row[]>([])
const onboards = ref<Row[]>([])
const offboards = ref<Row[]>([])
const groups = ref<Row[]>([])
const permissions = ref<Row[]>([])

const selectedUser = ref<number | null>(null)
const selectedRoles = ref<number[]>([])
const dataScope = reactive({ data_scope_type: 'self', workshop_id: 0, team_id: 0 })
const bindEmpId = ref<number | null>(null)

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

const permDomains = computed(() => {
  const set = new Set<string>()
  permissions.value.forEach((p) => { if (p.domain) set.add(String(p.domain)) })
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})

const filteredPerms = computed(() => {
  const kw = permKeyword.value.trim().toLowerCase()
  return permissions.value.filter((p) => {
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

const selectedPermCount = computed(() => rolePermIds.value.length)

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

const openDlg = ref(false)
const openForm = reactive({ employee_id: 0, login_name: '', password: 'ChangeMe123', role_ids: [] as number[] })

const onboardForm = reactive({ employee_id: null as number | null, remark: '', role_ids: [] as number[] })
const offboardForm = reactive({ employee_id: null as number | null, reason: '', revoke_permission: true })

const empFilter = ref<'all' | 'bound' | 'unbound'>('all')

const filteredEmployees = computed(() => {
  const list = employees.value
  if (empFilter.value === 'bound') return list.filter((e) => Number(e.user_id) > 0)
  if (empFilter.value === 'unbound') return list.filter((e) => !Number(e.user_id))
  return list
})

const unboundEmployees = computed(() => employees.value.filter((e) => !Number(e.user_id) && e.status === 'active'))

function empLabel(e: Row) {
  return `${e.emp_no || e.id} · ${e.name || ''}`
}

async function loadOverview() {
  const res = await iamApi.hrPermOverview()
  if (res.code === 1) overview.value = (res.data as Row) || {}
}

async function loadCore() {
  loading.value = true
  try {
    const results = await Promise.allSettled([
      iamApi.users(),
      iamApi.roles(),
      hrApi.employees(),
      iamApi.sessions(),
      iamApi.groups(),
      iamApi.hrPermOverview(),
      hrApi.onboards(),
      hrApi.offboards(),
      iamApi.permissions(),
    ])
    const val = <T,>(i: number): T | undefined => {
      const r = results[i]
      return r.status === 'fulfilled' ? (r.value as T) : undefined
    }
    const listOf = (env: { code?: number; data?: unknown } | undefined): Row[] => {
      if (!env || env.code !== 1) return []
      const d = env.data
      if (Array.isArray(d)) return d as Row[]
      if (d && typeof d === 'object' && Array.isArray((d as { list?: unknown }).list)) {
        return (d as { list: Row[] }).list
      }
      return []
    }
    users.value = listOf(val(0))
    roles.value = listOf(val(1))
    employees.value = listOf(val(2))
    sessions.value = listOf(val(3))
    groups.value = listOf(val(4))
    const ov = val<{ code?: number; data?: Row }>(5)
    overview.value = (ov?.code === 1 ? ov.data : undefined) || {}
    onboards.value = listOf(val(6))
    offboards.value = listOf(val(7))
    permissions.value = listOf(val(8))
    const permRes = val<{ code?: number; msg?: string }>(8)
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

async function syncPermissions() {
  const res = await iamApi.syncPermissions()
  if (res.code !== 1) return ElMessage.error(res.msg || '同步失败')
  ElMessage.success(`权限码已同步，共 ${(res.data as { total?: number })?.total ?? 0} 条`)
  await loadCore()
}

async function loadUserAuth(uid: number) {
  const res = await iamApi.getUser(uid)
  if (res.code !== 1) return ElMessage.error(res.msg)
  const data = res.data as Row
  const rs = (data.roles as Row[]) || []
  selectedRoles.value = rs.map((x) => Number(x.id))
  bindEmpId.value = Number(data.employee_id) || null
  const scope = (data.data_scope as Row) || {}
  dataScope.data_scope_type = String(scope.data_scope_type || 'self')
  dataScope.workshop_id = Number(scope.workshop_id) || 0
  dataScope.team_id = Number(scope.team_id) || 0
}

watch(selectedUser, (uid) => {
  if (uid) void loadUserAuth(uid)
})

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
  const res = await iamApi.setPermissions(selectedRoleId.value, { permission_ids: rolePermIds.value })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('权限码已保存')
}

async function saveRoleScopes() {
  if (!selectedRoleId.value) return
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

function goAuthorize(uid: number) {
  selectedUser.value = uid
  tab.value = 'authorize'
}

function goRole(rid: number) {
  selectedRoleId.value = rid
  tab.value = 'roles'
}

function openAccountDialog(emp?: Row) {
  openForm.employee_id = emp ? Number(emp.id) : 0
  openForm.login_name = emp ? String(emp.emp_no || '') : ''
  openForm.password = 'ChangeMe123'
  openForm.role_ids = []
  openDlg.value = true
}

async function submitOpenAccount() {
  if (!openForm.employee_id) return ElMessage.warning('请选择员工')
  const body: Record<string, unknown> = {
    login_name: openForm.login_name || undefined,
    password: openForm.password || undefined,
  }
  if (openForm.role_ids.length) body.role_ids = openForm.role_ids
  const res = await hrApi.openAccount(openForm.employee_id, body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已开户：${(res.data as Row)?.login_name || ''}`)
  openDlg.value = false
  await loadCore()
}

async function bindEmployee() {
  if (!selectedUser.value || !bindEmpId.value) return ElMessage.warning('请选择用户与员工')
  const res = await iamApi.bindEmployee(selectedUser.value, bindEmpId.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已绑定员工')
  await loadCore()
  await loadUserAuth(selectedUser.value)
}

async function unbindEmployee() {
  if (!selectedUser.value) return
  await ElMessageBox.confirm('确认解除用户与员工绑定？', '提示')
  const res = await iamApi.unbindEmployee(selectedUser.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已解绑')
  bindEmpId.value = null
  await loadCore()
}

async function saveRoles() {
  if (!selectedUser.value) return
  const res = await iamApi.setRoles(selectedUser.value, selectedRoles.value)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('角色已保存')
}

async function saveScope() {
  if (!selectedUser.value) return
  const res = await iamApi.setDataScope(selectedUser.value, { ...dataScope })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('数据范围已保存')
}

async function freeze(id: number, frozen: boolean) {
  const res = frozen ? await iamApi.unfreeze(id) : await iamApi.freeze(id, { reason: '人事工作台冻结' })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(frozen ? '已解冻' : '已冻结并踢下线')
  await loadCore()
}

async function revokeSession(id: number) {
  const res = await iamApi.revokeSession(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('会话已撤销')
  await loadCore()
}

async function createOnboard() {
  if (!onboardForm.employee_id) return ElMessage.warning('请选择员工')
  const res = await hrApi.createOnboard({
    employee_id: onboardForm.employee_id,
    remark: onboardForm.remark,
    role_ids: onboardForm.role_ids,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('入职单已创建')
  onboardForm.employee_id = null
  onboardForm.remark = ''
  onboardForm.role_ids = []
  await loadCore()
}

async function confirmOnboard(id: number) {
  const res = await hrApi.confirmOnboard(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('入职确认：已开户赋权')
  await loadCore()
}

async function createOffboard() {
  if (!offboardForm.employee_id) return ElMessage.warning('请选择员工')
  const res = await hrApi.createOffboard({
    employee_id: offboardForm.employee_id,
    reason: offboardForm.reason,
    revoke_permission: offboardForm.revoke_permission,
  })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('离职单已创建')
  offboardForm.employee_id = null
  offboardForm.reason = ''
  await loadCore()
}

async function confirmOffboard(id: number) {
  await ElMessageBox.confirm('确认离职？将冻结账号、清角色并踢下线', '离职收回')
  const res = await hrApi.confirmOffboard(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('离职确认：权限已收回')
  await loadCore()
}

onMounted(loadCore)
</script>

<template>
  <div v-loading="loading" class="hr-perm">
    <h2 class="title">权限分配工作台</h2>
    <p class="desc">
      人事闭环：角色管理 → 员工开户 → 绑定/用户授权 → 冻结与会话 → 入职赋权 / 离职收回。
      系统侧「自定义权限 / 菜单 / 登录控制」仍走系统管理模块。
    </p>

    <el-tabs v-model="tab">
      <el-tab-pane label="总览" name="overview">
        <div class="stats">
          <div class="stat" @click="tab = 'accounts'"><div class="label">用户</div><div class="value">{{ overview.users ?? '—' }}</div></div>
          <div class="stat" @click="tab = 'accounts'"><div class="label">已绑定</div><div class="value">{{ overview.bound_users ?? '—' }}</div></div>
          <div class="stat warn" @click="empFilter = 'unbound'; tab = 'accounts'"><div class="label">未开户员工</div><div class="value">{{ overview.unbound_employees ?? '—' }}</div></div>
          <div class="stat" @click="tab = 'roles'"><div class="label">角色</div><div class="value">{{ overview.roles ?? '—' }}</div></div>
          <div class="stat danger" @click="tab = 'freeze'"><div class="label">冻结</div><div class="value">{{ overview.frozen_users ?? '—' }}</div></div>
          <div class="stat" @click="tab = 'freeze'"><div class="label">会话</div><div class="value">{{ overview.active_sessions ?? '—' }}</div></div>
        </div>
        <el-alert type="info" :closable="false" show-icon
          title="闭环：自定义权限/菜单 → 权限分配（本页）→ 登录控制/冻结 → 操作日志 → 离职收回" />
        <ul class="steps">
          <li>角色 {{ roles.length }} · 分组 {{ groups.length }}</li>
          <li>员工 {{ employees.length }} · 入职单 {{ onboards.length }} · 离职单 {{ offboards.length }}</li>
        </ul>
        <el-button @click="loadCore(); loadOverview()">刷新总览</el-button>
      </el-tab-pane>

      <el-tab-pane label="员工·账户" name="accounts">
        <div class="row">
          <el-radio-group v-model="empFilter" size="small">
            <el-radio-button label="all">全部</el-radio-button>
            <el-radio-button label="unbound">未开户</el-radio-button>
            <el-radio-button label="bound">已开户</el-radio-button>
          </el-radio-group>
          <el-button type="primary" @click="openAccountDialog()">为员工开户</el-button>
          <el-button @click="loadCore">刷新</el-button>
        </div>
        <el-table :data="filteredEmployees" border stripe height="360">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="emp_no" label="工号" width="110" />
          <el-table-column prop="name" label="姓名" />
          <el-table-column label="类型" width="90">
            <template #default="{ row }">{{ empTypeLabel(row.emp_type) }}</template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="user_id" label="账号ID" width="90" />
          <el-table-column label="开户" width="90">
            <template #default="{ row }">
              <el-tag :type="row.has_account || row.user_id ? 'success' : 'info'" size="small">
                {{ row.has_account || row.user_id ? '已开' : '未开' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button v-if="!row.user_id" link type="primary" @click="openAccountDialog(row)">开户</el-button>
              <el-button v-if="row.user_id" link type="primary" @click="goAuthorize(Number(row.user_id))">授权</el-button>
            </template>
          </el-table-column>
        </el-table>

        <h3 class="sub">用户列表</h3>
        <el-table :data="users" border stripe height="280">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="login_name" label="登录名" />
          <el-table-column prop="name" label="员工名" />
          <el-table-column prop="employee_id" label="员工ID" width="90" />
          <el-table-column prop="user_type" label="类型" width="90" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button link type="primary" @click="goAuthorize(Number(row.id))">授权</el-button>
              <el-button link :type="row.status === 'frozen' ? 'success' : 'danger'"
                @click="freeze(Number(row.id), row.status === 'frozen')">
                {{ row.status === 'frozen' ? '解冻' : '冻结' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="角色管理" name="roles">
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
              <el-button type="primary" @click="saveRoleBasic">保存基本信息</el-button>
            </div>
            <el-form label-width="100px" inline class="role-basic">
              <el-form-item label="编码">
                <el-input v-model="roleForm.code" :disabled="roleForm.is_system" style="width:140px" />
              </el-form-item>
              <el-form-item label="名称">
                <el-input v-model="roleForm.name" style="width:140px" />
              </el-form-item>
              <el-form-item label="数据范围">
                <el-select v-model="roleForm.data_scope_type" style="width:140px">
                  <el-option label="本人 self" value="self" />
                  <el-option label="班组 team" value="team" />
                  <el-option label="车间 workshop" value="workshop" />
                  <el-option label="仓库 warehouse" value="warehouse" />
                  <el-option label="全部 all" value="all" />
                </el-select>
              </el-form-item>
              <el-form-item label="状态">
                <el-select v-model="roleForm.status" style="width:110px">
                  <el-option label="启用" value="active" />
                  <el-option label="停用" value="inactive" />
                </el-select>
              </el-form-item>
              <el-form-item label="备注">
                <el-input v-model="roleForm.remark" style="width:200px" />
              </el-form-item>
            </el-form>

            <el-row :gutter="16">
              <el-col :span="12">
                <h3 class="sub">仓范围</h3>
                <el-checkbox-group v-model="roleWhIds">
                  <el-checkbox v-for="w in warehouses" :key="String(w.id)" :value="Number(w.id)">
                    {{ w.name }} ({{ w.code }})
                  </el-checkbox>
                </el-checkbox-group>
              </el-col>
              <el-col :span="12">
                <h3 class="sub">工序范围</h3>
                <div v-for="p in processes" :key="String(p.id)" class="proc-row">
                  <template v-if="roleProcMap[Number(p.id)]">
                    <el-checkbox v-model="roleProcMap[Number(p.id)].checked">{{ p.name }}</el-checkbox>
                    <el-checkbox v-model="roleProcMap[Number(p.id)].can_report" :disabled="!roleProcMap[Number(p.id)].checked">可报工</el-checkbox>
                    <el-checkbox v-model="roleProcMap[Number(p.id)].can_dispatch" :disabled="!roleProcMap[Number(p.id)].checked">可派工</el-checkbox>
                  </template>
                </div>
              </el-col>
            </el-row>
            <el-button type="primary" style="margin-top:10px" @click="saveRoleScopes">保存仓/工序范围</el-button>

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
                  <el-button @click="syncPermissions">同步权限码</el-button>
                  <el-button type="primary" :disabled="!permissions.length" @click="saveRolePerms">保存权限码</el-button>
                </div>
              </div>

              <el-empty v-if="!filteredPerms.length" description="暂无权限码，请先点「同步权限码」或调整筛选" :image-size="64" />
              <div v-else class="perm-tree-wrap">
                <el-collapse v-model="permCollapse">
                  <el-collapse-item v-for="dom in permTree" :key="dom.domain" :name="dom.domain">
                    <template #title>
                      <div class="perm-domain-title" @click.stop>
                        <el-checkbox
                          :model-value="groupCheckState(dom.ids).checked"
                          :indeterminate="groupCheckState(dom.ids).indeterminate"
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

            <h3 class="sub">已绑定用户</h3>
            <el-table :data="roleBoundUsers" border size="small" max-height="180">
              <el-table-column prop="login_name" label="登录名" />
              <el-table-column prop="name" label="姓名" />
              <el-table-column prop="status" label="状态" width="90" />
              <el-table-column label="操作" width="90">
                <template #default="{ row }">
                  <el-button link type="primary" @click="goAuthorize(Number(row.id))">授权</el-button>
                </template>
              </el-table-column>
            </el-table>
          </section>
          <el-empty v-else description="请选择角色" />
        </div>
      </el-tab-pane>

      <el-tab-pane label="用户授权" name="authorize">
        <div class="row">
          <el-select v-model="selectedUser" filterable clearable placeholder="选择用户" style="width:220px">
            <el-option v-for="u in users" :key="String(u.id)" :label="`${u.login_name} (#${u.id})`" :value="Number(u.id)" />
          </el-select>
        </div>
        <template v-if="selectedUser">
          <el-row :gutter="16">
            <el-col :span="12">
              <h3 class="sub">绑定员工</h3>
              <div class="row">
                <el-select v-model="bindEmpId" filterable clearable placeholder="员工" style="width:260px">
                  <el-option v-for="e in employees" :key="String(e.id)" :label="empLabel(e)" :value="Number(e.id)" />
                </el-select>
                <el-button type="primary" @click="bindEmployee">绑定</el-button>
                <el-button @click="unbindEmployee">解绑</el-button>
              </div>
              <h3 class="sub">角色（并集）</h3>
              <el-checkbox-group v-model="selectedRoles">
                <el-checkbox v-for="r in roles" :key="String(r.id)" :label="Number(r.id)">
                  {{ r.name }} ({{ r.code }})
                </el-checkbox>
              </el-checkbox-group>
              <div class="row" style="margin-top:12px">
                <el-button type="primary" @click="saveRoles">保存角色</el-button>
              </div>
            </el-col>
            <el-col :span="12">
              <h3 class="sub">数据范围</h3>
              <el-form label-width="110px" style="max-width:360px">
                <el-form-item label="范围类型">
                  <el-select v-model="dataScope.data_scope_type" style="width:100%">
                    <el-option label="本人 self" value="self" />
                    <el-option label="车间 workshop" value="workshop" />
                    <el-option label="班组 team" value="team" />
                    <el-option label="全部 all" value="all" />
                  </el-select>
                </el-form-item>
                <el-form-item label="车间">
                  <WorkshopSelect v-model="dataScope.workshop_id" allow-zero zero-label="不限" style="width:100%" />
                </el-form-item>
                <el-form-item label="班组 ID">
                  <el-input-number v-model="dataScope.team_id" :min="0" placeholder="暂无班组主数据，请填ID" style="width:100%" />
                </el-form-item>
                <el-button type="primary" @click="saveScope">保存数据范围</el-button>
              </el-form>
              <h3 class="sub">快捷</h3>
              <el-button @click="tab = 'roles'">打开角色管理</el-button>
              <p class="muted" style="margin-top:8px">配置角色权限码 / 仓工序范围请到「角色管理」页签。</p>
            </el-col>
          </el-row>
        </template>
        <el-empty v-else description="请先选择用户" />
      </el-tab-pane>

      <el-tab-pane label="入职·离职" name="lifecycle">
        <el-row :gutter="20">
          <el-col :span="12">
            <h3 class="sub">入职开户</h3>
            <div class="row">
              <el-select v-model="onboardForm.employee_id" filterable clearable placeholder="未开户员工" style="width:220px">
                <el-option v-for="e in unboundEmployees" :key="String(e.id)" :label="empLabel(e)" :value="Number(e.id)" />
              </el-select>
              <el-input v-model="onboardForm.remark" placeholder="备注" style="width:140px" />
              <el-button type="primary" @click="createOnboard">建单</el-button>
            </div>
            <el-checkbox-group v-model="onboardForm.role_ids" style="margin-bottom:8px">
              <el-checkbox v-for="r in roles.slice(0, 8)" :key="'ob'+r.id" :label="Number(r.id)">{{ r.name }}</el-checkbox>
            </el-checkbox-group>
            <el-table :data="onboards" border size="small" height="260">
              <el-table-column prop="id" label="ID" width="60" />
              <el-table-column prop="employee_id" label="员工" width="80" />
              <el-table-column prop="status" label="状态" width="90" />
              <el-table-column prop="remark" label="备注" />
              <el-table-column label="操作" width="90">
                <template #default="{ row }">
                  <el-button v-if="row.status !== 'confirmed'" link type="primary" @click="confirmOnboard(Number(row.id))">确认开户</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-col>
          <el-col :span="12">
            <h3 class="sub">离职收回</h3>
            <div class="row">
              <el-select v-model="offboardForm.employee_id" filterable clearable placeholder="员工" style="width:220px">
                <el-option v-for="e in employees.filter(x => x.status !== 'left')" :key="'of'+e.id" :label="empLabel(e)" :value="Number(e.id)" />
              </el-select>
              <el-input v-model="offboardForm.reason" placeholder="原因" style="width:140px" />
              <el-checkbox v-model="offboardForm.revoke_permission">收回权限</el-checkbox>
              <el-button type="danger" @click="createOffboard">建单</el-button>
            </div>
            <el-table :data="offboards" border size="small" height="260">
              <el-table-column prop="id" label="ID" width="60" />
              <el-table-column prop="employee_id" label="员工" width="80" />
              <el-table-column prop="status" label="状态" width="90" />
              <el-table-column prop="reason" label="原因" />
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button v-if="row.status !== 'confirmed'" link type="danger" @click="confirmOffboard(Number(row.id))">确认离职</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane label="冻结·会话" name="freeze">
        <h3 class="sub">账户冻结</h3>
        <el-table :data="users" border stripe height="280">
          <el-table-column prop="login_name" label="登录名" />
          <el-table-column prop="name" label="姓名" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button link :type="row.status === 'frozen' ? 'success' : 'danger'"
                @click="freeze(Number(row.id), row.status === 'frozen')">
                {{ row.status === 'frozen' ? '解冻' : '冻结' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <h3 class="sub">在线会话</h3>
        <el-table :data="sessions" border size="small" height="220">
          <el-table-column prop="id" label="会话ID" width="90" />
          <el-table-column prop="user_id" label="用户" width="90" />
          <el-table-column prop="client_type" label="端" width="100" />
          <el-table-column prop="created_at" label="创建时间" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="revokeSession(Number(row.id))">踢下线</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="openDlg" title="员工开户" width="480px">
      <el-form label-width="100px">
        <el-form-item label="员工">
          <el-select v-model="openForm.employee_id" filterable style="width:100%">
            <el-option v-for="e in unboundEmployees" :key="'oa'+e.id" :label="empLabel(e)" :value="Number(e.id)" />
          </el-select>
        </el-form-item>
        <el-form-item label="登录名">
          <el-input v-model="openForm.login_name" placeholder="默认用工号" />
        </el-form-item>
        <el-form-item label="初始密码">
          <el-input v-model="openForm.password" show-password />
        </el-form-item>
        <el-form-item label="角色">
          <el-checkbox-group v-model="openForm.role_ids">
            <el-checkbox v-for="r in roles" :key="'oar'+r.id" :label="Number(r.id)">{{ r.name }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="openDlg = false">取消</el-button>
        <el-button type="primary" @click="submitOpenAccount">开户</el-button>
      </template>
    </el-dialog>

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
            <el-option label="车间 workshop" value="workshop" />
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
.stats { display: grid; grid-template-columns: repeat(6, 1fr); gap: 10px; margin-bottom: 14px; }
.stat {
  background: #f4f7f9; border: 1px solid #e2e8ec; border-radius: 8px; padding: 12px;
  cursor: pointer; transition: border-color .15s;
}
.stat:hover { border-color: #3d8b7a; }
.stat.warn { background: #fff8e8; }
.stat.danger { background: #fff0f0; }
.stat .label { font-size: 12px; color: #5c6b75; }
.stat .value { font-size: 22px; font-weight: 600; margin-top: 4px; color: #1a2b33; }
.steps { line-height: 1.8; color: #3d4f5a; }
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
  .stats { grid-template-columns: repeat(3, 1fr); }
  .role-layout { grid-template-columns: 1fr; }
  .perm-mod-row { grid-template-columns: 1fr; }
}
</style>
