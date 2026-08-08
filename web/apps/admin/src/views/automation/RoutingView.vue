<script setup lang="ts">
import FlowGraphEditorView from './FlowGraphEditorView.vue'
</script>

<template>
  <div class="page">
    <h2>工艺流程编排</h2>
    <p class="sub">拖拽节点连线配置生产工序；保存生产类流程时自动编译为工艺步骤供报工引擎使用。</p>
    <FlowGraphEditorView kind-filter="production" />
  </div>
</template>

<style scoped>
.page { padding: 0 4px; }
.sub { color: #64748b; font-size: 13px; margin: 0 0 12px; }
</style>
