<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { authApi, useAuthStore } from '@erp/shared'

const auth = useAuthStore()
const router = useRouter()
const saving = ref(false)
const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

const pwdErr: Record<string, string> = {
  INVALID_REQUEST: '请填写旧密码和新密码',
  OLD_PASSWORD_WRONG: '旧密码不正确',
  PASSWORD_TOO_SHORT: '新密码过短（至少 8 位，以登录策略为准）',
  PASSWORD_NEED_LETTER: '新密码须包含字母',
  PASSWORD_NEED_DIGIT: '新密码须包含数字',
  USER_NOT_FOUND: '账号不存在',
}

const displayName = computed(() => auth.user?.name || auth.user?.employee_name || '—')
const loginName = computed(() => auth.user?.login_name || '—')
const empNo = computed(() => auth.user?.emp_no || '—')
const roleText = computed(() => (auth.roles || []).filter(Boolean).join('、') || '—')

onMounted(() => {
  void auth.fetchMe()
})

function goLogin() {
  auth.logout()
  router.replace('/login')
}

async function onChangePassword() {
  if (!form.old_password || !form.new_password) {
    ElMessage.warning('请填写旧密码和新密码')
    return
  }
  if (form.new_password !== form.confirm_password) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  if (form.new_password === form.old_password) {
    ElMessage.warning('新密码不能与旧密码相同')
    return
  }
  saving.value = true
  try {
    const res = await authApi.changePassword(form.old_password, form.new_password)
    if (res.code !== 1) {
      ElMessage.error(pwdErr[res.msg] || res.msg || '修改失败')
      return
    }
    ElMessage.success('密码已修改，请重新登录')
    goLogin()
  } finally {
    saving.value = false
  }
}

async function onLogout() {
  try {
    await ElMessageBox.confirm('确定退出当前账号？', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  goLogin()
}
</script>

<template>
  <div class="acc">
    <header class="page-head">
      <div>
        <h2 class="title">个人中心</h2>
        <p class="desc">查看当前登录账号，修改密码或退出。修改密码后需重新登录。</p>
      </div>
      <div class="head-meta">
        <span class="meta-pill">{{ loginName }}</span>
      </div>
    </header>

    <section class="card">
      <h3 class="card-title">账号信息</h3>
      <dl class="info">
        <div><dt>登录名</dt><dd>{{ loginName }}</dd></div>
        <div><dt>姓名</dt><dd>{{ displayName }}</dd></div>
        <div><dt>工号</dt><dd>{{ empNo }}</dd></div>
        <div><dt>角色</dt><dd>{{ roleText }}</dd></div>
      </dl>
    </section>

    <section class="card">
      <h3 class="card-title">修改密码</h3>
      <p class="hint">新密码至少 8 位，须同时包含字母和数字（以系统登录策略为准）。</p>
      <el-form label-width="96px" class="pwd-form" @submit.prevent="onChangePassword">
        <el-form-item label="旧密码">
          <el-input v-model="form.old_password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="form.new_password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input v-model="form.confirm_password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" native-type="submit">保存新密码</el-button>
        </el-form-item>
      </el-form>
    </section>

    <section class="card">
      <h3 class="card-title">退出登录</h3>
      <p class="hint">退出后需重新输入账号密码进入后台。</p>
      <el-button type="danger" plain @click="onLogout">退出登录</el-button>
    </section>
  </div>
</template>

<style scoped>
.acc { display: flex; flex-direction: column; gap: 12px; max-width: 720px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.title { margin: 0 0 4px; font-size: 18px; font-weight: 600; color: #1f2a33; }
.desc { color: #5c6b75; font-size: 13px; margin: 0; line-height: 1.5; }
.head-meta { flex-shrink: 0; padding-top: 2px; }
.meta-pill {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 999px;
  background: #eef6f1;
  color: #2f6b4f;
  font-size: 12px;
  font-weight: 500;
}
.card {
  background: #fff;
  padding: 16px 18px;
  border-radius: 10px;
  border: 1px solid #e2e8ee;
}
.card-title { margin: 0 0 12px; font-size: 15px; font-weight: 600; color: #1f2a33; }
.hint { color: #5c6b75; font-size: 13px; margin: 0 0 12px; line-height: 1.5; }
.info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 16px;
  margin: 0;
}
.info div { min-width: 0; }
.info dt { font-size: 12px; color: #6b7a85; margin-bottom: 2px; }
.info dd { margin: 0; font-size: 14px; color: #1f2a33; word-break: break-all; }
.pwd-form { max-width: 420px; }
@media (max-width: 720px) {
  .info { grid-template-columns: 1fr; }
}
</style>
