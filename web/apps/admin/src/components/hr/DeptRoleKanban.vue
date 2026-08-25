<script setup lang="ts">
import { computed, ref } from 'vue'
import KanbanAssignPanel from './KanbanAssignPanel.vue'

type Row = Record<string, unknown>

export type RoleOption = {
  id: number
  name: string
  code: string
}

const props = defineProps<{
  modelValue: number[]
  roles: Row[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const keyword = ref('')

const roleOptions = computed<RoleOption[]>(() =>
  props.roles.map((r) => ({
    id: Number(r.id),
    name: String(r.name || r.code || ''),
    code: String(r.code || ''),
  })),
)

function matchKeyword(item: RoleOption, q: string) {
  if (!q) return true
  const hay = `${item.name} ${item.code}`.toLowerCase()
  return hay.includes(q)
}

function filterLeft(item: RoleOption) {
  return matchKeyword(item, keyword.value.trim().toLowerCase())
}

function filterRight(item: RoleOption) {
  return matchKeyword(item, keyword.value.trim().toLowerCase())
}
</script>

<template>
  <div class="role-kanban-wrap">
    <KanbanAssignPanel
      :model-value="modelValue"
      :options="roleOptions"
      :get-id="(item) => item.id"
      left-title="可选角色"
      right-title="本级基础角色"
      height="200px"
      compact
      :filter-left="filterLeft"
      :filter-right="filterRight"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <template #filters>
        <el-input
          v-model="keyword"
          size="small"
          clearable
          placeholder="角色名称 / 编码"
          style="width: 100%"
        />
      </template>

      <template #list-head>
        <div class="role-grid role-grid-head">
          <span>角色名</span>
          <span>编码</span>
        </div>
      </template>

      <template #item="{ item }">
        <div class="role-grid role-row" :title="`${item.name} (${item.code})`">
          <span class="role-name">{{ item.name }}</span>
          <span class="role-code">{{ item.code }}</span>
        </div>
      </template>
    </KanbanAssignPanel>
    <p class="hint">上级部门会自动继承全部子部门角色；此处只配置本部门直属权限。</p>
  </div>
</template>

<style scoped>
.role-kanban-wrap {
  width: 100%;
}
.role-grid {
  display: grid;
  grid-template-columns: 1fr 88px;
  gap: 6px;
  align-items: center;
  width: 100%;
  font-size: 12px;
  line-height: 1.2;
}
.role-grid-head {
  padding: 2px 28px 4px 4px;
  color: #8a9aa3;
  font-size: 11px;
  font-weight: 500;
}
.role-row {
  min-height: 24px;
}
.role-name {
  font-weight: 600;
  color: #334;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.role-code {
  color: #8a9aa3;
  font-size: 11px;
  font-family: ui-monospace, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hint {
  margin: 6px 0 0;
  color: #8a9aa3;
  font-size: 12px;
  line-height: 1.5;
}
</style>
