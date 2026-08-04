/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}

declare module '@erp/shared' {
  export * from '../../packages/shared/src/index'
}

declare module '@erp/shared/styles/tokens.css'
