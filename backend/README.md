# ACPone Go Backend

Go implementation of the ACPone gateway server.

## Build

```bash
cd go-backend
go build -o acpone ./cmd/acpone
```

## Run

```bash
# Use config file
./acpone -config acpone.config.json -port 3000

# With external web directory
./acpone -web ../web/dist -port 3000

# Default (uses embedded web files if available)
./acpone
```

## Configuration

See `acpone.config.example.json` for configuration options.

Config file locations (in order of priority):
1. Specified via `-config` flag
2. `./acpone.config.json`
3. `./acpone.json`
4. `~/.config/acpone/config.json`

## API Endpoints

- `GET /api/agents` - List available agents
- `POST /api/agents/update` - Update agent settings
- `GET /api/workspaces` - List workspaces
- `POST /api/workspaces/create` - Create workspace
- `GET /api/sessions` - List sessions
- `POST /api/sessions/new` - Create new session
- `GET /api/sessions/:id` - Get session details
- `DELETE /api/sessions/:id` - Delete session
- `POST /api/chat` - Chat (SSE stream)
- `POST /api/permission/confirm` - Confirm permission request

## Development

```bash
# Build for development
go build ./...

# Run with web directory
go run ./cmd/acpone -web ../web/dist
```
