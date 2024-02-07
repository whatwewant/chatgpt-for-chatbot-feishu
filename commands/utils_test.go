package commands

import (
	"testing"
	"time"
)

func TestTimeDuration(t *testing.T) {
	// ms
	if (&TimeDuration{Duration: 100 * time.Millisecond}).String() != "100ms" {
		t.Fatalf("expected 100ms, but got %s", (&TimeDuration{Duration: 100 * time.Millisecond}).String())
	}

	// s
	if (&TimeDuration{Duration: 2*time.Second + 123*time.Millisecond}).String() != "2.12s" {
		t.Fatalf("expected 2.12s, but got %s", (&TimeDuration{Duration: 2 * time.Second}).String())
	}

	// m
	if (&TimeDuration{Duration: 2*time.Minute + 12*time.Second}).String() != "2m 12s" {
		t.Fatalf("expected 2m 12s, but got %s", (&TimeDuration{Duration: 2 * time.Minute}).String())
	}
}
