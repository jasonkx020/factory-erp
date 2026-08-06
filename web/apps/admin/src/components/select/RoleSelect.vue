<script setup lang="ts">
import RefSelect from './RefSelect.vue'
import { loadRoles } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: string
  }>(),
  { placeholder: '选择角色', clearable: true, style: 'width:180px' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadRoles"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
