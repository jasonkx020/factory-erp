<script setup lang="ts">
import { computed, onMounted, ref, type StyleValue } from 'vue'

type Row = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    modelValue?: number | number[] | null
    load: () => Promise<Row[]>
    labelKey?: string
    valueKey?: string
    labelFn?: (row: Row) => string
    placeholder?: string
    clearable?: boolean
    filterable?: boolean
    multiple?: boolean
    disabled?: boolean
    style?: StyleValue
    allowZero?: boolean
    zeroLabel?: string
  }>(),
  {
    labelKey: 'name',
    valueKey: 'id',
    placeholder: '请选择',
    clearable: true,
    filterable: true,
    multiple: false,
    disabled: false,
    style: 'width:180px',
    allowZero: false,
    zeroLabel: '全部',
  },
)

const emit = defineEmits<{ 'update:modelValue': [number | number[] | null] }>()

const options = ref<Row[]>([])
const loading = ref(false)

async function refresh() {
  loading.value = true
  try {
    options.value = (await props.load()) || []
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
// load 函数引用稳定即可；不监听 load 避免重复请求


function rowLabel(row: Row) {
  if (props.labelFn) return props.labelFn(row)
  const name = row[props.labelKey]
  const code = row.code
  if (name != null && code != null && String(code)) return `${name} (${code})`
  if (name != null) return String(name)
  return String(row[props.valueKey] ?? '')
}

const inner = computed({
  get: () => props.modelValue ?? (props.multiple ? [] : null),
  set: (v) => emit('update:modelValue', v as number | number[] | null),
})
</script>

<template>
  <el-select
    v-model="inner"
    :placeholder="placeholder"
    :clearable="clearable"
    :filterable="filterable"
    :multiple="multiple"
    :disabled="disabled"
    :loading="loading"
    :style="style"
  >
    <el-option v-if="allowZero && !multiple" :label="zeroLabel" :value="0" />
    <el-option
      v-for="row in options"
      :key="String(row[valueKey])"
      :label="rowLabel(row)"
      :value="Number(row[valueKey])"
    />
  </el-select>
</template>
