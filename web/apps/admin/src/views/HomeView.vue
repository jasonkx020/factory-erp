<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePermStore, approvalApi, inventoryApi, productionApi, adminSpecialPath } from '@erp/shared'

const perm = usePermStore()
const router = useRouter()
const stats = ref({ tasks: 0, approval: 0, balances: 0 })

onMounted(async () => {
  const [t, a, b] = await Promise.all([
    productionApi.listTasks(),
    approvalApi.tasks(),
    inventoryApi.balances(),
  ])
  stats.value.tasks = ((t.data as { list?: unknown[] })?.list || []).length
  stats.value.approval = ((a.data as { list?: unknown[] })?.list || []).length
  stats.value.balances = ((b.data as { list?: unknown[] })?.list || []).length
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
  <div>
    <h2>管理端工作台</h2>
    <p class="desc">菜单与第 3 章核心功能表对应；现场台账与运维页已挂入各域菜单。</p>
    <div class="stats">
      <div class="stat"><div class="label">生产任务</div><div class="value">{{ stats.tasks }}</div></div>
      <div class="stat"><div class="label">审批任务</div><div class="value">{{ stats.approval }}</div></div>
      <div class="stat"><div class="label">库存行</div><div class="value">{{ stats.balances }}</div></div>
      <div class="stat click" @click="open('系统管理','业务闭环')"><div class="label">业务闭环</div><div class="value">演示</div></div>
    </div>
    <el-card style="margin:16px 0">
      <template #header>权限中心快捷入口</template>
      <el-button type="primary" @click="open('人事管理','权限分配')">权限分配</el-button>
      <el-button @click="open('系统管理','自定义权限')">自定义权限</el-button>
      <el-button @click="open('系统管理','自定义菜单')">自定义菜单</el-button>
      <el-button @click="open('系统管理','登录控制')">登录控制</el-button>
      <el-button @click="open('系统管理','账户冻结')">账户冻结</el-button>
    </el-card>
    <h3>十三大核心功能</h3>
    <div class="stats">
      <div
        v-for="d in perm.visibleMenus"
        :key="d.domain"
        class="stat click"
        @click="open(d.domain, d.modules[0])"
      >
        <div class="label">{{ d.domain }}</div>
        <div class="value" style="font-size:16px">{{ d.modules.length }} 模块</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.desc { color: #5c6b75; }
.stats { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 12px; }
.stat {
  background: #fff; border: 1px solid #d5dde3; border-radius: 8px; padding: 14px;
}
.stat.click { cursor: pointer; }
.stat.click:hover { border-color: #0d7a6f; }
.label { color: #5c6b75; font-size: 12px; }
.value { font-size: 24px; font-weight: 600; color: #0d7a6f; margin-top: 4px; }
</style>
