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

package testutils

import (
	"database/sql"
	"fmt"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	DatabaseUsernameAndPassword = "dbtest"
	DatabaseName                = "database_test"
)

func createTestDatabase() (string, func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16.3",
		Env: []string{
			fmt.Sprintf("POSTGRES_PASSWORD=%s", DatabaseUsernameAndPassword),
			fmt.Sprintf("POSTGRES_USER=%s", DatabaseUsernameAndPassword),
			fmt.Sprintf("POSTGRES_DB=%s", DatabaseName),
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return "", nil, fmt.Errorf("could not start resource: %w", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DatabaseUsernameAndPassword, DatabaseUsernameAndPassword, hostAndPort, DatabaseName)

	_ = resource.Expire(120) // Tell docker to hard kill the container in 120 seconds

	// Try to connect to the database
	err = pool.Retry(func() error {
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	if err != nil {
		return "", nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	return databaseURL, func() { _ = pool.Purge(resource) }, nil
}
