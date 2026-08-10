<script setup lang="ts">
/**
 * Desktop: render default slot (typically el-table).
 * Mobile: render MobileDataCards with provided columns / action slots.
 */
import { computed, useSlots } from 'vue'
import { useIsMobile } from '../../composables/useMediaQuery'
import MobileDataCards, { type MobileCardColumn } from './MobileDataCards.vue'

withDefaults(
  defineProps<{
    data: Record<string, unknown>[]
    loading?: boolean
    columns: MobileCardColumn[]
    emptyText?: string
  }>(),
  {
    loading: false,
    emptyText: '暂无数据',
  },
)

const isMobile = useIsMobile()
const slots = useSlots()
const forwardSlotNames = computed(() => Object.keys(slots).filter((n) => n !== 'default'))
</script>

<template>
  <div class="table-or-cards">
    <div v-show="!isMobile" class="desktop-table">
      <slot />
    </div>
    <MobileDataCards
      v-if="isMobile"
      :data="data"
      :loading="loading"
      :columns="columns"
      :empty-text="emptyText"
    >
      <template v-for="name in forwardSlotNames" :key="name" #[name]="scope">
        <slot :name="name" v-bind="scope || {}" />
      </template>
    </MobileDataCards>
  </div>
</template>
