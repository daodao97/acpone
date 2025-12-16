import { createApp } from 'vue'
import './style.css'
import 'markstream-vue/index.css'
import App from './App.vue'
import router from './router'
import { useTheme } from './composables/useTheme'

// Initialize theme on app startup
useTheme()

const app = createApp(App)
app.use(router)

// Navigation guard for setup check
router.beforeEach(async (to, _from, next) => {
  if (to.path === '/setup') {
    next()
    return
  }

  try {
    const res = await fetch('/api/setup/status')
    const data = await res.json()

    if (!data.ready && to.path !== '/setup') {
      next('/setup')
    } else {
      next()
    }
  } catch {
    // API failed, proceed anyway
    next()
  }
})

app.mount('#app')
