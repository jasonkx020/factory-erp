<script setup lang="ts">
import FlowGraphEditorView from './FlowGraphEditorView.vue'
</script>

<template>
  <div class="page">
    <h2>工艺流程编排</h2>
    <p class="sub">维护生产工艺流程；列表中点「编辑」进入画布编排工序。发布后将编译为过站步骤。</p>
    <FlowGraphEditorView kind-filter="production" />
  </div>
</template>

<style scoped>
.page { padding: 0 4px; }
.sub { color: #64748b; font-size: 13px; margin: 0 0 12px; }
</style>
