<script setup lang="ts">
import { watch } from 'vue'
import { RouterView } from 'vue-router'
import { useAuthStore, useNotifyStore } from '@erp/shared'

const auth = useAuthStore()
const notify = useNotifyStore()

watch(
  () => auth.isLoggedIn,
  (ok) => {
    if (ok) void notify.start()
    else notify.stop()
  },
  { immediate: true },
)
</script>

<template>
  <RouterView />
</template>
