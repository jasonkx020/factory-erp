<script setup lang="ts">
import type { StyleValue } from 'vue'
import RefSelect from './RefSelect.vue'
import { loadRoutings } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
  }>(),
  { placeholder: '选择工艺', clearable: true, style: 'width:180px' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadRoutings"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    label-key="name"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
