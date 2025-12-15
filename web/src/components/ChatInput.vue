<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { Agent, SlashCommand } from '../types'
import { useI18n } from '../composables/useI18n'

const emit = defineEmits<{
  send: [message: string]
}>()

const props = defineProps<{
  disabled: boolean
  agents: Agent[]
  commands: SlashCommand[]
  currentAgent: string
}>()

const { t } = useI18n()

const message = ref('')
const showMentions = ref(false)
const showCommands = ref(false)
const mentionQuery = ref('')
const commandQuery = ref('')
const selectedIndex = ref(0)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

// Global Escape key handler for dropdowns
function handleGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showCommands.value) {
      e.preventDefault()
      e.stopImmediatePropagation()
      removeTriggerText('/')
      showCommands.value = false
      textareaRef.value?.focus()
      return
    }
    if (showMentions.value) {
      e.preventDefault()
      e.stopImmediatePropagation()
      removeTriggerText('@')
      showMentions.value = false
      textareaRef.value?.focus()
      return
    }
  }
}

onMounted(() => {
  // Use window level with capture to catch events as early as possible
  window.addEventListener('keydown', handleGlobalKeydown, true)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown, true)
})

const filteredAgents = computed(() => {
  const query = mentionQuery.value.toLowerCase()
  return props.agents.filter(
    (a) => a.id.toLowerCase().includes(query) || a.name.toLowerCase().includes(query)
  )
})

const filteredCommands = computed(() => {
  const query = commandQuery.value.toLowerCase()
  return props.commands.filter(
    (c) => c.name.toLowerCase().includes(query) || c.description.toLowerCase().includes(query)
  )
})

function handleSubmit() {
  const text = message.value.trim()
  if (!text) return
  emit('send', text)
  message.value = ''
  showMentions.value = false
  showCommands.value = false
}

function handleInput(e: Event) {
  const target = e.target as HTMLTextAreaElement
  const value = target.value
  const cursorPos = target.selectionStart
  const textBeforeCursor = value.slice(0, cursorPos)

  // Check if we're typing a slash command (/ at start of line)
  const slashMatch = textBeforeCursor.match(/(?:^|\n)\/([\w\-:]*)$/)
  if (slashMatch) {
    showCommands.value = true
    showMentions.value = false
    commandQuery.value = slashMatch[1] || ''
    selectedIndex.value = 0
    return
  }

  // Check if we're typing after @
  const atMatch = textBeforeCursor.match(/@([\w\-/]*)$/)
  if (atMatch) {
    showMentions.value = true
    showCommands.value = false
    mentionQuery.value = atMatch[1] || ''
    selectedIndex.value = 0
    return
  }

  // Neither
  showMentions.value = false
  showCommands.value = false
  mentionQuery.value = ''
  commandQuery.value = ''
}

function removeTriggerText(trigger: '@' | '/') {
  if (!textareaRef.value) return

  const cursorPos = textareaRef.value.selectionStart
  const textBeforeCursor = message.value.slice(0, cursorPos)
  const textAfterCursor = message.value.slice(cursorPos)

  // Remove trigger and query text
  const pattern = trigger === '/' ? /(?:^|\n)\/[\w\-:]*$/ : /@[\w\-/]*$/
  const newTextBefore = textBeforeCursor.replace(pattern, trigger === '/' ? '' : '')

  message.value = newTextBefore + textAfterCursor

  // Reset queries
  mentionQuery.value = ''
  commandQuery.value = ''

  // Restore cursor position
  const newCursorPos = newTextBefore.length
  setTimeout(() => {
    textareaRef.value?.setSelectionRange(newCursorPos, newCursorPos)
    textareaRef.value?.focus()
  }, 0)
}

