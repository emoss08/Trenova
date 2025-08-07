/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package appctx_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAddContextToRequest(t *testing.T) {
	reqCtx := &appctx.RequestContext{
		OrgID:  pulid.MustNew("org"),
		BuID:   pulid.MustNew("bu"),
		UserID: pulid.MustNew("usr"),
	}

	t.Run("all fields present", func(t *testing.T) {
		type MyRequest struct {
			OrgID  pulid.ID
			BuID   pulid.ID
			UserID pulid.ID
			Other  string
		}
		req := &MyRequest{Other: "test"}

		appctx.AddContextToRequest(reqCtx, req)

		assert.Equal(t, reqCtx.OrgID, req.OrgID)
		assert.Equal(t, reqCtx.BuID, req.BuID)
		assert.Equal(t, reqCtx.UserID, req.UserID)
		assert.Equal(t, "test", req.Other)
	})

	t.Run("some fields present", func(t *testing.T) {
		type MyPartialRequest struct {
			OrgID  pulid.ID
			UserID pulid.ID
		}
		req := &MyPartialRequest{}

		appctx.AddContextToRequest(reqCtx, req)

		assert.Equal(t, reqCtx.OrgID, req.OrgID)
		assert.Equal(t, reqCtx.UserID, req.UserID)
	})

	t.Run("no fields present", func(t *testing.T) {
		type EmptyRequest struct{}
		req := &EmptyRequest{}
		originalReq := *req

		appctx.AddContextToRequest(reqCtx, req)

		assert.Equal(t, originalReq, *req)
	})

	t.Run("not a pointer to struct", func(t *testing.T) {
		// Should not panic
		appctx.AddContextToRequest(reqCtx, "not a struct")
		appctx.AddContextToRequest(reqCtx, 123)
		appctx.AddContextToRequest(reqCtx, nil)
	})

	t.Run("unexported fields", func(t *testing.T) {
		type UnexportedRequest struct {
			orgID  pulid.ID
			buID   pulid.ID
			userID pulid.ID
		}
		req := &UnexportedRequest{}
		originalReq := *req

		appctx.AddContextToRequest(reqCtx, req)
		assert.Equal(t, originalReq, *req)
	})
}
