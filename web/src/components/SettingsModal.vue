<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useSessionStore } from '../stores/session'
import { updateAgentPermission, updateAgentEnv } from '../api'
import { useTheme } from '../composables/useTheme'
import { useI18n } from '../composables/useI18n'
import type { Agent } from '../types'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: [] }>()

const store = useSessionStore()
const { agents, defaultAgent } = store
const { currentTheme, toggleTheme } = useTheme()
const { t, currentLang, setLang } = useI18n()

const saving = ref<string | null>(null)
const error = ref<string | null>(null)

// Env editing state
const editingEnv = ref<string | null>(null)
const envEdits = reactive<Record<string, { key: string; value: string }[]>>({})
const savingEnv = ref<string | null>(null)

// Initialize env edits when modal opens
watch(() => props.visible, (visible) => {
  if (visible) {
    agents.value.forEach(agent => {
      envEdits[agent.id] = Object.entries(agent.env || {}).map(([key, value]) => ({ key, value }))
    })
  }
})

function formatCommand(agent: { command?: string; args?: string[] }) {
  if (!agent.command) return '-'
  const args = agent.args?.join(' ') || ''
  return `${agent.command} ${args}`.trim()
}

async function togglePermission(agent: Agent) {
  const currentMode = agent.permissionMode || 'default'
  const newMode = currentMode === 'bypass' ? 'default' : 'bypass'

  saving.value = agent.id
  error.value = null

  const result = await updateAgentPermission(agent.id, newMode)

  if (result.success) {
    // Update local state
    agent.permissionMode = newMode
  } else {
    error.value = result.error || 'Failed to update'
  }

  saving.value = null
}

function toggleEnvEdit(agentId: string) {
  if (editingEnv.value === agentId) {
    editingEnv.value = null
  } else {
    editingEnv.value = agentId
  }
}

function addEnvVar(agentId: string) {
  if (!envEdits[agentId]) {
    envEdits[agentId] = []
  }
  envEdits[agentId].push({ key: '', value: '' })
}

function removeEnvVar(agentId: string, index: number) {
  envEdits[agentId]?.splice(index, 1)
}

async function saveEnv(agent: Agent) {
  savingEnv.value = agent.id
  error.value = null

  // Convert array to object, filtering out empty keys
  const env: Record<string, string> = {}
  for (const item of envEdits[agent.id] || []) {
    if (item.key.trim()) {
      env[item.key.trim()] = item.value
    }
  }

  const result = await updateAgentEnv(agent.id, env)

  if (result.success) {
    agent.env = env
    editingEnv.value = null
  } else {
    error.value = result.error || 'Failed to save env'
  }

  savingEnv.value = null
}

