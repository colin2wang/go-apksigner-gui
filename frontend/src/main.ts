import { createApp } from 'vue'
import TDesign from 'tdesign-vue-next'
import 'tdesign-vue-next/es/style/index.css'
import './style.css'
import App from './App.vue'
import { initStore } from './stores/app'

// TDesign 深色模式
document.documentElement.setAttribute('theme-mode', 'dark')

createApp(App).use(TDesign).mount('#app')

initStore().catch((err) => {
  console.error('初始化应用数据失败', err)
})
