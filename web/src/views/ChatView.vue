<script setup lang="ts">
import { onMounted, watch, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSessionStore } from '../stores/session'
import Sidebar from '../components/Sidebar.vue'
import ChatContainer from '../components/ChatContainer.vue'
import SettingsButton from '../components/SettingsButton.vue'

const route = useRoute()
const router = useRouter()
const store = useSessionStore()
const sidebarCollapsed = ref(false)

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

// Update URL when session changes
watch(store.currentSessionId, (sessionId) => {
  const currentPath = route.path
  const newPath = sessionId ? `/c/${sessionId}` : '/c'
  if (currentPath !== newPath) {
    router.push(newPath)
  }
})

// Handle route changes
watch(() => route.params.sessionId, async (sessionId) => {
  if (sessionId && sessionId !== store.currentSessionId.value) {
    await store.selectSession(sessionId as string)
  }
}, { immediate: true })

onMounted(async () => {
  // Get session ID from route
  const sessionIdFromRoute = route.params.sessionId as string | undefined

  await Promise.all([store.loadAgents(), store.loadWorkspaces()])

  // Load sessions but skip auto-select if route has sessionId
  await store.loadSessions(sessionIdFromRoute ? true : false)

  // Load session from route if present
  if (sessionIdFromRoute) {
    await store.selectSession(sessionIdFromRoute)
  }
})
</script>

<template>
  <div class="app-layout">
    <Sidebar v-show="!sidebarCollapsed" @collapse="toggleSidebar" />
    <main class="stage">
      <ChatContainer />
      <SettingsButton class="settings-trigger" />
    </main>
    <button v-if="sidebarCollapsed" class="sidebar-expand" @click="toggleSidebar" title="Show sidebar">
      <span class="expand-icon">☰</span>
    </button>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  background: var(--bg-root);
}

.stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  height: 100%;
  overflow: hidden;
  background: var(--bg-root);
}

.sidebar-expand {
  position: fixed;
  top: 16px;
  left: 16px;
  z-index: 100;
  width: 32px;
  height: 32px;
  border: 1px solid var(--bg-element);
  background: var(--bg-surface);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--duration-fast);
}

.sidebar-expand:hover {
  background: var(--bg-surface-hover);
  color: var(--text-primary);
  border-color: var(--text-tertiary);
}

.expand-icon {
  font-size: 14px;
  line-height: 1;
}

.settings-trigger {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 50;
  opacity: 0.5;
  transition: opacity var(--duration-fast);
}

.settings-trigger:hover {
  opacity: 1;
}
</style>
