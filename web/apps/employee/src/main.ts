import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import { Button, Field, Cell, CellGroup, NoticeBar, Empty, Toast } from 'vant'
import 'vant/lib/index.css'
import '@erp/shared/styles/tokens.css'
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(createPinia()).use(router).use(ElementPlus)
;[Button, Field, Cell, CellGroup, NoticeBar, Empty, Toast].forEach((c) => app.use(c))
app.mount('#app')
