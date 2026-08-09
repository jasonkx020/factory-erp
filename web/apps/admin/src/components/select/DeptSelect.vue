<script setup lang="ts">
import RefSelect from './RefSelect.vue'
import { loadDepartments } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: string
    allowZero?: boolean
    zeroLabel?: string
  }>(),
  {
    placeholder: '选择部门',
    clearable: true,
    style: 'width:180px',
    allowZero: false,
    zeroLabel: '未设置',
  },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadDepartments"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    :allow-zero="allowZero"
    :zero-label="zeroLabel"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