function handleKeydown(e: KeyboardEvent) {
  // Handle Escape to close dropdowns (prevent default blur behavior)
  if (e.key === 'Escape') {
    if (showCommands.value) {
      e.preventDefault()
      e.stopPropagation()
      removeTriggerText('/')
      showCommands.value = false
      return
    }
    if (showMentions.value) {
      e.preventDefault()
      e.stopPropagation()
      removeTriggerText('@')
      showMentions.value = false
      return
    }
  }

  // Handle commands dropdown navigation
  if (showCommands.value && filteredCommands.value.length > 0) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedIndex.value = (selectedIndex.value + 1) % filteredCommands.value.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedIndex.value =
        (selectedIndex.value - 1 + filteredCommands.value.length) %
        filteredCommands.value.length
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      const cmd = filteredCommands.value[selectedIndex.value]
      if (cmd) selectCommand(cmd)
      return
    }
  }

  // Handle mentions dropdown navigation
  if (showMentions.value && filteredAgents.value.length > 0) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedIndex.value = (selectedIndex.value + 1) % filteredAgents.value.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedIndex.value =
        (selectedIndex.value - 1 + filteredAgents.value.length) %
        filteredAgents.value.length
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      const agent = filteredAgents.value[selectedIndex.value]
      if (agent) selectAgent(agent)
      return
    }
  }

  // Submit on Enter (without Shift), but only if there's actual content
  if (e.key === 'Enter' && !e.shiftKey) {
    // Don't submit during IME composition (e.g., typing Chinese)
    if (e.isComposing) {
      return
    }
    // Don't submit if any dropdown is open
    if (showMentions.value || showCommands.value) {
      e.preventDefault()
      showMentions.value = false
      showCommands.value = false
      return
    }
    // Only submit if there's non-empty content
    if (message.value.trim()) {
      e.preventDefault()
      handleSubmit()
    }
  }
}

function selectAgent(agent: Agent) {
  if (!textareaRef.value) return

  const cursorPos = textareaRef.value.selectionStart
  const textBeforeCursor = message.value.slice(0, cursorPos)
  const textAfterCursor = message.value.slice(cursorPos)

  // Replace @query with @agentId
  const newTextBefore = textBeforeCursor.replace(/@[\w\-/]*$/, `@${agent.id} `)
  message.value = newTextBefore + textAfterCursor

  showMentions.value = false
  mentionQuery.value = ''

  // Focus and set cursor position
  textareaRef.value.focus()
  const newCursorPos = newTextBefore.length
  setTimeout(() => {
    textareaRef.value?.setSelectionRange(newCursorPos, newCursorPos)
  }, 0)
}

function selectCommand(cmd: SlashCommand) {
  if (!textareaRef.value) return

  const cursorPos = textareaRef.value.selectionStart
  const textBeforeCursor = message.value.slice(0, cursorPos)
  const textAfterCursor = message.value.slice(cursorPos)

  // Replace /query with /commandName
  const newTextBefore = textBeforeCursor.replace(/(?:^|\n)\/[\w\-:]*$/, `/${cmd.name} `)
  message.value = newTextBefore + textAfterCursor

  showCommands.value = false
  commandQuery.value = ''

  // Focus and set cursor position
  textareaRef.value.focus()
  const newCursorPos = newTextBefore.length
  setTimeout(() => {
    textareaRef.value?.setSelectionRange(newCursorPos, newCursorPos)
  }, 0)
}
</script>

