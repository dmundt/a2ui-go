package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dmundt/a2ui-go/internal/engine"
	"github.com/dmundt/a2ui-go/internal/store"
	"github.com/dmundt/a2ui-go/internal/stream"
	"github.com/dmundt/a2ui-go/mcp"
	"github.com/dmundt/a2ui-go/renderer"
)

func main() {
	templateDir, err := resolveDir("renderer", "templates")
	if err != nil {
		log.Fatalf("template registry: %v", err)
	}

	reg, err := renderer.NewRegistry(templateDir)
	if err != nil {
		log.Fatalf("template registry: %v", err)
	}

	uiDir := "github.com/dmundt/a2ui-go/internal/ui"
	if resolvedUI, err := resolveDir("github.com/dmundt/a2ui-go/internal/ui"); err == nil {
		uiDir = resolvedUI
	}
	log.Printf("resolved uiDir: %s", uiDir)

	r := renderer.New(reg)
	pageStore := store.NewPageStore()
	broker := stream.NewBroker()
	eng := engine.New(r, reg, pageStore, broker, uiDir)
	mcpHandlers := mcp.NewHandlers(eng, reg)

	mux := http.NewServeMux()
	engine.RegisterHTTPHandlers(mux, eng)

	staticDir := "static"
	if resolvedStatic, serr := resolveDir("static"); serr == nil {
		staticDir = resolvedStatic
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	server := &http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if shouldStartMCPStdio() {
		go func() {
			if err := mcp.StartStdio(ctx, mcpHandlers); err != nil {
				log.Printf("mcp stdio exited: %v", err)
			}
		}()
	}

	go func() {
		log.Printf("http server listening on %s (templates=%s static=%s)", server.Addr, templateDir, staticDir)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http listen: %v", err)
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func resolveDir(parts ...string) (string, error) {
	candidates := make([]string, 0, 8)
	rel := filepath.Join(parts...)
	candidates = append(candidates, rel)
	candidates = append(candidates, filepath.Join("..", rel))
	candidates = append(candidates, filepath.Join("github.com/dmundt/a2ui-go", rel))

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, rel))
		candidates = append(candidates, filepath.Join(exeDir, "..", rel))
		candidates = append(candidates, filepath.Join(exeDir, "github.com/dmundt/a2ui-go", rel))
	}

	for _, c := range candidates {
		if isDir(c) {
			abs, aerr := filepath.Abs(c)
			if aerr != nil {
				return c, nil
			}
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find directory %q from candidates: %v", rel, candidates)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func shouldStartMCPStdio() bool {
	if os.Getenv("ENABLE_MCP_STDIO") == "1" {
		return true
	}
	for _, arg := range os.Args[1:] {
		if arg == "--mcp-stdio" {
			return true
		}
	}
	return isPiped(os.Stdin) && isPiped(os.Stdout)
}

func isPiped(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}
