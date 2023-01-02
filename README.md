# Canary FSWatcher

Canary FSWatcher is a CLI tool which monitors a file or directory and fires a [canarytokens.org](https://canarytokens.org/generate) URL webhook whenever the target is accessed.

## Why?

Mainly for educational purposes as there's already a similar tool - [Canaryfy](https://github.com/thinkst/canaryfy). However it does not work on all operating systems unlike Canary FSWatcher. Moreover, canaryfy relies on a DNS canary token which is unreliable due to DNS caching - the probability of missing events is quite high. I know that the TTL is quite low (3 seconds at the time of writing this) but 
they're not always respected. Some DNS resolvers impose minimum TTL (check https://00f.net/2019/11/03/stop-using-low-dns-ttls/).

## How?

Canary FSWatcher uses the [fsnotify](https://github.com/fsnotify/fsnotify) cross-platform Go library. It supports Windows, Linux, macOS and more.

Both the full path name and the operation are included in the token request as headers:
```
X-Canary-Path-Name: /tmp/my-dir/my-file
X-Canary-Path-Op: WRITE
```

## Usage

```
Usage of canary-fswatcher:
  -linger duration
    	Time to wait for new events to arrive before pinging the token url (default 1s)
  -path string
    	File or directory to monitor for changes (default "/tmp")
  -token-url string
    	Canary token url generated from canarytokens.org to be pinged on events
```

## Creating a Systemd Service

We can use systemd to ensure that the binary is automatically started on boot or failures. Here's an example service file which can be used for this exact purpose. Make sure to modify the `ExecStart` line:

* Set the correct path to the `canary-fswatcher` binary on your machine
* Set the path to the directory or file that must be monitored via `-path` flag
* Set the URL of the token generated from [canarytokens.org](https://canarytokens.org/generate) via `-token-url` flag

*canary-fswatcher-daemon.service*
```
# Systemd service unit file for the Canary FSWatcher daemon

[Unit]
Description=Canary FSWatcher
After=network.target
StartLimitIntervalSec=0

[Service]
Restart=always
RestartSec=3
ExecStart=/usr/local/bin/canary-fswatcher -path <path> -token-url <url>
```

Create the file in `/etc/systemd/system/` and execute the following:

```sh
# Start the service
systemctl start canary-fswatcher-daemon

# Start the service automatically on boot
systemctl enable canary-fswatcher-daemon

# Check the service status
systemctl status canary-fswatcher-daemon
```
