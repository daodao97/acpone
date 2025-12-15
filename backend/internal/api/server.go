package api

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/daodao97/acpone/internal/agent"
	"github.com/daodao97/acpone/internal/config"
	"github.com/daodao97/acpone/internal/conversation"
	"github.com/daodao97/acpone/internal/router"
	"github.com/daodao97/acpone/internal/storage"
)

// Server is the HTTP server
type Server struct {
	config         *config.Config
	agents         *agent.Manager
	router         *router.Router
	conversations  *conversation.Manager
	sessionStore   *storage.SessionStore
	workspaceStore *storage.WorkspaceStore
	staticFS       fs.FS

	// Per-conversation agent sessions: convID -> agentID -> sessionID
	agentSessions map[string]map[string]string
	initialized   map[string]bool

	// Cached commands per agent
	agentCommands   map[string][]SlashCommand
	agentCommandsMu sync.RWMutex

	// Setup status cache
	setupStatus *SetupStatus
	setupMu     sync.RWMutex
	setupSubs   map[chan SetupStatus]struct{}
	setupSubsMu sync.RWMutex
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, staticFS fs.FS) *Server {
	s := &Server{
		config:         cfg,
		agents:         agent.NewManager(cfg),
		router:         router.New(cfg),
		conversations:  conversation.NewManager(),
		sessionStore:   storage.NewSessionStore(""),
		workspaceStore: storage.NewWorkspaceStore(""),
		staticFS:       staticFS,
		agentSessions:  make(map[string]map[string]string),
		initialized:    make(map[string]bool),
		agentCommands:  make(map[string][]SlashCommand),
		setupSubs:      make(map[chan SetupStatus]struct{}),
	}

	s.loadPersistedWorkspaces()
	s.initSetupStatus()
	go s.checkDependenciesAsync()
	return s
}

func (s *Server) loadPersistedWorkspaces() {
	persisted := s.workspaceStore.Load()
	for _, ws := range persisted {
		exists := false
		for _, existing := range s.config.Workspaces {
			if existing.ID == ws.ID {
				exists = true
				break
			}
		}
		if !exists {
			s.config.Workspaces = append(s.config.Workspaces, ws)
		}
	}
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	mux.HandleFunc("/api/setup/subscribe", s.handleSetupSubscribe)
	mux.HandleFunc("/api/setup/install", s.handleSetupInstall)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/update", s.handleAgentUpdate)
	mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/new", s.handleSessionNew)
	mux.HandleFunc("/api/sessions/", s.handleSessionByID)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/permission/confirm", s.handlePermissionConfirm)

	// Static files
	if s.staticFS != nil {
		fileServer := http.FileServer(http.FS(s.staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			// Try to open the file
			f, err := s.staticFS.Open(path)
			if err != nil {
				// SPA fallback: serve index.html content directly
				indexFile, err := s.staticFS.Open("index.html")
				if err != nil {
					http.NotFound(w, r)
					return
				}
				defer indexFile.Close()

				stat, err := indexFile.Stat()
				if err != nil {
					http.NotFound(w, r)
					return
				}

				// Serve index.html with correct content type
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
				return
			}
			f.Close()

			// Serve the actual file
			fileServer.ServeHTTP(w, r)
		})
	}

	return corsMiddleware(mux)
}

// Shutdown stops all agents
func (s *Server) Shutdown() error {
	return s.agents.Shutdown()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}

// StaticFS is embedded static files (set from main)
var StaticFS embed.FS
