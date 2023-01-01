package main

import (
	"flag"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/exp/slog"
)

type config struct {
	watchDir string
	tokenUrl string
}

func main() {
	var dir, token string

	flag.StringVar(&dir, "dir", "/tmp", "Directory to monitor for changes")
	flag.StringVar(&token, "token", "", "URL token to be triggered")
	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("could not create watcher", err)
		panic(err)
	}
	defer watcher.Close()

	go watchLoop(watcher, token)

	err = watcher.Add(dir)
	if err != nil {
		slog.Error("could not create watcher", err)
		panic(err)
	}

	// Block main goroutine forever.
	select {}
}

func watchLoop(watcher *fsnotify.Watcher, token string) {
	for {
		select {
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			slog.Error("encountered watch error", err)
		case e, ok := <-watcher.Events:
			if !ok {
				return
			}

			// todo: implement dedup
			if e.Has(fsnotify.Chmod) {
				continue
			}

			pingToken(e, token)
		}
	}
}

// todo: implement retry
func pingToken(e fsnotify.Event, url string) {
	resp, err := http.Head(url)
	if err == nil {
		defer resp.Body.Close()
		slog.Info("pinged canary token for", e)
		return
	}

	slog.Error("failed to ping canary token", err)
}
