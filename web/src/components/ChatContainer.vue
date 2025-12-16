<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'
import MarkdownRender from 'markstream-vue'
import { useSessionStore } from '../stores/session'
import { sendMessage } from '../api'
import type { StreamEvent, SessionUpdate, PermissionRequest, SlashCommand, MessageFile } from '../types'
import ChatMessage from './ChatMessage.vue'
import ToolCallItem from './ToolCallItem.vue'
import ChatInput from './ChatInput.vue'
import PermissionRequestVue from './PermissionRequest.vue'
import { useI18n } from '../composables/useI18n'

const store = useSessionStore()
const {
  currentSession,
  messages,
  currentAgent,
  currentWorkspace,
  isSending,
  streamItems,
  agents,
  commands,
} = store
const { t } = useI18n()

const chatContainer = ref<HTMLElement | null>(null)
const pendingPermission = ref<PermissionRequest | null>(null)

function scrollToBottom() {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

watch(messages, () => scrollToBottom(), { deep: true })
watch(streamItems, () => scrollToBottom(), { deep: true })
watch(pendingPermission, () => scrollToBottom())

// Check if current session is streaming
const isCurrentSessionStreaming = computed(() => {
  return store.sendingSessionId.value === currentSession.value?.id
})

async function handleSend(message: string, files: MessageFile[] = []) {
  store.setSending(true)
  store.commitStreamItems() // Move previous stream items to messages
  store.clearStreamItems()
  pendingPermission.value = null

  // Create session if none exists
  if (!currentSession.value) {
    await store.createNewSession()
  }

  // Track which session we're sending to
  const targetSessionId = store.currentSessionId.value
  store.setSendingSessionId(targetSessionId)

  // Store message with file info
  store.addUserMessage(message, files.length > 0 ? files : undefined)

  sendMessage(
    message,
    targetSessionId,
    currentWorkspace.value || null,
    files,
    (event: unknown) => {
      const data = event as StreamEvent & { permission_request?: PermissionRequest }
      // Always process events to update the target session's stream items
      // even if user switched to another session
      handleStreamEvent(data, targetSessionId)
    }
  )
}

interface ToolCallEvent {
  toolCallId: string
  toolName: string
  kind: string
  title: string
  description: string
  status: string
  input: string
  rawInput: string
  output: string
  error: string
}

function handleStreamEvent(
  data: StreamEvent & {
    permission_request?: PermissionRequest
    _eventType?: string
    commands?: SlashCommand[]
    agent?: string
  } & Partial<ToolCallEvent>,
  targetSessionId?: string | null
) {
  // Handle commands event (with agent info)
  if (data._eventType === 'commands' && data.commands) {
    const agentId = data.agent || currentAgent.value
    store.setCommands(agentId, data.commands)
    return
  }

  // Handle tool_call event from backend (direct format)
  if (data._eventType === 'tool_call' && data.toolCallId) {
    store.addToolCall({
      toolCallId: data.toolCallId,
      toolName: data.toolName || 'Tool',
      kind: data.kind || '',
      title: data.title || data.toolCallId,
      description: data.description || '',
      status: (data.status as 'pending' | 'completed' | 'error') || 'pending',
      input: data.input || '',
      rawInput: data.rawInput || '',
      output: data.output || '',
      error: data.error || '',
    }, targetSessionId || undefined)
    return
  }

  // Permission request - only show if on same session
  if ((data as unknown as { sessionId?: string; options?: unknown[] }).options) {
    if (store.currentSessionId.value === targetSessionId) {
      pendingPermission.value = data as unknown as PermissionRequest
    }
    return
  }

  if (data.conversationId) {
    store.setConversationId(data.conversationId)
    if (data.agent) {
      store.setAgent(data.agent)
    }
    // Capture agent sessionId for cancel functionality
    if (data.sessionId) {
      store.setAgentSessionId(data.sessionId)
    }
  }

  // Status message only (but not error messages)
  if (data.message && !data.update && !data.error) {
    return
  }

  // Session update
  const update = data.update || (data as unknown as SessionUpdate)

  if (update?.sessionUpdate) {
    switch (update.sessionUpdate) {
      case 'agent_message_chunk':
      case 'agent_thought_chunk':
        if (update.content?.type === 'text' && update.content.text) {
          store.addStreamingText(update.content.text, targetSessionId || undefined)
        }
        break

      case 'tool_call':
      case 'tool_call_update':
        handleToolUpdate(update, targetSessionId)
        break
    }
  }

  // Done
  if (data.stopReason) {
    finishStreaming(targetSessionId)
  }

  // Error
  if (data.error) {
    finishStreaming(targetSessionId)
    store.addErrorMessage(data.error)
  }
}

function handleToolUpdate(update: SessionUpdate, targetSessionId?: string | null) {
  const toolId = update.toolCallId
  if (!toolId) return

  const toolName = update._meta?.claudeCode?.toolName || update.kind || 'Tool'
  const title = update.title || toolId

  let status: 'pending' | 'completed' | 'error' = 'pending'
  const hasError = update.error || update._meta?.claudeCode?.error
  if (hasError) {
    status = 'error'
  } else if (update.status === 'completed') {
    status = 'completed'
  }

  let input = ''
  if (update.rawInput) {
    const ri = update.rawInput
    if (ri.command) input = String(ri.command)
    else if (ri.file_path) input = String(ri.file_path)
    else if (ri.pattern) input = String(ri.pattern)
    else if (ri.old_string) input = `old_string: ${String(ri.old_string).slice(0, 100)}...`
    else input = JSON.stringify(ri, null, 2)
  }

  let output = ''
  let error = ''

  if (update._meta?.claudeCode?.toolResponse) {
    const resp = update._meta.claudeCode.toolResponse
    if (resp.stdout) {
      output = resp.stdout
    } else if (resp.stderr) {
      error = resp.stderr
      status = 'error'
    } else if (resp.type === 'text' && resp.file?.content) {
      const content = resp.file.content
      output = `File: ${resp.file.filePath}\n${content.slice(0, 500)}${content.length > 500 ? '...' : ''}`
    }
  }

  if (update.error) {
    error = update.error
  } else if (update._meta?.claudeCode?.error) {
    error = update._meta.claudeCode.error
  }

  store.addToolCall({
    toolCallId: toolId,
    toolName,
    title,
    status,
    input,
    output,
    error,
  }, targetSessionId || undefined)
}

function finishStreaming(targetSessionId?: string | null) {
  // Finalize and commit stream items to the target session
  // This ensures content is saved even if user switched to another session
  store.finalizeStreamItems(currentAgent.value, targetSessionId || undefined)
  store.commitStreamItems(targetSessionId || undefined)
  store.setSending(false)
  store.setSendingSessionId(null)
  // Only clear permission if on same session
  if (store.currentSessionId.value === targetSessionId) {
    pendingPermission.value = null
  }
  store.loadSessions()
}

function handlePermissionConfirmed() {
  pendingPermission.value = null
}

async function handleCancel() {
  await store.cancelCurrentChat()
  finishStreaming()
}

const visibleMessages = computed(() => {
  // Fix: messages is a Ref, so we must access .value
  const msgs = messages.value || []
  const result: Array<{ msg: typeof msgs[0], hideTag: boolean }> = []
  let lastAgent: string | null = null
  let lastRole: string | null = null

  for (const msg of msgs) {
    // Filter logic: content exists OR toolCall exists OR is error OR not assistant
    const hasContent = !!msg.content
    const isVisible = hasContent || !!msg.toolCall || !!msg.isError || msg.role !== 'assistant'
    
    if (!isVisible) continue

    let hideTag = false
    if (msg.role === 'assistant') {
      // Hide tag if same agent as previous visible message
      if (lastRole === 'assistant' && msg.agent === lastAgent) {
        hideTag = true
      }
      lastAgent = msg.agent || null
      lastRole = 'assistant'
    } else {
      lastAgent = null
      lastRole = msg.role
    }
    
    result.push({ msg, hideTag })
  }
  return result
})
</script>

<template>
  <div class="chat-wrapper">
    <div ref="chatContainer" class="chat-container">
      <!-- Welcome message -->
      <div
        v-if="!currentSession || messages.length === 0"
        class="welcome-message"
      >
        <template v-if="!currentWorkspace">
          <p>{{ t('welcome.select_workspace') }}</p>
        </template>
        <template v-else>
          <p>{{ t('welcome.start') }}</p>
          <p>
            {{ t('welcome.mention') }}:
            <code v-for="a in agents" :key="a.id">@{{ a.id }}</code>
          </p>
        </template>
      </div>

      <!-- Messages -->
      <template v-for="(item, idx) in visibleMessages" :key="idx">
        <ChatMessage 
          :message="item.msg"
          :hide-agent-tag="item.hideTag"
        />
      </template>

      <!-- Streaming items (text and tool calls in order) -->
      <template v-for="(item, idx) in streamItems" :key="`stream-${idx}`">
        <ToolCallItem v-if="item.type === 'tool'" :tool="item.data" />
        <div v-else-if="item.type === 'text'" class="message assistant">
          <span class="agent-tag" :class="currentAgent">
            {{ currentAgent }}
          </span>
          <div class="content">
            <MarkdownRender :content="item.data" />
          </div>
        </div>
      </template>

      <!-- Permission request -->
      <PermissionRequestVue
        v-if="pendingPermission"
        :request="pendingPermission"
        :agent-id="currentAgent"
        @confirmed="handlePermissionConfirmed"
      />

      <!-- Loading indicator -->
      <div v-if="isCurrentSessionStreaming && !pendingPermission" class="loading-indicator">
        <div class="loading-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>
    </div>

    <ChatInput :disabled="isSending || !currentWorkspace" :is-sending="isSending" :agents="agents" :commands="commands" :current-agent="currentAgent" :current-workspace="currentWorkspace" @send="handleSend" @cancel="handleCancel" />
  </div>
</template>

<style scoped>
.chat-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

.chat-container {
  flex: 1;
  overflow-y: auto;
  padding: 40px 20px 20px;
  display: flex;
  flex-direction: column;
  gap: 0; /* Manual spacing via margins */
  width: 100%;
  max-width: 800px;
  margin: 0 auto;
  /* Visual scrollbar hiding but functional */
  scrollbar-width: none;
}

.chat-container::-webkit-scrollbar {
  display: none;
}

.welcome-message {
  text-align: center;
  color: var(--text-tertiary);
  padding: 80px 20px;
  font-size: 14px;
}

.welcome-message p {
  margin-bottom: 12px;
}

.welcome-message code {
  background: var(--bg-surface);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 12px;
}

/* Streaming Assistant Message - "The Void" Style */
.message.assistant {
  background: transparent;
  padding: 0;
  max-width: 100%;
  align-self: stretch;
  margin-bottom: 4px; /* Tight log spacing */
}

.agent-tag {
  display: inline-block;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 0;
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--text-tertiary); /* Minimalist tag */
}

