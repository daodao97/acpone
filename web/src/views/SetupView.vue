<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

interface DependencyItem {
  name: string
  command?: string
  package?: string
  status: 'checking' | 'ready' | 'missing' | 'not_installed' | 'installing' | 'error' | 'blocked'
  message?: string
  install?: string
}

const environment = ref<DependencyItem[]>([])
const agents = ref<DependencyItem[]>([])
const acpPackages = ref<DependencyItem[]>([])
const isReady = ref(false)
const isInstalling = ref(false)
const error = ref('')
let eventSource: EventSource | null = null

const canInstall = computed(() => {
  const envReady = environment.value.every(e => e.status === 'ready')
  const hasAgentMissing = agents.value.some(a => a.status === 'missing')
  const hasACPNotInstalled = acpPackages.value.some(p => p.status === 'not_installed')
  return envReady && (hasAgentMissing || hasACPNotInstalled) && !isInstalling.value
})

const hasBlocked = computed(() => {
  return acpPackages.value.some(p => p.status === 'blocked')
})

const hasMissing = computed(() => {
  return environment.value.some(e => e.status === 'missing') ||
         agents.value.some(a => a.status === 'missing')
})

function subscribeStatus() {
  eventSource = new EventSource('/api/setup/subscribe')

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      environment.value = data.environment || []
      agents.value = data.agents || []
      acpPackages.value = data.acpPackages || []
      isReady.value = data.ready

      if (data.ready) {
        eventSource?.close()
        router.replace('/')
      }
    } catch {
      // Ignore
    }
  }

  eventSource.onerror = () => {
    error.value = 'Connection lost, retrying...'
    eventSource?.close()
    setTimeout(subscribeStatus, 2000)
  }
}

async function startInstall() {
  isInstalling.value = true
  error.value = ''

  // Mark missing agents as installing
  agents.value.forEach(a => {
    if (a.status === 'missing') {
      a.status = 'installing'
    }
  })

  // Mark not_installed ACP packages as installing
  acpPackages.value.forEach(p => {
    if (p.status === 'not_installed') {
      p.status = 'installing'
    }
  })

  try {
    const res = await fetch('/api/setup/install', { method: 'POST' })
    const reader = res.body?.getReader()
    const decoder = new TextDecoder()

    if (!reader) throw new Error('No response body')

    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('event: ')) continue
        if (line.startsWith('data: ')) {
          const data = JSON.parse(line.slice(6))

          // Handle agent installation progress
          if (data.type === 'agent' && data.index !== undefined) {
            const agent = agents.value[data.index]
            if (agent) {
              if (data.status) agent.status = data.status
              if (data.message) agent.message = data.message
            }
          }

          // Handle ACP package installation progress
          if (data.type === 'acp' && data.index !== undefined) {
            const pkg = acpPackages.value[data.index]
            if (pkg) {
              if (data.status) pkg.status = data.status
              if (data.message) pkg.message = data.message
            }
          }

          if (data.success !== undefined) {
            if (data.success) {
              isReady.value = true
              setTimeout(() => router.replace('/'), 500)
            } else if (data.error) {
              error.value = data.error
            }
          }
        }
      }
    }
  } catch {
    error.value = 'Installation failed'
  } finally {
    isInstalling.value = false
  }
}

onMounted(() => subscribeStatus())
onUnmounted(() => eventSource?.close())

function getStatusIcon(status: string) {
  switch (status) {
    case 'ready': return '✓'
    case 'not_installed': return '○'
    case 'installing':
    case 'checking': return '◐'
    case 'error':
    case 'missing': return '✗'
    case 'blocked': return '⊘'
    default: return '○'
  }
}

function getStatusClass(status: string) {
  switch (status) {
    case 'ready': return 'success'
    case 'error':
    case 'missing': return 'error'
    case 'installing':
    case 'checking': return 'pending'
    case 'blocked': return 'blocked'
    default: return ''
  }
}

function isUrl(str: string) {
  return str?.startsWith('http://') || str?.startsWith('https://')
}
</script>

