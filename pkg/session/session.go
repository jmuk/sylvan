package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
)

const (
	sessionIDsFile  = "session-ids.txt"
	sessionMetaFile = "session.toml"
)

type sessionMeta struct {
	SessionID  string    `toml:"session_id"`
	Timestamp  time.Time `toml:"timestamp"`
	WorkingDir string    `toml:"path"`
}

type logger struct {
	f *os.File
	l *slog.Logger
}

func newLogger(p string, opts *slog.HandlerOptions) (*logger, error) {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &logger{
		f: f,
		l: slog.New(slog.NewJSONHandler(f, opts)),
	}, nil
}

func (h *logger) Close() error {
	return h.f.Close()
}

type Session struct {
	meta        sessionMeta
	sessionPath string

	loggers map[string]*logger
	files   map[string]*os.File

	initialized bool
}

type sessionKeyT struct{}

var sessionKey = sessionKeyT{}

func (s *Session) With(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey).(*Session)
	return s, ok
}

func (s *Session) ID() string {
	return s.meta.SessionID
}

func (s *Session) Timestamp() time.Time {
	return s.meta.Timestamp
}

func (s *Session) updateSessionsFile(workingDir string) error {
	sessionsFile := filepath.Join(workingDir, sessionIDsFile)
	sessionsContent, err := os.ReadFile(sessionsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		sessionsContent = []byte{}
	}
	var lines []string
	for line := range strings.Lines(string(sessionsContent)) {
		line = strings.TrimSpace(line)
		if line == s.meta.SessionID {
			return nil
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	lines = append(lines, s.meta.SessionID)
	return os.WriteFile(sessionsFile, []byte(strings.Join(lines, "\n")), 0644)
}

func (s *Session) Init() error {
	if s.initialized {
		return nil
	}
	if s.meta.SessionID == "" {
		s.initialized = true
		// likely generated on tempdir; skipping.
		return nil
	}

	cacheDir, err := getCacheBase()
	if err != nil {
		return err
	}

	workingDir := getWorkingDir(cacheDir, s.meta.WorkingDir)
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return err
	}

	if err := s.updateSessionsFile(workingDir); err != nil {
		return err
	}

	if err := os.MkdirAll(s.sessionPath, 0755); err != nil {
		return err
	}

	metaFile := filepath.Join(s.sessionPath, sessionMetaFile)
	encodedMeta, err := toml.Marshal(s.meta)
	if err != nil {
		return err
	}
	if err := os.WriteFile(metaFile, encodedMeta, 0644); err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *Session) HistoryFile() string {
	return filepath.Join(s.sessionPath, "history.json")
}

func (s *Session) logPath() string {
	return filepath.Join(s.sessionPath, "logs")
}

func (s *Session) GetLogFile(filename string) (io.Writer, error) {
	if f, ok := s.files[filename]; ok {
		return f, nil
	}
	if strings.Contains(filename, "/") {
		return nil, fmt.Errorf("malformed filename %s", filename)
	}
	if err := os.MkdirAll(s.logPath(), 0755); err != nil {
		return nil, err
	}
	p := filepath.Join(s.logPath(), filename)
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	s.files[filename] = f
	return f, nil
}

func (s *Session) GetLogger(name string) (*slog.Logger, error) {
	l, ok := s.loggers[name]
	if ok {
		return l.l, nil
	}
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("malformed log name %s", name)
	}
	var pathName string = name
	if !strings.Contains(name, ".") {
		pathName = name + ".jsonl"
	}
	if err := os.MkdirAll(s.logPath(), 0755); err != nil {
		return nil, err
	}
	l, err := newLogger(filepath.Join(s.logPath(), pathName), nil)
	if err != nil {
		return nil, err
	}
	s.loggers[name] = l
	return l.l, nil
}

func (s *Session) Close() error {
	if !s.initialized {
		return nil
	}
	var allerr error
	for name, l := range s.loggers {
		err := l.Close()
		if err != nil {
			allerr = errors.Join(allerr, fmt.Errorf("failed to close %s: %w", name, err))
		}
	}
	for name, f := range s.files {
		err := f.Close()
		if err != nil {
			allerr = errors.Join(allerr, fmt.Errorf("failed to close %s: %w", name, err))
		}
	}
	return allerr
}

func getCacheBase() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "sylvan"), nil
}

func getWorkingDir(cacheDir, p string) string {
	h := sha256.Sum256([]byte(p))
	hhex := hex.EncodeToString(h[:])
	return filepath.Join(cacheDir, "paths", hhex)
}

func ListSessions(cwd string) ([]*Session, error) {
	cacheDir, err := getCacheBase()
	if err != nil {
		return nil, err
	}
	workingDir := getWorkingDir(cacheDir, cwd)
	if finfo, err := os.Stat(workingDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	} else if !finfo.IsDir() {
		return nil, fmt.Errorf("path %s is not a dir", workingDir)
	}

	sessionsFile := filepath.Join(workingDir, sessionIDsFile)
	content, err := os.ReadFile(sessionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var results []*Session
	for line := range strings.Lines(string(content)) {
		session, err := newFromID(strings.TrimSpace(line), cacheDir)
		if err != nil {
			continue
		}
		if session.meta.WorkingDir == cwd {
			results = append(results, session)
		}
	}
	return results, nil
}

func NewFromID(sessionID string) (*Session, error) {
	cacheDir, err := getCacheBase()
	if err != nil {
		return nil, err
	}
	return newFromID(sessionID, cacheDir)
}

func newFromID(sessionID, cacheDir string) (*Session, error) {
	if _, err := uuid.Parse(sessionID); err != nil {
		return nil, fmt.Errorf("illformed session ID %s: %w", sessionID, err)
	}
	sessionDir := filepath.Join(cacheDir, "sessions", sessionID)
	if finfo, err := os.Stat(sessionDir); err != nil {
		return nil, err
	} else if !finfo.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", sessionDir)
	}

	metaFile := filepath.Join(sessionDir, sessionMetaFile)
	metadata, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}
	var m sessionMeta
	if err := toml.Unmarshal(metadata, &m); err != nil {
		return nil, err
	}
	return &Session{
		meta:        m,
		sessionPath: sessionDir,
		loggers:     map[string]*logger{},
		files:       map[string]*os.File{},
	}, nil
}

func New(cwd string) (*Session, error) {
	now := time.Now()
	cacheDir, err := getCacheBase()
	if err != nil {
		log.Printf("Failed to obtain the user cache dir: %v", err)
		log.Printf("Falls back to the temporary directory...")
		tempDir, err := os.MkdirTemp("", "sylvan")
		if err != nil {
			return nil, err
		}
		log.Printf("Session stored at: %s", tempDir)
		return &Session{
			meta: sessionMeta{
				Timestamp: now,
			},
			sessionPath: tempDir,
			loggers:     map[string]*logger{},
			files:       map[string]*os.File{},
		}, nil
	}

	sessionUUID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sessionPath := filepath.Join(cacheDir, "sessions", sessionUUID.String())
	return &Session{
		meta: sessionMeta{
			SessionID:  sessionUUID.String(),
			Timestamp:  now,
			WorkingDir: cwd,
		},
		sessionPath: sessionPath,
		loggers:     map[string]*logger{},
		files:       map[string]*os.File{},
	}, nil
}
