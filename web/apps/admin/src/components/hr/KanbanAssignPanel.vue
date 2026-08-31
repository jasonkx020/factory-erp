<script setup lang="ts" generic="T">
import { computed, ref } from 'vue'
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue'

type Side = 'left' | 'right'

const props = withDefaults(
  defineProps<{
    modelValue: number[]
    options: T[]
    getId: (item: T) => number
    leftTitle?: string
    rightTitle?: string
    height?: string
    compact?: boolean
    filterLeft?: (item: T) => boolean
    filterRight?: (item: T) => boolean
    searchMatch?: (item: T, query: string) => boolean
  }>(),
  {
    leftTitle: '可选',
    rightTitle: '已选',
    height: '360px',
    compact: false,
    filterLeft: undefined,
    filterRight: undefined,
    searchMatch: undefined,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const dragOverSide = ref<Side | null>(null)
const dragging = ref<{ id: number; from: Side } | null>(null)
const leftSearch = ref('')
const rightSearch = ref('')

const selectedSet = computed(() => new Set(props.modelValue))

function matchColumnSearch(item: T, query: string) {
  const q = query.trim()
  if (!q) return true
  if (props.searchMatch) return props.searchMatch(item, q)
  return true
}

const availableList = computed(() =>
  props.options.filter((item) => {
    const id = props.getId(item)
    if (selectedSet.value.has(id)) return false
    if (props.filterLeft && !props.filterLeft(item)) return false
    if (!matchColumnSearch(item, leftSearch.value)) return false
    return true
  }),
)

const selectedList = computed(() => {
  const byId = new Map(props.options.map((item) => [props.getId(item), item]))
  return props.modelValue
    .map((id) => byId.get(id))
    .filter((item): item is T => !!item)
    .filter((item) => (props.filterRight ? props.filterRight(item) : true))
    .filter((item) => matchColumnSearch(item, rightSearch.value))
})

function emitIds(next: number[]) {
  emit('update:modelValue', next)
}

function addId(id: number) {
  if (id <= 0 || selectedSet.value.has(id)) return
  emitIds([...props.modelValue, id])
}

function removeId(id: number) {
  if (!selectedSet.value.has(id)) return
  emitIds(props.modelValue.filter((v) => v !== id))
}

function toggleId(id: number, from: Side) {
  if (from === 'left') addId(id)
  else removeId(id)
}

function onDragStart(id: number, from: Side, e: DragEvent) {
  dragging.value = { id, from }
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(id))
  }
}

function onDragEnd() {
  dragging.value = null
  dragOverSide.value = null
}

function onDragOver(side: Side, e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dragOverSide.value = side
}

function onDragLeave(side: Side) {
  if (dragOverSide.value === side) dragOverSide.value = null
}

function onDrop(side: Side, e: DragEvent) {
  e.preventDefault()
  const raw = dragging.value || { id: Number(e.dataTransfer?.getData('text/plain')), from: null as Side | null }
  const id = raw.id
  const from = raw.from
  dragOverSide.value = null
  dragging.value = null
  if (!id) return
  if (side === 'right' && from === 'left') addId(id)
  else if (side === 'left' && from === 'right') removeId(id)
}

function moveToRight(id: number) {
  addId(id)
}

function moveToLeft(id: number) {
  removeId(id)
}
</script>

