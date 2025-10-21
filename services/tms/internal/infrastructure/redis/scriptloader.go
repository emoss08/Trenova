package redis

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed scripts/*.lua
var scriptsF5 embed.FS

// ScriptLoader loads and manages redis Lua scripts
type ScriptLoader struct {
	redis       *Connection
	scripts     map[string]string // Script name -> SHA Mapping
	scriptNames map[string]string // SHA -> script name (for debugging)
	mu          sync.RWMutex
}

func NewScriptLoader(client *Connection) *ScriptLoader {
	return &ScriptLoader{
		redis:       client,
		scripts:     make(map[string]string),
		scriptNames: make(map[string]string),
	}
}

// LoadScripts loads all Lua scripts from the embedded filesystem into Redis
func (sl *ScriptLoader) LoadScripts(ctx context.Context) error {
	entries, err := scriptsF5.ReadDir("scripts")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err = sl.loadScript(ctx, entry); err != nil {
			if strings.Contains(err.Error(), "circuit breaker is open") {
				continue
			}
			return err
		}
	}

	return nil
}

func (sl *ScriptLoader) UnloadScripts() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.scripts = make(map[string]string)
	sl.scriptNames = make(map[string]string)
	return nil
}

func (sl *ScriptLoader) loadScript(ctx context.Context, entry fs.DirEntry) error {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
		return nil
	}

	scriptName := strings.TrimSuffix(entry.Name(), ".lua")
	scriptPath := filepath.Join("scripts", entry.Name())

	scriptBytes, err := scriptsF5.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %s", scriptPath)
	}

	scriptContent := string(scriptBytes)

	sha, err := sl.redis.Client().ScriptLoad(ctx, scriptContent).Result()
	if err != nil {
		return fmt.Errorf("failed to load script: %s", scriptName)
	}

	sl.mu.Lock()
	sl.scripts[scriptName] = sha
	sl.scriptNames[sha] = scriptName
	sl.mu.Unlock()

	return nil
}

func (sl *ScriptLoader) EvalSHA(
	ctx context.Context,
	scriptName string,
	keys []string,
	args ...any,
) (any, error) {
	sl.mu.RLock()
	sha, exists := sl.scripts[scriptName]
	sl.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("script %s not found", scriptName)
	}

	var result any
	var err error

	result, err = sl.redis.Client().EvalSha(ctx, sha, keys, args...).Result()

	if err != nil && isNoScriptError(err) {
		sha, err = sl.reloadScript(ctx, scriptName)
		if err != nil {
			return nil, err
		}

		result, err = sl.redis.Client().EvalSha(ctx, sha, keys, args...).Result()
		if err != nil {
			return nil, err
		}
		return result, err
	}

	if err != nil {
		return nil, err
	}

	return result, err
}

func (sl *ScriptLoader) reloadScript(ctx context.Context, scriptName string) (string, error) {
	entries, err := scriptsF5.ReadDir("scripts")
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		sm := strings.TrimSuffix(entry.Name(), ".lua")
		if sm == scriptName {
			if err = sl.loadScript(ctx, entry); err != nil {
				return "", err
			}
			break
		}
	}

	sl.mu.RLock()
	sha, exists := sl.scripts[scriptName]
	sl.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("script %s not found after reload", scriptName)
	}

	return sha, nil
}

func (sl *ScriptLoader) GetScriptSHA(scriptName string) (string, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	sha, exists := sl.scripts[scriptName]
	return sha, exists
}

func (sl *ScriptLoader) GetScriptName(sha string) (string, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	name, exists := sl.scriptNames[sha]
	return name, exists
}

func isNoScriptError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "noscript")
}
