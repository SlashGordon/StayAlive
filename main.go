/*
MIT License - SlashGordon

Permission is granted to use, copy, modify, and/or distribute this software for any purpose with or without fee, subject to the inclusion of the above copyright notice and this permission notice in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	opts, err := loadOptions(os.Args[1:], os.LookupEnv, os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "stay-alive:", err)
		os.Exit(2)
	}

	out := io.Discard
	if opts.Verbose {
		out = os.Stderr
	}
	logger := log.New(out, "", log.Ltime)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Printf("waiting for %s of inactivity", opts.IdleAfter)
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	NewJiggler(opts.Config, robotMouse{}, logger, rng).Run(ctx)
}