<template>
  <form class="input-container" @submit.prevent="handleSubmit">
    <div class="input-wrapper">
      <textarea ref="textareaRef" v-model="message" :placeholder="t('input.placeholder')" rows="1" :disabled="disabled"
        @input="handleInput" @keydown="handleKeydown"></textarea>

      <div class="action-bar">
        <button type="submit" :disabled="disabled || !message.trim()">
          {{ disabled ? 'Sending...' : 'Send' }}
        </button>
      </div>

      <!-- Commands dropdown -->
      <div v-if="showCommands && filteredCommands.length > 0" class="dropdown command-dropdown" @mousedown.prevent>
        <div class="dropdown-header">
          <span class="agent-badge">{{ currentAgent }}</span>
          <span class="header-text">Available Commands</span>
        </div>
        <div v-for="(cmd, idx) in filteredCommands" :key="cmd.name" class="dropdown-item"
          :class="{ selected: idx === selectedIndex }" @click="selectCommand(cmd)" @mouseenter="selectedIndex = idx">
          <span class="cmd-name">/{{ cmd.name }}</span>
          <span class="cmd-desc">{{ cmd.description }}</span>
        </div>
      </div>

      <!-- Mention dropdown -->
      <div v-if="showMentions && filteredAgents.length > 0" class="dropdown mention-dropdown" @mousedown.prevent>
        <div v-for="(agent, idx) in filteredAgents" :key="agent.id" class="dropdown-item"
          :class="{ selected: idx === selectedIndex }" @click="selectAgent(agent)" @mouseenter="selectedIndex = idx">
          <span class="mention-id">@{{ agent.id }}</span>
          <span class="mention-name">{{ agent.name }}</span>
        </div>
      </div>
    </div>
  </form>
</template>

<style scoped>
.input-container {
  display: flex;
  justify-content: center;
  padding: 0 40px 40px;
  position: relative;
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
}

.input-wrapper {
  flex: 1;
  position: relative;
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-lg);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  transition: box-shadow var(--duration-normal) var(--ease-snappy), border-color var(--duration-normal);
  display: flex;
  flex-direction: column;
}

.input-wrapper:focus-within {
  border-color: var(--accent-subtle);
  box-shadow: 0 12px 48px rgba(0, 0, 0, 0.4);
}

textarea {
  width: 100%;
  background: transparent;
  border: none;
  color: var(--text-primary);
  padding: 16px;
  resize: none;
  font-size: 15px;
  font-family: inherit;
  line-height: 1.5;
  outline: none;
  min-height: 56px;
  box-sizing: border-box;
}

textarea::placeholder {
  color: var(--text-tertiary);
}

.action-bar {
  display: flex;
  justify-content: flex-end;
  padding: 8px 12px;
}

button {
  background: var(--text-primary);
  color: var(--bg-root);
  border: none;
  padding: 6px 16px;
  border-radius: var(--radius-pill);
  cursor: pointer;
  font-weight: 600;
  font-size: 13px;
  transition: all var(--duration-fast);
  opacity: 0;
  transform: translateY(4px);
  pointer-events: none;
}

.input-wrapper:focus-within button,
button:not(:disabled) {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

button:hover:not(:disabled) {
  background: #fff;
  box-shadow: 0 0 12px rgba(255, 255, 255, 0.3);
}

button:disabled {
  background: var(--bg-element);
  color: var(--text-tertiary);
  cursor: not-allowed;
  opacity: 1 !important;
  /* Keep visible if disabled (sending) */
}

/* Dropdowns */
.dropdown {
  position: absolute;
  bottom: 100%;
  left: 0;
  right: 0;
  background: var(--bg-element);
  border: 1px solid var(--accent-subtle);
  border-radius: var(--radius-lg);
  margin-bottom: 8px;
  max-height: 300px;
  overflow-y: auto;
  z-index: 100;
  box-shadow: 0 -8px 24px rgba(0, 0, 0, 0.3);
}

.dropdown-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--accent-subtle);
  font-size: 11px;
  background: var(--bg-surface);
  position: sticky;
  top: 0;
}

.agent-badge {
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-weight: 700;
  font-size: 10px;
  text-transform: uppercase;
  background: var(--text-tertiary);
  color: var(--bg-root);
}

.header-text {
  color: var(--text-secondary);
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  cursor: pointer;
  transition: background var(--duration-fast);
  border-left: 2px solid transparent;
}

.dropdown-item:hover {
  background: var(--bg-surface-hover);
}

.dropdown-item.selected {
  background: var(--bg-surface-hover);
  border-left-color: var(--text-primary);
}

.mention-id,
.cmd-name {
  color: var(--text-primary);
  font-weight: 600;
  font-family: var(--font-mono);
  font-size: 13px;
}

.mention-name,
.cmd-desc {
  color: var(--text-secondary);
  font-size: 13px;
}
</style>
