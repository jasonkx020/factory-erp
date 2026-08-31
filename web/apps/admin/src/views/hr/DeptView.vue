<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hrApi, iamApi } from '@erp/shared'
import OrgChartNode from './OrgChartNode.vue'
import DeptMemberKanban from '../../components/hr/DeptMemberKanban.vue'
import DeptRoleKanban from '../../components/hr/DeptRoleKanban.vue'
import WorkshopTeamAssign from '../../components/hr/WorkshopTeamAssign.vue'

type Row = Record<string, unknown>
type Member = { id: number; emp_no: string; name: string; job_title_name?: string; login_name?: string; has_account?: boolean }
type TreeNode = Row & { children?: TreeNode[] }

const loading = ref(false)
const viewMode = ref<'chart' | 'tree'>('chart')
const flatList = ref<Row[]>([])
const treeData = ref<TreeNode[]>([])
const employees = ref<Row[]>([])
const roles = ref<Row[]>([])
const dlg = ref(false)
const detailDlg = ref(false)
const editingId = ref<number | null>(null)
const detail = ref<Row | null>(null)
const form = reactive({
  code: '',
  name: '',
  status: 'active',
  parent_id: 0,
  dept_type: 'normal',
  employee_ids: [] as number[],
  role_ids: [] as number[],
  teams: [] as { id?: number; code: string; name: string }[],
})
const teamMembers = ref<Record<number, number[]>>({})

const errLabel: Record<string, string> = {
  NAME_REQUIRED: '请填写部门名称',
  CODE_DUPLICATE: '部门编码已存在',
  DEPT_IN_USE: '仍有在职员工属于该部门，无法删除',
  DEPT_HAS_CHILDREN: '请先删除或移走子部门',
  MAX_DEPTH_EXCEEDED: '组织层级最多 3 层',
  PARENT_INVALID: '上级部门无效',
  PARENT_NOT_FOUND: '上级部门不存在',
  PARENT_IS_DESCENDANT: '上级部门不能是当前部门的子级',
  WORKSHOP_PARENT_REQUIRED: '车间必须挂在行政部门下',
  WORKSHOP_PARENT_MUST_BE_NORMAL: '车间的上级必须是行政部门',
  WORKSHOP_NO_CHILDREN: '车间节点不能再挂子部门，班组请在车间内维护',
  TEAM_NOT_IN_WORKSHOP: '班组不属于当前车间',
  NOT_FOUND: '部门不存在',
}

const levelStats = computed(() => {
  const stats = { l1: 0, l2: 0, l3: 0 }
  for (const d of flatList.value) {
    const lv = Number(d.level || 1)
    if (lv === 1) stats.l1++
    else if (lv === 2) stats.l2++
    else if (lv === 3) stats.l3++
  }
  return stats
})

const descendantIds = computed(() => {
  if (!editingId.value) return new Set<number>()
  const set = new Set<number>()
  const walk = (nodes: TreeNode[]) => {
    for (const n of nodes) {
      const id = Number(n.id)
      set.add(id)
      if (n.children?.length) walk(n.children)
    }
  }
  const findSubtree = (nodes: TreeNode[]): TreeNode[] | null => {
    for (const n of nodes) {
      if (Number(n.id) === editingId.value) return [n]
      if (n.children?.length) {
        const hit = findSubtree(n.children)
        if (hit) return hit
      }
    }
    return null
  }
  const sub = findSubtree(treeData.value)
  if (sub) walk(sub)
  return set
})

const parentOptions = computed(() =>
  flatList.value.filter((d) => {
    const level = Number(d.level || 1)
    const id = Number(d.id)
    if (level >= 3) return false
    if (String(d.dept_type || 'normal') === 'workshop') return false
    if (editingId.value && (id === editingId.value || descendantIds.value.has(id))) return false
    return true
  }),
)

const formLevel = computed(() => {
  if (form.parent_id > 0) {
    const parent = flatList.value.find((d) => Number(d.id) === form.parent_id)
    return Number(parent?.level || 0) + 1
  }
  return 1
})

const treeProps = { children: 'children', label: 'name' }

async function loadMeta() {
  const [empRes, roleRes] = await Promise.all([hrApi.employees(), iamApi.roles()])
  if (empRes.code === 1) employees.value = ((empRes.data as { list?: Row[] })?.list) || []
  if (roleRes.code === 1) roles.value = roleRes.data?.list || []
}

