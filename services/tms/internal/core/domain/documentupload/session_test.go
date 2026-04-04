package documentupload

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestSessionIsSupersededByNewerArtifacts(t *testing.T) {
	lineageID := pulid.MustNew("doc_")
	currentID := pulid.MustNew("dus_")
	newerID := pulid.MustNew("dus_")

	current := &Session{
		ID:        currentID,
		LineageID: &lineageID,
		CreatedAt: 100,
	}

	newerSession := &Session{
		ID:        newerID,
		LineageID: &lineageID,
		CreatedAt: 101,
	}

	if !current.IsSupersededByNewerArtifacts([]*Session{newerSession}, nil) {
		t.Fatal("expected newer active session to supersede current session")
	}

	current.StoragePath = "same-object"
	if current.IsSupersededByNewerArtifacts(nil, []*document.Document{{
		StoragePath: "same-object",
		CreatedAt:   101,
	}}) {
		t.Fatal("expected same storage path version to be ignored")
	}

	if !current.IsSupersededByNewerArtifacts(nil, []*document.Document{{
		StoragePath: "different-object",
		CreatedAt:   101,
	}}) {
		t.Fatal("expected newer document version to supersede current session")
	}
}

func TestSessionMarkSuperseded(t *testing.T) {
	session := &Session{}

	session.MarkSuperseded(123)

	if session.Status != StatusCanceled {
		t.Fatalf("expected status %q, got %q", StatusCanceled, session.Status)
	}
	if session.FailureCode != "SUPERSEDED_BY_NEWER_SESSION" {
		t.Fatalf("unexpected failure code %q", session.FailureCode)
	}
	if session.FailureMessage == "" {
		t.Fatal("expected failure message to be set")
	}
	if session.LastActivityAt != 123 {
		t.Fatalf("expected last activity 123, got %d", session.LastActivityAt)
	}
}
