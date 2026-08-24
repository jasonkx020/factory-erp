<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { VueFlow, type Edge, type Node, type Connection } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { ElMessage } from 'element-plus'
import { productionApi, inventoryApi, productApi } from '@erp/shared'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import DesktopOnlyGate from '../../components/mobile/DesktopOnlyGate.vue'

const props = withDefaults(
  defineProps<{ kindFilter?: string }>(),
  { kindFilter: '' },
)

type Row = Record<string, unknown>

const graphs = ref<Row[]>([])
const currentId = ref<number | null>(null)
const meta = ref({
  code: '',
  name: '',
  kind: 'production' as string,
  status: 'draft',
  routing_id: 0,
  product_id: 0,
  version_no: 'V1',
})
const nodes = ref<Node[]>([])
const edges = ref<Edge[]>([])
const selected = ref<Node | null>(null)
const selectedEdge = ref<Edge | null>(null)
const processes = ref<Row[]>([])
const warehouses = ref<Row[]>([])
const products = ref<Row[]>([])
const compiledSteps = ref<Row[]>([])
const loading = ref(false)
const saving = ref(false)

const kindOptions = [
  { value: 'production', label: '生产工艺' },
  { value: 'purchase_gate', label: '过磅入厂' },
  { value: 'purchase_stockin', label: '过磅入库' },
]

const roleOptions = [
  { value: 'purchase', label: '采购' },
  { value: 'qc', label: '质检' },
  { value: 'warehouse', label: '仓管' },
  { value: 'finance', label: '财务' },
]

const actionOptions = [
  { value: 'submit', label: '提交' },
  { value: 'qc_deduct', label: '确认扣损' },
  { value: 'warehouse_confirm', label: '入库确认' },
  { value: 'pay', label: '付款' },
]

const filteredKinds = computed(() => {
  if (!props.kindFilter) return kindOptions
  if (props.kindFilter === 'purchase') {
    return kindOptions.filter((k) => k.value.startsWith('purchase'))
  }
  return kindOptions.filter((k) => k.value === props.kindFilter || k.value.startsWith(props.kindFilter))
})

function onConnectHandler(params: Connection) {
  if (!params.source || !params.target) return
  if (params.source === params.target) return
  const dup = edges.value.some(
    (e) => e.source === params.source && e.target === params.target,
  )
  if (dup) {
    ElMessage.warning('这两点之间已有连线，请先删除再重画')
    return
  }
  // 同一源点若尚无默认边，新边标为默认；已有默认则新边为旁路
  const hasDefault = edges.value.some(
    (e) => e.source === params.source && (e.data as Row)?.is_default !== false,
  )
  const isDefault = !hasDefault
  edges.value = [
    ...edges.value,
    {
      id: `e_${params.source}_${params.target}_${Date.now()}`,
      source: params.source,
      target: params.target,
      sourceHandle: params.sourceHandle ?? undefined,
      targetHandle: params.targetHandle ?? undefined,
      data: { is_default: isDefault },
      label: isDefault ? '默认' : '旁路',
      animated: true,
      selectable: true,
      focusable: true,
    },
  ]
  selected.value = null
  selectedEdge.value = edges.value[edges.value.length - 1]
}

async function loadLists() {
  loading.value = true
  try {
    const q = props.kindFilter === 'purchase'
      ? ''
      : props.kindFilter
        ? `kind=${encodeURIComponent(props.kindFilter)}`
        : ''
    const [g, p, w, prod] = await Promise.all([
      productionApi.listFlowGraphs(q || undefined),
      productionApi.processes(),
      inventoryApi.warehouses(),
      productApi.list(),
    ])
    let list = ((g.data as { list?: Row[] })?.list) || []
    if (props.kindFilter === 'purchase') {
      list = list.filter((x) => String(x.kind || '').startsWith('purchase'))
    }
    graphs.value = list
    processes.value = ((p.data as { list?: Row[] })?.list) || []
    warehouses.value = ((w.data as { list?: Row[] })?.list) || []
    products.value = ((prod.data as { list?: Row[] })?.list) || []
    if (graphs.value.length && !currentId.value) {
      await openGraph(Number(graphs.value[0].id))
    }
  } finally {
    loading.value = false
  }
}

