<script setup lang="ts">
import { computed } from 'vue'
import type { StyleValue } from 'vue'
import RefSelect from './RefSelect.vue'
import { loadSuppliers } from './entitySelects'

const props = withDefaults(
  defineProps<{
    modelValue?: number | null
    placeholder?: string
    clearable?: boolean
    style?: StyleValue
    /** person | enterprise；空则全部 */
    partyKind?: string
  }>(),
  { placeholder: '选择供应商', clearable: true, style: 'width:180px', partyKind: '' },
)

defineEmits<{ 'update:modelValue': [number | null] }>()

const load = computed(() => {
  const kind = props.partyKind
  return () => loadSuppliers(kind || undefined)
})
</script>

<template>
  <RefSelect
    :model-value="modelValue"
    :load="load"
    :placeholder="placeholder"
    :clearable="clearable"
    :style="style"
    @update:model-value="$emit('update:modelValue', $event as number | null)"
  />
</template>
