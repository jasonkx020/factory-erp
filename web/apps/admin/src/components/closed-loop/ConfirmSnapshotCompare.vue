<script setup lang="ts">
defineProps<{
  imageUrl?: string
  title?: string
  draft?: Record<string, unknown> | null
  fields: { key: string; label: string; highlight?: boolean }[]
  modelValue: Record<string, unknown>
}>()
defineEmits<{ 'update:modelValue': [Record<string, unknown>] }>()
</script>

<template>
  <div class="confirm-side">
    <div class="pane photo">
      <div class="pane-title">{{ title || '原图证据' }}</div>
      <img v-if="imageUrl" :src="imageUrl" alt="evidence" class="img" />
      <div v-else class="empty">暂无照片 — 请先上传必填证据</div>
      <div v-if="draft && Object.keys(draft).length" class="draft">
        <div class="pane-title">OCR/自动草稿（仅供对照）</div>
        <pre>{{ JSON.stringify(draft, null, 2) }}</pre>
      </div>
    </div>
    <div class="pane fields">
      <div class="pane-title">确认数值</div>
      <el-form label-width="96px">
        <el-form-item
          v-for="f in fields"
          :key="f.key"
          :label="f.label"
          :class="{ hi: f.highlight }"
        >
          <el-input
            :model-value="String(modelValue[f.key] ?? '')"
            @update:model-value="(v: string) => $emit('update:modelValue', { ...modelValue, [f.key]: v })"
          />
        </el-form-item>
        <slot />
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.confirm-side {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  align-items: start;
}
@media (max-width: 900px) {
  .confirm-side { grid-template-columns: 1fr; }
}
.pane {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px;
  background: #fafbfc;
}
.pane-title { font-size: 13px; color: #556; margin-bottom: 8px; }
.img { width: 100%; max-height: 320px; object-fit: contain; background: #111; border-radius: 4px; }
.empty { color: #999; font-size: 13px; padding: 40px 0; text-align: center; }
.draft { margin-top: 12px; }
.draft pre { font-size: 11px; background: #fff; padding: 8px; border-radius: 4px; max-height: 120px; overflow: auto; }
.hi :deep(.el-form-item__label) { color: #b45309; font-weight: 600; }
</style>
