package util

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/internal/util/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/rotisserie/eris"

	"github.com/google/uuid"
)

func ParseBody(c *fiber.Ctx, body any) error {
	if err := c.BodyParser(body); err != nil {
		return eris.Wrap(err, "Error parsing request body")
	}

	return nil
}

var validatorInstance *Validator

func init() {
	var err error
	validatorInstance, err = NewValidator()
	if err != nil {
		log.Fatalf("Failed to initialize validator: %v", err)
	}
}

// ParseBodyAndValidate parses the request body into the given struct and validates it using the given validator.
// If the body is invalid, it writes a 400 response with the validation error.
func ParseBodyAndValidate(c *fiber.Ctx, body any) error {
	logging := logger.GetLogger()
	if err := ParseBody(c, body); err != nil {
		return err
	}

	if err := validatorInstance.Validate(body); err != nil {
		var validationErr *ValidationError
		switch {
		case errors.As(err, &validationErr):
			return c.Status(http.StatusBadRequest).JSON(validationErr)
		default:
			genericErr := c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
			if genericErr != nil {
				wrappedErr := eris.Wrap(genericErr, "Error encoding generic error response")
				logging.WithError(wrappedErr).Error("Error encoding generic error response")
			}
			return err
		}
	}

	return nil
}

// GetSessionDetails retrieves user ID, organization ID, and business unit ID from the session.
func GetSessionDetails(sess *session.Session) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	// Safely attempt to retrieve and type assert each UUID from the session
	userID, ok := getSessionUUID(sess, CTXUserID)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	orgID, ok := getSessionUUID(sess, CTXOrganizationID)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	buID, ok := getSessionUUID(sess, CTXBusinessUnitID)
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

func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "Failed to start transaction")
		return wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "Failed to rollback transaction")
				log.Printf("Failed to rollback transaction: %v", wrappedErr)
			}
			panic(v)
		}
	}()

	if err = fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			err = eris.Wrap(err, "Failed to rollback transaction")
			log.Printf("Failed to rollback transaction: %v", err)
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		err = eris.Wrap(err, "Failed to commit transaction")
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	return nil
}
