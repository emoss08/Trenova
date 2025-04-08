package redis

import (
	"context"
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rotisserie/eris"
)

//go:embed scripts/*.lua
var scriptsF5 embed.FS

// ScriptLoader loads and manages redis Lua scripts
type ScriptLoader struct {
	redis       *Client
	scripts     map[string]string // Script name -> SHA Mapping
	scriptNames map[string]string // SHA -> script name (for debugging)
	mu          sync.RWMutex
}

func NewScriptLoader(client *Client) *ScriptLoader {
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
		return eris.Wrap(err, "failed to read scripts directory")
	}

	for _, entry := range entries {
		if err = sl.loadScript(ctx, entry); err != nil {
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
		return eris.Wrapf(err, "failed to read script: %s", scriptPath)
	}

	scriptContent := string(scriptBytes)
	sha, err := sl.redis.ScriptLoad(ctx, scriptContent).Result()
	if err != nil {
		return eris.Wrapf(err, "failed to load script: %s", scriptName)
	}

	sl.mu.Lock()
	sl.scripts[scriptName] = sha
	sl.scriptNames[sha] = scriptName
	sl.mu.Unlock()

	return nil
}

func (sl *ScriptLoader) EvalSHA(ctx context.Context, scriptName string, keys []string, args ...any) (any, error) {
	sl.mu.RLock()
	sha, exists := sl.scripts[scriptName]
	sl.mu.RUnlock()

	if !exists {
		return nil, eris.Errorf("script %s not found", scriptName)
	}

	result, err := sl.redis.EvalSha(ctx, sha, keys, args...).Result()
	if err != nil && isNoScriptError(err) {
		// Script not found in Redis, try to reload it
		sha, err = sl.reloadScript(ctx, scriptName)
		if err != nil {
			return nil, err
		}

		// Retry with the reloaded script
		result, err = sl.redis.EvalSha(ctx, sha, keys, args...).Result()
	}

	return result, err
}

// reloadScript attempts to reload a script into Redis
func (sl *ScriptLoader) reloadScript(ctx context.Context, scriptName string) (string, error) {
	entries, err := scriptsF5.ReadDir("scripts")
	if err != nil {
		return "", eris.Wrap(err, "failed to read scripts directory")
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
		return "", eris.Errorf("script %s not found after reload", scriptName)
	}

	return sha, nil
}

// GetScriptSHA returns the SHA for a script name
func (sl *ScriptLoader) GetScriptSHA(scriptName string) (string, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	sha, exists := sl.scripts[scriptName]
	return sha, exists
}

// GetScriptName returns the name for a script SHA
func (sl *ScriptLoader) GetScriptName(sha string) (string, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	name, exists := sl.scriptNames[sha]
	return name, exists
}

// isNoScriptError checks if the error is a Redis NOSCRIPT error
func isNoScriptError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "noscript")
}
