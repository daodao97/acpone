# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ACPone is an ACP (Anthropic Client Protocol) Gateway Chat interface with a Go backend and Vue 3 + TypeScript frontend. It provides a web-based chat interface for communicating with AI agents (like Claude Code, Codex) through JSON-RPC, with support for multiple workspaces, sessions, and agent routing.

The project also includes a desktop tray application for macOS/Linux/Windows.

## Build Commands

### Frontend (web/)
```bash
cd web
npm install              # Install dependencies
npm run dev              # Dev server on :5173 (proxies API to :3000)
npm run build            # Build to web/dist
```

### Backend (backend/)
```bash
cd backend
go build -o acpone ./cmd/acpone              # Build web server binary
go run ./cmd/acpone                          # Run with embedded web
go run ./cmd/acpone -web ../web/dist         # Run with external web dir
go run ./cmd/acpone -port 8080               # Custom port (default: 3000)
```

### Desktop App (backend/)
```bash
cd backend
go build -o acpone-desktop ./cmd/desktop     # Build desktop tray app
```

### Full Build (embedded single binary)
```bash
# Build web assets and embed into Go binary
cd web && npm run build && cd ../backend && go build -o acpone ./cmd/acpone
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Vue 3 Frontend                            │
│  ┌─────────┐ ┌──────────────┐ ┌─────────────┐ ┌──────────────┐  │
│  │ Sidebar │ │ChatContainer │ │ ChatInput   │ │SettingsModal │  │
│  └────┬────┘ └──────┬───────┘ └──────┬──────┘ └──────────────┘  │
│       │             │                │                           │
│       └─────────────┼────────────────┘                           │
│                     ▼                                            │
│              ┌─────────────┐                                     │
│              │session.ts   │  (Reactive store)                   │
│              └──────┬──────┘                                     │
│                     ▼                                            │
│              ┌─────────────┐                                     │
│              │  api/       │  HTTP + SSE                         │
│              └──────┬──────┘                                     │
└─────────────────────┼───────────────────────────────────────────┘
                      │ HTTP/SSE
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Go Backend                                │
│  ┌─────────────┐                                                 │
│  │ api/server  │ ──► Routes: /api/chat, /api/sessions, etc.     │
│  └──────┬──────┘                                                 │
│         │                                                        │
│    ┌────┴────┐                                                   │
│    ▼         ▼                                                   │
│ ┌──────┐ ┌────────┐                                              │
│ │Router│ │Session │ (storage/)                                   │
│ └──┬───┘ │Storage │                                              │
│    │     └────────┘                                              │
│    ▼                                                             │
│ ┌────────────────┐                                               │
│ │ Agent Manager  │                                               │
│ └───────┬────────┘                                               │
│         │ JSON-RPC                                               │
│         ▼                                                        │
│ ┌────────────────┐                                               │
│ │ Agent Process  │ (subprocess: claude-code, codex, etc.)       │
│ └────────────────┘                                               │
└─────────────────────────────────────────────────────────────────┘
```

## Key Data Flow

### Chat Message Flow
1. User sends message via `ChatInput` → `POST /api/chat` (SSE)
2. Backend `Router` selects agent based on @mentions or keywords
3. `AgentManager` spawns/connects to agent process via JSON-RPC
4. Agent streams responses → Backend relays via SSE → Frontend updates UI
5. Stream items (text/tool calls) are held in `streamItems` ref until completion
6. On next user message, `commitStreamItems()` moves stream content to messages array

### Permission Flow
Agent requests permission → Backend sends SSE event → `PermissionRequest.vue` displays → User confirms → `POST /api/permission/confirm` → Agent proceeds

### File Upload Flow
1. User uploads file via ChatInput → `POST /api/upload` with multipart form
2. Backend stores file in `.acpone-uploads/` directory in workspace
3. File path is added to chat request and formatted as `@filename` reference in prompt
4. Agent can access uploaded files via file path
5. On session end or manual cleanup → `POST /api/upload/cleanup` removes upload directory

## Key Files

| Path | Purpose |
|------|---------|
| `backend/cmd/acpone/main.go` | Web server entry point, embeds web assets |
| `backend/cmd/desktop/main.go` | Desktop tray app entry point |
| `backend/internal/api/chat.go` | SSE chat handler |
| `backend/internal/api/files.go` | File upload/download/cleanup handlers |
| `backend/internal/agent/manager.go` | Agent lifecycle management |
| `backend/internal/agent/rpc.go` | JSON-RPC communication with agents |
| `backend/internal/router/router.go` | Message routing to agents via @mention/keywords |
| `backend/internal/storage/session.go` | Session persistence to disk |
| `backend/internal/storage/workspace.go` | Workspace management |
| `web/embed.go` | Embeds `web/dist/*` into Go binary via `//go:embed` |
| `web/src/stores/session.ts` | Central state management (agents, sessions, messages) |
| `web/src/api/index.ts` | API client with SSE handling |
| `web/src/components/ChatContainer.vue` | Main chat UI with message rendering |
| `web/src/components/ChatInput.vue` | Input field with @mention, /command, file upload |
| `web/src/components/Sidebar.vue` | Session list and workspace selector |
| `gotray/` | Cross-platform system tray library |

