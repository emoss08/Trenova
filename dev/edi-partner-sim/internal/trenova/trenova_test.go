package trenova

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCents(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]int64{"1250": 125000, "1250.5": 125050, "0.07": 7,
		"1850.00": 185000} {
		if got, ok := Cents(text); !ok || got != want {
			t.Fatalf("Cents(%q) = %d, want %d", text, got, want)
		}
	}
	if _, ok := Cents("lots"); ok {
		t.Fatal("Cents read lots")
	}
}

func TestAProblemNamesItsFields(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"title":"Invalid","errors":[{"path":"tests",` +
			`"message":"1 tests fail"}]}`))
	}))
	defer server.Close()
	client, err := New(server.URL+"/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.VersionAction(context.Background(), "ldv_1", "certify")
	var problem *Problem
	if !errors.As(err, &problem) || !errors.Is(err, ErrTrenova) ||
		!strings.Contains(err.Error(), "tests: 1 tests fail") {
		t.Fatalf("the problem: %v", err)
	}
}

func TestTheCLIMustBeABuiltBinary(t *testing.T) {
	t.Parallel()
	if err := (&CLI{}).Check(); !errors.Is(err, ErrNoCLI) {
		t.Fatalf("no binary: %v", err)
	}
	plain := filepath.Join(t.TempDir(), "trenova")
	if err := os.WriteFile(plain, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (&CLI{Binary: plain}).Check(); err == nil ||
		!strings.Contains(err.Error(), "not an executable") {
		t.Fatalf("a file that cannot run: %v", err)
	}
}
