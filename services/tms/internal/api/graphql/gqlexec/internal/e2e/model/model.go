package model

import (
	"context"
	"errors"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

type Node interface {
	IsNode()
}

type SearchResult interface {
	IsSearchResult()
}

type Owner struct {
	ID   string
	Name string
}

type Truck struct {
	ID       string
	Name     string
	Status   Status
	Owner    *Owner
	Tags     []string
	Capacity *int
}

func (*Truck) IsNode()         {}
func (*Truck) IsSearchResult() {}

func (t *Truck) DisplayName(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.Name == "" {
		return "", errors.New("truck has no name")
	}
	return t.ID + " " + t.Name, nil
}

type Driver struct {
	ID      string
	Name    string
	Rating  float64
	TruckID string
}

func (*Driver) IsNode()         {}
func (*Driver) IsSearchResult() {}
