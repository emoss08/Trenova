package database

import (
	"github.com/emoss08/trenova/ent"
)

var client *ent.Client

func GetClient() *ent.Client {
	return client
}

func SetClient(newClient *ent.Client) {
	client = newClient
}

func NewEntClient(dsn string) (*ent.Client, error) {
	entClient, err := ent.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return entClient, nil
}
