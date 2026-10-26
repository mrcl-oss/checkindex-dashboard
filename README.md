# Index Usage Dashboard

A local Go web application that displays the index-usage reports returned by the public `checkIndex/report` API.

## Prerequisites

- Go 1.27.3 or newer.
- Complete the existing checkIndex setup before starting the dashboard. At runtime, the library reads `$HOME/.checkIndex/config.json` and may create a starter configuration if the file is absent. Configure MongoDB access through checkIndex's supported setup flow; do not put credentials in this application's environment variables or source code.
- MongoDB connectivity for the configured profiles when the dashboard is running. Starting the application begins report collection.

## Quick Start with Docker (Recommended)

You don't need to clone the repository or install Go. Pull and run the pre-built image directly from Docker Hub:

```sh
docker run -d \
  --name checkindex-dashboard \
  --restart unless-stopped \
  -p 6767:6767 \
  -v "${HOME}/.checkIndex:/home/appuser/.checkIndex" \
  --add-host host.docker.internal:host-gateway \
  mrcl29/checkindex-dashboard:latest
```

Open <http://localhost:6767> in your browser.

- `-v "${HOME}/.checkIndex:/home/appuser/.checkIndex"`: Mounts your local checkIndex configuration so profile settings persist across restarts.
- `--add-host host.docker.internal:host-gateway`: Allows the container to connect to local MongoDB instances using `host.docker.internal`.
- Open the in-dashboard **⚙️ Configuration** panel to configure database profiles directly from your browser without manually editing files.

To view logs or stop the container:

```sh
docker logs -f checkindex-dashboard
docker stop checkindex-dashboard
```

## Run with Docker Compose

If you prefer Docker Compose:

```sh
docker compose up -d
```

Open <http://localhost:6767> (or your custom port defined in `.env`).

To stop:

```sh
docker compose down
```

## Run Locally from Source

### Prerequisites

- Go 1.27.3 or newer.
- MongoDB connectivity for configured profiles.

From this directory:

```sh
go run ./cmd/checkindex-dashboard
```

The default listener is `127.0.0.1:6767`; open <http://127.0.0.1:6767>. `Ctrl+C` and `SIGTERM` trigger graceful HTTP shutdown.

## Configuration with .env

You can configure the application using a `.env` file in the root directory (see `.env.example`):

```sh
cp .env.example .env
```

To change the port to, for example, 80:

```env
PORT=80
```

Both local Go runs (`go run`) and Docker Compose automatically load the `.env` file.

## In-UI Configuration

Click the **⚙️ Configuration** button in the top sticky bar to view and customize `~/.checkIndex/config.json` directly from your browser. This web interface translates the CLI configuration menu (`checkcluster config` / `checkindex config`):
- **Shared settings**: adjust database name (`dbName`), auth source (`authSource`), and global host mappings.
- **Cluster profiles**: add, modify, or remove cluster profiles (`initialUri`, `username`, `password`, per-profile host mappings). Existing passwords are preserved on disk if left blank during edits.
- **Raw JSON toggle**: review or edit the configuration in raw JSON format with live bidirectional synchronization.
- **Instant refresh**: saving persists changes safely via atomic writes and immediately triggers report collection.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `6767` | Port number for the HTTP server. Overrides the port in `DefaultListenAddress`. |
| `CHECKINDEX_DASHBOARD_LISTEN_ADDR` | `127.0.0.1:6767` | Full HTTP listener in `host:port` form. Takes precedence over `PORT`. |
| `CHECKINDEX_DASHBOARD_REFRESH_INTERVAL` | `60s` | Positive Go duration controlling background collection and page refresh, for example `30s` or `2m`. |

For example, to collect every 30 seconds while keeping the default local listener:

```sh
CHECKINDEX_DASHBOARD_REFRESH_INTERVAL=30s go run ./cmd/checkindex-dashboard
```

## Safe verification

The test suite uses injected synthetic collectors and `httptest`; `go vet` and `go build` only inspect or compile code. None of these checks start the dashboard, read the checkIndex configuration, or connect to MongoDB:

```sh
go test ./...
go vet ./...
go build ./cmd/checkindex-dashboard
```

Do not use the run commands above as verification checks: starting the dashboard invokes report collection for the locally configured profiles.
