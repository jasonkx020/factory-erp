<script setup lang="ts">
import type { StyleValue } from 'vue'
import RefSelect from './RefSelect.vue'
import { loadSalesOrders, orderLabel } from './entitySelects'

withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
  }>(),
  { placeholder: '选择销售订单', clearable: true, style: 'width:220px' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="loadSalesOrders"
    :label-fn="orderLabel"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
