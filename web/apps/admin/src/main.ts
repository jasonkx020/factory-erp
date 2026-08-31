import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import '@erp/shared/styles/tokens.css'
import '@erp/shared/styles/factory.css'
import './style.css'
import App from './App.vue'
import router from './router'
import { vPerm } from '@erp/shared'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)
app.directive('perm', vPerm)
app.mount('#app')
