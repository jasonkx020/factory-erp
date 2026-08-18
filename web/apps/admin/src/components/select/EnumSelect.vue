<script setup lang="ts">
import { computed, type StyleValue } from 'vue'
import type { FormOption } from '@erp/shared'

const props = withDefaults(
  defineProps<{
    modelValue?: string | null
    options: FormOption[]
    placeholder?: string
    clearable?: boolean
    filterable?: boolean
    disabled?: boolean
    style?: StyleValue
  }>(),
  {
    placeholder: '请选择',
    clearable: true,
    filterable: false,
    disabled: false,
    style: 'width:160px',
  },
)

const emit = defineEmits<{ 'update:modelValue': [string | null] }>()

const inner = computed({
  get: () => props.modelValue ?? null,
  set: (v) => emit('update:modelValue', v),
})
</script>

<template>
  <el-select
    v-model="inner"
    :placeholder="placeholder"
    :clearable="clearable"
    :filterable="filterable"
    :disabled="disabled"
    :style="style"
  >
    <el-option v-for="o in options" :key="o.value" :label="o.label" :value="o.value" />
  </el-select>
</template>