function parseGraph(raw: unknown): { nodes: Node[]; edges: Edge[] } {
  let obj: { nodes?: Array<Node & { type?: string }>; edges?: Edge[] } = {}
  if (typeof raw === 'string') {
    try {
      obj = JSON.parse(raw || '{}')
    } catch {
      obj = {}
    }
  } else if (raw && typeof raw === 'object') {
    obj = raw as { nodes?: Array<Node & { type?: string }>; edges?: Edge[] }
  }
  const ns = (obj.nodes || []).map((n) => {
    const sem = String(n.type && n.type !== 'default' ? n.type : (n.data as Row)?._nodeType || 'process_step')
    const data = { ...(n.data as object), _nodeType: sem, label: String((n.data as Row)?.label || sem) } as Row
    return {
      id: n.id,
      type: 'default' as const,
      position: n.position || { x: 0, y: 0 },
      data,
      label: String(data.label),
      style: nodeStyle(sem),
      draggable: true,
      selectable: true,
    }
  })
  const es = (obj.edges || []).map((e) => ({
    ...e,
    label: (e.data as Row)?.is_default === false ? '旁路' : '默认',
    animated: true,
    selectable: true,
    focusable: true,
  }))
  return { nodes: ns, edges: es }
}


async function loadCompiledSteps(routingId: number) {
  compiledSteps.value = []
  if (!routingId) return
  const res = await productionApi.flowRules(routingId)
  if (res.code !== 1) return
  const data = (res.data || {}) as Row
  const steps = (data.steps as Row[]) || []
  compiledSteps.value = steps
}

