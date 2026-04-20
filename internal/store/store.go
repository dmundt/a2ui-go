package store

import (
	"sort"
	"sync"
	"time"

	"github.com/dmundt/au2ui-go/a2ui"
)

// Page holds the state for one A2UI surface.
type Page struct {
	ID         string
	Components map[string]*a2ui.Component // flat adjacency-list buffer
	DataModel  a2ui.DataModel
	RootID     string // set by beginRendering
	CatalogID  string // set by beginRendering
	Ready      bool   // true after beginRendering fires
	HTML       string
	UpdatedAt  time.Time
	Ended      bool
}

// PageStore is an in-memory deterministic page repository.
type PageStore struct {
	mu    sync.RWMutex
	pages map[string]Page
}

// NewPageStore builds a thread-safe page store.
func NewPageStore() *PageStore {
	return &PageStore{pages: make(map[string]Page)}
}

// Upsert inserts or replaces a page.
func (s *PageStore) Upsert(p Page) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pages[p.ID] = p
}

// Get returns a page by id.
func (s *PageStore) Get(id string) (Page, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pages[id]
	return p, ok
}

// Delete removes a page by id.
func (s *PageStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pages, id)
}

// List returns all pages in deterministic id order.
func (s *PageStore) List() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.pages))
	for id := range s.pages {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]Page, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.pages[id])
	}
	return out
}
