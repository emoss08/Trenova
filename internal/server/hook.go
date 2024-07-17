// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
