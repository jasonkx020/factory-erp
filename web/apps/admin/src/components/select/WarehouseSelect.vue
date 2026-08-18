<script setup lang="ts">
import type { StyleValue } from 'vue'
import RefSelect from './RefSelect.vue'
import { loadWarehouses } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
    allowZero?: boolean
    zeroLabel?: string
  }>(),
  {
    placeholder: '选择仓库',
    clearable: true,
    style: 'width:180px',
    allowZero: false,
    zeroLabel: '全部',
  },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadWarehouses"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    :allow-zero="allowZero"
    :zero-label="zeroLabel"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