async function load() {
  loading.value = true
  try {
    const res = await hrApi.departments()
    if (res.code !== 1) {
      ElMessage.error(res.msg || '加载失败')
      flatList.value = []
      treeData.value = []
      return
    }
    const data = (res.data || {}) as { list?: Row[]; tree?: TreeNode[] }
    flatList.value = data.list || []
    treeData.value = data.tree || []
  } finally {
    loading.value = false
  }
}

function resetTeamMembers() {
  teamMembers.value = {}
}

function loadTeamMembersFromTeams(teams: Row[]) {
  const m: Record<number, number[]> = {}
  for (const t of teams) {
    const id = Number(t.id) || 0
    if (id > 0) m[id] = ((t.employee_ids as number[]) || []).map(Number)
  }
  teamMembers.value = m
}

function buildTeamMembersBody() {
  return form.teams
    .filter((t) => Number(t.id) > 0)
    .map((t) => ({
      team_id: Number(t.id),
      team_code: t.code,
      employee_ids: teamMembers.value[Number(t.id)] || [],
    }))
}

function resetForm(parentId = 0) {
  Object.assign(form, {
    code: '',
    name: '',
    status: 'active',
    parent_id: parentId,
    dept_type: 'normal',
    employee_ids: [],
    role_ids: [],
    teams: [],
  })
  resetTeamMembers()
}

function isWorkshopNode(row: Row) {
  return String(row.dept_type || 'normal') === 'workshop'
}

function canAddChild(row: Row) {
  return Number(row.level) < 3 && !isWorkshopNode(row)
}

function openCreate(parentId = 0) {
  editingId.value = null
  resetForm(parentId)
  if (parentId > 0) {
    const parent = flatList.value.find((d) => Number(d.id) === parentId)
    if (Number(parent?.level || 1) >= 2) form.dept_type = 'workshop'
  }
  dlg.value = true
}

async function openEdit(row: Row) {
  editingId.value = Number(row.id)
  resetForm()
  const res = await hrApi.getDepartment(editingId.value)
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '加载失败')
  const data = (res.data || {}) as Row
  Object.assign(form, {
    code: String(data.code || ''),
    name: String(data.name || ''),
    status: String(data.status || 'active'),
    parent_id: Number(data.parent_id || 0),
    dept_type: String(data.dept_type || 'normal'),
    employee_ids: ((data.members as Member[]) || []).map((m) => Number(m.id)),
    role_ids: ((data.role_ids as number[]) || []).map(Number),
    teams: ((data.teams as { id?: number; code: string; name: string; employee_ids?: number[] }[]) || []).map((t) => ({
      id: Number(t.id) || undefined,
      code: String(t.code || ''),
      name: String(t.name || ''),
    })),
  })
  loadTeamMembersFromTeams((data.teams as Row[]) || [])
  dlg.value = true
}

async function openDetail(row: Row) {
  const res = await hrApi.getDepartment(Number(row.id))
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '加载失败')
  detail.value = (res.data || {}) as Row
  detailDlg.value = true
}

async function save() {
  if (!form.name.trim()) return ElMessage.warning('请填写部门名称')
  if (!form.parent_id) form.dept_type = 'normal'
  const body: Record<string, unknown> = {
    name: form.name.trim(),
    status: form.status,
    parent_id: form.parent_id || 0,
    dept_type: form.dept_type,
    employee_ids: form.employee_ids,
    role_ids: form.role_ids,
  }
  if (form.dept_type === 'workshop') {
    body.teams = form.teams.filter((t) => t.name.trim())
    body.team_members = buildTeamMembersBody()
  }
  let res
  if (editingId.value) {
    res = await hrApi.updateDepartment(editingId.value, body)
  } else {
    if (form.code.trim()) body.code = form.code.trim()
    res = await hrApi.createDepartment(body)
  }
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '保存失败')
  ElMessage.success(editingId.value ? '已保存' : '已创建')
  dlg.value = false
  await load()
}

async function deactivate(row: Row) {
  await ElMessageBox.confirm(`将「${row.name}」设为停用？`, '提示')
  const res = await hrApi.updateDepartment(Number(row.id), { status: 'inactive' })
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '失败')
  ElMessage.success('已停用')
  await load()
}

