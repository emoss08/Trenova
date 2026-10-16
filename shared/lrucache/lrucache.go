package lrucache

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Add(key K, value V) bool
	Remove(key K) bool
	Purge()
	Len() int
}

func New[K comparable, V any](size int) (Cache[K, V], error) {
	cache, err := lru.New[K, V](size)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func NewExpirable[K comparable, V any](size int, ttl time.Duration) Cache[K, V] {
	return expirable.NewLRU[K, V](size, nil, ttl)
}
