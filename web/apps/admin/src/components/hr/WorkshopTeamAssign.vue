<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import KanbanAssignPanel from './KanbanAssignPanel.vue'

type Row = Record<string, unknown>

export type TeamRow = { id?: number; code: string; name: string }

export type MemberOption = {
  id: number
  emp_no: string
  name: string
  job_title_name?: string
  team_id?: number
  team_name?: string
}

const props = defineProps<{
  teams: TeamRow[]
  workshopMemberIds: number[]
  employees: Row[]
  teamMembers: Record<number, number[]>
}>()

const emit = defineEmits<{
  'update:teams': [value: TeamRow[]]
  'update:teamMembers': [value: Record<number, number[]>]
}>()

const activeTeamKey = ref('')
const addDlg = ref(false)
const teamForm = reactive({ code: '', name: '' })

const teamsWithKey = computed(() =>
  props.teams
    .filter((t) => t.name.trim())
    .map((t, i) => ({
      ...t,
      key: t.id ? String(t.id) : `new-${i}-${t.code || t.name}`,
      teamId: Number(t.id) || 0,
    })),
)

const teamNameMap = computed(() => {
  const m = new Map<number, string>()
  for (const t of props.teams) {
    const id = Number(t.id) || 0
    if (id > 0) m.set(id, String(t.name || t.code || `#${id}`))
  }
  return m
})

const memberOptions = computed<MemberOption[]>(() => {
  const pool = new Set(props.workshopMemberIds.map(Number))
  return props.employees
    .filter((e) => pool.has(Number(e.id)) && String(e.status || '') !== 'left')
    .map((e) => {
      const teamId = Number(e.team_id) || 0
      return {
        id: Number(e.id),
        emp_no: String(e.emp_no || ''),
        name: String(e.name || ''),
        job_title_name: String(e.job_title_name || ''),
        team_id: teamId,
        team_name: teamId > 0 ? teamNameMap.value.get(teamId) || '' : '',
      }
    })
})

watch(
  teamsWithKey,
  (list) => {
    if (!list.length) {
      activeTeamKey.value = ''
      return
    }
    if (!list.some((t) => t.key === activeTeamKey.value)) {
      activeTeamKey.value = list[0].key
    }
  },
  { immediate: true },
)

function currentMemberIds(team: (typeof teamsWithKey.value)[0]) {
  const tid = team.teamId
  if (tid > 0) return props.teamMembers[tid] || []
  return []
}

function setTeamMembers(team: { teamId: number }, ids: number[]) {
  const tid = team.teamId
  if (tid <= 0) return
  const next: Record<number, number[]> = {}
  for (const t of teamsWithKey.value) {
    if (t.teamId > 0) next[t.teamId] = [...(props.teamMembers[t.teamId] || [])]
  }
  for (const id of Object.keys(next).map(Number)) {
    next[id] = next[id].filter((eid) => !ids.includes(eid))
  }
  next[tid] = [...ids]
  emit('update:teamMembers', next)
}

function matchNameSearch(item: MemberOption, q: string) {
  const query = q.trim().toLowerCase()
  if (!query) return true
  return item.name.toLowerCase().includes(query) || item.emp_no.toLowerCase().includes(query)
}

function otherTeamLabel(item: MemberOption, currentTeamId: number) {
  if (!item.team_id || item.team_id === currentTeamId) return ''
  return item.team_name || `班组#${item.team_id}`
}

function openAddTeam() {
  teamForm.code = ''
  teamForm.name = ''
  addDlg.value = true
}

function confirmAddTeam() {
  const name = teamForm.name.trim()
  if (!name) return ElMessage.warning('请填写班组名称')
  const code = teamForm.code.trim()
  const next = [...props.teams, { code, name }]
  emit('update:teams', next)
  addDlg.value = false
  const idx = next.length - 1
  const key = `new-${idx}-${code || name}`
  activeTeamKey.value = key
  ElMessage.success('已添加班组，保存车间后生效')
}

