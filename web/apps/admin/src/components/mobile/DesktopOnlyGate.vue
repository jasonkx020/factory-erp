<script setup lang="ts">
import { useIsMobile } from '../../composables/useMediaQuery'

withDefaults(
  defineProps<{
    title?: string
    message?: string
  }>(),
  {
    title: '请使用桌面浏览器',
    message: '该功能需在桌面浏览器操作。',
  },
)

const isMobile = useIsMobile()
</script>

<template>
  <div v-if="isMobile" class="desktop-only-gate">
    <div class="gate-card">
      <div class="gate-title">{{ title }}</div>
      <p class="gate-msg">{{ message }}</p>
      <slot name="hint" />
    </div>
  </div>
  <slot v-else />
</template>

<style scoped>
.desktop-only-gate {
  padding: 8px 0;
}
.gate-card {
  background: #fff;
  border: 1px solid #e2e8ec;
  border-radius: 8px;
  padding: 28px 20px;
  text-align: center;
}
.gate-title {
  font-size: 16px;
  font-weight: 600;
  color: #2c3e50;
  margin-bottom: 8px;
}
.gate-msg {
  margin: 0;
  font-size: 13px;
  color: #5c6b75;
  line-height: 1.5;
}
</style>
