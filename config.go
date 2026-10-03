package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"time"
)

// envPrefix is prepended to every environment variable name.
const envPrefix = "STAYALIVE_"

// Options is everything the command line and environment can set.
type Options struct {
	Config
	Verbose bool
}

// loadOptions reads settings from defaults, then environment variables, then
// flags. A flag wins over its environment variable.
func loadOptions(args []string, lookupEnv func(string) (string, bool), output io.Writer) (Options, error) {
	opts := Options{
		Config: Config{
			IdleAfter: 30 * time.Second,
			Interval:  30 * time.Second,
			Radius:    100,
			Poll:      time.Second,
		},
	}

	env := envReader{lookup: lookupEnv}
	opts.IdleAfter = env.duration("IDLE", opts.IdleAfter)
	opts.Interval = env.duration("INTERVAL", opts.Interval)
	opts.Radius = env.int("RADIUS", opts.Radius)
	opts.Poll = env.duration("POLL", opts.Poll)
	opts.Verbose = env.bool("VERBOSE", opts.Verbose)
	if env.err != nil {
		return opts, env.err
	}

	fs := flag.NewFlagSet("stay-alive", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.DurationVar(&opts.IdleAfter, "idle", opts.IdleAfter, "start moving the cursor after this much time without input (env "+envPrefix+"IDLE)")
	fs.DurationVar(&opts.Interval, "interval", opts.Interval, "longest pause between two moves while you stay idle (env "+envPrefix+"INTERVAL)")
	fs.IntVar(&opts.Radius, "radius", opts.Radius, "maximum distance of a move in pixels (env "+envPrefix+"RADIUS)")
	fs.DurationVar(&opts.Poll, "poll", opts.Poll, "how often to check for input (env "+envPrefix+"POLL)")
	fs.BoolVar(&opts.Verbose, "v", opts.Verbose, "log moves and detected activity (env "+envPrefix+"VERBOSE)")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	return opts, opts.validate()
}

func (c Config) validate() error {
	switch {
	case c.IdleAfter <= 0:
		return errors.New("idle time must be positive")
	case c.Interval <= 0:
		return errors.New("interval must be positive")
	case c.Radius <= 0:
		return errors.New("radius must be positive")
	case c.Poll <= 0:
		return errors.New("poll interval must be positive")
	}
	return nil
}

// envReader parses prefixed environment variables and collects all errors.
type envReader struct {
	lookup func(string) (string, bool)
	err    error
}

func (e *envReader) get(name string) (string, bool) {
	v, ok := e.lookup(envPrefix + name)
	return v, ok && v != ""
}

func (e *envReader) fail(name, value string, err error) {
	e.err = errors.Join(e.err, fmt.Errorf("%s%s=%q: %w", envPrefix, name, value, err))
}

// duration accepts Go durations like "90s" or "2m", or a plain number of seconds.
func (e *envReader) duration(name string, def time.Duration) time.Duration {
	v, ok := e.get(name)
	if !ok {
		return def
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		e.fail(name, v, errors.New("want a duration like 90s or 2m"))
		return def
	}
	return d
}

func (e *envReader) int(name string, def int) int {
	v, ok := e.get(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		e.fail(name, v, errors.New("want a whole number"))
		return def
	}
	return n
}

func (e *envReader) bool(name string, def bool) bool {
	v, ok := e.get(name)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		e.fail(name, v, errors.New("want true or false"))
		return def
	}
	return b
}
