package util

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/google/uuid"
)

var (
	validatorInstance *Validator
	once              sync.Once
)

// GetValidatorInstance safely initializes Validator instance only once
func GetValidatorInstance() *Validator {
	once.Do(func() {
		var err error
		validatorInstance, err = NewValidator()
		if err != nil {
			log.Fatalf("Failed to initialize validator: %v", err)
		}
	})
	return validatorInstance
}

func ParseBody(c *fiber.Ctx, body any) error {
	if err := c.BodyParser(body); err != nil {
		return fiber.ErrBadRequest
	}
	return nil
}

// ParseBodyAndValidate parses the request body into the given struct and validates it using the given validator.
func ParseBodyAndValidate(c *fiber.Ctx, body any) error {
	if err := ParseBody(c, body); err != nil {
		return err
	}
	validator := GetValidatorInstance()

	if err := validator.Validate(body); err != nil {
		var validationErr *ValidationError
		switch {
		case errors.As(err, &validationErr):
			return c.Status(http.StatusBadRequest).JSON(validationErr)
		default:
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
		}
	}
	return nil
}

// GetSessionDetails retrieves user ID, organization ID, and business unit ID from the session.
func GetSessionDetails(sess *session.Session) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	// Safely attempt to retrieve and type assert each UUID from the session
	userID, ok := getSessionUUID(sess, string(CTXUserID))
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	orgID, ok := getSessionUUID(sess, string(CTXOrganizationID))
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	buID, ok := getSessionUUID(sess, string(CTXBusinessUnitID))
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	return userID, orgID, buID, true
}

// getSessionUUID safely retrieves a UUID from the session based on the provided key
func getSessionUUID(sess *session.Session, key string) (uuid.UUID, bool) {
	raw := sess.Get(key)
	if raw == nil {
		return uuid.Nil, false
	}
	if id, ok := raw.(uuid.UUID); ok {
		return id, true
	}
	return uuid.Nil, false
}

// WithTx executes the given function within a transaction.
func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	// TODO(WOLFRED): Change logging to zerolog. We could use the one from the server.Logger.
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Failed to rollback transaction: %v", err)
			}
			panic(v)
		}
	}()

	if err = fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Failed to rollback transaction: %v", err)
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	return nil
}

func ConvertToInt(value string, defaultCount int) int {
	if value == "" {
		return defaultCount
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return defaultCount
	}
	return count
}
