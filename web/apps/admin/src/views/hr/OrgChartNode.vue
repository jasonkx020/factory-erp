<script setup lang="ts">
import { computed } from 'vue'

type Row = Record<string, unknown>
type TreeNode = Row & { children?: TreeNode[] }

const props = defineProps<{
  node: TreeNode
  isRoot?: boolean
}>()

const emit = defineEmits<{
  action: [action: string, node: Row]
}>()

const level = computed(() => Number(props.node.level || 1))
const levelClass = computed(() => `level-${level.value}`)
const isWorkshop = computed(() => String(props.node.dept_type || '') === 'workshop')
const hasChildren = computed(() => (props.node.children?.length || 0) > 0)

function act(action: string) {
  emit('action', action, props.node)
}
</script>

<template>
  <div class="branch" :class="{ 'is-root': isRoot }">
    <div class="card" :class="[levelClass, { workshop: isWorkshop }]" @click="act('detail')">
      <div class="card-head">
        <el-tag size="small" :type="level === 1 ? 'primary' : level === 2 ? 'success' : 'warning'">
          {{ node.level_label || `L${level}` }}
        </el-tag>
        <el-tag v-if="isWorkshop" size="small" type="danger" effect="plain">车间</el-tag>
        <el-tag v-if="node.status !== 'active'" size="small" type="info">停用</el-tag>
      </div>
      <div class="card-name">{{ node.name }}</div>
      <div class="card-code">{{ node.code }}</div>
      <div v-if="node.parent_name" class="card-parent">
        <span class="parent-arrow">↑</span> {{ node.parent_name }}
      </div>
      <div v-else-if="!isRoot && level > 1" class="card-parent muted">上级未标注</div>
      <div class="card-stats">
        <span v-if="hasChildren">{{ node.child_count ?? node.children?.length ?? 0 }} 个子部门</span>
        <span>{{ node.member_count ?? 0 }} 人</span>
        <span>{{ node.effective_role_count ?? 0 }} 项权限</span>
      </div>
      <div class="card-actions" @click.stop>
        <el-button v-if="level < 3 && !isWorkshop" link type="primary" size="small" @click="act('child')">+ 子部门</el-button>
        <el-button link type="primary" size="small" @click="act('edit')">编辑</el-button>
        <el-button v-if="node.status === 'active'" link size="small" @click="act('deactivate')">停用</el-button>
        <el-button link type="danger" size="small" @click="act('remove')">删除</el-button>
      </div>
    </div>

    <div v-if="hasChildren" class="children-wrap">
      <div class="connector-down" />
      <div class="children-row">
        <div v-for="child in node.children" :key="String(child.id)" class="child-slot">
          <div class="connector-up" />
          <OrgChartNode :node="child" @action="(a, n) => emit('action', a, n)" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.branch {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
}

.card {
  min-width: 168px;
  max-width: 220px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 2px solid #d5dde3;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  cursor: pointer;
  transition: box-shadow 0.15s, border-color 0.15s;
}

.card:hover {
  box-shadow: 0 4px 14px rgba(22, 119, 255, 0.12);
  border-color: #91caff;
}

.card.level-1 { border-color: #91caff; background: linear-gradient(180deg, #f0f7ff 0%, #fff 100%); }
.card.level-2 { border-color: #b7eb8f; background: linear-gradient(180deg, #f6ffed 0%, #fff 100%); }
.card.level-3 { border-color: #ffe58f; background: linear-gradient(180deg, #fffbe6 0%, #fff 100%); }
.card.workshop { border-color: #d3adf7; background: linear-gradient(180deg, #f9f0ff 0%, #fff 100%); }

.card-head { display: flex; gap: 6px; margin-bottom: 6px; }
.card-name { font-size: 15px; font-weight: 700; color: #1f2933; line-height: 1.3; margin-bottom: 2px; }
.card-code { font-size: 11px; color: #8a9aa3; margin-bottom: 6px; }
.card-parent { font-size: 11px; color: #5c6b75; margin-bottom: 6px; padding: 3px 6px; background: rgba(0,0,0,0.03); border-radius: 4px; }
.card-parent.muted { color: #b0bac2; }
.parent-arrow { color: #1677ff; font-weight: 700; }
.card-stats { display: flex; flex-wrap: wrap; gap: 6px; font-size: 11px; color: #8a9aa3; margin-bottom: 6px; }
.card-actions { display: flex; flex-wrap: wrap; gap: 2px; border-top: 1px dashed #e8eef2; padding-top: 6px; }

.children-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  margin-top: 0;
}

.connector-down {
  width: 2px;
  height: 24px;
  background: #b0bac2;
}

.children-row {
  display: flex;
  gap: 20px;
  position: relative;
  padding-top: 0;
}

.children-row::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: #b0bac2;
}

.child-slot {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  padding-top: 0;
}

.connector-up {
  width: 2px;
  height: 24px;
  background: #b0bac2;
  margin-bottom: 0;
}

/* 单个子节点时横线不超出 */
.children-row:has(.child-slot:only-child)::before {
  left: 50%;
  right: 50%;
}

.branch.is-root + .branch.is-root {
  margin-left: 0;
}
</style>
