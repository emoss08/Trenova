/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dbutil

import (
	"context"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/uptrace/bun"
)

// OperationType represents the type of database operation
type OperationType int

const (
	// ReadOperation indicates a read-only operation
	ReadOperation OperationType = iota
	// WriteOperation indicates a write operation
	WriteOperation
)

// ConnectionSelector helps select the appropriate database connection
type ConnectionSelector struct {
	conn db.Connection
	// Cache for performance
	readDB  atomic.Pointer[bun.DB]
	writeDB atomic.Pointer[bun.DB]
}

// NewConnectionSelector creates a new connection selector
func NewConnectionSelector(conn db.Connection) *ConnectionSelector {
	return &ConnectionSelector{conn: conn}
}

// GetDB returns the appropriate database connection based on the operation type
func (cs *ConnectionSelector) GetDB(ctx context.Context, opType OperationType) (*bun.DB, error) {
	switch opType {
	case ReadOperation:
		return cs.conn.ReadDB(ctx)
	case WriteOperation:
		return cs.conn.WriteDB(ctx)
	default:
		// * Default to write connection for safety
		return cs.conn.WriteDB(ctx)
	}
}

// Read is a convenience method for getting a read connection with caching
func (cs *ConnectionSelector) Read(ctx context.Context) (*bun.DB, error) {
	// Try to get cached connection first
	if cached := cs.readDB.Load(); cached != nil {
		return cached, nil
	}

	// Get fresh connection and cache it
	dba, err := cs.conn.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	cs.readDB.Store(dba)
	return dba, nil
}

// Write is a convenience method for getting a write connection with caching
func (cs *ConnectionSelector) Write(ctx context.Context) (*bun.DB, error) {
	// Try to get cached connection first
	if cached := cs.writeDB.Load(); cached != nil {
		return cached, nil
	}

	// Get fresh connection and cache it
	dba, err := cs.conn.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	cs.writeDB.Store(dba)
	return dba, nil
}

// InferOperationType attempts to infer the operation type from common method names
func InferOperationType(methodName string) OperationType {
	// * Common read operation prefixes
	readPrefixes := []string{
		"Get", "List", "Find", "Search", "Query", "Select",
		"Count", "Exists", "Has", "Is", "Check", "Fetch",
	}

	for _, prefix := range readPrefixes {
		if len(methodName) >= len(prefix) && methodName[:len(prefix)] == prefix {
			return ReadOperation
		}
	}

	// * Default to write operation for safety
	return WriteOperation
}

// TransactionHelper provides utilities for handling transactions
type TransactionHelper struct {
	conn db.Connection
}

// NewTransactionHelper creates a new transaction helper
func NewTransactionHelper(conn db.Connection) *TransactionHelper {
	return &TransactionHelper{conn: conn}
}

// RunInTx executes a function within a transaction
// ! Transactions always use the write connection
func (th *TransactionHelper) RunInTx(
	ctx context.Context,
	fn func(ctx context.Context, tx bun.Tx) error,
) error {
	dba, err := th.conn.WriteDB(ctx)
	if err != nil {
		return err
	}

	return dba.RunInTx(ctx, nil, fn)
}
