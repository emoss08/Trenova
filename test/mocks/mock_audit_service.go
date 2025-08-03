/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
)

type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) LogAction(
	params *services.LogActionParams,
	opts ...services.LogOption,
) error {
	args := m.Called(params, opts)
	return args.Error(0)
}

func (m *MockAuditService) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*audit.Entry], error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*ports.ListResult[*audit.Entry]), args.Error(1)
}

func (m *MockAuditService) ListByResourceID(
	ctx context.Context,
	opts repositories.ListByResourceIDRequest,
) (*ports.ListResult[*audit.Entry], error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*ports.ListResult[*audit.Entry]), args.Error(1)
}

func (m *MockAuditService) GetByID(
	ctx context.Context,
	opts repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*audit.Entry), args.Error(1)
}

func (m *MockAuditService) LiveStream(
	c *fiber.Ctx,
) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockAuditService) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	args := m.Called(resource, fields)
	return args.Error(0)
}

func (m *MockAuditService) GetServiceStatus() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAuditService) SetDefaultField(
	key string,
	value any,
) {
	// * do nothing
	_ = m.Called(key, value)
}

func (m *MockAuditService) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuditService) Stop() error {
	args := m.Called()
	return args.Error(0)
}