<template>
  <div class="kanban-assign" :class="{ 'is-compact': compact }">
    <div v-if="$slots.filters" class="kanban-filters">
      <slot name="filters" />
    </div>

    <div class="kanban-summary">
      已选 {{ modelValue.length }} / 共 {{ options.length }}
      <span class="kanban-tip">拖拽或双击移动 · 悬停显示操作按钮</span>
    </div>

    <div class="kanban-columns">
      <div
        class="kanban-column"
        :class="{ 'is-drag-over': dragOverSide === 'left' }"
        @dragover="onDragOver('left', $event)"
        @dragleave="onDragLeave('left')"
        @drop="onDrop('left', $event)"
      >
        <div class="kanban-column-head">
          <span class="kanban-column-title">{{ leftTitle }}</span>
          <el-input
            v-if="searchMatch"
            v-model="leftSearch"
            size="small"
            clearable
            class="kanban-column-search"
            placeholder="搜姓名"
            @click.stop
          />
          <el-tag size="small" type="info" effect="plain">{{ availableList.length }}</el-tag>
        </div>
        <div v-if="$slots['list-head']" class="kanban-list-head">
          <slot name="list-head" />
        </div>
        <div class="kanban-list" :style="{ height }">
          <div v-if="!availableList.length" class="kanban-empty">
            {{ leftSearch.trim() ? '无匹配人员' : '暂无可选项' }}
          </div>
          <div
            v-for="item in availableList"
            :key="getId(item)"
            class="kanban-card"
            draggable="true"
            @dragstart="onDragStart(getId(item), 'left', $event)"
            @dragend="onDragEnd"
            @dblclick="toggleId(getId(item), 'left')"
          >
            <div class="kanban-card-body">
              <slot name="item" :item="item" side="left" />
            </div>
            <el-button
              class="kanban-card-action"
              link
              type="primary"
              :icon="ArrowRight"
              title="加入"
              @click.stop="moveToRight(getId(item))"
            />
          </div>
        </div>
      </div>

      <div
        class="kanban-column kanban-column-selected"
        :class="{ 'is-drag-over': dragOverSide === 'right' }"
        @dragover="onDragOver('right', $event)"
        @dragleave="onDragLeave('right')"
        @drop="onDrop('right', $event)"
      >
        <div class="kanban-column-head">
          <span class="kanban-column-title">{{ rightTitle }}</span>
          <el-input
            v-if="searchMatch"
            v-model="rightSearch"
            size="small"
            clearable
            class="kanban-column-search"
            placeholder="搜姓名"
            @click.stop
          />
          <el-tag size="small" type="success" effect="plain">{{ selectedList.length }}</el-tag>
        </div>
        <div v-if="$slots['list-head']" class="kanban-list-head">
          <slot name="list-head" />
        </div>
        <div class="kanban-list" :style="{ height }">
          <div v-if="!selectedList.length" class="kanban-empty">
            {{ rightSearch.trim() ? '无匹配人员' : '拖入或双击左侧行添加' }}
          </div>
          <div
            v-for="item in selectedList"
            :key="getId(item)"
            class="kanban-card is-selected"
            draggable="true"
            @dragstart="onDragStart(getId(item), 'right', $event)"
            @dragend="onDragEnd"
            @dblclick="toggleId(getId(item), 'right')"
          >
            <el-button
              class="kanban-card-action"
              link
              type="danger"
              :icon="ArrowLeft"
              title="移出"
              @click.stop="moveToLeft(getId(item))"
            />
            <div class="kanban-card-body">
              <slot name="item" :item="item" side="right" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kanban-assign {
  width: 100%;
}
.kanban-filters {
  margin-bottom: 8px;
}
.kanban-summary {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
  font-size: 12px;
  color: #5c6b75;
}
.kanban-tip {
  color: #8a9aa3;
}
.kanban-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.kanban-column {
  border: 1px solid #e8eef2;
  border-radius: 6px;
  background: #fafcfd;
  transition: border-color 0.15s, box-shadow 0.15s;
  min-width: 0;
}
.kanban-column.is-drag-over {
  border-color: #1677ff;
  box-shadow: 0 0 0 2px rgba(22, 119, 255, 0.12);
}
.kanban-column-selected {
  background: #f6ffed;
  border-color: #d9f7be;
}
.kanban-column-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-bottom: 1px solid #e8eef2;
  font-size: 12px;
  font-weight: 600;
  color: #334;
}
.kanban-column-title {
  flex-shrink: 0;
  white-space: nowrap;
}
.kanban-column-search {
  flex: 1;
  min-width: 0;
}
.kanban-column-search :deep(.el-input__wrapper) {
  padding: 0 6px;
}
.kanban-list-head {
  padding: 4px 8px 0;
  border-bottom: 1px solid #eef2f5;
  background: #f3f6f8;
}
.kanban-list {
  overflow-y: auto;
  padding: 4px;
}
.kanban-empty {
  padding: 16px 8px;
  text-align: center;
  color: #8a9aa3;
  font-size: 12px;
}
.kanban-card {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 6px 8px;
  margin-bottom: 4px;
  border: 1px solid #e8eef2;
  border-radius: 4px;
  background: #fff;
  cursor: grab;
  user-select: none;
  transition: border-color 0.15s, background 0.15s;
}
.kanban-card:last-child {
  margin-bottom: 0;
}
.kanban-card:hover {
  border-color: #91caff;
  background: #f0f7ff;
}
.kanban-card.is-selected {
  background: #fff;
  border-color: #d9f7be;
}
.kanban-card.is-selected:hover {
  background: #f6ffed;
}
.kanban-card:active {
  cursor: grabbing;
}
.kanban-card-body {
  flex: 1;
  min-width: 0;
}
.kanban-card-action {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.12s;
}
.kanban-card:hover .kanban-card-action {
  opacity: 1;
}

/* 紧凑模式：单行表格风，一屏可浏览更多人员 */
.is-compact .kanban-column-head {
  padding: 4px 6px;
  gap: 4px;
}
.is-compact .kanban-list {
  padding: 2px 4px;
}
.is-compact .kanban-card {
  padding: 0 4px;
  margin-bottom: 1px;
  min-height: 26px;
  border-radius: 2px;
  border-color: transparent;
  background: transparent;
}
.is-compact .kanban-card:nth-child(odd) {
  background: rgba(255, 255, 255, 0.7);
}
.is-compact .kanban-card:hover {
  border-color: #91caff;
  background: #e6f4ff;
}
.is-compact .kanban-card.is-selected:nth-child(odd) {
  background: rgba(255, 255, 255, 0.85);
}
.is-compact .kanban-card.is-selected:hover {
  background: #f6ffed;
  border-color: #b7eb8f;
}
.is-compact .kanban-card-action {
  width: 22px;
  height: 22px;
  padding: 0;
  margin: 0;
}
.is-compact .kanban-list-head {
  padding: 2px 4px 0;
}
</style>
