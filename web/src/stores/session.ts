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

// Stream items for current response (tool calls interleaved with messages)
const streamItems = ref<StreamItem[]>([])

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
    // Commit any pending stream items before switching
    commitStreamItems()
    const session = await api.fetchSession(id)
    if (session) {
      currentSession.value = session
      currentAgent.value = session.activeAgent
      streamItems.value = []
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
    streamItems.value = []
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

function addToolCall(tool: ToolCall) {
  // Add to stream items in order
  const existing = streamItems.value.find(
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
    streamItems.value.push({ type: 'tool', data: tool })
  }
}

function addStreamingText(text: string) {
  // Find last text item or create new one
  const lastItem = streamItems.value[streamItems.value.length - 1]
  if (lastItem && lastItem.type === 'text') {
    lastItem.data += text
  } else {
    streamItems.value.push({ type: 'text', data: text })
  }
}

function clearStreamItems() {
  streamItems.value = []
}

// Track agent for pending stream items
const pendingStreamAgent = ref<string>('')

function finalizeStreamItems(agent: string) {
  // Don't move items immediately - just mark the agent
  // Items will be moved when user sends next message (in commitStreamItems)
  pendingStreamAgent.value = agent
}

function commitStreamItems() {
  if (!currentSession.value || streamItems.value.length === 0) return

  const agent = pendingStreamAgent.value || 'claude'

  // Move stream items to messages
  for (const item of streamItems.value) {
    if (item.type === 'text') {
      currentSession.value.messages.push({
        role: 'assistant',
        content: item.data,
        agent,
      })
    } else if (item.type === 'tool') {
      currentSession.value.messages.push({
        role: 'assistant',
        content: '',
        agent,
        toolCall: item.data,
      })
    }
  }

  streamItems.value = []
  pendingStreamAgent.value = ''
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
  streamItems.value = []
}

function setSending(value: boolean) {
  isSending.value = value
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
    cancelCurrentChat,
  }
}
