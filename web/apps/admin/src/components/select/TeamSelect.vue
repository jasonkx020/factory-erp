<script setup lang="ts">
import { computed, onMounted, ref, watch, type StyleValue } from 'vue'
import { loadWorkTeams } from './entitySelects'

type Row = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    modelValue?: number | null
    deptId?: number | null
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
    allowZero?: boolean
    zeroLabel?: string
  }>(),
  {
    placeholder: '选择班组',
    clearable: true,
    style: 'width:180px',
    allowZero: true,
    zeroLabel: '未设置',
  },
)

const emit = defineEmits<{ 'update:modelValue': [number | null] }>()

const options = ref<Row[]>([])
const loading = ref(false)

async function refresh() {
  loading.value = true
  try {
    const ws = Number(props.deptId) || 0
    options.value = (await loadWorkTeams(ws > 0 ? ws : undefined)) || []
    const cur = Number(props.modelValue) || 0
    if (cur > 0 && !options.value.some((r) => Number(r.id) === cur)) {
      emit('update:modelValue', props.allowZero ? 0 : null)
    }
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
watch(() => props.deptId, refresh)

const inner = computed({
  get: () => props.modelValue ?? (props.allowZero ? 0 : null),
  set: (v) => emit('update:modelValue', v as number | null),
})

function rowLabel(row: Row) {
  const name = row.name
  const code = row.code
  if (name != null && code != null && String(code)) return `${name} (${code})`
  if (name != null) return String(name)
  return String(row.id ?? '')
}
</script>

<template>
  <el-select
    v-model="inner"
    :placeholder="placeholder"
    :clearable="clearable"
    filterable
    :loading="loading"
    :style="style"
  >
    <el-option v-if="allowZero" :label="zeroLabel" :value="0" />
    <el-option
      v-for="row in options"
      :key="String(row.id)"
      :label="rowLabel(row)"
      :value="Number(row.id)"
    />
  </el-select>
</template>
