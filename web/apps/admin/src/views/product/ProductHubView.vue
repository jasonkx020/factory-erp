<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { productApi, BASE_UNIT_OPTIONS } from '@erp/shared'
import { ProductSelect, ProcessSelect, RoutingSelect, EnumSelect } from '../../components/select'

type Row = Record<string, unknown>

const route = useRoute()
const TITLE_MAP: Record<string, string> = {
  products: '产品档案',
  units: '产品单位管理',
  'app-sorts': 'APP产品排序',
  specs: '生产规格绑定',
}

const active = computed(() => String(route.params.section || 'products'))
const title = computed(() => TITLE_MAP[active.value] || '产品管理')

const loading = ref(false)
const list = ref<Row[]>([])
const products = ref<Row[]>([])
const units = ref<Row[]>([])
const selectedProductId = ref<number | null>(null)

const productForm = reactive({
  code: '',
  name: '',
  category: '',
  product_type: 'finished',
  spec_text: '',
  barcode: '',
  cost_price: 0,
  sale_price: 0,
  base_unit: 'kg',
  is_batch_managed: true,
  is_box_managed: false,
})

const unitForm = reactive({
  unit_name: '袋',
  factor_to_base: 25,
  is_base: false,
  is_purchase: true,
  is_sale: true,
  is_stock: true,
})

const specForm = reactive({
  product_id: null as number | null,
  spec_code: '',
  routing_id: null as number | null,
  process_id: null as number | null,
  wage: 0,
  remark: '',
})

async function loadMeta() {
  const p = await productApi.list()
  products.value = ((p.data as { list?: Row[] })?.list) || []
  if (products.value[0]) {
    const id = Number(products.value[0].id)
    if (!selectedProductId.value) selectedProductId.value = id
    if (!specForm.product_id) specForm.product_id = id
  }
}

async function refresh() {
  loading.value = true
  try {
    if (active.value === 'products') {
      const res = await productApi.list()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
      products.value = list.value
    } else if (active.value === 'units') {
      if (!selectedProductId.value && products.value[0]) {
        selectedProductId.value = Number(products.value[0].id)
      }
      if (!selectedProductId.value) {
        units.value = []
        return
      }
      const res = await productApi.units(selectedProductId.value)
      if (res.code !== 1) return ElMessage.error(res.msg)
      units.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'app-sorts') {
      const res = await productApi.appSorts()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
    } else if (active.value === 'specs') {
      const res = await productApi.specs()
      if (res.code !== 1) return ElMessage.error(res.msg)
      list.value = ((res.data as { list?: Row[] })?.list) || []
    }
  } finally {
    loading.value = false
  }
}

async function createProduct() {
  if (!productForm.code || !productForm.name) return ElMessage.warning('填写编码和名称')
  const res = await productApi.create({ ...productForm })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`已建档 ${(res.data as Row)?.code}`)
  Object.assign(productForm, {
    code: '',
    name: '',
    category: '',
    product_type: 'finished',
    spec_text: '',
    barcode: '',
    cost_price: 0,
    sale_price: 0,
    base_unit: 'kg',
    is_batch_managed: true,
    is_box_managed: false,
  })
  await loadMeta()
  await refresh()
}

