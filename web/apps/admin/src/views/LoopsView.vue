<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  productionApi,
  inventoryApi,
  payrollApi,
  purchaseApi,
  salesApi,
  financeApi,
  iamApi,
  productApi,
  systemApi,
} from '@erp/shared'

const log = ref<string[]>([])
const busy = ref(false)

function push(msg: string) {
  log.value.unshift(`${new Date().toLocaleTimeString()} ${msg}`)
}

async function runPiecework() {
  busy.value = true
  try {
    // ensure product + wage rate + stock for requisition
    let productId = 0
    const pl = await productApi.list()
    const plist = (pl.data as { list?: { id: number }[] })?.list || []
    if (plist.length) productId = plist[0].id
    else {
      const c = await productApi.create({ code: `PW${Date.now()}`, name: '计件原料', product_type: 'raw' })
      productId = Number((c.data as { id: number })?.id)
    }
    await payrollApi.createWageRate({ process_id: 1, rate: 2.5 }) // 同工序会停用旧 active，不堆重复
    const txn = await inventoryApi.createTxn({
      doc_type: 'purchase_in',
      warehouse_id: 1,
      lines: [{ product_id: productId, qty: 100, direction: 'in' }],
    })
    const tid = Number((txn.data as { id: number })?.id)
    await inventoryApi.postTxn(tid)
    push(`库存入库过账 #${tid}`)

    const task = await productionApi.createTask({ doc_no: `PT${Date.now()}` })
    const taskId = Number((task.data as { id: number })?.id)
    push(`任务单 #${taskId}`)

    const disp = await productionApi.createDispatch({
      task_id: taskId, process_id: 1, worker_id: 1, qty: 10,
    })
    const dispId = Number((disp.data as { id: number })?.id)
    push(`派工 #${dispId}`)

    const req = await productionApi.createRequisition({
      task_id: taskId, warehouse_id: 1, txn_type: 'consume',
      lines: [{ product_id: productId, qty: 5 }],
    })
    const reqId = Number((req.data as { id: number })?.id)
    await productionApi.postRequisition(reqId)
    push(`领料过账 #${reqId}`)

    const settle = await productionApi.daySettlePiecework({ biz_date: new Date().toISOString().slice(0, 10) })
    push(`计件日结 ${settle.code === 1 ? 'OK' : settle.msg}`)

    const sheet = await payrollApi.calcSheet({ remark: 'piecework-loop' })
    push(`工资核算 ${sheet.code === 1 ? 'OK' : sheet.msg}`)
    ElMessage.success('生产计件闭环完成（领料日结，无旧报工）')
  } catch (e) {
    push(`ERROR ${e}`)
    ElMessage.error('计件闭环失败')
  } finally {
    busy.value = false
  }
}

async function runPurchase() {
  busy.value = true
  try {
    const created = await purchaseApi.createSupplier({
      code: `S${Date.now()}`, name: '闭环供应商', status: 'qualified', supplier_type: 'raw',
    })
    const sid = Number((created.data as { id?: number })?.id)
    push(`供应商 #${sid} 已合格`)
    const req = await purchaseApi.createRequest({ title: '采购申请闭环', qty: 20, supplier_id: sid })
    push(`采购申请 ${(req.data as { id?: number })?.id}`)
    let productId = 1
    const pl = await productApi.list()
    const list = (pl.data as { list?: { id: number }[] })?.list || []
    if (list.length) productId = list[0].id
    const ib = await purchaseApi.createInbound({
      supplier_id: sid, qty: 20, product_id: productId, price: 1.8, warehouse_id: 1,
    })
    const ibId = Number((ib.data as { id: number })?.id)
    const posted = await purchaseApi.postInbound(ibId)
    push(`采购入库过账 #${ibId} ${posted.code === 1 ? 'posted' : posted.msg}`)
    const qc = await purchaseApi.createQc({ inbound_id: ibId, product_id: productId, qty_check: 20, supplier_id: sid })
    const qcId = Number((qc.data as { id: number })?.id)
    await purchaseApi.passQc(qcId)
    push(`来料质检 #${qcId} pass`)
    ElMessage.success('采购入库闭环完成')
  } catch (e) {
    push(`ERROR ${e}`)
  } finally {
    busy.value = false
  }
}

