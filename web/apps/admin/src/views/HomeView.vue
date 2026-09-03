<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePermStore, approvalApi, inventoryApi, productionApi, adminSpecialPath } from '@erp/shared'

const perm = usePermStore()
const router = useRouter()
const stats = ref({ tasks: 0, approval: 0, balances: 0 })

const processChain = ref<Array<{ name: string; run?: boolean }>>([])
const processChainHint = ref('未配置工艺')

onMounted(async () => {
  const [t, a, b, line] = await Promise.all([
    productionApi.listTasks(),
    approvalApi.tasks(),
    inventoryApi.balances(),
    productionApi.plantLinePreview(),
  ])
  stats.value.tasks = ((t.data as { list?: unknown[] })?.list || []).length
  stats.value.approval = ((a.data as { list?: unknown[] })?.list || []).length
  stats.value.balances = ((b.data as { list?: unknown[] })?.list || []).length
  const steps = (line.data as { steps?: Array<{ name: string; run?: boolean }>; message?: string })?.steps
  if (Array.isArray(steps) && steps.length) {
    processChain.value = steps.map((s) => ({ name: s.name, run: !!s.run }))
    processChainHint.value = ''
  } else {
    processChain.value = []
    processChainHint.value = (line.data as { message?: string })?.message || '未配置工艺'
  }
})

function open(domain: string, module: string) {
  const special = adminSpecialPath(domain, module)
  if (special) {
    router.push(special)
    return
  }
  router.push(`/m/${encodeURIComponent(domain)}/${encodeURIComponent(module)}`)
}
</script>

<template>
  <div class="home">
    <header class="hero factory-panel">
      <div class="hero-main">
        <div class="hero-top">
          <p class="eyebrow">木薯加工厂 · 产线管控台</p>
          <span class="factory-status run">产线运行</span>
        </div>
        <h2>管理端工作台</h2>
        <p class="desc">鲜薯入厂 → 工艺流程加工 → 成品入库。菜单对应业务域；现场领退料在员工 App 完成。工艺在「工艺流程」维护，开工时选择。</p>
      </div>
      <div class="factory-conveyor" aria-label="工艺步骤预览">
        <template v-if="processChain.length">
          <div
            v-for="(s, i) in processChain"
            :key="s.name"
            class="node"
            :class="{ active: s.run }"
          >
            <div class="seq">S{{ i + 1 }}</div>
            <div class="nm">{{ s.name }}</div>
          </div>
        </template>
        <div v-else class="node">
          <div class="seq">—</div>
          <div class="nm">{{ processChainHint }}</div>
        </div>
      </div>
    </header>

    <div class="stats">
      <div class="factory-kpi">
        <div class="label">生产任务</div>
        <div class="value">{{ stats.tasks }}</div>
      </div>
      <div class="factory-kpi warn">
        <div class="label">审批待办</div>
        <div class="value">{{ stats.approval }}</div>
      </div>
      <div class="factory-kpi ok">
        <div class="label">库存行</div>
        <div class="value">{{ stats.balances }}</div>
      </div>
      <div class="factory-kpi idle click" @click="open('系统管理','业务闭环')">
        <div class="label">业务闭环</div>
        <div class="value" style="font-size:16px">演示</div>
      </div>
    </div>

    <section class="factory-panel quick-panel">
      <div class="factory-panel-head">
        <h3 class="title">权限中心快捷入口</h3>
      </div>
      <div class="factory-panel-body quick-actions">
        <el-button type="primary" @click="open('人事管理','角色管理')">角色管理</el-button>
        <el-button @click="open('系统管理','自定义权限')">自定义权限</el-button>
        <el-button @click="open('系统管理','自定义菜单')">自定义菜单</el-button>
        <el-button @click="open('系统管理','登录控制')">登录控制</el-button>
        <el-button @click="open('系统管理','账户冻结')">账户冻结</el-button>
      </div>
    </section>

    <h3 class="section-title">业务功能域</h3>
    <div class="stats domains">
      <div
        v-for="d in perm.visibleMenus"
        :key="d.domain"
        class="domain-card"
        @click="open(d.domain, d.modules[0])"
      >
        <div class="label">{{ d.domain }}</div>
        <div class="mods">{{ d.modules.length }} 模块</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home { max-width: 1100px; }
.hero {
  margin-bottom: 16px;
  padding: 16px 16px 14px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.hero-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.eyebrow {
  margin: 0;
  font-size: 12px;
  letter-spacing: 0.08em;
  color: var(--accent, #1f7a4d);
  font-weight: 700;
  text-transform: uppercase;
}
h2 {
  margin: 8px 0 6px;
  font-size: 22px;
  color: var(--sidebar, #14352a);
}
.desc {
  margin: 0;
  max-width: 56ch;
  color: var(--muted, #5a6e64);
  font-size: 13px;
  line-height: 1.5;
}
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.factory-kpi.click { cursor: pointer; }
.factory-kpi.click:hover { border-color: var(--accent, #1f7a4d); }
.quick-panel { margin-bottom: 18px; }
.quick-actions { display: flex; flex-wrap: wrap; gap: 8px; }
.section-title {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 700;
  color: var(--sidebar, #14352a);
  letter-spacing: 0.02em;
}
.domain-card {
  background: var(--panel, #fff);
  border: 1px solid var(--border, #cfdcd4);
  border-radius: 8px;
  padding: 12px 14px;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.domain-card:hover {
  border-color: var(--accent, #1f7a4d);
  box-shadow: var(--shadow-soft);
}
.domain-card .label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text, #1a2e24);
}
.domain-card .mods {
  margin-top: 6px;
  font-size: 12px;
  color: var(--muted, #5a6e64);
  font-variant-numeric: tabular-nums;
}
@media (max-width: 768px) {
  .stats { grid-template-columns: repeat(2, 1fr); gap: 8px; }
}
</style>
