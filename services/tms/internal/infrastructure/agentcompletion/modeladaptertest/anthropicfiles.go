package modeladaptertest

import (
	"context"
	"errors"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

// fileStore stands in for the conversation's files and counts its reads.
type FileStore struct {
	Mu    sync.Mutex
	Files map[string]*modeladapter.MessageFileContent
	Reads map[string]int
}

func (s *FileStore) Source(
	_ context.Context,
	documentID string,
) (*modeladapter.MessageFileContent, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Reads[documentID]++
	content, ok := s.Files[documentID]
	if !ok {
		return nil, errors.New("not found")
	}

	return content, nil
}

func (s *FileStore) ReadsOf(documentID string) int {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	return s.Reads[documentID]
}
