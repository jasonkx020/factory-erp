<script setup lang="ts">
import RefSelect from './RefSelect.vue'
import { loadCustomers } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: string
  }>(),
  { placeholder: '选择客户', clearable: true, style: 'width:180px' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadCustomers"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
