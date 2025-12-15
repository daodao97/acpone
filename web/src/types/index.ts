export interface Agent {
  id: string
  name: string
  permissionMode?: 'default' | 'bypass' | string
  command?: string
  args?: string[]
  commands?: SlashCommand[]
  env?: Record<string, string>
}

export interface Workspace {
  id: string
  name: string
  path: string
}

export interface SessionMeta {
  id: string
  title: string
  activeAgent: string
  workspaceId?: string
  messageCount: number
  createdAt: number
  updatedAt: number
}

export interface Message {
  role: 'user' | 'assistant'
  content: string
  agent?: string
  toolCall?: ToolCall
  timestamp?: number
  isError?: boolean
}

export interface Session {
  id: string
  title: string
  messages: Message[]
  activeAgent: string
  workspaceId?: string
  createdAt: number
  updatedAt: number
}

export interface ToolCall {
  toolCallId: string
  toolName: string
  kind?: string
  title: string
  description?: string
  status: 'pending' | 'completed' | 'error'
  input?: string
  rawInput?: string
  output?: string
  error?: string
}

// Unified stream item for rendering in order
export type StreamItem =
  | { type: 'text'; data: string }
  | { type: 'tool'; data: ToolCall }

export interface StreamEvent {
  conversationId?: string
  agent?: string
  sessionId?: string
  isNew?: boolean
  message?: string
  update?: SessionUpdate
  sessionUpdate?: string
  stopReason?: string
  error?: string
}

export interface SessionUpdate {
  sessionUpdate: string
  content?: { type: string; text?: string }
  toolCallId?: string
  title?: string
  status?: string
  kind?: string
  rawInput?: Record<string, unknown>
  error?: string
  _meta?: {
    claudeCode?: {
      toolName?: string
      toolResponse?: {
        stdout?: string
        stderr?: string
        type?: string
        file?: { filePath: string; content: string }
      }
      error?: string
    }
  }
}

export interface PermissionRequest {
  sessionId: string
  options: Array<{
    optionId: string
    name: string
    kind: 'allow_once' | 'allow_always' | 'reject_once' | 'reject_always'
  }>
  toolCall: {
    toolCallId: string
    rawInput?: Record<string, unknown>
    status?: string
    title?: string
    kind?: string
  }
}

export interface SlashCommand {
  name: string
  description: string
  input?: {
    hint?: string
  } | null
}
