<script setup lang="ts">
import type { StyleValue } from 'vue'
import RefSelect from './RefSelect.vue'
import { loadProdTasks, taskLabel } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | number[] | null
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
    multiple?: boolean
  }>(),
  {
    placeholder: '选择生产任务',
    clearable: true,
    style: 'width:220px',
    multiple: false,
  },
)

defineEmits<{ 'update:modelValue': [number | number[] | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadProdTasks"
    :label-fn="taskLabel"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    :multiple="multiple"
    @update:model-value="$emit('update:modelValue', $event)"
  />
</template>
