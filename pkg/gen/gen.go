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

package gen

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/pkg/utils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// CodeGeneratable is an interface for models that can generate unique codes
type CodeGeneratable interface {
	TableName() string
	GetCodePrefix(pattern string) string
	GenerateCode(pattern string, counter int) string
}

// CounterManager manages counters for code generation
type CounterManager struct {
	mu           sync.Mutex
	lastCounters map[string]map[string]int // model -> orgID:prefix -> counter
}

func NewCounterManager() *CounterManager {
	return &CounterManager{
		lastCounters: make(map[string]map[string]int),
	}
}

func (cm *CounterManager) IncrementCounter(modelName, orgID, prefix string) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", orgID, prefix)
	if cm.lastCounters[modelName] == nil {
		cm.lastCounters[modelName] = make(map[string]int)
	}
	cm.lastCounters[modelName][key]++
	return cm.lastCounters[modelName][key]
}

func (cm *CounterManager) SetCounter(modelName, orgID, prefix string, value int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", orgID, prefix)
	if cm.lastCounters[modelName] == nil {
		cm.lastCounters[modelName] = make(map[string]int)
	}
	cm.lastCounters[modelName][key] = value
}

type CodeChecker struct {
	DB bun.IDB
}

func (cc *CodeChecker) Exists(ctx context.Context, tableName string, code string, orgID uuid.UUID) (bool, error) {
	exists, err := cc.DB.NewSelect().Table(tableName).Where("code = ? AND organization_id = ?", code, orgID).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("error checking code uniqueness: %w", err)
	}
	return exists, nil
}

type CodeGenerator struct {
	CounterManager *CounterManager
	CodeChecker    *CodeChecker
}

func NewCodeGenerator(cm *CounterManager, cc *CodeChecker) *CodeGenerator {
	return &CodeGenerator{
		CounterManager: cm,
		CodeChecker:    cc,
	}
}

func (cg *CodeGenerator) GenerateUniqueCode(ctx context.Context, model CodeGeneratable, pattern string, orgID uuid.UUID) (string, error) {
	tableName := model.TableName()
	prefix := model.GetCodePrefix(pattern)

	maxAttempts := 1000 // Arbitrary limit to prevent infinite loops
	for attempt := 0; attempt < maxAttempts; attempt++ {
		counter := cg.CounterManager.IncrementCounter(tableName, orgID.String(), prefix)

		code := model.GenerateCode(pattern, counter)
		code = utils.EnsureFixedLength(code, 10)

		exists, err := cg.CodeChecker.Exists(ctx, tableName, code, orgID)
		if err != nil {
			return "", err
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxAttempts)
}

type CodeInitializer struct {
	DB bun.IDB
}

func (ci *CodeInitializer) Initialize(ctx context.Context, cm *CounterManager, models ...CodeGeneratable) error {
	for _, model := range models {
		tableName := model.TableName()
		var codes []struct {
			Code           string    `bun:"code"`
			OrganizationID uuid.UUID `bun:"organization_id"`
		}
		err := ci.DB.NewSelect().Table(tableName).Column("code", "organization_id").Scan(ctx, &codes)
		if err != nil {
			return fmt.Errorf("error fetching codes for %s: %w", tableName, err)
		}

		for _, code := range codes {
			if code.Code == "" {
				// Skip empty codes
				continue
			}

			if len(code.Code) < 4 {
				return fmt.Errorf("invalid code length for %s: %s", tableName, code.Code)
			}

			prefix := code.Code[:4]
			counter := 0

			// Try to parse the counter from the rest of the code
			counterStr := strings.TrimPrefix(code.Code, prefix)
			_, err := fmt.Sscanf(counterStr, "%d", &counter)
			if err != nil {
				// If parsing fails, set counter to 0 and log a warning
				fmt.Printf("Warning: Unable to parse counter for code %s in table %s: %v\n", code.Code, tableName, err)
				counter = 0
			}

			currentCounter := cm.IncrementCounter(tableName, code.OrganizationID.String(), prefix)
			if counter > currentCounter {
				cm.SetCounter(tableName, code.OrganizationID.String(), prefix, counter)
			}
		}
	}

	return nil
}
