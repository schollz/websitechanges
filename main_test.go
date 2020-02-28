package main

import (
	"testing"

	log "github.com/schollz/logger"
)

func TestRun(t *testing.T) {
	log.SetLevel("trace")
	// err := run("https://rwtxt.com/public/testtest", "#rendered")
	// if err != nil {
	// 	t.Errorf("errors: %s", err.Error())
	// }
	err := Watch([]Watcher{
		Watcher{"https://www.nytimes.com", "span.balancedHeadline"},
	})
	if err != nil {
		t.Errorf("errors: %s", err.Error())
	}
}