async function openGraph(id: number) {
  const res = await productionApi.getFlowGraph(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  const d = (res.data || {}) as Row
  currentId.value = id
  let graphProductId = Number(d.product_id || 0)
  try {
    const raw = typeof d.graph_json === 'string' ? JSON.parse(String(d.graph_json || '{}')) : (d.graph_json || {})
    const gm = (raw as Row).meta as Row | undefined
    if (!graphProductId && gm?.product_id) graphProductId = Number(gm.product_id)
  } catch { /* ignore */ }
  meta.value = {
    code: String(d.code || ''),
    name: String(d.name || ''),
    kind: String(d.kind || 'production'),
    status: String(d.status || 'draft'),
    routing_id: Number(d.routing_id || 0),
    product_id: graphProductId,
    version_no: String(d.version_no || 'V1'),
  }
  const parsed = parseGraph(d.graph_json)
  nodes.value = parsed.nodes
  edges.value = parsed.edges
  selected.value = null
  selectedEdge.value = null
  await loadCompiledSteps(meta.value.routing_id)
}

function onNodeClick(ev: { node: Node }) {
  selected.value = ev.node
  selectedEdge.value = null
}

function onEdgeClick(ev: { edge: Edge }) {
  selectedEdge.value = ev.edge
  selected.value = null
}

function onPaneClick() {
  selected.value = null
  selectedEdge.value = null
}

function syncSelectedEdge() {
  if (!selectedEdge.value) return
  const id = selectedEdge.value.id
  const i = edges.value.findIndex((e) => e.id === id)
  if (i < 0) return
  const isDefault = (selectedEdge.value.data as Row)?.is_default !== false
  // 设为默认时，同 source 其它边改为旁路
  if (isDefault) {
    edges.value = edges.value.map((e) => {
      if (e.source !== selectedEdge.value!.source) return e
      if (e.id === id) {
        return { ...e, data: { ...(e.data as object), is_default: true }, label: '默认' }
      }
      return { ...e, data: { ...(e.data as object), is_default: false }, label: '旁路' }
    })
  } else {
    edges.value = edges.value.map((e) =>
      e.id === id
        ? { ...e, data: { ...(e.data as object), is_default: false }, label: '旁路' }
        : e,
    )
  }
  selectedEdge.value = edges.value.find((e) => e.id === id) || null
}

function removeSelectedEdge() {
  if (!selectedEdge.value) return
  const id = selectedEdge.value.id
  edges.value = edges.value.filter((e) => e.id !== id)
  selectedEdge.value = null
}

function onKeyDelete(ev: KeyboardEvent) {
  const tag = (ev.target as HTMLElement)?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
  if (ev.key !== 'Delete' && ev.key !== 'Backspace') return
  if (selectedEdge.value) {
    ev.preventDefault()
    removeSelectedEdge()
    return
  }
  if (selected.value) {
    ev.preventDefault()
    removeSelected()
  }
}

function syncSelectedData() {
  if (!selected.value) return
  const i = nodes.value.findIndex((n) => n.id === selected.value!.id)
  if (i >= 0) {
    nodes.value[i] = {
      ...nodes.value[i],
      data: { ...(selected.value.data as object) },
      label: String((selected.value.data as Row)?.label || selected.value.type),
    }
  }
}

function addNode(type: string) {
  const id = `${type}_${Date.now()}`
  const data: Row =
    type === 'process_step'
      ? { label: '工序', process_id: Number(processes.value[0]?.id || 0), auto_next: true, is_piecework: false, is_inbound_checkpoint: false, checkpoint_bind_warehouse: false }
      : type === 'role_task'
        ? { label: '岗位任务', role_code: 'qc', action: 'qc_deduct' }
        : type === 'gateway_xor'
          ? { label: '网关' }
          : { label: type === 'start' ? '开始' : type === 'end' ? '结束' : type }
  const n: Node = {
    id,
    type: 'default',
    position: { x: 120 + nodes.value.length * 40, y: 80 + (nodes.value.length % 5) * 60 },
    data: { ...data, _nodeType: type },
    label: String(data.label),
    style: nodeStyle(type),
    draggable: true,
    selectable: true,
  }
  nodes.value = [...nodes.value, n]
  selected.value = n
  selectedEdge.value = null
}

function nodeStyle(type: string): Record<string, string> {
  switch (type) {
    case 'start':
      return { background: '#d1fae5', border: '1px solid #059669', borderRadius: '20px', padding: '8px 16px' }
    case 'end':
      return { background: '#fee2e2', border: '1px solid #dc2626', borderRadius: '20px', padding: '8px 16px' }
    case 'role_task':
      return { background: '#e0f2fe', border: '1px solid #0284c7', borderRadius: '8px', padding: '8px 12px' }
    case 'gateway_xor':
      // 勿写 transform：会覆盖 Vue Flow 的 translate，导致无法拖动
      return { background: '#fef3c7', border: '2px solid #d97706', borderRadius: '4px', padding: '10px 14px', minWidth: '72px', textAlign: 'center', fontWeight: '600' }
    default:
      return { background: '#f1f5f9', border: '1px solid #64748b', borderRadius: '8px', padding: '8px 12px' }
  }
}

function semanticType(n: Node): string {
  const d = (n.data || {}) as Row
  if (d._nodeType) return String(d._nodeType)
  if (n.id === 'start' || String(n.label).includes('开始')) return 'start'
  if (n.id === 'end' || String(n.label).includes('结束')) return 'end'
  // seeded graphs use type field on node
  const raw = (n as Node & { type?: string }).type
  if (raw && raw !== 'default' && raw !== 'input' && raw !== 'output') return raw
  // check data from API: nodes have type at top level when loaded
  return String((n as Row).type || d.type || 'process_step')
}

function removeSelected() {
  if (!selected.value) return
  const id = selected.value.id
  nodes.value = nodes.value.filter((n) => n.id !== id)
  edges.value = edges.value.filter((e) => e.source !== id && e.target !== id)
  selected.value = null
  selectedEdge.value = null
}

function buildGraphJSON() {
  const outNodes = nodes.value.map((n) => {
    const t = semanticType(n)
    const data = { ...(n.data as object) } as Row
    delete data._nodeType
    return {
      id: n.id,
      type: t,
      position: n.position,
      data,
    }
  })
  const outEdges = edges.value.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    data: { is_default: (e.data as Row)?.is_default !== false },
  }))
  return { nodes: outNodes, edges: outEdges, meta: { product_id: meta.value.product_id || 0 } }
}