async function runSales() {
  busy.value = true
  try {
    await salesApi.createInquiry({ customer: '闭环客户', product: '演示', qty: 5 })
    push('询价已创建')
    let productId = 1
    const pl = await productApi.list()
    const list = (pl.data as { list?: { id: number }[] })?.list || []
    if (list.length) productId = list[0].id
    // ensure stock
    const txn = await inventoryApi.createTxn({
      doc_type: 'purchase_in', warehouse_id: 3,
      lines: [{ product_id: productId, qty: 50, direction: 'in' }],
    })
    await inventoryApi.postTxn(Number((txn.data as { id: number }).id))

    const order = await salesApi.createOrder({
      customer: '闭环客户', warehouse_id: 3,
      lines: [{ product_id: productId, qty: 3 }],
    })
    const oid = Number((order.data as { id: number })?.id)
    push(`销售订单 #${oid}（已建占用）`)

    const pre = await salesApi.createPreShip({
      order_id: oid, warehouse_id: 3, txn_type: 'sale_out',
      lines: [{ product_id: productId, qty: 3 }],
    })
    const pid = Number((pre.data as { id: number })?.id)
    const conf = await salesApi.confirmPreShip(pid, {
      warehouse_id: 3, txn_type: 'sale_out',
      lines: [{ product_id: productId, qty: 3 }],
    })
    push(`预发货确认 ${conf.code === 1 ? 'shipped' : conf.msg}`)

    const wo = await financeApi.createWriteoff({ order_id: oid, amount: 100 })
    push(`收款核单 ${wo.code === 1 ? 'OK' : wo.msg}`)
    ElMessage.success('销售出库闭环完成')
  } catch (e) {
    push(`ERROR ${e}`)
  } finally {
    busy.value = false
  }
}

async function runPerm() {
  busy.value = true
  try {
    const perms = await iamApi.permissions()
    push(`权限码 ${(perms.data as { list?: unknown[] })?.list?.length || 0}`)
    const menus = await iamApi.menus()
    push(`菜单 ${(menus.data as { list?: unknown[] })?.list?.length || 0}`)
    const users = await iamApi.users()
    const list = (users.data as { list?: { id: number; login_name: string }[] })?.list || []
    const target = list.find((u) => u.login_name !== 'admin') || list[0]
    if (target) {
      await iamApi.setRoles(target.id, [1])
      push(`用户 ${target.login_name} 授权角色 1`)
      await iamApi.freeze(target.id)
      push(`冻结 ${target.login_name}`)
      await iamApi.unfreeze(target.id)
      push(`解冻 ${target.login_name}`)
    }
    const logs = await systemApi.logs()
    push(`操作日志拉取 ${logs.code === 1 ? 'OK' : logs.msg}`)
    ElMessage.success('权限闭环完成')
  } catch (e) {
    push(`ERROR ${e}`)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="loops">
    <h2>业务闭环（设计文档第 9 章）</h2>
    <p class="desc">一键跑通计件 / 采购入库 / 销售出库 / 权限四条闭环，结果写入真实 API。</p>
    <div class="btns">
      <el-button type="primary" :loading="busy" @click="runPiecework">1. 生产计件闭环</el-button>
      <el-button type="success" :loading="busy" @click="runPurchase">2. 采购入库闭环</el-button>
      <el-button type="warning" :loading="busy" @click="runSales">3. 销售出库闭环</el-button>
      <el-button type="info" :loading="busy" @click="runPerm">4. 权限闭环</el-button>
    </div>
    <el-card style="margin-top:16px">
      <template #header>执行日志</template>
      <div v-for="(l, i) in log" :key="i" class="line">{{ l }}</div>
      <div v-if="!log.length" class="muted">尚未执行</div>
    </el-card>
  </div>
</template>

<style scoped>
.loops { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.desc { color: #5c6b75; font-size: 13px; }
.btns { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 12px; }
.line { font-family: Consolas, monospace; font-size: 12px; padding: 2px 0; }
.muted { color: #999; }
</style>
