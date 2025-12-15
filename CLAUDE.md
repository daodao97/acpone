# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ACPone is an ACP (Anthropic Protocol) Gateway Chat interface with a Go backend and Vue 3 + TypeScript frontend. It provides a web-based chat interface for communicating with AI agents (like Claude Code) through JSON-RPC, with support for multiple workspaces, sessions, and agent routing.

## Build Commands

### Frontend (web/)
```bash
cd web
npm install              # Install dependencies
npm run dev              # Dev server on :5173 (proxies API to :3000)
npm run build            # Build to web/dist
npm run build:togo       # Build embedded into backend/cmd/acpone/web
```

### Backend (backend/)
```bash
cd backend
go build -o acpone ./cmd/acpone              # Build binary
go run ./cmd/acpone                          # Run with embedded web
go run ./cmd/acpone -web ../web/dist         # Run with external web dir
go run ./cmd/acpone -port 8080               # Custom port (default: 3000)
```

### Full Build (embedded single binary)
```bash
cd web && npm run build:togo && cd ../backend && go build -o acpone ./cmd/acpone
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
│              │session.ts   │  (Pinia-style store)                │
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
5. On completion, `finalizeStreamItems()` moves stream content to messages

### Permission Flow
Agent requests permission → Backend sends SSE event → `PermissionRequest.vue` displays → User confirms → `POST /api/permission/confirm` → Agent proceeds

## Key Files

| Path | Purpose |
|------|---------|
| `backend/cmd/acpone/main.go` | Entry point, embeds web assets |
| `backend/internal/api/chat.go` | SSE chat handler |
| `backend/internal/agent/manager.go` | Agent lifecycle |
| `backend/internal/agent/rpc.go` | JSON-RPC communication |
| `backend/internal/router/router.go` | Message routing to agents |
| `web/src/stores/session.ts` | Central state management |
| `web/src/api/index.ts` | API client with SSE handling |
| `web/src/components/ChatContainer.vue` | Main chat UI |

## Configuration

Config file search order (first found wins):
1. `./acpone.config.json`
2. `./acpone.json`
3. `~/.acpone/acpone.config.json` (auto-created on first run)
4. `~/.config/acpone/config.json`

```json
{
  "agents": [
    { "id": "claude", "name": "Claude Code", "command": "npx", "args": ["@anthropics/claude-code", "--acp"] }
  ],
  "defaultAgent": "claude",
  "routing": { "keywords": { "use codex": "codex" }, "meta": true },
  "workspaces": [{ "id": "default", "name": "Default", "path": "." }]
}
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/agents` | List agents |
| POST | `/api/agents/update` | Update agent settings |
| GET | `/api/workspaces` | List workspaces |
| POST | `/api/workspaces` | Create workspace |
| GET | `/api/sessions` | List sessions |
| POST | `/api/sessions/new` | Create session |
| GET | `/api/sessions/:id` | Get session |
| DELETE | `/api/sessions/:id` | Delete session |
| POST | `/api/chat` | Chat (SSE stream) |
| POST | `/api/permission/confirm` | Confirm permission |

## Development Notes

- Frontend dev server (`:5173`) proxies `/api/*` to backend (`:3000`)
- Session data stored in `~/.config/acpone/sessions/`
- Agent permission modes: `default` (user confirms) or `bypass` (auto-approve)
- SSE events: `message`, `tool_call`, `error`, `commands`, `permission_request`