## Configuration

Config file search order (first found wins):
1. `./acpone.config.json`
2. `./acpone.json`
3. `~/.acpone/acpone.config.json` (auto-created on first run)
4. `~/.config/acpone/config.json`

```json
{
  "agents": [
    {
      "id": "claude",
      "name": "Claude Code",
      "command": "npx",
      "args": ["@anthropics/claude-code", "--acp"],
      "permissionMode": "default",
      "env": {
        "ANTHROPIC_API_KEY": "sk-...",
        "API_TIMEOUT_MS": "600000"
      }
    }
  ],
  "defaultAgent": "claude",
  "defaultWorkspace": "default",
  "routing": {
    "keywords": {
      "@claude": "claude",
      "@codex": "codex"
    },
    "meta": true
  },
  "workspaces": [
    { "id": "default", "name": "Default", "path": "." }
  ]
}
```

### Agent Permission Modes
- `default`: User confirms each tool call (recommended)
- `bypass`: Auto-approve all tool calls (use with caution)

### Routing Strategies
- `@agent-id`: Direct mention (highest priority)
- `keywords`: Keyword matching from config (e.g., "use codex" → codex agent)
- `meta`: Meta-routing (agent can route to other agents)

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/agents` | List agents with their configs |
| POST | `/api/agents/update` | Update agent settings |
| GET | `/api/workspaces` | List workspaces |
| POST | `/api/workspaces` | Create workspace |
| GET | `/api/sessions` | List all sessions |
| POST | `/api/sessions/new` | Create new session |
| GET | `/api/sessions/:id` | Get session with messages |
| DELETE | `/api/sessions/:id` | Delete session |
| POST | `/api/chat` | Send message (SSE stream) |
| POST | `/api/cancel` | Cancel current chat |
| POST | `/api/permission/confirm` | Confirm permission request |
| GET | `/api/files` | List files in workspace |
| POST | `/api/upload` | Upload files (multipart form) |
| POST | `/api/upload/cleanup` | Remove upload directory |

### SSE Events (from /api/chat)
- `session`: Session info (conversationId, sessionId, agent)
- `status`: Status message (e.g., "Processing...")
- `message`: Streaming text chunks
- `tool_call`: Tool execution updates
- `commands`: Available slash commands for agent
- `error`: Error message
- `permission_request`: Permission confirmation needed
- `done`: Chat completion (includes stopReason)

## Development Notes

### Frontend Development
- Dev server (`:5173`) proxies `/api/*` to backend (`:3000`)
- Uses Vue 3 Composition API with `ref()` and `computed()`
- No Pinia - uses a custom reactive store pattern in `session.ts`
- Markdown rendering via `markstream-vue` package
- TypeScript strict mode enabled

### Backend Development
- Session data stored in `~/.config/acpone/sessions/`
- Uploaded files stored in `<workspace>/.acpone-uploads/`
- Agent processes are long-running subprocesses
- JSON-RPC 2.0 communication over stdin/stdout
- Each conversation can have multiple agent sessions (one per agent)

### State Management
- `streamItems` holds in-progress tool calls and text chunks
- `commitStreamItems()` is called before sending next user message
- `finalizeStreamItems()` marks completion but doesn't move items yet
- Messages array only contains finalized assistant responses

### Agent Switching
- User can switch agents mid-conversation via @mention
- Each agent gets its own session ID within the conversation
- Context summary is sent to new agent when switching
- Previous messages remain in shared conversation history

### Cross-Platform Considerations
- Desktop app auto-detects and adds common tool paths (npm, cargo, go, etc.)
- Different icon formats: PNG for macOS/Linux, ICO for Windows
- Tray menu implementation varies by OS (handled by `gotray/` package)

## File Structure Constraints

遵循以下代码架构原则：
- Go 文件：每个文件不超过 250 行
- TypeScript/Vue 文件：每个文件不超过 200 行
- 每层文件夹中的文件不超过 8 个（超过需要规划子文件夹）
- 避免循环依赖、重复代码、过度复杂性
