import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '../api/endpoints'
import { clearTokens, getAccessToken, setTokens } from '../api/client'
import type { LoginUser, MeData } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(getAccessToken())
  const user = ref<LoginUser | null>(null)
  const roles = ref<string[]>([])
  const permissions = ref<string[]>([])
  const menus = ref<MeData['menus']>([])
  const fieldPolicies = ref<MeData['field_policies']>([])
  const loading = ref(false)
  const error = ref('')

  const isLoggedIn = computed(() => !!accessToken.value)

  async function login(loginName: string, password: string, clientType = 'web') {
    loading.value = true
    error.value = ''
    try {
      const r = await authApi.login(loginName, password, clientType)
      if (r.code !== 1 || !r.data) {
        error.value = r.msg || 'LOGIN_FAILED'
        return false
      }
      setTokens(r.data.access_token, r.data.refresh_token)
      accessToken.value = r.data.access_token
      user.value = r.data.user || null
      roles.value = r.data.roles || []
      permissions.value = r.data.permissions || []
      await fetchMe()
      return true
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    const r = await authApi.me()
    if (r.code !== 1 || !r.data) {
      if (r.msg === 'UNAUTHORIZED') logout()
      return false
    }
    user.value = { ...r.data.user, employee_id: user.value?.employee_id, name: user.value?.name }
    roles.value = r.data.roles || []
    permissions.value = r.data.permissions || []
    menus.value = r.data.menus || []
    fieldPolicies.value = r.data.field_policies || []
    return true
  }

  function logout() {
    clearTokens()
    accessToken.value = ''
    user.value = null
    roles.value = []
    permissions.value = []
    menus.value = []
    fieldPolicies.value = []
  }

  function hasPerm(code: string) {
    if (permissions.value.includes('*:*:*')) return true
    if (roles.value.includes('sys_admin') || roles.value.includes('系统管理员')) return true
    return permissions.value.includes(code)
  }

  function fieldVisible(key: string) {
    const hit = fieldPolicies.value.find((p) => p.field_key === key)
    if (!hit) return true
    return hit.visible
  }

  return {
    accessToken, user, roles, permissions, menus, fieldPolicies,
    loading, error, isLoggedIn,
    login, logout, fetchMe, hasPerm, fieldVisible,
  }
})
