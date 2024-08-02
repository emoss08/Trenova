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
	"log"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	DatabaseUsernameAndPassword = "dbtest"
	DatabaseName                = "database_test"
)

func retryWithBackoff(attempts int, initialSleep time.Duration, fn func() error) error {
	var err error
	sleep := initialSleep
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		log.Printf("Attempt %d failed, retrying in %v: %v", i+1, sleep, err)
		time.Sleep(sleep)
		sleep = sleep * 2
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func initDatabase() (string, func(), error) {
	var err error
	dbOnce.Do(func() {
		sharedDBURL, dbCleanup, err = createTestDatabase()
	})
	if err != nil {
		return "", nil, err
	}
	return sharedDBURL, nil, nil
}

func createTestDatabase() (string, func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	log.Println("Starting PostgreSQL container...")
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16.3",
		Env: []string{
			fmt.Sprintf("POSTGRES_PASSWORD=%s", DatabaseUsernameAndPassword),
			fmt.Sprintf("POSTGRES_USER=%s", DatabaseUsernameAndPassword),
			fmt.Sprintf("POSTGRES_DB=%s", DatabaseName),
			"listen_addresses = '*'",
		},
	}, func(cfg *docker.HostConfig) {
		cfg.AutoRemove = true
		cfg.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DatabaseUsernameAndPassword, DatabaseUsernameAndPassword, hostAndPort, DatabaseName)
	log.Println("PostgreSQL container started. Connecting to database on url: ", databaseURL)

	// Increase the max wait time
	pool.MaxWait = 300 * time.Second // 5 minutes

	// Add an initial delay before attempting to connect
	time.Sleep(5 * time.Second)

	log.Println("Attempting to connect to the database...")
	err = retryWithBackoff(15, 2*time.Second, func() error {
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}
		err = db.Ping()
		if err == nil {
			log.Println("Successfully connected to the database.")
		}
		return err
	})
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	return databaseURL, func() {
		log.Println("Purging database container...")
		if err = pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
		log.Println("Database container purged successfully.")
	}, nil
}