/* Loading */
.loading-indicator {
  display: flex;
  justify-content: flex-start; /* Align left */
  padding: 16px 0;
}

.loading-dots {
  display: flex;
  gap: 4px;
  align-items: center;
}

.loading-dots span {
  width: 4px;
  height: 4px;
  background: var(--text-tertiary);
  border-radius: 50%;
  animation: loading 1.2s infinite ease-in-out;
}

.loading-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.loading-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes loading {
  0%,
  80%,
  100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}

/* Markdown Styles override for this context */
.content {
  font-size: 15px;
  line-height: 1.7;
  color: var(--text-primary);
}

.content :deep(pre) {
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-md);
  padding: 16px;
  overflow-x: auto;
  margin: 16px 0;
}

.content :deep(code) {
  font-family: var(--font-mono);
  font-size: 13px;
  background: var(--bg-surface);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.content :deep(p) {
  margin-bottom: 16px;
}

.content :deep(ul),
.content :deep(ol) {
  margin-bottom: 16px;
  padding-left: 1.2em;
}

.content :deep(li) {
  margin-bottom: 4px;
}

.content :deep(blockquote) {
  border-left: 2px solid var(--accent-subtle);
  margin: 16px 0;
  padding-left: 16px;
  color: var(--text-secondary);
  font-style: italic;
}

.content :deep(a) {
  color: var(--text-primary);
  text-decoration: underline;
  text-decoration-color: var(--text-tertiary);
}

.content :deep(a:hover) {
  text-decoration-color: var(--text-primary);
}
</style>
