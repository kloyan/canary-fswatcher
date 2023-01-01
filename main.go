package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

var backoffPolicy = []time.Duration{0, 250, 500, 1_000}

type config struct {
	watcher  *fsnotify.Watcher
	tokenUrl string
}

func main() {
	var path, tokenUrl string
	flag.StringVar(&path, "path", "/tmp", "File or directory to monitor for changes")
	flag.StringVar(&tokenUrl, "token-url", "", "URL canary token to be triggered on events")
	flag.Parse()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Panicf("path %s does not exist", path)
	}

	if _, err := url.ParseRequestURI(tokenUrl); err != nil {
		log.Panicf("url %s is invalid: %v", tokenUrl, err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panicf("could not create watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		log.Panicf("could not monitor path: %v", err)
	}

	c := config{watcher: watcher, tokenUrl: tokenUrl}
	if err := c.startWatchLoop(watcher); err != nil {
		log.Panicf("watch loop failed: %v", err)
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