async function save(publish = false) {
  if (!meta.value.code || !meta.value.name) return ElMessage.warning('请填写编码与名称')
  saving.value = true
  try {
    const body: Record<string, unknown> = {
      code: meta.value.code,
      name: meta.value.name,
      kind: meta.value.kind,
      status: publish ? 'active' : meta.value.status,
      version_no: meta.value.version_no,
      routing_id: meta.value.routing_id || undefined,
      product_id: meta.value.product_id || undefined,
      graph_json: buildGraphJSON(),
    }
    let res
    if (currentId.value) {
      res = await productionApi.updateFlowGraph(currentId.value, body)
    } else {
      res = await productionApi.createFlowGraph(body)
    }
    if (res.code !== 1) return ElMessage.error(res.msg)
    ElMessage.success(publish ? '已发布并编译工艺步骤' : '已保存（草稿不编译过站步骤）')
    const id = Number((res.data as Row)?.id || currentId.value)
    await loadLists()
    if (id) await openGraph(id)
  } finally {
    saving.value = false
  }
}

async function createNew() {
  const kind = filteredKinds.value[0]?.value || 'production'
  currentId.value = null
  meta.value = {
    code: `FLOW_${Date.now().toString().slice(-6)}`,
    name: '新建流程',
    kind,
    status: 'draft',
    routing_id: 0,
    version_no: 'V1',
  }
  nodes.value = [
    {
      id: 'start',
      type: 'default',
      position: { x: 40, y: 80 },
      data: { label: '开始', _nodeType: 'start' },
      label: '开始',
      style: nodeStyle('start'),
    },
    {
      id: 'end',
      type: 'default',
      position: { x: 480, y: 80 },
      data: { label: '结束', _nodeType: 'end' },
      label: '结束',
      style: nodeStyle('end'),
    },
  ]
  edges.value = []
  selected.value = null
  selectedEdge.value = null
}

watch(
  () => props.kindFilter,
  () => {
    currentId.value = null
    loadLists()
  },
)

onMounted(() => {
  loadLists()
  window.addEventListener('keydown', onKeyDelete)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDelete)
})
</script>

