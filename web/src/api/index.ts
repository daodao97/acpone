import type { Agent, Session, SessionMeta, Workspace } from '../types'

const API_BASE = '/api'

export async function fetchAgents(): Promise<{ agents: Agent[]; default: string }> {
  const res = await fetch(`${API_BASE}/agents`)
  const data = await res.json()
  const agents = (data.agents || []).map(
    (a: { id: string; name: string; permissionMode?: string; command?: string; args?: string[]; commands?: unknown[]; env?: Record<string, string> }) => ({
      id: a.id,
      name: a.name,
      permissionMode: a.permissionMode || 'default',
      command: a.command,
      args: a.args,
      commands: a.commands,
      env: a.env || {},
    })
  )
  return { agents, default: data.default }
}

export async function fetchWorkspaces(): Promise<{ workspaces: Workspace[]; default: string }> {
  const res = await fetch(`${API_BASE}/workspaces`)
  const data = await res.json()
  return { workspaces: data.workspaces || [], default: data.default }
}

export interface FileInfo {
  path: string
  name: string
  isDir: boolean
}

export async function fetchWorkspaceFiles(
  workspaceId: string,
  query: string = '',
  limit: number = 50
): Promise<FileInfo[]> {
  const params = new URLSearchParams({ workspaceId, q: query, limit: String(limit) })
  const res = await fetch(`${API_BASE}/workspaces/files?${params}`)
  const data = await res.json()
  return data.files || []
}

export async function createWorkspace(
  name: string,
  path: string
): Promise<{ workspace: Workspace; error?: string }> {
  const res = await fetch(`${API_BASE}/workspaces`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, path }),
  })
  const data = await res.json()
  if (!res.ok) {
    return { workspace: null as unknown as Workspace, error: data.error || 'Failed to create workspace' }
  }
  return { workspace: data.workspace }
}

export async function fetchSessions(): Promise<SessionMeta[]> {
  const res = await fetch(`${API_BASE}/sessions`)
  const data = await res.json()
  return data.sessions || []
}

export async function fetchSession(id: string): Promise<Session | null> {
  const res = await fetch(`${API_BASE}/sessions/${id}`)
  if (!res.ok) return null
  const data = await res.json()
  return data.session || null
}

export async function createSession(workspaceId?: string): Promise<SessionMeta> {
  const res = await fetch(`${API_BASE}/sessions/new`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workspaceId }),
  })
  const data = await res.json()
  return data.session
}

export async function deleteSession(id: string): Promise<void> {
  await fetch(`${API_BASE}/sessions/${id}`, { method: 'DELETE' })
}

export async function confirmPermission(
  agentId: string,
  toolCallId: string,
  optionId: string
): Promise<void> {
  await fetch(`${API_BASE}/permission/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentId, toolCallId, optionId }),
  })
}

export async function updateAgentPermission(
  agentId: string,
  permissionMode: string
): Promise<{ success: boolean; error?: string }> {
  const res = await fetch(`${API_BASE}/agents/update`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentId, permissionMode }),
  })
  const data = await res.json()
  if (!res.ok) {
    return { success: false, error: data.error || 'Failed to update agent' }
  }
  return { success: true }
}

export async function updateAgentEnv(
  agentId: string,
  env: Record<string, string>
): Promise<{ success: boolean; error?: string }> {
  const res = await fetch(`${API_BASE}/agents/update`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentId, env, updateEnv: true }),
  })
  const data = await res.json()
  if (!res.ok) {
    return { success: false, error: data.error || 'Failed to update env' }
  }
  return { success: true }
}

export async function cancelChat(
  agentId: string,
  sessionId: string
): Promise<{ success: boolean; error?: string }> {
  const res = await fetch(`${API_BASE}/chat/cancel`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentId, sessionId }),
  })
  const data = await res.json()
  if (!res.ok) {
    return { success: false, error: data.error || 'Failed to cancel' }
  }
  return { success: true }
}

export interface UploadedFile {
  name: string
  path: string
  size: number
}

export async function uploadFiles(
  files: File[],
  workspaceId: string
): Promise<{ success: boolean; files?: UploadedFile[]; error?: string }> {
  const formData = new FormData()
  formData.append('workspaceId', workspaceId)
  files.forEach((file) => formData.append('files', file))

  const res = await fetch(`${API_BASE}/upload`, {
    method: 'POST',
    body: formData,
  })
  const data = await res.json()
  if (!res.ok) {
    return { success: false, error: data.error || 'Failed to upload' }
  }
  return { success: true, files: data.files }
}

export async function cleanupFiles(
  workspaceId: string
): Promise<{ success: boolean; error?: string }> {
  const res = await fetch(`${API_BASE}/upload/cleanup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ workspaceId }),
  })
  const data = await res.json()
  if (!res.ok) {
    return { success: false, error: data.error || 'Failed to cleanup' }
  }
  return { success: true }
}

export function sendMessage(
  message: string,
  conversationId: string | null,
  workspaceId: string | null,
  files: string[],
  onEvent: (event: unknown) => void
): AbortController {
  const controller = new AbortController()

  fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message, conversationId, workspaceId, files }),
    signal: controller.signal,
  })
    .then(async (response) => {
      const reader = response.body?.getReader()
      if (!reader) return

      const decoder = new TextDecoder()
      let buffer = ''
      let currentEventType = 'message'

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEventType = line.slice(7).trim()
          } else if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))
              if (currentEventType === 'error') {
                onEvent({ error: data.message || 'Unknown error', ...data })
              } else if (currentEventType === 'commands') {
                // Commands event: data is { agent, commands }
                onEvent({ _eventType: 'commands', ...data })
              } else {
                // Pass event type to callback
                onEvent({ _eventType: currentEventType, ...data })
              }
            } catch {
              // ignore parse errors
            }
            currentEventType = 'message'
          }
        }
      }
    })
    .catch((err) => {
      if (err.name !== 'AbortError') {
        onEvent({ error: err.message })
      }
    })

  return controller
}
