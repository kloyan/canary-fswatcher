package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
)

var backoffPolicy = []time.Duration{0, 250, 500, 1_000}

type config struct {
	watcher  *fsnotify.Watcher
	tokenUrl string
}

func main() {
	var dir, token string

	flag.StringVar(&dir, "path", "/tmp", "File or directory to monitor for changes")
	flag.StringVar(&token, "token-url", "", "URL token to be triggered")
	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("could not create watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatalf("could not add watch dir: %v", err)
	}

	c := config{watcher: watcher, tokenUrl: token}
	if err := c.startWatchLoop(watcher); err != nil {
		log.Fatalf("watch loop failure: %v", err)
	}
}

func (c *config) startWatchLoop(watcher *fsnotify.Watcher) error {
	for {
		select {
		case err, ok := <-watcher.Errors:
			// Indicates that the Errors channel was closed
			if !ok {
				return nil
			}

			return err
		case e, ok := <-watcher.Events:
			// Indicates that the Events channel was closed
			if !ok {
				return nil
			}

			// todo: implement dedup
			if e.Has(fsnotify.Chmod) {
				continue
			}

			c.pingWithRetry(e)
		}
	}
}

func (c *config) pingWithRetry(event fsnotify.Event) {
	req, _ := http.NewRequest("HEAD", c.tokenUrl, nil)
	req.Header.Add("X-Canary-Path-Name", event.Name)
	req.Header.Add("X-Canary-Path-Op", event.Op.String())

	for i, b := range backoffPolicy {
		time.Sleep(b * time.Millisecond)

		err := c.ping(req)
		if err == nil {
			log.Printf("ping successful for %s", event.Name)
			return
		}

		log.Printf("ping failed on attempt %d: %v", i, err)
	}

	log.Printf("ping skipped due to earlier failure")
}

func (c *config) ping(req *http.Request) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	return nil
}