function getEnvCount(agent: Agent): number {
  return Object.keys(agent.env || {}).length
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="emit('close')">
      <div class="modal-content">
        <div class="modal-header">
          <h2>{{ t('settings.title') }}</h2>
          <button class="close-btn" @click="emit('close')">&times;</button>
        </div>

        <div class="modal-body">
          <div v-if="error" class="error-message">{{ error }}</div>

          <section class="section">
            <h3>{{ t('settings.agents') }}</h3>
            <p class="section-desc">
              {{ t('settings.agents.desc') }}
              Default: <code>{{ defaultAgent }}</code>
            </p>

            <div class="agent-list">
              <div
                v-for="agent in agents"
                :key="agent.id"
                class="agent-card"
                :class="{ default: agent.id === defaultAgent }"
              >
                <div class="agent-card-header">
                  <span class="agent-name" :class="agent.id">
                    {{ agent.name }}
                  </span>
                  <span v-if="agent.id === defaultAgent" class="default-badge">
                    {{ t('settings.default') }}
                  </span>
                </div>

                <div class="agent-card-body">
                  <div class="info-row">
                    <span class="info-label">ID:</span>
                    <code class="info-value">{{ agent.id }}</code>
                  </div>
                  <div class="info-row">
                    <span class="info-label">{{ t('settings.permission') }}:</span>
                    <div class="permission-group" :class="{ disabled: saving === agent.id }">
                      <button
                        class="permission-btn default"
                        :class="{ active: (agent.permissionMode || 'default') === 'default' }"
                        :disabled="saving === agent.id"
                        @click="agent.permissionMode !== 'default' && togglePermission(agent)"
                      >
                        {{ t('settings.permission.default') }}
                      </button>
                      <button
                        class="permission-btn bypass"
                        :class="{ active: agent.permissionMode === 'bypass' }"
                        :disabled="saving === agent.id"
                        @click="agent.permissionMode !== 'bypass' && togglePermission(agent)"
                      >
                        {{ t('settings.permission.bypass') }}
                      </button>
                    </div>
                  </div>
                  <div class="info-row">
                    <span class="info-label">Command:</span>
                    <code class="info-value command">{{ formatCommand(agent) }}</code>
                  </div>
                  <div class="info-row env-row">
                    <span class="info-label">{{ t('settings.env') }}:</span>
                    <button class="env-toggle" @click="toggleEnvEdit(agent.id)">
                      {{ getEnvCount(agent) }} variables
                      <span class="toggle-icon">{{ editingEnv === agent.id ? '▲' : '▼' }}</span>
                    </button>
                  </div>

                  <!-- Env Editor -->
                  <div v-if="editingEnv === agent.id" class="env-editor">
                    <div v-for="(item, idx) in envEdits[agent.id]" :key="idx" class="env-item">
                      <input
                        v-model="item.key"
                        class="env-input env-key"
                        placeholder="KEY"
                      />
                      <span class="env-eq">=</span>
                      <input
                        v-model="item.value"
                        class="env-input env-value"
                        placeholder="value"
                      />
                      <button class="env-remove" @click="removeEnvVar(agent.id, idx)">×</button>
                    </div>
                    <div class="env-actions">
                      <button class="env-add" @click="addEnvVar(agent.id)">+ Add</button>
                      <button
                        class="env-save"
                        :disabled="savingEnv === agent.id"
                        @click="saveEnv(agent)"
                      >
                        {{ savingEnv === agent.id ? t('settings.saving') : t('settings.save') }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Permission Legend moved inside Agents section -->
            <div class="permission-legend">
              <div class="legend-header">{{ t('settings.permission.mode') }}</div>
              <div class="mode-list">
                <div class="mode-item">
                  <span class="permission-badge default">{{ t('settings.permission.default') }}</span>
                  <span class="mode-desc">
                    {{ t('settings.permission.default.desc') }}
                  </span>
                </div>
                <div class="mode-item">
                  <span class="permission-badge bypass">{{ t('settings.permission.bypass') }}</span>
                  <span class="mode-desc">
                    {{ t('settings.permission.bypass.desc') }}
                  </span>
                </div>
              </div>
            </div>

          </section>

          <section class="section">
            <h3>{{ t('settings.appearance') }}</h3>
            
            <div class="setting-row">
              <span class="setting-label">{{ t('settings.theme') }}</span>
              <button class="setting-btn" @click="toggleTheme">
                {{ currentTheme === 'dark' ? t('settings.theme.dark') : t('settings.theme.light') }}
              </button>
            </div>

            <div class="setting-row">
              <span class="setting-label">{{ t('settings.language') }}</span>
              <div class="lang-group">
                <button 
                  class="lang-btn" 
                  :class="{ active: currentLang === 'en' }"
                  @click="setLang('en')"
                >
                  English
                </button>
                <button 
                  class="lang-btn" 
                  :class="{ active: currentLang === 'zh' }"
                  @click="setLang('zh')"
                >
                  中文
                </button>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-lg);
  width: 90%;
  max-width: 680px;
  max-height: 85vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
  color: var(--text-primary);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid var(--bg-element);
  background: var(--bg-surface-hover);
}

.modal-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-primary);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-tertiary);
  font-size: 20px;
  cursor: pointer;
  padding: 4px;
  line-height: 1;
  border-radius: 4px;
  transition: all var(--duration-fast);
}

.close-btn:hover {
  color: var(--text-primary);
  background: var(--bg-element);
}

.modal-body {
  padding: 24px;
  overflow-y: auto;
}

.error-message {
  background: rgba(207, 51, 51, 0.1);
  color: #ff8888;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  margin-bottom: 16px;
  font-size: 13px;
  border: 1px solid var(--accent-error);
}

.section {
  margin-bottom: 32px;
}

.section:last-child {
  margin-bottom: 0;
}

.section h3 {
  margin: 0 0 12px 0;
  font-size: 12px;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-weight: 700;
}