async function removeTeam(team: (typeof teamsWithKey.value)[0]) {
  const label = team.name || team.code || '该班组'
  await ElMessageBox.confirm(`删除班组「${label}」？已分配成员将变为未分班。`, '删除班组', { type: 'warning' })
  const tid = team.teamId
  const nextTeams = props.teams.filter((t) => {
    if (tid > 0) return Number(t.id) !== tid
    return !(t.name === team.name && (t.code || '') === (team.code || ''))
  })
  emit('update:teams', nextTeams)
  if (tid > 0) {
    const nextMembers = { ...props.teamMembers }
    delete nextMembers[tid]
    emit('update:teamMembers', nextMembers)
  }
}

function onTabRemove(name: string | number) {
  const team = teamsWithKey.value.find((t) => t.key === String(name))
  if (team) removeTeam(team)
}
</script>

<template>
  <div class="workshop-team-assign">
    <div class="tabs-with-add">
      <el-tabs
        v-if="teamsWithKey.length"
        v-model="activeTeamKey"
        type="card"
        class="team-tabs"
        @tab-remove="onTabRemove"
      >
        <el-tab-pane
          v-for="team in teamsWithKey"
          :key="team.key"
          :label="team.name || team.code || '班组'"
          :name="team.key"
          closable
        >
          <p v-if="!workshopMemberIds.length" class="hint pane-hint">
            请先在「部门成员」中加入本车间人员，再分配班组成员。
          </p>
          <KanbanAssignPanel
            v-else-if="team.teamId > 0"
            :model-value="currentMemberIds(team)"
            :options="memberOptions"
            :get-id="(item) => item.id"
            left-title="未在本班"
            :right-title="`${team.name || '本班'}成员`"
            height="280px"
            compact
            :search-match="matchNameSearch"
            @update:model-value="setTeamMembers(team, $event)"
          >
            <template #list-head>
              <div class="member-grid member-grid-head">
                <span>工号</span>
                <span>姓名</span>
                <span>岗位</span>
                <span>备注</span>
              </div>
            </template>
            <template #item="{ item }">
              <div
                class="member-grid member-row"
                :title="[item.emp_no, item.name, item.job_title_name].filter(Boolean).join(' · ')"
              >
                <span class="col-no">{{ item.emp_no || '—' }}</span>
                <span class="col-name">{{ item.name }}</span>
                <span class="col-title">{{ item.job_title_name || '—' }}</span>
                <span class="col-meta">
                  <span
                    v-if="otherTeamLabel(item, team.teamId)"
                    class="tag-other"
                    :title="`当前在：${otherTeamLabel(item, team.teamId)}`"
                  >
                    他班
                  </span>
                </span>
              </div>
            </template>
          </KanbanAssignPanel>
          <p v-else class="hint pane-hint">新建班组需先保存车间，再次编辑即可分配成员。</p>
        </el-tab-pane>
      </el-tabs>
      <div v-else class="tabs-empty-bar" />

      <el-tooltip content="添加班组" placement="top">
        <el-button class="tabs-add-btn" type="primary" circle size="small" :icon="Plus" @click="openAddTeam" />
      </el-tooltip>
    </div>

    <p v-if="!teamsWithKey.length" class="hint empty-hint">暂无班组，点击右侧 + 添加。</p>

    <el-dialog v-model="addDlg" title="添加班组" width="400px" append-to-body destroy-on-close>
      <el-form label-width="72px" @submit.prevent="confirmAddTeam">
        <el-form-item label="编码">
          <el-input v-model="teamForm.code" placeholder="可选，空则自动生成" maxlength="32" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="teamForm.name" placeholder="班组名称" maxlength="64" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDlg = false">取消</el-button>
        <el-button type="primary" @click="confirmAddTeam">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.workshop-team-assign {
  width: 100%;
}
.tabs-with-add {
  display: flex;
  align-items: flex-start;
  gap: 6px;
}
.team-tabs {
  flex: 1;
  min-width: 0;
}
.team-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}
.tabs-add-btn {
  flex-shrink: 0;
  margin-top: 2px;
}
.tabs-empty-bar {
  flex: 1;
  min-height: 32px;
}
.pane-hint {
  padding: 8px 0;
}
.hint {
  margin: 0;
  color: #8a9aa3;
  font-size: 12px;
  line-height: 1.5;
}
.empty-hint {
  margin-top: 6px;
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
</style>
