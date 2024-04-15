package handlers_test

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/test"
)

//func TestCreateAccessorialCharge(t *testing.T) {
//	test.WithTestServer(t, func(s *api.Server) {
//	})
//}

func TestGetAccessorialCharges(t *testing.T) {
	test.WithTestServer(t, func(s *api.Server) {
		res := test.PerformRequest(t, s, "GET", "/api/accessorial-charges", nil, nil)

		fmt.Printf("Response body: %s\n", res.Body.String())
		require.Equal(t, fiber.StatusOK, res.Result().StatusCode)
	})
}
