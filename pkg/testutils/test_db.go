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
	"runtime"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
)

const (
	DatabaseUsernameAndPassword = "dbtest"
	DatabaseName                = "database_test"
)

type Database struct {
	db *bun.DB
}

func New(db *sql.DB, verbose bool) *Database {
	d := new(Database)
	d.db = bun.NewDB(db, pgdialect.New())

	// configuration of database
	maxOpenConnections := 4 * runtime.GOMAXPROCS(0)
	d.db.SetMaxOpenConns(maxOpenConnections)
	d.db.SetMaxIdleConns(maxOpenConnections)

	if verbose {
		d.db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	return d
}

func initDatabase() (string, func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
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
		// set AutoRemove to true so that stopped container goes away by itself
		cfg.AutoRemove = true
		cfg.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", DatabaseUsernameAndPassword, DatabaseUsernameAndPassword, hostAndPort, DatabaseName)
	log.Println("Connecting to database on url: ", databaseURL)

	_ = resource.Expire(120) // Tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	return databaseURL, func() {
		// You can't defer this because os.Exit doesn't care for defer
		if err = pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}, err
}
