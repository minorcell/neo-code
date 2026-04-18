package memo

import (
	"context"
	"errors"
	"sync"
)

type memoryTestStore struct {
	mu sync.Mutex

	indexes map[Scope]*Index
	topics  map[Scope]map[string]string

	err            error
	saveIndexErr   error
	saveTopicErr   error
	deleteTopicErr error

	loadIndexCalls   int
	saveIndexCalls   int
	saveTopicCalls   int
	deleteTopicCalls int

	deletedTopics []string
}

func newMemoryTestStore() *memoryTestStore {
	return &memoryTestStore{
		indexes: make(map[Scope]*Index),
		topics: map[Scope]map[string]string{
			ScopeUser:    {},
			ScopeProject: {},
		},
	}
}

func (s *memoryTestStore) LoadIndex(_ context.Context, scope Scope) (*Index, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadIndexCalls++
	if s.err != nil {
		return nil, s.err
	}
	index, ok := s.indexes[scope]
	if !ok || index == nil {
		return &Index{Entries: []Entry{}}, nil
	}
	return cloneIndex(index), nil
}

func (s *memoryTestStore) SaveIndex(_ context.Context, scope Scope, index *Index) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.saveIndexCalls++
	if s.saveIndexErr != nil {
		return s.saveIndexErr
	}
	s.indexes[scope] = cloneIndex(index)
	return nil
}

func (s *memoryTestStore) LoadTopic(_ context.Context, scope Scope, filename string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return "", s.err
	}
	content, ok := s.topics[scope][filename]
	if !ok {
		return "", errors.New("not found")
	}
	return content, nil
}

func (s *memoryTestStore) SaveTopic(_ context.Context, scope Scope, filename string, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.saveTopicCalls++
	if s.saveTopicErr != nil {
		return s.saveTopicErr
	}
	if s.topics[scope] == nil {
		s.topics[scope] = map[string]string{}
	}
	s.topics[scope][filename] = content
	return nil
}

func (s *memoryTestStore) DeleteTopic(_ context.Context, scope Scope, filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteTopicCalls++
	s.deletedTopics = append(s.deletedTopics, scopedTopicKey(scope, filename))
	if s.deleteTopicErr != nil {
		return s.deleteTopicErr
	}
	delete(s.topics[scope], filename)
	return nil
}

func (s *memoryTestStore) ListTopics(_ context.Context, scope Scope) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	result := make([]string, 0, len(s.topics[scope]))
	for name := range s.topics[scope] {
		result = append(result, name)
	}
	return result, nil
}
