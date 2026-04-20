package store

import (
	"testing"
	"time"

	"github.com/dmundt/a2ui-go/a2ui"
)

func TestPageStoreCRUDAndListOrder(t *testing.T) {
	s := NewPageStore()
	now := time.Now().UTC()

	s.Upsert(Page{ID: "b", DataModel: a2ui.DataModel{"v": 2}, UpdatedAt: now})
	s.Upsert(Page{ID: "a", DataModel: a2ui.DataModel{"v": 1}, UpdatedAt: now.Add(time.Second)})

	if p, ok := s.Get("a"); !ok || p.DataModel["v"] != 1 {
		t.Fatalf("expected page a with value 1, got %#v ok=%v", p, ok)
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(list))
	}
	if list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("expected deterministic ID ordering [a b], got [%s %s]", list[0].ID, list[1].ID)
	}

	s.Delete("a")
	if _, ok := s.Get("a"); ok {
		t.Fatalf("expected page a to be deleted")
	}
}
