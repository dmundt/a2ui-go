package mcp_test

import (
	"context"
	"testing"
	"time"

	"github.com/dmundt/a2ui-go/internal/engine"
	"github.com/dmundt/a2ui-go/internal/store"
	"github.com/dmundt/a2ui-go/internal/stream"
	"github.com/dmundt/a2ui-go/mcp"
	"github.com/dmundt/a2ui-go/renderer"
)

func TestStartStdioReturnsAfterContextCancel(t *testing.T) {
	reg, err := renderer.NewRegistry("../renderer/templates")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	r := renderer.New(reg)
	eng := engine.New(r, reg, store.NewPageStore(), stream.NewBroker(), "../internal/ui")
	h := mcp.NewHandlers(eng, reg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- mcp.StartStdio(ctx, h)
	}()

	select {
	case <-done:
		// Returning quickly is expected with a canceled context.
	case <-time.After(2 * time.Second):
		t.Fatalf("StartStdio did not return after context cancellation")
	}
}