<template>
  <div class="setup-container">
    <div class="setup-card">
      <div class="setup-header">
        <h1>ACPone Setup</h1>
        <p class="subtitle">Checking dependencies...</p>
      </div>

      <!-- Environment Section -->
      <div v-if="environment.length > 0" class="section">
        <h3 class="section-title">Environment</h3>
        <div class="items-list">
          <div
            v-for="item in environment"
            :key="item.command"
            class="item"
            :class="getStatusClass(item.status)"
          >
            <span class="status-icon">{{ getStatusIcon(item.status) }}</span>
            <div class="item-info">
              <span class="item-name">{{ item.name }}</span>
              <span class="item-detail">{{ item.command }}</span>
            </div>
            <div class="item-status">
              <span>{{ item.message }}</span>
              <template v-if="item.install && item.status === 'missing'">
                <a v-if="isUrl(item.install)" :href="item.install" target="_blank" class="install-link">
                  Download Node.js →
                </a>
                <span v-else class="install-hint">{{ item.install }}</span>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- Agents Section -->
      <div v-if="agents.length > 0" class="section">
        <h3 class="section-title">Agents</h3>
        <div class="items-list">
          <div
            v-for="item in agents"
            :key="item.command"
            class="item"
            :class="getStatusClass(item.status)"
          >
            <span class="status-icon">{{ getStatusIcon(item.status) }}</span>
            <div class="item-info">
              <span class="item-name">{{ item.name }}</span>
              <span class="item-detail">{{ item.command }}</span>
            </div>
            <div class="item-status">
              <span>{{ item.message }}</span>
              <template v-if="item.install && item.status === 'missing'">
                <a v-if="isUrl(item.install)" :href="item.install" target="_blank" class="install-link">
                  Download →
                </a>
                <span v-else class="install-hint">{{ item.install }}</span>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- ACP Packages Section -->
      <div v-if="acpPackages.length > 0" class="section">
        <h3 class="section-title">ACP Packages</h3>
        <div class="items-list">
          <div
            v-for="item in acpPackages"
            :key="item.package"
            class="item"
            :class="getStatusClass(item.status)"
          >
            <span class="status-icon">{{ getStatusIcon(item.status) }}</span>
            <div class="item-info">
              <span class="item-name">{{ item.name }}</span>
              <span class="item-detail">{{ item.package }}</span>
            </div>
            <span class="item-status">{{ item.message }}</span>
          </div>
        </div>
      </div>

      <div v-if="error" class="error-message">{{ error }}</div>

      <div class="setup-actions">
        <button
          v-if="canInstall"
          class="install-btn"
          @click="startInstall"
        >
          Install Dependencies
        </button>
        <button
          v-if="isInstalling"
          class="install-btn"
          disabled
        >
          <span class="spinner"></span>
          Installing...
        </button>
        <button
          v-if="(hasBlocked || hasMissing) && !canInstall && !isInstalling && !isReady"
          class="install-btn blocked"
          disabled
        >
          Install Prerequisites First
        </button>
        <button
          v-if="isReady"
          class="continue-btn"
          @click="router.replace('/c')"
        >
          Continue to Chat
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.setup-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-root);
  padding: 20px;
}

.setup-card {
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-lg);
  padding: 40px;
  max-width: 550px;
  width: 100%;
  box-shadow: var(--shadow-lg);
}

.setup-header {
  text-align: center;
  margin-bottom: 24px;
}

.setup-header h1 {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 14px;
  margin: 0;
}

.section {
  margin-bottom: 20px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin: 0 0 12px;
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--bg-surface-hover);
  border-radius: var(--radius-md);
  border: 1px solid var(--bg-element);
}

.item.success { border-color: var(--status-success-border); }
.item.error { border-color: var(--status-error-border); }
.item.pending { border-color: var(--status-warning-border); }
.item.blocked { border-color: var(--status-muted-border); opacity: 0.7; }

.status-icon {
  font-size: 14px;
  width: 20px;
  text-align: center;
}

.item.success .status-icon { color: var(--status-success); }
.item.error .status-icon { color: var(--status-error); }
.item.pending .status-icon { color: var(--status-warning); animation: pulse 1s infinite; }
.item.blocked .status-icon { color: var(--status-muted); }

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.item-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.item-detail {
  font-size: 11px;
  color: var(--text-tertiary);
  font-family: var(--font-mono);
}

.item-status {
  font-size: 11px;
  color: var(--text-secondary);
  text-align: right;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.install-hint {
  font-size: 10px;
  color: var(--text-tertiary);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.install-link {
  font-size: 11px;
  color: var(--accent-success);
  text-decoration: none;
  border: none;
  font-weight: 500;
  transition: opacity var(--duration-fast);
}

.install-link:hover {
  opacity: 0.8;
  border: none;
}

.error-message {
  background: var(--status-error-bg);
  color: var(--status-error);
  padding: 12px;
  border-radius: var(--radius-md);
  font-size: 13px;
  margin-bottom: 16px;
  border: 1px solid var(--status-error-border);
}

.setup-actions {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

.install-btn,
.continue-btn {
  padding: 12px 32px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--duration-fast);
  display: flex;
  align-items: center;
  gap: 8px;
}

.install-btn {
  background: var(--accent-primary);
  color: var(--bg-root);
  border: none;
}

.install-btn:hover:not(:disabled) { opacity: 0.9; }
.install-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.install-btn.blocked { background: var(--text-tertiary); }

.continue-btn {
  background: var(--accent-success);
  color: var(--bg-root);
  border: none;
}

.continue-btn:hover { opacity: 0.9; }

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid transparent;
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
