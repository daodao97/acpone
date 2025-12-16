import { ref, computed } from 'vue'
import type {
  Agent,
  Session,
  SessionMeta,
  ToolCall,
  StreamItem,
  Workspace,
  SlashCommand,
  MessageFile,
} from '../types'
import * as api from '../api'

// State
const sessions = ref<SessionMeta[]>([])
const agents = ref<Agent[]>([])
const workspaces = ref<Workspace[]>([])
const commandsByAgent = ref<Record<string, SlashCommand[]>>({})
const defaultAgent = ref<string>('claude')
const defaultWorkspace = ref<string>('')
const currentSession = ref<Session | null>(null)
const currentAgent = ref<string>('claude')
const currentWorkspace = ref<string>('')
const isLoading = ref(false)
const isSending = ref(false)
const agentSessionId = ref<string | null>(null)
const sendingSessionId = ref<string | null>(null) // Track which session is currently streaming

// Stream items stored per session (tool calls interleaved with messages)
const streamItemsBySession = ref<Record<string, StreamItem[]>>({})

// Session cache to preserve messages during streaming (prevents loss when switching sessions)
const sessionCache = ref<Record<string, Session>>({})

// Computed: current session's stream items
const streamItems = computed(() => {
  const sessionId = currentSession.value?.id
  if (!sessionId) return []
  return streamItemsBySession.value[sessionId] || []
})

// Computed
const currentSessionId = computed(() => currentSession.value?.id || null)
const messages = computed(() => currentSession.value?.messages || [])
const currentWorkspaceInfo = computed(() =>
  workspaces.value.find((w) => w.id === currentWorkspace.value)
)
const currentAgentInfo = computed(() =>
  agents.value.find((a) => a.id === currentAgent.value)
)
// Filter sessions by current workspace
const filteredSessions = computed(() => {
  if (!currentWorkspace.value) return sessions.value
  return sessions.value.filter(
    (s) => s.workspaceId === currentWorkspace.value || !s.workspaceId
  )
})

// Actions
async function loadAgents() {
  const data = await api.fetchAgents()
  agents.value = data.agents
  defaultAgent.value = data.default
  currentAgent.value = data.default

  // Initialize commands from agents (cached on server)
  for (const agent of data.agents) {
    if (agent.commands && agent.commands.length > 0) {
      commandsByAgent.value[agent.id] = agent.commands
    }
  }
}

async function loadWorkspaces() {
  const data = await api.fetchWorkspaces()
  workspaces.value = data.workspaces
  defaultWorkspace.value = data.default || ''

  // Auto-select workspace
  if (!currentWorkspace.value) {
    if (data.default) {
      currentWorkspace.value = data.default
    } else if (data.workspaces.length > 0 && data.workspaces[0]) {
      // Select first workspace if no default
      currentWorkspace.value = data.workspaces[0].id
    }
  }
}

async function addWorkspace(name: string, path: string): Promise<string | null> {
  const result = await api.createWorkspace(name, path)
  if (result.error) {
    return result.error
  }
  workspaces.value.push(result.workspace)
  currentWorkspace.value = result.workspace.id
  return null
}

async function loadSessions(skipAutoSelect = false) {
  isLoading.value = true
  try {
    sessions.value = await api.fetchSessions()
    if (!skipAutoSelect) {
      const first = sessions.value[0]
      if (first && !currentSession.value) {
        await selectSession(first.id)
      }
    }
  } finally {
    isLoading.value = false
  }
}

