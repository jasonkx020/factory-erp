<script setup lang="ts">
import { terminalUrl } from '@erp/shared'

const cards = [
  {
    key: 'admin' as const,
    badge: '后台 · 管理端',
    title: '管理后台',
    desc: '13 域菜单与权限中心：采购入库、生产派工、库存过账、销售出库与 IAM。',
  },
  {
    key: 'workshop' as const,
    badge: '前台 · 车间 Pad',
    title: '车间工作台',
    desc: '产线节点：清洗 → 去皮计件 → 收货卡点 → 切断去芯 → 切块装袋。',
  },
  {
    key: 'worker' as const,
    badge: '前台 · 工人端',
    title: '工人端',
    desc: '去皮 / 去芯 / 切块扫码报工、联动领料、本人工资与考勤。',
  },
  {
    key: 'sales' as const,
    badge: '前台 · 销售端',
    title: '销售外勤端',
    desc: '袋装木薯丁等成品：下单协同、询价、客户跟进与任务提醒。',
  },
  {
    key: 'boss' as const,
    badge: '前台 · 看板',
    title: '老板驾驶舱',
    desc: '驾驶舱 · 生产看板 · 产线实况 · 经营报表只读。',
  },
  {
    key: 'customer' as const,
    badge: '用户端 · 自助',
    title: '客户自助端',
    desc: '自助下单、询价、我的订单、复购与发货进度。',
  },
]

const flow = ['鲜木薯入库', '清洗', '去皮计件', '收货卡点', '切断去芯', '切块装袋', '成品出库']
</script>

<template>
  <div class="page">
    <header class="hero">
      <p class="eyebrow">木薯加工 · 计件产线</p>
      <h1 class="brand">木薯加工厂 ERP</h1>
      <p class="lead">选择终端登录。功能落在设计文档 13 大域内，产线由生产管理 + 库存管理配置，不另立流程系统。</p>
      <ol class="flow" aria-label="产线节点">
        <li v-for="(step, i) in flow" :key="step" :class="{ accent: i === 2 || i === 3 }">{{ step }}</li>
      </ol>
    </header>

    <section class="terminals">
      <h2 class="section-title">选择登录平台</h2>
      <div class="grid">
        <a
          v-for="c in cards"
          :key="c.key"
          class="card"
          :href="terminalUrl(c.key)"
        >
          <span class="badge">{{ c.badge }}</span>
          <h3>{{ c.title }}</h3>
          <p>{{ c.desc }}</p>
          <span class="enter">进入登录 →</span>
        </a>
      </div>
    </section>

    <footer class="foot">
      演示账号 <code>admin</code> / <code>admin123</code>
      · API <code>127.0.0.1:18080</code>
      · 开发请先启动目标终端再点选
    </footer>
  </div>
</template>

<style scoped>
/* 木薯意象：薯皮肉色、薯皮褐、叶色绿；全部纯色，无渐变 */
.page {
  --soil: #2a241c;
  --bark: #6b5344;
  --flesh: #f3ebe0;
  --starch: #e7dfd2;
  --leaf: #2f6b45;
  --leaf-deep: #1f4d32;
  --panel: #fffdf9;
  --line: #d4c8b8;
  --ink: #1e2a22;
  --mute: #5c655c;

  min-height: 100vh;
  margin: 0;
  background: var(--starch);
  color: var(--ink);
  font-family: "Source Han Sans SC", "PingFang SC", "Microsoft YaHei", "Segoe UI", sans-serif;
}

.hero {
  background: var(--soil);
  color: var(--flesh);
  padding: 48px 24px 40px;
}

.eyebrow {
  margin: 0 0 10px;
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #c4b49a;
}

.brand {
  margin: 0 0 12px;
  font-size: clamp(28px, 5vw, 40px);
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #fff;
  line-height: 1.15;
}

.lead {
  margin: 0 0 28px;
  max-width: 640px;
  font-size: 15px;
  line-height: 1.65;
  color: #d9cfc0;
}

.flow {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.flow li {
  padding: 6px 12px;
  background: #3a3228;
  color: #e8dfd0;
  font-size: 12px;
  border: 1px solid #4a4034;
}

.flow li.accent {
  background: var(--leaf);
  border-color: var(--leaf);
  color: #fff;
}

.terminals {
  max-width: 1080px;
  margin: 0 auto;
  padding: 36px 24px 24px;
}

.section-title {
  margin: 0 0 16px;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--bark);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 14px;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 20px 20px 18px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-left: 4px solid var(--leaf);
  color: inherit;
  text-decoration: none;
  transition: border-color 0.15s, background 0.15s;
}

.card:hover {
  border-color: var(--leaf);
  background: var(--flesh);
}

.badge {
  align-self: flex-start;
  font-size: 11px;
  padding: 2px 8px;
  background: #e2eee6;
  color: var(--leaf-deep);
}

.card h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--soil);
}

.card p {
  margin: 0;
  flex: 1;
  font-size: 13px;
  line-height: 1.55;
  color: var(--mute);
}

.enter {
  margin-top: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--leaf);
}

.foot {
  max-width: 1080px;
  margin: 0 auto;
  padding: 8px 24px 40px;
  font-size: 12px;
  color: var(--mute);
  line-height: 1.6;
}

.foot code {
  font-size: 11px;
  padding: 1px 5px;
  background: var(--flesh);
  border: 1px solid var(--line);
  color: var(--soil);
}
</style>
