<script setup lang="ts">
import RefSelect from './RefSelect.vue'
import { loadEmployees } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    disabled?: boolean
    style?: string
  }>(),
  { placeholder: '选择员工（可搜索）', clearable: true, disabled: false, style: 'width:180px' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()

function employeeLabel(row: Record<string, unknown>) {
  const name = String(row.name ?? '')
  const no = String(row.emp_no ?? row.code ?? '')
  if (name && no) return `${name} (${no})`
  return name || no || String(row.id ?? '')
}
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadEmployees"
    :label-fn="employeeLabel"
    :placeholder="placeholder"
    :clearable="clearable"
    :disabled="disabled"
    :style="style"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