async function selectSession(id: string) {
  isLoading.value = true
  try {
    // Save current session to cache before switching (preserves uncommitted messages)
    if (currentSession.value) {
      sessionCache.value[currentSession.value.id] = currentSession.value
    }

    // Check if target session is in cache and has streaming content or pending items
    const streamItems = streamItemsBySession.value[id]
    const hasStreamingContent = streamItems !== undefined && streamItems.length > 0
    const cachedSession = sessionCache.value[id]

    let session: Session | null = null
    if (cachedSession && hasStreamingContent) {
      // Use cached version to preserve streaming state
      session = cachedSession
    } else {
      // Fetch from server
      session = await api.fetchSession(id)
      // Merge with cache if exists (preserve any messages added during streaming)
      if (cachedSession && session) {
        // If cached has more messages, use cached messages
        if (cachedSession.messages.length > session.messages.length) {
          session.messages = cachedSession.messages
        }
      }
    }

    if (session) {
      currentSession.value = session
      currentAgent.value = session.activeAgent
      // Auto-select workspace based on session's workspace
      if (session.workspaceId && session.workspaceId !== currentWorkspace.value) {
        currentWorkspace.value = session.workspaceId
      }
    }
  } finally {
    isLoading.value = false
  }
}

async function createNewSession() {
  isLoading.value = true
  try {
    const meta = await api.createSession(currentWorkspace.value || undefined)
    currentSession.value = {
      id: meta.id,
      title: meta.title,
      messages: [],
      activeAgent: meta.activeAgent,
      workspaceId: meta.workspaceId,
      createdAt: meta.createdAt,
      updatedAt: meta.updatedAt,
    }
    currentAgent.value = meta.activeAgent
    // Initialize empty stream items for new session
    streamItemsBySession.value[meta.id] = []
    await loadSessions()
  } finally {
    isLoading.value = false
  }
}

async function removeSession(id: string) {
  await api.deleteSession(id)
  if (currentSession.value?.id === id) {
    currentSession.value = null
  }
  // Clean up all data for deleted session
  delete streamItemsBySession.value[id]
  delete pendingStreamAgentBySession.value[id]
  delete sessionCache.value[id]
  await loadSessions()
}

function addUserMessage(content: string, files?: MessageFile[]) {
  if (!currentSession.value) return
  currentSession.value.messages.push({
    role: 'user',
    content,
    files: files && files.length > 0 ? files : undefined,
  })
}

function addAssistantMessage(content: string, agent: string) {
  if (!currentSession.value) return
  currentSession.value.messages.push({ role: 'assistant', content, agent })
}

function addErrorMessage(content: string) {
  if (!currentSession.value) return
  currentSession.value.messages.push({ role: 'assistant', content, isError: true })
}

function addToolCall(tool: ToolCall, sessionId?: string) {
  // Use provided sessionId or sendingSessionId or currentSession
  const targetId = sessionId || sendingSessionId.value || currentSession.value?.id
  if (!targetId) return

  // Ensure array exists for this session
  if (!streamItemsBySession.value[targetId]) {
    streamItemsBySession.value[targetId] = []
  }
  const items = streamItemsBySession.value[targetId]

  // Add to stream items in order
  const existing = items.find(
    (item) => item.type === 'tool' && item.data.toolCallId === tool.toolCallId
  )
  if (existing && existing.type === 'tool') {
    // Merge: preserve existing fields if new ones are empty
    // Don't overwrite description with output-like content (when status is completed)
    const shouldKeepDescription = tool.status === 'completed' && existing.data.description
    const merged: ToolCall = {
      ...existing.data,
      ...tool,
      title: (tool.title && !tool.title.startsWith('toolu_')) ? tool.title : existing.data.title,
      description: shouldKeepDescription ? existing.data.description : (tool.description || existing.data.description),
      input: tool.input || existing.data.input,
      rawInput: tool.rawInput || existing.data.rawInput,
      output: tool.output || existing.data.output,
      error: tool.error || existing.data.error,
    }
    existing.data = merged
  } else {
    // Add new tool call
    items.push({ type: 'tool', data: tool })
  }
}

function addStreamingText(text: string, sessionId?: string) {
  // Use provided sessionId or sendingSessionId or currentSession
  const targetId = sessionId || sendingSessionId.value || currentSession.value?.id
  if (!targetId) return

  // Ensure array exists for this session
  if (!streamItemsBySession.value[targetId]) {
    streamItemsBySession.value[targetId] = []
  }
  const items = streamItemsBySession.value[targetId]

  // Find last text item or create new one
  const lastItem = items[items.length - 1]
  if (lastItem && lastItem.type === 'text') {
    lastItem.data += text
  } else {
    items.push({ type: 'text', data: text })
  }
}

