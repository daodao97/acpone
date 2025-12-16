<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import type { Agent, SlashCommand } from '../types'
import { useI18n } from '../composables/useI18n'
import { fetchWorkspaceFiles, uploadFiles, type FileInfo, type UploadedFile } from '../api'

const emit = defineEmits<{
  send: [message: string, files: string[]]
  cancel: []
}>()

const props = defineProps<{
  disabled: boolean
  isSending: boolean
  agents: Agent[]
  commands: SlashCommand[]
  currentAgent: string
  currentWorkspace: string
}>()

const { t } = useI18n()

const message = ref('')
const showMentions = ref(false)
const showCommands = ref(false)
const mentionQuery = ref('')
const commandQuery = ref('')
const selectedIndex = ref(0)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

// File suggestions
const files = ref<FileInfo[]>([])
const isLoadingFiles = ref(false)
let fileSearchTimeout: ReturnType<typeof setTimeout> | null = null

// File upload
const uploadedFiles = ref<UploadedFile[]>([])
const isDragging = ref(false)
const isUploading = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

// Fetch files when mention query changes
watch(mentionQuery, async (query) => {
  if (!showMentions.value || !props.currentWorkspace) {
    files.value = []
    return
  }

  // Debounce file search
  if (fileSearchTimeout) {
    clearTimeout(fileSearchTimeout)
  }

  fileSearchTimeout = setTimeout(async () => {
    isLoadingFiles.value = true
    try {
      files.value = await fetchWorkspaceFiles(props.currentWorkspace, query, 20)
    } catch {
      files.value = []
    } finally {
      isLoadingFiles.value = false
    }
  }, 150)
})

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

// Filtered files (already filtered from API, but limit display)
const filteredFiles = computed(() => files.value.slice(0, 15))

// Combined mention items count for navigation
const totalMentionItems = computed(() => filteredAgents.value.length + filteredFiles.value.length)

const filteredCommands = computed(() => {
  const query = commandQuery.value.toLowerCase()
  return props.commands.filter(
    (c) => c.name.toLowerCase().includes(query) || c.description.toLowerCase().includes(query)
  )
})

function handleSubmit() {
  const text = message.value.trim()
  if (!text && uploadedFiles.value.length === 0) return
  const filePaths = uploadedFiles.value.map(f => f.path)
  emit('send', text, filePaths)
  message.value = ''
  uploadedFiles.value = []
  showMentions.value = false
  showCommands.value = false
}

function handleCancel() {
  emit('cancel')
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
  const atMatch = textBeforeCursor.match(/@([\w\-/\.]*)$/)
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
  const pattern = trigger === '/' ? /(?:^|\n)\/[\w\-:]*$/ : /@[\w\-/\.]*$/
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

  // Handle mentions dropdown navigation (agents + files)
  if (showMentions.value && totalMentionItems.value > 0) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedIndex.value = (selectedIndex.value + 1) % totalMentionItems.value
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedIndex.value =
        (selectedIndex.value - 1 + totalMentionItems.value) %
        totalMentionItems.value
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      const agentCount = filteredAgents.value.length
      if (selectedIndex.value < agentCount) {
        const agent = filteredAgents.value[selectedIndex.value]
        if (agent) selectAgent(agent)
      } else {
        const fileIndex = selectedIndex.value - agentCount
        const file = filteredFiles.value[fileIndex]
        if (file) selectFile(file)
      }
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
  const newTextBefore = textBeforeCursor.replace(/@[\w\-/\.]*$/, `@${agent.id} `)
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

function selectFile(file: FileInfo) {
  if (!textareaRef.value) return

  const cursorPos = textareaRef.value.selectionStart
  const textBeforeCursor = message.value.slice(0, cursorPos)
  const textAfterCursor = message.value.slice(cursorPos)

  // Replace @query with @filePath
  const newTextBefore = textBeforeCursor.replace(/@[\w\-/\.]*$/, `@${file.path} `)
  message.value = newTextBefore + textAfterCursor

  showMentions.value = false
  mentionQuery.value = ''
  files.value = []

  // Focus and set cursor position
  textareaRef.value.focus()
  const newCursorPos = newTextBefore.length
  setTimeout(() => {
    textareaRef.value?.setSelectionRange(newCursorPos, newCursorPos)
  }, 0)
}

// File upload handlers
async function handleFileUpload(fileList: FileList | File[]) {
  if (!props.currentWorkspace || isUploading.value) return

  const filesToUpload = Array.from(fileList)
  if (filesToUpload.length === 0) return

  isUploading.value = true
  try {
    const result = await uploadFiles(filesToUpload, props.currentWorkspace)
    if (result.success && result.files) {
      uploadedFiles.value = [...uploadedFiles.value, ...result.files]
    }
  } catch (err) {
    console.error('Failed to upload files:', err)
  } finally {
    isUploading.value = false
  }
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  isDragging.value = true
}

function handleDragLeave(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false
}

async function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false

  if (e.dataTransfer?.files) {
    await handleFileUpload(e.dataTransfer.files)
  }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) {
    handleFileUpload(input.files)
    input.value = '' // Reset to allow selecting same file again
  }
}

