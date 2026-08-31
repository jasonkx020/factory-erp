<script setup lang="ts">
import { computed, ref } from 'vue'
import KanbanAssignPanel from './KanbanAssignPanel.vue'

type Row = Record<string, unknown>

export type MemberOption = {
  id: number
  emp_no: string
  name: string
  job_title_name?: string
  emp_type?: string
  login_name?: string
  has_account?: boolean
  otherDepts?: string
}

const props = defineProps<{
  modelValue: number[]
  employees: Row[]
  flatList: Row[]
  editingDeptId?: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const empType = ref('')
const accountFilter = ref<'all' | 'yes' | 'no'>('all')

const empTypeOptions = computed(() => {
  const set = new Set<string>()
  for (const e of props.employees) {
    const t = String(e.emp_type || '').trim()
    if (t) set.add(t)
  }
  return Array.from(set).sort()
})

const memberOptions = computed<MemberOption[]>(() =>
  props.employees
    .filter((e) => String(e.status || '') !== 'left')
    .map((e) => {
      const deptIds = ((e.dept_ids as number[]) || []).map(Number)
      const editingId = props.editingDeptId || 0
      const inCurrent = editingId ? deptIds.includes(editingId) : false
      const otherNames = deptIds
        .filter((id) => id !== editingId)
        .map((id) => String(props.flatList.find((d) => Number(d.id) === id)?.name || `#${id}`))
      return {
        id: Number(e.id),
        emp_no: String(e.emp_no || ''),
        name: String(e.name || ''),
        job_title_name: String(e.job_title_name || ''),
        emp_type: String(e.emp_type || ''),
        login_name: String(e.login_name || ''),
        has_account: Boolean(e.has_account),
        otherDepts: !inCurrent && otherNames.length ? otherNames.join('、') : '',
      }
    }),
)

const filteredAvailableCount = computed(() => {
  const selected = new Set(props.modelValue)
  return memberOptions.value.filter((item) => !selected.has(item.id) && matchFilters(item)).length
})

function matchNameSearch(item: MemberOption, q: string) {
  const query = q.trim().toLowerCase()
  if (!query) return true
  return item.name.toLowerCase().includes(query) || item.emp_no.toLowerCase().includes(query)
}

function matchFilters(item: MemberOption) {
  if (empType.value && item.emp_type !== empType.value) return false
  if (accountFilter.value === 'yes' && !item.has_account) return false
  if (accountFilter.value === 'no' && item.has_account) return false
  return true
}

function filterLeft(item: MemberOption) {
  return matchFilters(item)
}

function itemTitle(item: MemberOption) {
  const parts = [
    item.emp_no,
    item.name,
    item.job_title_name,
    item.emp_type ? empTypeLabel[item.emp_type] || item.emp_type : '',
    item.has_account ? item.login_name || '已开户' : '未开户',
    item.otherDepts ? `还在：${item.otherDepts}` : '',
  ].filter(Boolean)
  return parts.join(' · ')
}

function addFiltered() {
  const selected = new Set(props.modelValue)
  const next = [...props.modelValue]
  for (const item of memberOptions.value) {
    if (selected.has(item.id)) continue
    if (!matchFilters(item)) continue
    next.push(item.id)
    selected.add(item.id)
  }
  emit('update:modelValue', next)
}

function clearSelected() {
  emit('update:modelValue', [])
}

const empTypeLabel: Record<string, string> = {
  formal: '正式',
  temp: '临时',
  outsource: '外包',
  intern: '实习',
  office: '行政',
  piece: '计件',
  fixed: '固定',
  warehouse: '仓储',
}
</script>

<template>
  <KanbanAssignPanel
    :model-value="modelValue"
    :options="memberOptions"
    :get-id="(item) => item.id"
    left-title="可选人员"
    right-title="本部门成员"
    height="420px"
    compact
    :filter-left="filterLeft"
    :search-match="matchNameSearch"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #filters>
      <div class="member-toolbar">
        <div class="member-filters">
          <el-select v-model="empType" size="small" clearable placeholder="用工类型" style="width: 100px">
            <el-option label="全部" value="" />
            <el-option
              v-for="t in empTypeOptions"
              :key="t"
              :label="empTypeLabel[t] || t"
              :value="t"
            />
          </el-select>
          <el-select v-model="accountFilter" size="small" placeholder="账号" style="width: 90px">
            <el-option label="全部" value="all" />
            <el-option label="已开户" value="yes" />
            <el-option label="未开户" value="no" />
          </el-select>
        </div>
        <div class="member-bulk">
          <el-button size="small" link type="primary" :disabled="!filteredAvailableCount" @click="addFiltered">
            加入筛选结果 ({{ filteredAvailableCount }})
          </el-button>
          <el-button size="small" link type="danger" :disabled="!modelValue.length" @click="clearSelected">
            清空已选
          </el-button>
        </div>
      </div>
    </template>

    <template #list-head>
      <div class="member-grid member-grid-head">
        <span>工号</span>
        <span>姓名</span>
        <span>岗位</span>
        <span>备注</span>
      </div>
    </template>

    <template #item="{ item }">
      <el-tooltip :content="itemTitle(item)" placement="top" :show-after="400">
        <div class="member-grid member-row">
          <span class="col-no">{{ item.emp_no || '—' }}</span>
          <span class="col-name">{{ item.name }}</span>
          <span class="col-title">{{ item.job_title_name || '—' }}</span>
          <span class="col-meta">
            <span v-if="item.otherDepts" class="tag-other" title="兼任其他部门">兼</span>
            <span v-if="!item.has_account" class="tag-muted">无账号</span>
          </span>
        </div>
      </el-tooltip>
    </template>
  </KanbanAssignPanel>
</template>

<style scoped>
.member-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}
.member-filters {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}
.member-bulk {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
.member-grid {
  display: grid;
  grid-template-columns: 68px minmax(56px, 0.9fr) minmax(64px, 1fr) 36px;
  gap: 4px;
  align-items: center;
  width: 100%;
  font-size: 12px;
  line-height: 1.2;
}
.member-grid-head {
  padding: 2px 28px 4px 4px;
  color: #8a9aa3;
  font-size: 11px;
  font-weight: 500;
}
.member-row {
  min-height: 24px;
}
.col-no {
  color: #5c6b75;
  font-family: ui-monospace, monospace;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.col-name {
  font-weight: 600;
  color: #334;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.col-title {
  color: #5c6b75;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.col-meta {
  display: flex;
  gap: 2px;
  justify-content: flex-end;
}
.tag-other {
  display: inline-block;
  padding: 0 4px;
  border-radius: 2px;
  background: #fff7e6;
  color: #d46b08;
  font-size: 10px;
  line-height: 16px;
}
.tag-muted {
  display: inline-block;
  padding: 0 4px;
  border-radius: 2px;
  background: #f0f2f5;
  color: #8a9aa3;
  font-size: 10px;
  line-height: 16px;
}
</style>