<template>
  <DesktopOnlyGate message="工艺流程图编辑需在桌面浏览器操作。">
  <div class="flow-editor" v-loading="loading">
    <aside class="left">
      <div class="toolbar">
        <el-button type="primary" size="small" @click="createNew">新建</el-button>
        <el-button size="small" :loading="saving" @click="save(false)">保存</el-button>
        <el-button type="success" size="small" :loading="saving" @click="save(true)">发布</el-button>
      </div>
      <el-form label-position="top" size="small" class="meta">
        <el-form-item label="编码"><el-input v-model="meta.code" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="meta.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="meta.kind" style="width:100%">
            <el-option v-for="k in filteredKinds" :key="k.value" :label="k.label" :value="k.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="meta.status" style="width:100%">
            <el-option label="草稿" value="draft" />
            <el-option label="启用" value="active" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="meta.kind === 'production'" label="绑定产品">
          <el-select v-model="meta.product_id" clearable style="width:100%" placeholder="可选">
            <el-option
              v-for="p in products"
              :key="String(p.id)"
              :label="String(p.name || p.code || p.id)"
              :value="Number(p.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="meta.kind === 'production' && meta.routing_id" label="工艺ID">
          <el-input :model-value="String(meta.routing_id)" disabled />
        </el-form-item>
      </el-form>
      <div v-if="meta.kind === 'production'" class="compiled-steps">
        <h4>已编译步骤（发布后生效）</h4>
        <p v-if="!compiledSteps.length" class="muted">尚未编译或仍为草稿。点「发布」后同步到过站。</p>
        <el-table v-else :data="compiledSteps" size="small" max-height="220">
          <el-table-column prop="seq_no" label="序" width="44" />
          <el-table-column prop="step_code" label="编码" width="64" />
          <el-table-column prop="step_name" label="名称" min-width="100" />
          <el-table-column label="计件" width="50">
            <template #default="{ row }">{{ row.is_piecework ? '是' : '' }}</template>
          </el-table-column>
          <el-table-column label="卡点" width="50">
            <template #default="{ row }">{{ row.is_inbound_checkpoint ? '是' : '' }}</template>
          </el-table-column>
          <el-table-column prop="output_product_name" label="产出产物" min-width="100" />
          <el-table-column label="绑仓" width="50">
            <template #default="{ row }">{{ row.checkpoint_bind_warehouse ? '是' : '' }}</template>
          </el-table-column>
        </el-table>
      </div>
      <el-table
        :data="graphs"
        size="small"
        highlight-current-row
        height="280"
        @current-change="(r: Row | null) => r && openGraph(Number(r.id))"
      >
        <el-table-column prop="name" label="流程" />
        <el-table-column prop="kind" label="类型" width="100" />
        <el-table-column prop="status" label="状态" width="70" />
      </el-table>
      <div class="palette">
        <div class="ph">节点库（点击添加）</div>
        <el-button size="small" @click="addNode('start')">开始</el-button>
        <el-button size="small" @click="addNode('end')">结束</el-button>
        <el-button v-if="meta.kind === 'production'" size="small" @click="addNode('process_step')">工序</el-button>
        <el-button v-if="meta.kind.startsWith('purchase')" size="small" @click="addNode('role_task')">岗位</el-button>
        <el-button size="small" @click="addNode('gateway_xor')" title="多分支汇合/分叉，自动走标为「默认」的那条边">网关</el-button>
      </div>
      <p class="ph tip">提示：点击连线后可删除；Delete 键删除选中项；删线后可重新拖拽连线。</p>
    </aside>
    <main class="canvas">
      <VueFlow
        v-model:nodes="nodes"
        v-model:edges="edges"
        fit-view-on-init
        :default-viewport="{ zoom: 0.9 }"
        :edges-updatable="true"
        :elements-selectable="true"
        @node-click="onNodeClick"
        @edge-click="onEdgeClick"
        @pane-click="onPaneClick"
        @connect="onConnectHandler"
      >
        <Background />
        <Controls />
        <MiniMap />
      </VueFlow>
    </main>
    <aside class="right">
      <h3>属性</h3>
      <template v-if="selectedEdge">
        <el-form label-position="top" size="small">
          <el-form-item label="连线">
            <el-input :model-value="`${selectedEdge.source} → ${selectedEdge.target}`" disabled />
          </el-form-item>
          <el-form-item label="默认路径">
            <el-switch
              :model-value="(selectedEdge.data as Row)?.is_default !== false"
              @change="(v: string | number | boolean) => {
                if (!selectedEdge) return
                selectedEdge.data = { ...(selectedEdge.data as object), is_default: !!v }
                syncSelectedEdge()
              }"
            />
            <div class="hint">自动流转只走「默认」边；旁路需业务上传 next_node_id 才走。</div>
          </el-form-item>
          <el-button type="danger" size="small" @click="removeSelectedEdge">删除连线</el-button>
        </el-form>
      </template>
      <template v-else-if="selected">
        <el-form label-position="top" size="small" @change="syncSelectedData">
          <el-form-item label="节点ID"><el-input :model-value="selected.id" disabled /></el-form-item>
          <el-form-item label="显示名">
            <el-input v-model="(selected.data as Row).label" @change="syncSelectedData" />
          </el-form-item>
          <template v-if="semanticType(selected) === 'process_step'">
            <el-form-item label="工序">
              <el-select v-model="(selected.data as Row).process_id" filterable style="width:100%" @change="syncSelectedData">
                <el-option v-for="p in processes" :key="String(p.id)" :label="String(p.name)" :value="Number(p.id)" />
              </el-select>
            </el-form-item>
            <el-form-item label="产出产物">
              <el-select v-model="(selected.data as Row).output_product_id" clearable filterable style="width:100%" placeholder="选择产物" @change="syncSelectedData">
                <el-option
                  v-for="p in products"
                  :key="String(p.id)"
                  :label="`${p.code || ''} · ${p.name || p.id}`"
                  :value="Number(p.id)"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="计件"><el-switch v-model="(selected.data as Row).is_piecework" @change="syncSelectedData" /></el-form-item>
            <el-form-item label="卡点"><el-switch v-model="(selected.data as Row).is_inbound_checkpoint" @change="syncSelectedData" /></el-form-item>
            <el-form-item label="卡点绑仓(先入后出)">
              <el-switch v-model="(selected.data as Row).checkpoint_bind_warehouse" @change="syncSelectedData" />
            </el-form-item>
            <el-form-item label="自动下步"><el-switch v-model="(selected.data as Row).auto_next" @change="syncSelectedData" /></el-form-item>
            <el-form-item label="自动入库"><el-switch v-model="(selected.data as Row).auto_stock_in" @change="syncSelectedData" /></el-form-item>
            <el-form-item label="自动出库"><el-switch v-model="(selected.data as Row).auto_stock_out" @change="syncSelectedData" /></el-form-item>
            <el-form-item label="仓库">
              <el-select v-model="(selected.data as Row).warehouse_id" clearable style="width:100%" @change="syncSelectedData">
                <el-option v-for="w in warehouses" :key="String(w.id)" :label="String(w.name)" :value="Number(w.id)" />
              </el-select>
            </el-form-item>
          </template>
          <template v-if="semanticType(selected) === 'role_task'">
            <el-form-item label="角色">
              <el-select v-model="(selected.data as Row).role_code" style="width:100%" @change="syncSelectedData">
                <el-option v-for="r in roleOptions" :key="r.value" :label="r.label" :value="r.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="动作">
              <el-select v-model="(selected.data as Row).action" style="width:100%" @change="syncSelectedData">
                <el-option v-for="a in actionOptions" :key="a.value" :label="a.label" :value="a.value" />
              </el-select>
            </el-form-item>
          </template>
          <template v-if="semanticType(selected) === 'gateway_xor'">
            <p class="hint">
              网关用于「一分多」：从网关拉出多条线到不同岗位/工序。
              自动流转只走标为「默认」的那条；其它线作为人工改道（next_node_id）备用。
              串行流程一般不需要网关，直接节点连节点即可。
            </p>
          </template>
          <el-button type="danger" size="small" @click="removeSelected">删除节点</el-button>
        </el-form>
      </template>
      <p v-else class="hint">点击节点或连线编辑；Delete 删除选中项；删线后可重新拖拽连线。</p>
    </aside>
  </div>
  </DesktopOnlyGate>
