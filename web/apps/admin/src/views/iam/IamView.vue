<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { iamApi } from '@erp/shared'
import HrPermView from './HrPermView.vue'

const props = defineProps<{ module: string }>()

const users = ref<Record<string, unknown>[]>([])
const roles = ref<Record<string, unknown>[]>([])
const groups = ref<Record<string, unknown>[]>([])
const permissions = ref<Record<string, unknown>[]>([])
const menus = ref<Record<string, unknown>[]>([])
const fieldPolicies = ref<Record<string, unknown>[]>([])
const loginPolicy = ref<Record<string, unknown>>({})
const loading = ref(false)

async function loadAll() {
  if (props.module === '权限分配') return
  loading.value = true
  try {
    const [u, r, g, p, m, f, lp] = await Promise.all([
      iamApi.users(),
      iamApi.roles(),
      iamApi.groups(),
      iamApi.permissions(),
      iamApi.menus(),
      iamApi.fieldPolicies(),
      iamApi.loginPolicy(),
    ])
    users.value = (u.data as { list?: Record<string, unknown>[] })?.list || []
    roles.value = (r.data as { list?: Record<string, unknown>[] })?.list || []
    groups.value = (g.data as { list?: Record<string, unknown>[] })?.list || []
    permissions.value = (p.data as { list?: Record<string, unknown>[] })?.list || []
    menus.value = (m.data as { list?: Record<string, unknown>[] })?.list || []
    fieldPolicies.value = (f.data as { list?: Record<string, unknown>[] })?.list || []
    loginPolicy.value = (lp.data as Record<string, unknown>) || {}
  } finally {
    loading.value = false
  }
}

async function freeze(id: number, frozen: boolean) {
  const res = frozen ? await iamApi.unfreeze(id) : await iamApi.freeze(id)
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success(frozen ? '已解冻' : '已冻结')
  await loadAll()
}

async function saveLoginPolicy() {
  const res = await iamApi.saveLoginPolicy({ ...loginPolicy.value })
  if (res.code !== 1) return ElMessage.error(res.msg)
  ElMessage.success('登录策略已保存')
}

onMounted(loadAll)
</script>

<template>
  <HrPermView v-if="module === '权限分配'" />
  <div v-else v-loading="loading" class="iam">
    <h2 class="title">{{ module }}</h2>
    <p class="desc">权限能力落在「人事·权限分配」与「系统·自定义权限/菜单/登录控制/账户冻结」，不另立核心功能名。</p>

    <template v-if="module === '自定义权限' || module === '成本隐藏'">
      <h3>权限码字典</h3>
      <el-table :data="permissions.slice(0, 50)" border height="280">
        <el-table-column prop="code" label="权限码" min-width="220" />
        <el-table-column prop="domain" label="域" />
        <el-table-column prop="module" label="模块" />
        <el-table-column prop="action" label="动作" />
      </el-table>
      <h3 style="margin-top:16px">字段策略（成本隐藏）</h3>
      <el-table :data="fieldPolicies" border>
        <el-table-column prop="role_id" label="角色ID" width="90" />
        <el-table-column prop="field_key" label="字段" />
        <el-table-column prop="field_name" label="名称" />
        <el-table-column prop="visible" label="可见" />
        <el-table-column prop="editable" label="可编辑" />
      </el-table>
    </template>

    <template v-else-if="module === '自定义菜单'">
      <el-table :data="menus" border>
        <el-table-column prop="role_id" label="角色ID" width="90" />
        <el-table-column prop="domain" label="域" />
        <el-table-column prop="module" label="模块" />
        <el-table-column prop="menu_key" label="菜单键" />
        <el-table-column prop="visible" label="可见" />
        <el-table-column prop="sort_no" label="排序" />
      </el-table>
    </template>

    <template v-else-if="module === '登录控制'">
      <el-form label-width="140px" style="max-width:480px">
        <el-form-item label="最大失败次数">
          <el-input-number v-model="loginPolicy.max_fail_count" :min="1" />
        </el-form-item>
        <el-form-item label="锁定分钟">
          <el-input-number v-model="loginPolicy.lock_minutes" :min="1" />
        </el-form-item>
        <el-form-item label="会话 TTL(分)">
          <el-input-number v-model="loginPolicy.session_ttl_min" :min="10" />
        </el-form-item>
        <el-form-item label="密码最短长度">
          <el-input-number v-model="loginPolicy.password_min_len" :min="6" />
        </el-form-item>
        <el-button type="primary" @click="saveLoginPolicy">保存策略</el-button>
      </el-form>
    </template>

    <template v-else-if="module === '账户冻结'">
      <el-table :data="users" border>
        <el-table-column prop="login_name" label="登录名" />
        <el-table-column prop="name" label="姓名" />
        <el-table-column prop="status" label="状态" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button
              link
              :type="row.status === 'frozen' ? 'success' : 'danger'"
              @click="freeze(Number(row.id), row.status === 'frozen')"
            >{{ row.status === 'frozen' ? '解冻' : '冻结' }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </template>
  </div>
</template>

<style scoped>
.iam { background: #fff; padding: 16px; border-radius: 8px; border: 1px solid #d5dde3; }
.title { margin: 0 0 4px; }
.desc { color: #5c6b75; font-size: 13px; margin: 0 0 12px; }
.row { display: flex; gap: 8px; margin-bottom: 12px; }
.steps { line-height: 1.8; }
</style>
