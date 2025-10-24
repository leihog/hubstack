# HubStack

A beautiful, responsive web dashboard for your services built with Go.

## Features

- Modern, responsive grid layout that adapts to screen size
- Mobile-friendly design
- Fast and lightweight
- Simple configuration via YAML file
- Hot-reload: Edit templates and config without recompiling

## Installation

1. Make sure you have Go installed (1.21 or later)
2. Clone this repository
3. Install dependencies:

```bash
go mod download
```

## Building

### Using Make (recommended):

```bash
# Build for current platform
make build

# Build for Linux (amd64)
make build-linux

# Build for Linux (arm64) - useful for Raspberry Pi
make build-linux-arm

# Clean build artifacts
make clean

# Show all available targets
make help
```

Binaries are output to the `bin/` directory.

### Manual build:

```bash
# Build for current platform
go build -o bin/hubstack

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o bin/hubstack-linux-amd64
```

## Usage

### Local Development

**Run with default port (8080):**

```bash
# Using make
make run

# Or run the binary
./bin/hubstack

# Or run directly with go
go run ./cmd/hubstack
```

**Run with custom port:**

```bash
# Using make
make run-port PORT=3000

# Or run the binary
./bin/hubstack -port 3000

# Or run directly with go
go run ./cmd/hubstack -port 3000
```

### Docker Deployment

**Test locally with Docker:**

```bash
# Build and run in Docker on port 80
make docker-local

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

**Deploy to remote host:**

```bash
# Deploy to remote Linux host (requires SSH access and Docker installed)
make deploy-docker HOST=192.168.1.50

# With custom user and path
make deploy-docker HOST=192.168.1.50 USER=myuser DEPLOY_PATH=/home/myuser/hubstack

# View logs from remote host
make remote-logs HOST=192.168.1.50

# Restart remote container (useful after editing services.yml)
make remote-restart HOST=192.168.1.50

# Stop remote container
make remote-stop HOST=192.168.1.50
```

**What happens during deployment:**
1. Builds the Linux binary
2. Copies all necessary files to the remote host via rsync
3. Builds the Docker image on the remote host
4. Starts the container with docker-compose

**Requirements for remote deployment:**
- SSH access to the remote host
- Docker and docker-compose installed on the remote host
- rsync installed locally and on the remote host

**Updating configuration after deployment:**

Since `config.yml` and `templates/` are mounted as volumes, you can:

```bash
# Option 1: Edit locally and redeploy
vim config.yml
make deploy-docker HOST=192.168.1.50

# Option 2: Edit directly on the server and restart
ssh user@192.168.1.50
cd /opt/hubstack
vim config.yml
docker-compose restart

# Option 3: Use the make target
make restart-remote HOST=192.168.1.50
```

## Configuration

Edit `config.yml` to customize your dashboard and services:

```yaml
title: "HubStack Services"  # Optional: customize the page title
subtitle: "Your custom subtitle"  # Optional: customize the subtitle (random if not set)

services:
  - name: "Service Name"
    url: "http://example.com:8080"
    icon: "http://example.com:8080/favicon.ico"
    description: "Description of the service"
```

The `title` field is optional and defaults to "HubStack" if not specified. The `subtitle` field is also optional - if not specified, a random subtitle will be shown each time the page loads.
The Icon field in service configs is optional. When Icon is left empty, Hubstack will attempt to autodetect the icon on startup. 

Services can also be added through the webinterface.

## Customization

The HTML template is located in `templates/index.html` and can be customized to change the look and feel of the dashboard.
You can modify the following files without needing to recompile:

- **`config.yml`** - Change the page title, add, remove, or modify services. Changes take effect on restart.
- **`templates/index.html`** - Customize the HTML, CSS, and layout. Changes take effect on restart.

Simply restart the application to see your changes.

## Graceful Shutdown

The application handles termination signals gracefully:

- **SIGINT** (Ctrl+C) - Gracefully stops the server
- **SIGTERM** (from Docker, systemd, etc.) - Gracefully stops the server

When a signal is received, the server:
1. Stops accepting new connections
2. Waits up to 30 seconds for in-flight requests to complete
3. Closes all connections and exits cleanly

This ensures no requests are dropped during deployments or restarts.

## Requirements

- Go 1.21 or later
- gopkg.in/yaml.v3
