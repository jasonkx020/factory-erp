<script setup lang="ts">
import type { StyleValue } from 'vue'
import RefSelect from './RefSelect.vue'
import { loadProcesses } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    disabled?: boolean
    style?: StyleValue
  }>(),
  { placeholder: '选择工序', clearable: true, disabled: false, style: 'width:180px' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadProcesses"
    :placeholder="placeholder"
    :clearable="clearable"
    :disabled="disabled"
    :style="style"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
