// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go4.org/syncutil"
)

var onStart appHooks

func OnStart(name string, fn HookFunc) {
	onStart.add(newHook(name, fn))
}

// ----------------------------------------------------------------------------

type HookFunc func(ctx context.Context, app *Server) error

type appHooks struct {
	mu    sync.Mutex
	hooks []appHook
}

func (hs *appHooks) add(hook appHook) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.hooks = append(hs.hooks, hook)
}

func (hs *appHooks) Run(ctx context.Context, app *Server) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	var group syncutil.Group
	for _, h := range hs.hooks {
		h := h
		group.Go(func() error {
			err := h.run(ctx, app)
			if err != nil {
				fmt.Printf("hook=%q failed: %s\n", h.name, err)
			}
			return err
		})
	}
	return group.Err()
}

type appHook struct {
	name string
	fn   HookFunc
}

func newHook(name string, fn HookFunc) appHook {
	return appHook{name: name, fn: fn}
}

func (h appHook) run(ctx context.Context, app *Server) error {
	const timeout = 30 * time.Second //

	done := make(chan struct{})
	errc := make(chan error)

	go func() {
		start := time.Now()
		if err := h.fn(ctx, app); err != nil {
			errc <- err
			return
		}

		if d := time.Since(start); d > time.Second {
			fmt.Printf("hook=%q took=%s\n", h.name, d)
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case err := <-errc:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("hook=%q timeout after %s", h.name, timeout)
	}
}
