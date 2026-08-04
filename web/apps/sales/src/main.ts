import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { Button, Field, Cell, CellGroup, NoticeBar, Steps, Step, Toast } from 'vant'
import 'vant/lib/index.css'
import '@erp/shared/styles/tokens.css'
import App from './App.vue'
import router from './router'
const app = createApp(App)
app.use(createPinia()).use(router)
;[Button, Field, Cell, CellGroup, NoticeBar, Steps, Step, Toast].forEach(c => app.use(c))
app.mount('#app')