</template>

<style scoped>
.flow-editor {
  display: grid;
  grid-template-columns: 260px 1fr 260px;
  gap: 12px;
  min-height: 640px;
  height: calc(100vh - 160px);
}
.left, .right {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px;
  overflow: auto;
}
.canvas {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fafafa;
  min-height: 560px;
}
.toolbar { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; }
.meta { margin-bottom: 8px; }
.palette { margin-top: 12px; display: flex; flex-wrap: wrap; gap: 6px; }
.ph { width: 100%; font-size: 12px; color: #64748b; margin-bottom: 4px; }
.tip { margin-top: 8px; line-height: 1.4; }
.hint { color: #64748b; font-size: 13px; line-height: 1.5; }
h3 { margin: 0 0 8px; font-size: 14px; }
/* 连线盖在节点之上，避免被节点挡住看不清/点不到 */
.canvas :deep(.vue-flow__edges) {
  z-index: 5 !important;
}
.canvas :deep(.vue-flow__edge) {
  pointer-events: stroke;
}
.canvas :deep(.vue-flow__edge-path) {
  stroke-width: 2.5;
}
.canvas :deep(.vue-flow__edge.selected .vue-flow__edge-path),
.canvas :deep(.vue-flow__edge:focus .vue-flow__edge-path) {
  stroke: #dc2626 !important;
  stroke-width: 3.5;
}
.canvas :deep(.vue-flow__nodes) {
  z-index: 1 !important;
}
.canvas :deep(.vue-flow__node) {
  z-index: 1;
}
.canvas :deep(.vue-flow__edge-textwrapper),
.canvas :deep(.vue-flow__edge-text) {
  z-index: 6;
  pointer-events: none;
}

.compiled-steps {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.compiled-steps h4 {
  margin: 0 0 6px;
  font-size: 13px;
}
.compiled-steps .muted {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
