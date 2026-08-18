<script setup lang="ts">
export type MobileCardColumn = {
  prop: string
  label: string
  /** Use as card title when no #title slot */
  primary?: boolean
  hideOnCard?: boolean
  format?: (value: unknown, row: Record<string, unknown>) => string
}

const props = withDefaults(
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

const primaryCol = () => props.columns.find((c) => c.primary) || props.columns.find((c) => !c.hideOnCard)
const fieldCols = () =>
  props.columns.filter((c) => !c.hideOnCard && c.prop !== primaryCol()?.prop)

function cell(row: Record<string, unknown>, col: MobileCardColumn) {
  const v = row[col.prop]
  if (col.format) return col.format(v, row)
  if (v == null || v === '') return '—'
  return String(v)
}
</script>

<template>
  <div v-loading="loading" class="mobile-cards">
    <div v-if="!loading && (!data || data.length === 0)" class="mobile-cards-empty">
      {{ emptyText }}
    </div>
    <article v-for="(row, idx) in data" :key="(row.id as string | number) ?? idx" class="mobile-card">
      <header class="mobile-card-head">
        <div class="mobile-card-title">
          <slot name="title" :row="row">
            {{ primaryCol() ? cell(row, primaryCol()!) : `#${idx + 1}` }}
          </slot>
        </div>
        <div v-if="$slots.extra" class="mobile-card-extra">
          <slot name="extra" :row="row" />
        </div>
      </header>
      <dl class="mobile-card-fields">
        <template v-for="col in fieldCols()" :key="col.prop">
          <div class="mobile-card-field">
            <dt>{{ col.label }}</dt>
            <dd>
              <slot :name="`field-${col.prop}`" :row="row" :value="row[col.prop]">
                {{ cell(row, col) }}
              </slot>
            </dd>
          </div>
        </template>
      </dl>
      <footer v-if="$slots.actions" class="mobile-card-actions">
        <slot name="actions" :row="row" />
      </footer>
    </article>
  </div>
</template>

<style scoped>
.mobile-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 48px;
}
.mobile-cards-empty {
  padding: 28px 12px;
  text-align: center;
  color: #98a2a8;
  font-size: 13px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e2e8ec;
}
.mobile-card {
  background: #fff;
  border: 1px solid #e2e8ec;
  border-radius: 8px;
  padding: 12px;
}
.mobile-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.mobile-card-title {
  font-size: 15px;
  font-weight: 600;
  color: #1a252f;
  word-break: break-word;
  min-width: 0;
}
.mobile-card-extra {
  flex-shrink: 0;
  font-size: 12px;
}
.mobile-card-fields {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.mobile-card-field {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 8px;
  font-size: 13px;
  line-height: 1.4;
}
.mobile-card-field dt {
  margin: 0;
  color: #8a969e;
}
.mobile-card-field dd {
  margin: 0;
  color: #2c3e50;
  word-break: break-word;
}
.mobile-card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #eef1f4;
}
</style>