async function activate(id: number) {
  const res = await productApi.activate(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已启用')
  await refresh()
}

async function deactivate(id: number) {
  const res = await productApi.deactivate(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已停用')
  await refresh()
}

async function removeProduct(id: number) {
  await ElMessageBox.confirm('确认删除该产品？', '提示', { type: 'warning' })
  const res = await productApi.remove(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已删除')
  await refresh()
}

async function addUnit() {
  if (!selectedProductId.value) return ElMessage.warning('先选择产品')
  if (!unitForm.unit_name) return ElMessage.warning('填写单位名')
  const next = [
    ...units.value.map((u) => ({
      unit_name: u.unit_name,
      is_base: !!u.is_base,
      factor_to_base: Number(u.factor_to_base) || 1,
      is_purchase: !!u.is_purchase,
      is_sale: !!u.is_sale,
      is_stock: !!u.is_stock,
    })),
    { ...unitForm },
  ]
  const res = await productApi.replaceUnits(selectedProductId.value, { units: next })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('单位已保存')
  unitForm.unit_name = ''
  await refresh()
}

async function removeUnit(idx: number) {
  if (!selectedProductId.value) return
  const next = units.value
    .filter((_, i) => i !== idx)
    .map((u) => ({
      unit_name: u.unit_name,
      is_base: !!u.is_base,
      factor_to_base: Number(u.factor_to_base) || 1,
      is_purchase: !!u.is_purchase,
      is_sale: !!u.is_sale,
      is_stock: !!u.is_stock,
    }))
  const res = await productApi.replaceUnits(selectedProductId.value, { units: next })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已移除')
  await refresh()
}

async function setBaseUnit(idx: number) {
  if (!selectedProductId.value) return
  const next = units.value.map((u, i) => ({
    unit_name: u.unit_name,
    is_base: i === idx,
    factor_to_base: i === idx ? 1 : Number(u.factor_to_base) || 1,
    is_purchase: !!u.is_purchase,
    is_sale: !!u.is_sale,
    is_stock: !!u.is_stock,
  }))
  const res = await productApi.replaceUnits(selectedProductId.value, { units: next })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已设为基本单位')
  await refresh()
}

async function saveSorts() {
  const items = list.value.map((r, i) => ({
    product_id: Number(r.product_id),
    sort_no: Number(r.sort_no) || (i + 1) * 10,
    is_visible: r.is_visible !== false && r.is_visible !== 0,
    channel: 'app',
  }))
  const res = await productApi.saveAppSorts({ channel: 'app', items })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('排序已保存')
  await refresh()
}

function moveSort(idx: number, dir: -1 | 1) {
  const j = idx + dir
  if (j < 0 || j >= list.value.length) return
  const arr = [...list.value]
  const tmp = arr[idx]
  arr[idx] = arr[j]
  arr[j] = tmp
  list.value = arr.map((r, i) => ({ ...r, sort_no: (i + 1) * 10 }))
}

async function createSpec() {
  if (!specForm.product_id || !specForm.spec_code) return ElMessage.warning('选择产品并填写规格编码')
  const body: Record<string, unknown> = {
    product_id: specForm.product_id,
    spec_code: specForm.spec_code,
    routing_id: specForm.routing_id || undefined,
    remark: specForm.remark,
  }
  if (specForm.process_id) {
    body.process_id = specForm.process_id
    body.wage = specForm.wage
  }
  const res = await productApi.createSpec(body)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(`规格 ${(res.data as Row)?.spec_code}`)
  specForm.spec_code = ''
  specForm.remark = ''
  await refresh()
}

async function removeSpec(id: number) {
  await ElMessageBox.confirm('确认删除该规格绑定？', '提示', { type: 'warning' })
  const res = await productApi.removeSpec(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('已删除')
  await refresh()
}

onMounted(async () => {
  await loadMeta()
  await refresh()
})
watch(active, refresh)
watch(selectedProductId, () => {
  if (active.value === 'units') refresh()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <div class="head">
      <h2>{{ title }}</h2>
      <el-button size="small" @click="refresh">刷新</el-button>
    </div>

    <!-- 产品档案 -->
    <template v-if="active === 'products'">
      <el-card header="新建产品档案" class="mb">
        <el-form inline size="small">
          <el-form-item label="编码"><el-input v-model="productForm.code" style="width:120px" /></el-form-item>
          <el-form-item label="名称"><el-input v-model="productForm.name" style="width:160px" /></el-form-item>
          <el-form-item label="类型">
            <el-select v-model="productForm.product_type" style="width:110px">
              <el-option label="原料" value="raw" />
              <el-option label="半成品" value="semi" />
              <el-option label="成品" value="finished" />
              <el-option label="辅料" value="aux" />
              <el-option label="废料" value="scrap" />
            </el-select>
          </el-form-item>
          <el-form-item label="分类"><el-input v-model="productForm.category" style="width:100px" placeholder="自由文本" /></el-form-item>
          <el-form-item label="规格"><el-input v-model="productForm.spec_text" style="width:120px" /></el-form-item>
          <el-form-item label="条码"><el-input v-model="productForm.barcode" style="width:120px" /></el-form-item>
          <el-form-item label="成本"><el-input-number v-model="productForm.cost_price" :min="0" :step="0.01" /></el-form-item>
          <el-form-item label="售价"><el-input-number v-model="productForm.sale_price" :min="0" :step="0.01" /></el-form-item>
          <el-form-item label="基本单位">
            <EnumSelect v-model="productForm.base_unit" :options="BASE_UNIT_OPTIONS" :clearable="false" style="width:110px" />
          </el-form-item>
          <el-form-item label="批次"><el-switch v-model="productForm.is_batch_managed" /></el-form-item>
          <el-form-item label="箱码"><el-switch v-model="productForm.is_box_managed" /></el-form-item>
          <el-button type="primary" @click="createProduct">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="code" label="编码" width="110" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="product_type" label="类型" width="90" />
        <el-table-column prop="category" label="分类" width="90" />
        <el-table-column prop="spec_text" label="规格" width="120" />
        <el-table-column prop="cost_price" label="成本" width="90" />
        <el-table-column prop="sale_price" label="售价" width="90" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status!=='active'" link type="success" @click="activate(Number(row.id))">启用</el-button>
            <el-button v-if="row.status==='active'" link type="warning" @click="deactivate(Number(row.id))">停用</el-button>
            <el-button link type="danger" @click="removeProduct(Number(row.id))">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 单位管理 -->
    <template v-else-if="active === 'units'">
      <el-card header="选择产品" class="mb">
        <ProductSelect v-model="selectedProductId" style="width:280px" placeholder="选择产品" />
      </el-card>
      <el-card header="添加单位（换算到基本单位）" class="mb">
        <el-form inline size="small">
          <el-form-item label="单位名"><el-input v-model="unitForm.unit_name" style="width:100px" /></el-form-item>
          <el-form-item label="换算系数"><el-input-number v-model="unitForm.factor_to_base" :min="0.0001" :step="0.1" /></el-form-item>
          <el-form-item label="采购"><el-switch v-model="unitForm.is_purchase" /></el-form-item>
          <el-form-item label="销售"><el-switch v-model="unitForm.is_sale" /></el-form-item>
          <el-form-item label="库存"><el-switch v-model="unitForm.is_stock" /></el-form-item>
          <el-button type="primary" @click="addUnit">添加并保存</el-button>
        </el-form>
        <p class="hint">例：袋 → kg，系数 25 表示 1 袋 = 25 kg</p>
      </el-card>
      <el-table :data="units" size="small">
        <el-table-column prop="unit_name" label="单位" width="100" />
        <el-table-column prop="factor_to_base" label="换算系数" width="110" />
        <el-table-column label="基本" width="80">
          <template #default="{ row }">{{ row.is_base ? '是' : '' }}</template>
        </el-table-column>
        <el-table-column label="采/销/存" width="120">
          <template #default="{ row }">
            {{ row.is_purchase ? '采' : '' }}{{ row.is_sale ? '销' : '' }}{{ row.is_stock ? '存' : '' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="{ $index, row }">
            <el-button v-if="!row.is_base" link type="primary" @click="setBaseUnit($index)">设为基本</el-button>
            <el-button link type="danger" @click="removeUnit($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- APP 排序 -->
    <template v-else-if="active === 'app-sorts'">
      <el-alert class="mb" type="info" :closable="false" title="调整顺序后点击「保存排序」生效；可控制小程序/APP 是否可见" />
      <div class="mb">
        <el-button type="primary" size="small" @click="saveSorts">保存排序</el-button>
      </div>
      <el-table :data="list" size="small">
        <el-table-column prop="sort_no" label="序号" width="80" />
        <el-table-column prop="product_code" label="编码" width="110" />
        <el-table-column prop="product_name" label="名称" min-width="160" />
        <el-table-column prop="product_type" label="类型" width="90" />
        <el-table-column label="可见" width="90">
          <template #default="{ row }">
            <el-switch v-model="row.is_visible" />
          </template>
        </el-table-column>
        <el-table-column label="调整" width="160">
          <template #default="{ $index }">
            <el-button link @click="moveSort($index, -1)">上移</el-button>
            <el-button link @click="moveSort($index, 1)">下移</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 生产规格绑定 -->
    <template v-else-if="active === 'specs'">
      <el-card header="新建规格绑定（挂工艺路线 / 工序工资）" class="mb">
        <el-form inline size="small">
          <el-form-item label="产品">
            <ProductSelect v-model="specForm.product_id" style="width:200px" />
          </el-form-item>
          <el-form-item label="规格码"><el-input v-model="specForm.spec_code" style="width:120px" placeholder="如 20kg袋" /></el-form-item>
          <el-form-item label="工艺路线">
            <RoutingSelect v-model="specForm.routing_id" style="width:180px" placeholder="可选" />
          </el-form-item>
          <el-form-item label="工序"><ProcessSelect v-model="specForm.process_id" style="width:160px" /></el-form-item>
          <el-form-item label="计件工资"><el-input-number v-model="specForm.wage" :min="0" :step="0.01" /></el-form-item>
          <el-form-item label="备注"><el-input v-model="specForm.remark" style="width:140px" /></el-form-item>
          <el-button type="primary" @click="createSpec">新建</el-button>
        </el-form>
      </el-card>
      <el-table :data="list" size="small">
        <el-table-column prop="product_code" label="产品编码" width="110" />
        <el-table-column prop="product_name" label="产品名称" min-width="140" />
        <el-table-column prop="spec_code" label="规格码" width="120" />
        <el-table-column prop="routing_id" label="工艺路线" width="100" />
        <el-table-column prop="process_wage_bind_json" label="工资绑定" min-width="160" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="80" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeSpec(Number(row.id))">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.head h2 { margin: 0; font-size: 18px; }
.mb { margin-bottom: 12px; }
.hint { margin: 8px 0 0; color: #888; font-size: 12px; }
</style>
