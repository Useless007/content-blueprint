package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"ContentBlueprint/internal/companionmcp"
	"ContentBlueprint/internal/facebookcompanion"
	"ContentBlueprint/internal/workbench"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := facebookcompanion.NewStore("")
	if err != nil {
		fail(err)
	}

	arguments := os.Args[1:]
	if len(arguments) > 0 && (arguments[0] == "--version" || arguments[0] == "version") {
		fmt.Println(facebookcompanion.MCPServerVersion)
		return
	}
	if len(arguments) > 0 && strings.HasPrefix(arguments[0], "chrome-extension://") {
		if err := facebookcompanion.ValidateNativeOrigin(arguments[0]); err != nil {
			fail(err)
		}
		if err := facebookcompanion.RunNativeHost(ctx, os.Stdin, os.Stdout, store); err != nil {
			fail(err)
		}
		return
	}
	if len(arguments) == 0 || arguments[0] == "mcp" {
		growthStore, err := workbench.NewStore("")
		if err != nil {
			fail(fmt.Errorf("prepare Growth Workbench storage: %w", err))
		}
		if err := companionmcp.Run(ctx, store, growthStore); err != nil && ctx.Err() == nil {
			fail(err)
		}
		return
	}

	fail(fmt.Errorf("unsupported mode; run without arguments (or with 'mcp') for the MCP server"))
}

func fail(err error) {
	// stdout is reserved for MCP or Chrome Native Messaging protocol frames.
	_, _ = fmt.Fprintf(os.Stderr, "content-blueprint-companion: %v\n", err)
	os.Exit(1)
}