function clearStreamItems(sessionId?: string) {
  const targetId = sessionId || currentSession.value?.id
  if (targetId) {
    streamItemsBySession.value[targetId] = []
  }
}

// Track agent for pending stream items (per session)
const pendingStreamAgentBySession = ref<Record<string, string>>({})

function finalizeStreamItems(agent: string, sessionId?: string) {
  // Don't move items immediately - just mark the agent
  // Items will be moved when user sends next message (in commitStreamItems)
  const targetId = sessionId || sendingSessionId.value || currentSession.value?.id
  if (targetId) {
    pendingStreamAgentBySession.value[targetId] = agent
  }
}

function commitStreamItems(sessionId?: string) {
  const targetId = sessionId || currentSession.value?.id
  if (!targetId) return

  const items = streamItemsBySession.value[targetId]
  if (!items || items.length === 0) return

  // Find the session to commit to (check current session first, then cache)
  let session: Session | null = null
  if (targetId === currentSession.value?.id) {
    session = currentSession.value
  } else if (sessionCache.value[targetId]) {
    session = sessionCache.value[targetId]
  }

  if (!session) return

  const agent = pendingStreamAgentBySession.value[targetId] || 'claude'

  // Move stream items to messages
  for (const item of items) {
    if (item.type === 'text') {
      session.messages.push({
        role: 'assistant',
        content: item.data,
        agent,
      })
    } else if (item.type === 'tool') {
      session.messages.push({
        role: 'assistant',
        content: '',
        agent,
        toolCall: item.data,
      })
    }
  }

  // Update cache
  sessionCache.value[targetId] = session

  streamItemsBySession.value[targetId] = []
  delete pendingStreamAgentBySession.value[targetId]
}

function setConversationId(id: string) {
  if (currentSession.value) {
    currentSession.value.id = id
  }
}

function setAgent(agent: string) {
  currentAgent.value = agent
}

function setWorkspace(workspaceId: string) {
  currentWorkspace.value = workspaceId
  // Clear current session to show new conversation page
  currentSession.value = null
  // No need to clear streamItems - they are stored per session
}

function setSending(value: boolean) {
  isSending.value = value
}

function setSendingSessionId(id: string | null) {
  sendingSessionId.value = id
}

function setAgentSessionId(id: string | null) {
  agentSessionId.value = id
}

async function cancelCurrentChat() {
  if (!agentSessionId.value || !currentAgent.value) return false
  const result = await api.cancelChat(currentAgent.value, agentSessionId.value)
  if (result.success) {
    isSending.value = false
  }
  return result.success
}

function setCommands(agentId: string, newCommands: SlashCommand[]) {
  commandsByAgent.value[agentId] = newCommands
}

// Computed: current agent's commands
const commands = computed(() => commandsByAgent.value[currentAgent.value] || [])

export function useSessionStore() {
  return {
    // State
    sessions,
    agents,
    workspaces,
    commands,
    defaultAgent,
    defaultWorkspace,
    currentSession,
    currentAgent,
    currentWorkspace,
    isLoading,
    isSending,
    streamItems,
    // Computed
    currentSessionId,
    messages,
    currentWorkspaceInfo,
    currentAgentInfo,
    filteredSessions,
    // Actions
    loadAgents,
    loadWorkspaces,
    addWorkspace,
    loadSessions,
    selectSession,
    createNewSession,
    removeSession,
    addUserMessage,
    addAssistantMessage,
    addErrorMessage,
    addToolCall,
    addStreamingText,
    clearStreamItems,
    finalizeStreamItems,
    commitStreamItems,
    setConversationId,
    setAgent,
    setWorkspace,
    setSending,
    setCommands,
    agentSessionId,
    setAgentSessionId,
    sendingSessionId,
    setSendingSessionId,
    cancelCurrentChat,
  }
}