async function remove(row: Row) {
  await ElMessageBox.confirm(`删除「${row.name}」？存在子部门或在职员工时将无法删除。`, '删除确认', {
    type: 'warning',
  })
  const res = await hrApi.removeDepartment(Number(row.id))
  if (res.code !== 1) return ElMessage.error(errLabel[res.msg] || res.msg || '删除失败')
  ElMessage.success('已删除')
  await load()
}

function onChartAction(action: string, node: Row) {
  if (action === 'detail') openDetail(node)
  else if (action === 'edit') openEdit(node)
  else if (action === 'child') openCreate(Number(node.id))
  else if (action === 'deactivate') deactivate(node)
  else if (action === 'remove') remove(node)
}

onMounted(async () => {
  await Promise.all([loadMeta(), load()])
})
</script>

<template>
  <div v-loading="loading" class="org">
    <h2 class="title">公司架构</h2>
    <p class="desc">
      树形展示部门父子关系，最多 3 层。车间是第三层（或挂在行政部门下）的节点类型；班组在车间内维护。上级部门自动汇总全部子部门权限。
    </p>

    <div class="toolbar">
      <div class="toolbar-left">
        <el-button type="primary" @click="openCreate(0)">新建一级部门</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
      <div class="toolbar-right">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button value="chart">架构图</el-radio-button>
          <el-radio-button value="tree">树形列表</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <div v-if="flatList.length" class="stats-bar">
      <span>共 {{ flatList.length }} 个部门</span>
      <el-tag size="small" type="primary" effect="plain">一级 {{ levelStats.l1 }}</el-tag>
      <el-tag size="small" type="success" effect="plain">二级 {{ levelStats.l2 }}</el-tag>
      <el-tag size="small" type="warning" effect="plain">三级 {{ levelStats.l3 }}</el-tag>
      <span class="legend-hint">连线表示上下级归属关系</span>
    </div>

    <div v-if="treeData.length && viewMode === 'chart'" class="chart-wrap">
      <div class="org-chart">
        <OrgChartNode
          v-for="node in treeData"
          :key="String(node.id)"
          :node="node"
          :is-root="true"
          @action="onChartAction"
        />
      </div>
    </div>

    <el-tree
      v-else-if="treeData.length && viewMode === 'tree'"
      :data="treeData"
      :props="treeProps"
      node-key="id"
      default-expand-all
      highlight-current
      class="org-tree"
    >
      <template #default="{ node, data }">
        <div class="tree-node" :class="`tree-level-${data.level || 1}`">
          <div class="tree-main">
            <span class="tree-indent">{{ '│ '.repeat(Math.max(0, Number(data.level || 1) - 1)) }}</span>
            <span class="tree-name" @click="openDetail(data)">{{ data.name }}</span>
            <el-tag size="small" :type="Number(data.level) === 1 ? 'primary' : Number(data.level) === 2 ? 'success' : 'warning'">
              {{ data.level_label || `L${data.level}` }}
            </el-tag>
            <el-tag v-if="isWorkshopNode(data)" size="small" type="danger" effect="plain">车间</el-tag>
            <span v-if="data.parent_name" class="tree-parent">↑ {{ data.parent_name }}</span>
            <span class="tree-meta">{{ data.code }}</span>
            <span v-if="data.has_children" class="tree-meta">{{ data.child_count }} 个子部门</span>
            <span class="tree-meta">{{ data.member_count ?? 0 }} 人</span>
            <el-tag :type="data.status === 'active' ? 'success' : 'info'" size="small">
              {{ data.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </div>
          <div class="tree-actions">
            <el-button v-if="canAddChild(data)" link type="primary" @click.stop="openCreate(Number(data.id))">
              子部门
            </el-button>
            <el-button link type="primary" @click.stop="openDetail(data)">详情</el-button>
            <el-button link type="primary" @click.stop="openEdit(data)">编辑</el-button>
            <el-button v-if="data.status === 'active'" link @click.stop="deactivate(data)">停用</el-button>
            <el-button link type="danger" @click.stop="remove(data)">删除</el-button>
          </div>
        </div>
      </template>
    </el-tree>

    <el-empty v-else description="暂无组织节点，请先创建一级部门" />

    <el-dialog
      v-model="dlg"
      :title="editingId ? '编辑部门' : (form.parent_id ? '新建子部门' : '新建一级部门')"
      width="min(920px, 96vw)"
    >
      <el-form label-width="108px">
        <el-form-item label="上级部门">
          <el-select v-model="form.parent_id" clearable placeholder="无（一级部门）" style="width:100%">
            <el-option :value="0" label="无（一级部门）" />
            <el-option
              v-for="d in parentOptions"
              :key="String(d.id)"
              :value="Number(d.id)"
              :label="`${d.path || d.name} (${d.level_label})`"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="层级">
          <el-tag type="info">{{ formLevel === 1 ? '一级' : formLevel === 2 ? '二级' : '三级' }}</el-tag>
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="form.code" :disabled="!!editingId" :placeholder="editingId ? '' : '可选，空则自动生成'" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="64" placeholder="部门名称" />
        </el-form-item>
        <el-form-item v-if="form.parent_id" label="节点类型">
          <el-radio-group v-model="form.dept_type">
            <el-radio value="normal">行政部门</el-radio>
            <el-radio value="workshop">车间</el-radio>
          </el-radio-group>
          <p class="hint">车间必须挂在行政部门下，不能再挂子部门；班组在车间内维护。</p>
        </el-form-item>
        <el-form-item label="部门成员">
          <DeptMemberKanban
            v-model="form.employee_ids"
            :employees="employees"
            :flat-list="flatList"
            :editing-dept-id="editingId"
          />
        </el-form-item>
        <el-form-item v-if="form.dept_type === 'workshop'" label="班组与成员">
          <WorkshopTeamAssign
            v-model:teams="form.teams"
            v-model:team-members="teamMembers"
            :workshop-member-ids="form.employee_ids"
            :employees="employees"
          />
          <p class="hint">Tab 右侧 + 添加班组；在各班组 Tab 内拖拽分配成员，最后统一保存车间。</p>
        </el-form-item>
        <el-form-item v-if="editingId" label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="本级基础角色">
          <DeptRoleKanban v-model="form.role_ids" :roles="roles" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="save">{{ editingId ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailDlg" :title="detail ? `部门详情 · ${detail.name}` : '部门详情'" size="560px">
      <template v-if="detail">
        <div class="detail-relation">
          <div v-if="detail.parent_name" class="relation-row">
            <span class="relation-label">上级部门</span>
            <span class="relation-value">{{ detail.parent_name }}</span>
          </div>
          <div v-else class="relation-row">
            <span class="relation-label">上级部门</span>
            <span class="relation-value muted">无（一级部门）</span>
          </div>
          <div class="relation-row">
            <span class="relation-label">架构路径</span>
            <span class="relation-value">{{ detail.path || detail.name }}</span>
          </div>
          <div class="relation-row">
            <span class="relation-label">直属子部门</span>
            <span class="relation-value">{{ detail.child_count ?? 0 }} 个</span>
          </div>
        </div>

        <div class="detail-meta">
          <el-tag size="small" type="info">{{ detail.level_label }}</el-tag>
          <el-tag v-if="isWorkshopNode(detail)" size="small" type="danger" effect="plain">车间</el-tag>
          <span>直属 {{ detail.member_count ?? 0 }} 人</span>
          <el-tag :type="detail.status === 'active' ? 'success' : 'info'" size="small">
            {{ detail.status === 'active' ? '启用' : '停用' }}
          </el-tag>
        </div>

        <h4 class="sub">本级基础角色</h4>
        <div v-if="(detail.base_roles as Row[])?.length" class="role-tags">
          <el-tag v-for="r in (detail.base_roles as Row[])" :key="'d'+r.id" size="small">{{ r.name || r.code }}</el-tag>
        </div>
        <p v-else class="hint">本级未配置角色。</p>

        <h4 class="sub">继承子部门角色</h4>
        <div v-if="(detail.inherited_roles as Row[])?.length" class="role-tags">
          <el-tag v-for="r in (detail.inherited_roles as Row[])" :key="'i'+r.id" size="small" type="warning">
            {{ r.name || r.code }}
          </el-tag>
        </div>
        <p v-else class="hint">无下级继承角色。</p>

        <h4 class="sub">有效权限（本级 + 全部子部门）</h4>
        <div v-if="(detail.effective_roles as Row[])?.length" class="role-tags">
          <el-tag v-for="r in (detail.effective_roles as Row[])" :key="'e'+r.id" size="small" type="success">
            {{ r.name || r.code }}
          </el-tag>
        </div>
        <p v-else class="hint">暂无有效权限。</p>

        <template v-if="isWorkshopNode(detail)">
          <h4 class="sub">班组</h4>
          <el-table :data="(detail.teams as Row[]) || []" border stripe empty-text="暂无班组" size="small">
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" label="名称" min-width="120" />
            <el-table-column prop="status" label="状态" width="80">
              <template #default="{ row }">{{ row.status === 'active' ? '启用' : '停用' }}</template>
            </el-table-column>
          </el-table>
        </template>

        <h4 class="sub">直属成员</h4>
        <el-table :data="(detail.members as Member[]) || []" border stripe empty-text="暂无直属成员" size="small">
          <el-table-column prop="emp_no" label="工号" width="100" />
          <el-table-column prop="name" label="姓名" min-width="100" />
          <el-table-column prop="job_title_name" label="岗位" min-width="100" />
          <el-table-column label="账号" min-width="120">
            <template #default="{ row }">
              <span v-if="row.has_account">{{ row.login_name || '已开户' }}</span>
              <el-tag v-else size="small" type="info">未开户</el-tag>
            </template>
          </el-table-column>
        </el-table>
        <div class="detail-actions">
          <el-button type="primary" @click="detail && openEdit(detail); detailDlg = false">编辑</el-button>
          <el-button
            v-if="canAddChild(detail)"
            @click="detail && openCreate(Number(detail.id)); detailDlg = false"
          >
            添加子部门
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.org { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.toolbar { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.toolbar-left, .toolbar-right { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.stats-bar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; padding: 8px 12px; background: #f6f9fb; border-radius: 6px; font-size: 13px; color: #5c6b75; flex-wrap: wrap; }
.legend-hint { margin-left: auto; color: #8a9aa3; font-size: 12px; }
.chart-wrap { overflow-x: auto; border: 1px solid #e8eef2; border-radius: 8px; padding: 24px 16px; background: linear-gradient(180deg, #fafcfd 0%, #fff 100%); }
.org-chart { display: flex; justify-content: center; gap: 32px; min-width: min-content; }
.org-tree { border: 1px solid #e8eef2; border-radius: 8px; padding: 8px 12px; }
.org-tree :deep(.el-tree-node__content) { height: auto; min-height: 40px; padding: 4px 0; }
.org-tree :deep(.el-tree-node__expand-icon) { font-size: 16px; color: #1677ff; }
.tree-node { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; padding: 4px 0; }
.tree-level-1 { border-left: 3px solid #1677ff; padding-left: 8px; }
.tree-level-2 { border-left: 3px solid #52c41a; padding-left: 8px; }
.tree-level-3 { border-left: 3px solid #faad14; padding-left: 8px; }
.tree-main { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; min-width: 0; }
.tree-indent { color: #d0d7de; font-family: monospace; font-size: 12px; user-select: none; }
.tree-name { font-weight: 600; cursor: pointer; color: #1677ff; }
.tree-parent { color: #8a9aa3; font-size: 12px; background: #f0f4f8; padding: 1px 6px; border-radius: 4px; }
.tree-meta { color: #8a9aa3; font-size: 12px; }
.tree-actions { display: flex; gap: 2px; flex-shrink: 0; flex-wrap: wrap; justify-content: flex-end; }
.hint { margin: 6px 0 0; color: #8a9aa3; font-size: 12px; line-height: 1.5; }
.detail-relation { background: #f6f9fb; border-radius: 8px; padding: 12px 14px; margin-bottom: 14px; }
.relation-row { display: flex; gap: 12px; padding: 4px 0; font-size: 13px; }
.relation-label { width: 72px; flex-shrink: 0; color: #8a9aa3; }
.relation-value { color: #334; word-break: break-all; }
.relation-value.muted { color: #8a9aa3; }
.detail-meta { display: flex; gap: 12px; align-items: center; margin-bottom: 16px; color: #5c6b75; font-size: 13px; flex-wrap: wrap; }
.sub { margin: 16px 0 8px; font-size: 14px; color: #334; }
.role-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; }
.detail-actions { margin-top: 16px; display: flex; gap: 8px; }
.team-row { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
</style>
