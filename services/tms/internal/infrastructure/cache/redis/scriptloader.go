/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
			// If loading fails due to circuit breaker, continue but log warning
			if strings.Contains(err.Error(), "circuit breaker is open") {
				// Continue loading other scripts, this one will be loaded later
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
		return eris.Wrapf(err, "failed to read script: %s", scriptPath)
	}

	scriptContent := string(scriptBytes)

	var sha string
	executeErr := sl.redis.executeWithCircuitBreaker(ctx, "SCRIPT_LOAD_"+scriptName, func() error {
		var e error
		sha, e = sl.redis.ScriptLoad(ctx, scriptContent).Result()
		return e
	})

	if executeErr != nil {
		return eris.Wrapf(executeErr, "failed to load script: %s", scriptName)
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
		return nil, eris.Errorf("script %s not found", scriptName)
	}

	var result any
	var err error

	executeErr := sl.redis.executeWithCircuitBreaker(ctx, "EVALSHA_"+scriptName, func() error {
		result, err = sl.redis.EvalSha(ctx, sha, keys, args...).Result()
		return err
	})

	if (executeErr != nil && isNoScriptError(executeErr)) || (err != nil && isNoScriptError(err)) {
		sha, err = sl.reloadScript(ctx, scriptName)
		if err != nil {
			return nil, err
		}

		executeErr = sl.redis.executeWithCircuitBreaker(
			ctx,
			"EVALSHA_"+scriptName+"_RETRY",
			func() error {
				result, err = sl.redis.EvalSha(ctx, sha, keys, args...).Result()
				return err
			},
		)

		if executeErr != nil {
			return nil, executeErr
		}
		return result, err
	}

	if executeErr != nil {
		return nil, executeErr
	}

	return result, err
}

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
