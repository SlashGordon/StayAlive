package main

import (
	"io"
	"testing"
	"time"
)

func envMap(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestLoadOptionsDefaults(t *testing.T) {
	opts, err := loadOptions(nil, envMap(nil), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.IdleAfter != 30*time.Second || opts.Interval != 30*time.Second || opts.Radius != 100 {
		t.Errorf("unexpected defaults: %+v", opts)
	}
}

func TestLoadOptionsFromEnv(t *testing.T) {
	opts, err := loadOptions(nil, envMap(map[string]string{
		"STAYALIVE_IDLE":     "2m",
		"STAYALIVE_INTERVAL": "45",
		"STAYALIVE_RADIUS":   "250",
		"STAYALIVE_VERBOSE":  "true",
	}), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.IdleAfter != 2*time.Minute {
		t.Errorf("IdleAfter = %s, want 2m", opts.IdleAfter)
	}
	if opts.Interval != 45*time.Second {
		t.Errorf("Interval = %s, want 45s (plain number means seconds)", opts.Interval)
	}
	if opts.Radius != 250 || !opts.Verbose {
		t.Errorf("Radius = %d, Verbose = %v", opts.Radius, opts.Verbose)
	}
}

func TestFlagWinsOverEnv(t *testing.T) {
	opts, err := loadOptions([]string{"-idle", "10s"}, envMap(map[string]string{"STAYALIVE_IDLE": "2m"}), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.IdleAfter != 10*time.Second {
		t.Errorf("IdleAfter = %s, want flag value 10s", opts.IdleAfter)
	}
}

func TestLoadOptionsRejectsBadValues(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"bad duration": {"STAYALIVE_IDLE": "soon"},
		"bad int":      {"STAYALIVE_RADIUS": "big"},
		"bad bool":     {"STAYALIVE_VERBOSE": "maybe"},
		"zero idle":    {"STAYALIVE_IDLE": "0"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := loadOptions(nil, envMap(env), io.Discard); err == nil {
				t.Error("want error")
			}
		})
	}
}
