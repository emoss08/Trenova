package middleware

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestIdempotencyMiddleware(t *testing.T) {
	type args struct {
		next http.Handler
	}
	tests := []struct {
		name string
		args args
		want http.Handler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IdempotencyMiddleware(tt.args.next); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IdempotencyMiddleware() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	type args struct {
		slice []string
		item  string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.slice, tt.args.item); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_copyHeader(t *testing.T) {
	type args struct {
		dst http.Header
		src http.Header
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyHeader(tt.args.dst, tt.args.src)
		})
	}
}

func Test_getExcludedMethods(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getExcludedMethods(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getExcludedMethods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getExcludedPaths(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getExcludedPaths(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getExcludedPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getIdempotencyKeyTTL(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIdempotencyKeyTTL(); got != tt.want {
				t.Errorf("getIdempotencyKeyTTL() = %v, want %v", got, tt.want)
			}
		})
	}
}