function removeUploadedFile(file: UploadedFile) {
  uploadedFiles.value = uploadedFiles.value.filter(f => f.path !== file.path)
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function triggerFileSelect() {
  fileInputRef.value?.click()
}
</script>

<template>
  <form class="input-container" @submit.prevent="handleSubmit"
    @dragover="handleDragOver" @dragleave="handleDragLeave" @drop="handleDrop">
    <div class="input-wrapper" :class="{ dragging: isDragging }">
      <!-- Hidden file input -->
      <input ref="fileInputRef" type="file" multiple class="hidden-file-input" @change="handleFileSelect" />

      <!-- Uploaded files preview -->
      <div v-if="uploadedFiles.length > 0" class="uploaded-files">
        <div v-for="file in uploadedFiles" :key="file.path" class="uploaded-file">
          <svg class="file-icon-small" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
          </svg>
          <span class="file-name">{{ file.name }}</span>
          <span class="file-size">{{ formatFileSize(file.size) }}</span>
          <button type="button" class="remove-file-btn" @click="removeUploadedFile(file)">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      </div>

      <!-- Drag overlay -->
      <div v-if="isDragging" class="drag-overlay">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
          <polyline points="17 8 12 3 7 8"></polyline>
          <line x1="12" y1="3" x2="12" y2="15"></line>
        </svg>
        <span>{{ t('input.dropFiles') }}</span>
      </div>

      <textarea ref="textareaRef" v-model="message" :placeholder="t('input.placeholder')" rows="1" :disabled="disabled"
        @input="handleInput" @keydown="handleKeydown"></textarea>

      <div class="action-bar">
        <button type="button" class="attach-btn" @click="triggerFileSelect" :disabled="disabled || isUploading">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"></path>
          </svg>
        </button>
        <div class="action-spacer"></div>
        <button v-if="isSending" type="button" class="cancel-btn" @click="handleCancel">
          Cancel
        </button>
        <button v-else type="submit" :disabled="disabled || (!message.trim() && uploadedFiles.length === 0)">
          Send
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

      <!-- Mention dropdown (Agents + Files) -->
      <div v-if="showMentions && totalMentionItems > 0" class="dropdown mention-dropdown" @mousedown.prevent>
        <!-- Agents section -->
        <template v-if="filteredAgents.length > 0">
          <div class="dropdown-section-header">Agents</div>
          <div v-for="(agent, idx) in filteredAgents" :key="'agent-' + agent.id" class="dropdown-item"
            :class="{ selected: idx === selectedIndex }" @click="selectAgent(agent)" @mouseenter="selectedIndex = idx">
            <span class="mention-icon agent-icon">@</span>
            <span class="mention-id">{{ agent.id }}</span>
            <span class="mention-name">{{ agent.name }}</span>
          </div>
        </template>

        <!-- Files section -->
        <template v-if="filteredFiles.length > 0">
          <div class="dropdown-section-header">Files</div>
          <div v-for="(file, idx) in filteredFiles" :key="'file-' + file.path" class="dropdown-item"
            :class="{ selected: idx + filteredAgents.length === selectedIndex }" @click="selectFile(file)" @mouseenter="selectedIndex = idx + filteredAgents.length">
            <span class="mention-icon file-icon">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
              </svg>
            </span>
            <span class="mention-id file-path">{{ file.path }}</span>
          </div>
        </template>

        <!-- Loading indicator -->
        <div v-if="isLoadingFiles" class="dropdown-loading">Loading files...</div>
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
}

.cancel-btn {
  background: var(--bg-element);
  color: var(--text-secondary);
  opacity: 1 !important;
  transform: translateY(0) !important;
  pointer-events: auto !important;
}

.cancel-btn:hover {
  background: #ef4444;
  color: #fff;
  box-shadow: 0 0 12px rgba(239, 68, 68, 0.3);
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

/* Section headers */
.dropdown-section-header {
  padding: 6px 12px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--text-tertiary);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--accent-subtle);
  position: sticky;
  top: 0;
}

/* Mention icons */
.mention-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.agent-icon {
  font-weight: 700;
  font-size: 14px;
  color: var(--text-primary);
}

.file-icon {
  color: var(--text-secondary);
}

.file-path {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-loading {
  padding: 10px 12px;
  color: var(--text-tertiary);
  font-size: 12px;
  text-align: center;
}

/* File upload styles */
.hidden-file-input {
  display: none;
}

.input-wrapper.dragging {
  border-color: var(--accent-subtle);
  border-style: dashed;
  background: rgba(var(--accent-rgb, 100, 100, 255), 0.05);
}

.drag-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(var(--bg-surface-rgb, 30, 30, 30), 0.95);
  border-radius: var(--radius-lg);
  color: var(--text-primary);
  font-size: 14px;
  z-index: 10;
}

.uploaded-files {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 16px 0;
}

.uploaded-file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: var(--bg-element);
  border-radius: var(--radius-sm);
  font-size: 12px;
}

.file-icon-small {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.file-name {
  color: var(--text-primary);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: var(--text-tertiary);
  font-size: 11px;
}

.remove-file-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  opacity: 1 !important;
  transform: none !important;
  pointer-events: auto !important;
}

.remove-file-btn:hover {
  color: #ef4444;
  background: transparent;
  box-shadow: none;
}

.attach-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: transparent;
  color: var(--text-secondary);
  opacity: 1 !important;
  transform: none !important;
  pointer-events: auto !important;
}

.attach-btn:hover:not(:disabled) {
  color: var(--text-primary);
  background: var(--bg-element);
  box-shadow: none;
}

.action-bar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
}

.action-spacer {
  flex: 1;
}
</style>
