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

	flag.StringVar(&dir, "dir", "/tmp", "Directory to monitor for changes")
	flag.StringVar(&token, "token", "", "URL token to be triggered")
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

	c.watchLoop(watcher, token)
}

func (c *config) watchLoop(watcher *fsnotify.Watcher, token string) error {
	for {
		select {
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return err
		case e, ok := <-watcher.Events:
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

func (c *config) pingWithRetry(e fsnotify.Event) {
	for _, b := range backoffPolicy {
		time.Sleep(time.Millisecond * b)

		err := c.ping(e)
		if err == nil {
			log.Printf("successfully pinged canary token for %s -> %s", e.Op, e.Name)
			return
		}

		log.Printf("failed to ping canary token: %v", err)
	}

	log.Printf("skipped canary ping due to earlier failure")
}

func (c *config) ping(e fsnotify.Event) error {
	req, err := http.NewRequest("HEAD", c.tokenUrl, nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-Canary-Path-Name", e.Name)
	req.Header.Add("X-Canary-Path-Op", e.Op.String())

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