.section-desc {
  margin: 0 0 16px 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.section-desc code {
  background: var(--bg-element);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: var(--font-mono);
}

.agent-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.agent-card {
  background: var(--bg-surface-hover);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-md);
  padding: 16px;
  transition: border-color var(--duration-fast);
}

.agent-card.default {
  border-color: var(--text-tertiary);
}

.agent-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.agent-name {
  font-size: 14px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 4px;
  color: white;
  background: var(--bg-element);
}

.agent-name.claude {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.agent-name.codex {
  background: linear-gradient(135deg, #10a37f 0%, #1a7f5a 100%);
}

.default-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--bg-element);
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 600;
}

.agent-card-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}

.info-label {
  color: var(--text-secondary);
  min-width: 80px;
}

.info-value {
  background: var(--bg-root);
  padding: 3px 8px;
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-primary);
}

.info-value.command {
  word-break: break-all;
}

/* Permission Button Group */
.permission-group {
  display: flex;
  background: var(--bg-element);
  padding: 2px;
  border-radius: 6px;
}

.permission-group.disabled {
  opacity: 0.6;
}

.permission-btn {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  cursor: pointer;
  transition: all var(--duration-fast);
}

.permission-btn:disabled {
  cursor: not-allowed;
}

.permission-btn.default.active {
  background: rgba(111, 207, 151, 0.15);
  color: #6fcf97;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.permission-btn.bypass.active {
  background: rgba(207, 51, 51, 0.15);
  color: #f0a0a0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.permission-btn:not(.active):hover:not(:disabled) {
  color: var(--text-secondary);
}

/* Env Editor Styles */
.env-row {
  align-items: flex-start;
}

.env-toggle {
  background: var(--bg-root);
  border: 1px solid var(--bg-element);
  color: var(--text-secondary);
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all var(--duration-fast);
}

.env-toggle:hover {
  border-color: var(--text-tertiary);
  color: var(--text-primary);
}

.toggle-icon {
  font-size: 8px;
  color: var(--text-tertiary);
}

.env-editor {
  margin-top: 8px;
  padding: 12px;
  background: var(--bg-root);
  border-radius: var(--radius-md);
  border: 1px solid var(--bg-element);
}

.env-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.env-input {
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  color: var(--text-primary);
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-family: var(--font-mono);
  transition: border-color var(--duration-fast);
}

.env-input:focus {
  outline: none;
  border-color: var(--text-tertiary);
}

.env-key {
  width: 180px;
}

.env-value {
  flex: 1;
}

.env-eq {
  color: var(--text-tertiary);
  font-family: var(--font-mono);
}

.env-remove {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
  transition: color var(--duration-fast);
}

.env-remove:hover {
  color: var(--accent-error);
}

.env-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--bg-element);
}

.env-add {
  background: transparent;
  border: 1px dashed var(--text-tertiary);
  color: var(--text-secondary);
  padding: 6px 12px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all var(--duration-fast);
}

.env-add:hover {
  border-color: var(--text-secondary);
  color: var(--text-primary);
}

.env-save {
  background: var(--text-primary);
  border: none;
  color: var(--bg-root);
  padding: 6px 16px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity var(--duration-fast);
}

.env-save:hover:not(:disabled) {
  opacity: 0.9;
}

.env-save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Settings Rows */
.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-surface-hover);
  padding: 12px 16px;
  border-radius: var(--radius-md);
  border: 1px solid var(--bg-element);
  margin-bottom: 12px;
}

.setting-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.setting-btn {
  background: var(--bg-element);
  border: 1px solid transparent;
  color: var(--text-primary);
  padding: 6px 12px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  min-width: 100px;
  transition: all var(--duration-fast);
}

.setting-btn:hover {
  background: var(--bg-surface);
  border-color: var(--text-tertiary);
}

.lang-group {
  display: flex;
  background: var(--bg-element);
  padding: 2px;
  border-radius: 6px;
}

.lang-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all var(--duration-fast);
}

.lang-btn.active {
  background: var(--bg-surface);
  color: var(--text-primary);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.permission-legend {
  margin-top: 24px;
  padding: 16px;
  background: var(--bg-surface-hover);
  border-radius: var(--radius-md);
  border: 1px solid var(--bg-element);
}

.legend-header {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 12px;
}
</style>
