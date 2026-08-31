<script setup lang="ts">
import { computed, onMounted, ref, watch, type StyleValue } from 'vue'
import { ElMessage } from 'element-plus'
import { hrApi } from '@erp/shared'
import { loadJobTitles, ensureJobTitle } from './entitySelects'

type Row = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    modelValue?: number | null
    empType?: string
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
    allowZero?: boolean
    zeroLabel?: string
    disabled?: boolean
  }>(),
  {
    placeholder: '选择岗位',
    clearable: true,
    style: 'width:180px',
    allowZero: true,
    zeroLabel: '未设置',
    disabled: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [number | null] }>()

const options = ref<Row[]>([])
const loading = ref(false)
const selectVal = ref<string | number>('')

async function refresh() {
  loading.value = true
  try {
    options.value = (await loadJobTitles(props.empType)) || []
    const cur = Number(props.modelValue) || 0
    if (cur > 0 && !options.value.some((r) => Number(r.id) === cur)) {
      const detail = await hrApi.getJobTitle(cur)
      if (detail.code === 1 && detail.data) {
        options.value = [detail.data as Row, ...options.value]
      }
    }
    selectVal.value = cur > 0 ? cur : props.allowZero ? 0 : ''
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
watch(() => props.empType, refresh)
watch(
  () => props.modelValue,
  (v) => {
    const cur = Number(v) || 0
    selectVal.value = cur > 0 ? cur : props.allowZero ? 0 : ''
  },
)

async function onCreate(name: string) {
  const trimmed = name.trim()
  if (!trimmed) return
  const res = await ensureJobTitle(trimmed, props.empType)
  if (!res) {
    ElMessage.error('岗位入库失败')
    return
  }
  await refresh()
  const id = Number(res.id) || 0
  selectVal.value = id
  emit('update:modelValue', id > 0 ? id : null)
}

watch(selectVal, (v) => {
  if (typeof v === 'string') {
    void onCreate(v)
    return
  }
  const n = Number(v) || 0
  emit('update:modelValue', n > 0 ? n : props.allowZero ? 0 : null)
})

const inner = computed({
  get: () => selectVal.value,
  set: (v) => {
    selectVal.value = v ?? (props.allowZero ? 0 : '')
  },
})
</script>

<template>
  <el-select
    v-model="inner"
    :placeholder="placeholder"
    :clearable="clearable"
    filterable
    allow-create
    default-first-option
    :loading="loading"
    :disabled="disabled"
    :style="style"
  >
    <el-option v-if="allowZero" :label="zeroLabel" :value="0" />
    <el-option
      v-for="row in options"
      :key="String(row.id)"
      :label="String(row.name || row.id)"
      :value="Number(row.id)"
    />
  </el-select>
</template>
