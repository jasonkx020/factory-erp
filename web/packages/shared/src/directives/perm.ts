import type { Directive } from 'vue'
import { useAuthStore } from '../stores/auth'

/** v-perm="'生产管理:扫码报工:新增'" — 无权限则移除节点 */
export const vPerm: Directive<HTMLElement, string> = {
  mounted(el, binding) {
    const auth = useAuthStore()
    const code = binding.value
    if (code && !auth.hasPerm(code)) {
      el.style.display = 'none'
      el.setAttribute('data-perm-denied', code)
    }
  },
  updated(el, binding) {
    const auth = useAuthStore()
    const code = binding.value
    if (code && !auth.hasPerm(code)) {
      el.style.display = 'none'
    } else {
      el.style.display = ''
      el.removeAttribute('data-perm-denied')
    }
  },
}
