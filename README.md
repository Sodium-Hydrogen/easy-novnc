# easy-novnc
An easy way to run a [noVNC](https://github.com/novnc/noVNC) instance and proxy with a single binary.

## Features
- Modifications to noVNC to make mobile device use even better.
- Clean start page.
- CIDR whitelist/blacklist.
- Optionally allow connections to arbitrary hosts (and ports).
- Ensures the target port is a VNC server to prevent tunneling to unauthorized ports.
- Can be configured using environment variables or command line flags (but works out-of-the box).
- IPv6 support.
- Single binary, no dependencies.
- Easy setup.
- Optional [client](./wstcp) for local TCP connections tunneled through WebSockets.

## Installation
- You can build your own binaries with go 1.20 or newer using `go get github.com/sodium-hydrogen/easy-novnc` or by cloning this repo and running `go build`.

## Usage
```
Usage: ./easy-novnc [options]

Options:
  -a, --addr string              The address to listen on (env NOVNC_ADDR) (default ":8080")
  -H, --arbitrary-hosts          Allow connection to other hosts (env NOVNC_ARBITRARY_HOSTS)
  -P, --arbitrary-ports          Allow connections to arbitrary ports (requires arbitrary-hosts) (env NOVNC_ARBITRARY_PORTS)
  -u, --basic-ui                 Hide connection options from the main screen (env NOVNC_BASIC_UI)
  -C, --cidr-blacklist strings   CIDR blacklist for when arbitrary hosts are enabled (comma separated) (conflicts with whitelist) (env NOVNC_CIDR_BLACKLIST)
  -c, --cidr-whitelist strings   CIDR whitelist for when arbitrary hosts are enabled (comma separated) (conflicts with blacklist) (env NOVNC_CIDR_WHITELIST)
      --default-view-only        Use view-only by default (env NOVNC_DEFAULT_VIEW_ONLY)
      --help                     Show this help text
  -h, --host string              The host/ip to connect to by default (env NOVNC_HOST) (default "localhost")
  -m, --min-height uint16        The minimum height of the noVNC window (env NOVNC_MIN_HEIGHT)
  -w, --min-width uint16         The minimum width of the noVNC window (env NOVNC_MIN_WIDTH)
      --no-url-password          Do not allow password in URL params (env NOVNC_NO_URL_PASSWORD)
      --novnc-params strings     Extra URL params for noVNC (advanced) (comma separated key-value pairs) (e.g. resize=remote) (env NOVNC_PARAMS)
  -p, --port uint16              The port to connect to by default (env NOVNC_PORT) (default 5900)
  -v, --verbose                  Show extra log info (env NOVNC_VERBOSE)
```
