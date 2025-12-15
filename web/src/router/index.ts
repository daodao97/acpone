import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/c'
    },
    {
      path: '/setup',
      name: 'setup',
      component: () => import('../views/SetupView.vue')
    },
    {
      path: '/c',
      name: 'chat',
      component: () => import('../views/ChatView.vue')
    },
    {
      path: '/c/:sessionId',
      name: 'chat-session',
      component: () => import('../views/ChatView.vue')
    }
  ]
})

export default router
