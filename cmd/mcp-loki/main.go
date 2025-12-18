package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/lexfrei/mcp-loki/internal/config"
	"github.com/lexfrei/mcp-loki/internal/loki"
	"github.com/lexfrei/mcp-loki/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName        = "mcp-loki"
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// version is set via ldflags at build time.
var version = "dev"

func main() {
	err := run()
	if err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	lokiClient := loki.NewClient(
		cfg.LokiURL,
		cfg.Username,
		cfg.Password,
		cfg.Token,
		cfg.OrgID,
	)

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    serverName,
			Version: version,
		},
		nil,
	)

	registerTools(server, lokiClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	if cfg.HTTPEnabled() {
		go runHTTPServer(ctx, server, cfg.HTTPPort)
	}

	err := server.Run(ctx, &mcp.StdioTransport{})
	if err != nil && ctx.Err() == nil {
		return errors.Wrap(err, "server run failed")
	}

	return nil
}

func registerTools(server *mcp.Server, client *loki.Client) {
	mcp.AddTool(server, tools.QueryTool(), tools.NewQueryHandler(client))
	mcp.AddTool(server, tools.LabelsTool(), tools.NewLabelsHandler(client))
	mcp.AddTool(server, tools.SeriesTool(), tools.NewSeriesHandler(client))
	mcp.AddTool(server, tools.StatsTool(), tools.NewStatsHandler(client))
}

func runHTTPServer(ctx context.Context, server *mcp.Server, port string) {
	handler := mcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcp.Server {
			return server
		},
		nil,
	)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, shutdownTimeout)
		defer shutdownCancel()

		err := httpServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("HTTP server listening on :%s", port)

	err := httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server error: %v", err)
	}
}
