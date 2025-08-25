# SiteWatch

A lightweight Go service for monitoring dual-line network connectivity with real-time dashboard, detailed logging, Prometheus metrics export and Serverguard integration.

## Table of Contents

- [Description](#description)
- [Quickstart](#quickstart)
- [Usage](#usage)
- [Authentication](#authentication)
- [API Reference](#api-reference)
- [Web Dashboard](#web-dashboard)
- [Monitoring Integration](#monitoring-integration)
- [Environment Variables](#environment-variables)
- [Configuration](#configuration)
- [Development](#development)
- [Production Deployment](#production-deployment)

## Description

SiteWatch continuously monitors network connectivity at multiple sites by pinging primary and optionally secondary IP addresses. It provides:

- **Flexible monitoring** - Supports both dual-line and single-line configurations
- **Dual-line monitoring** - Both IPs must be reachable for site success
- **Single-line monitoring** - Only primary IP required for site success
- **SLA monitoring** - Provider-specific SLA targets with visual indicators and metrics
- **Real-time web dashboard** - Live status overview with interactive charts and logs
- **Interactive charts** - Latency trends, uptime history, and SLA compliance tracking
- **Detailed ping logging** - Ring-buffer storage with filtering and live updates
- **Prometheus metrics** - Native metrics export including SLA targets for monitoring
- **Serverguard compatibility** - HTTP endpoints returning "success"/"failure"
- **REST API** - JSON endpoints for site status, details, and logs
- **Token-based Authentication** - Secure API access with granular permissions and UI session management
- **Container-ready** - Docker support for development and production

**Key Technologies**: Go 1.23, Fiber v2, HTMX, TailwindCSS, Prometheus client, Docker

## Quickstart

### Prerequisites

- Go 1.23+ (for local development)
- Docker (for containerized deployment)

### Get Started

1. **Clone and navigate to the project**
   ```bash
   git clone git@github.com:4gh0rn/sitewatch.git
   cd sitewatch
   ```

2. **Start development environment**
   ```bash
   make dev
   ```

3. **Access the service**
   - Web Dashboard: http://localhost:8080
   - Health check: http://localhost:8080/health
   - API: http://localhost:8080/api/sites

## Usage

### Web Dashboard

The service includes a modern web dashboard with two main sections:

**Dashboard Tab:**
- Real-time site status overview (dual-line and single-line sites)
- Live metrics and statistics
- Quick access to API endpoints
- Auto-refresh every 5 seconds

**Logs Tab:**
- Detailed ping log history (last 1000 entries)
- Real-time filtering by site, status, and limit
- Live updates with auto-refresh toggle
- Professional table view with latency details

### Docker Development

Start complete development environment with live-reload:
```bash
make dev
```

### Docker Production

Build and run production container:
```bash
make docker-build
make docker-run
```

### Configuration Files

Create configuration files from examples:
```bash
cp configs/config.example.yaml configs/config.yaml
cp configs/sites.example.yaml configs/sites.yaml
```

Edit `configs/sites.yaml` to configure your monitoring targets:

**Dual-Line Configuration with SLA monitoring:**
```yaml
sites:
  - id: "site-001"
    name: "Main Office Berlin"
    location: "Berlin, DE"
    primary_ip: "192.168.1.1"
    secondary_ip: "192.168.1.2"  # Both IPs required
    primary_provider: "Telekom"   # Optional: Provider name
    secondary_provider: "Vodafone" # Optional: Provider name
    interval: 30  # seconds
    enabled: true
    sla:
      primary:
        uptime: 99.9        # Primary provider SLA target (%)
        max_latency: 50     # Max expected latency (ms)
        restoration: 240    # Max restoration time (minutes)
      secondary:
        uptime: 99.5        # Secondary provider SLA target (%)
        max_latency: 80     # Max expected latency (ms)
        restoration: 480    # Max restoration time (minutes)
      combined:
        uptime: 99.95       # Combined SLA with redundancy
```

**Single-Line Configuration with SLA:**
```yaml
  - id: "site-002"
    name: "Remote Branch"
    location: "Munich, DE"
    primary_ip: "remote.example.com"
    primary_provider: "Deutsche Glasfaser"  # Optional
    # secondary_ip omitted = single-line mode
    interval: 60  # seconds
    enabled: true
    sla:
      primary:
        uptime: 99.8        # Single provider SLA target
        max_latency: 30     # Expected latency
        restoration: 360    # Restoration time (6h)
```

## Authentication

SiteWatch supports comprehensive token-based authentication for secure API access and UI session management:

- **UI Authentication**: Automatic cookie-based sessions for dashboard access
- **API Authentication**: Bearer token authentication with granular permissions
- **Token Management**: CLI tools for generating and managing tokens
- **Backwards Compatible**: Authentication disabled by default for easy setup

### Permission Levels

### Detailed Permission Matrix

| Endpoint | Method | `metrics` | `read` | `test` | `admin` | Description |
|----------|--------|-----------|--------|--------|---------|-------------|
| `/health` | GET | Yes | Yes | Yes | Yes | Service health check |
| `/metrics` | GET | Yes | No | No | Yes | Prometheus metrics export |
| `/api/sites` | GET | No | Yes | Yes | Yes | All sites status overview |
| `/api/sites/{id}/status` | GET | No | Yes | Yes | Yes | Serverguard compatible status |
| `/api/sites/{id}/details` | GET | No | Yes | Yes | Yes | Detailed site information |
| `/api/logs` | GET | No | Yes | Yes | Yes | Ping logs with filtering |
| `/api/sites/{id}/test` | POST | No | No | Yes | Yes | Manual connection test (API) |
| `/ui/test/{id}` | POST | No | No | No | No | Manual connection test (UI) |
| `/ui/*` | ALL | No | No | No | No | UI routes use cookie auth |
| Future admin endpoints | ALL | No | No | No | Yes | Administrative functions |

**Permission Summary:**
- **`metrics`**: Only metrics access - perfect for Telegraf/monitoring tools that need Prometheus data
- **`read`**: Full read access to all API endpoints - ideal for Serverguard and status monitoring
- **`test`**: Read access plus manual testing capabilities - perfect for development and debugging
- **`admin`**: Complete system access including all permissions and future management features

**Use Cases:**
- **Telegraf**: `metrics` permission → only gets `/metrics` endpoint
- **Serverguard**: `read` permission → gets all site status and details
- **Developer**: `test` permission → can run manual tests and access all read endpoints
- **System Admin**: `admin` permission → full access to everything

### Token Generation

Generate secure API tokens using the CLI tool:

```bash
# Generate a new API token
go run tools/token-gen/main.go generate -name "Telegraf Monitoring" -permissions metrics

# Generate UI secret for session management
go run tools/token-gen/main.go ui-secret

# List available commands
go run tools/token-gen/main.go --help
```

### Configuration

Enable authentication in `configs/config.yaml`:

```yaml
auth:
  enabled: true
  ui:
    secret: "your-generated-ui-secret-here"      # Generate with: go run tools/token-gen/main.go ui-secret
    session_name: "sitewatch_session"           # Cookie name for UI sessions
    expires_hours: 24                           # Session expiration (hours)
  api:
    tokens:
      - token: "sw_telegraf_a1b2c3d4e5f6..."    # Generate with: go run tools/token-gen/main.go generate
        name: "Telegraf Monitoring"
        permissions: ["metrics"]                 # Available: metrics, read, test, admin
        expires: "2025-12-31"                   # Optional expiration (YYYY-MM-DD)
      - token: "sw_admin_f6e5d4c3b2a1..."
        name: "Admin Access"
        permissions: ["admin"]
        # expires: null                          # Never expires
```

### API Usage with Authentication

**Without authentication (returns 401):**
```bash
curl http://localhost:8080/api/sites
# {"error":"Authentication required"}
```

**With Bearer token:**
```bash
curl -H "Authorization: Bearer sw_telegraf_a1b2c3d4e5f6..." http://localhost:8080/api/sites
# Returns site data
```

**Metrics endpoint with token:**
```bash
curl -H "Authorization: Bearer sw_telegraf_a1b2c3d4e5f6..." http://localhost:8080/metrics
# Returns Prometheus metrics
```

### UI Dashboard Access

The web dashboard automatically handles authentication via secure HttpOnly cookies:

1. **First visit to `/`**: Cookie is automatically set with 24h expiration
2. **Subsequent requests**: Cookie validates all UI access (`/ui/*`)
3. **Session expiry**: After 24h, revisit `/` to get new cookie
4. **HTMX requests**: Work seamlessly with existing cookies

No manual login required - just visit the dashboard and authentication is handled automatically.

### Integration Examples

**Telegraf with metrics permission:**
```toml
[[inputs.prometheus]]
  urls = ["http://sitewatch:8080/metrics"]
  metric_version = 2
  [inputs.prometheus.headers]
    Authorization = "Bearer sw_telegraf_a1b2c3d4e5f6..."  # metrics permission
```

**Serverguard with read permission:**
```yaml
checks:
  - name: "Berlin Site Health"
    type: "web"
    url: "http://sitewatch:8080/api/sites/site-001/status"
    headers:
      Authorization: "Bearer sw_monitor_x1y2z3..."  # read permission
    expect: "OK"
    interval: 60s
    timeout: 10s
```

## API Reference

### Core Endpoints

| Endpoint | Method | Description | Response |
|----------|--------|-------------|----------|
| `/` | GET | Web dashboard (main UI) | HTML |
| `/health` | GET | Service health check | JSON status |
| `/api/sites` | GET | All sites with status overview | JSON array |
| `/api/sites/{id}/status` | GET | Serverguard compatible status | `OK`/`FAILURE` |
| `/api/sites/{id}/details` | GET | Detailed site information | JSON object |
| `/api/logs` | GET | Ping logs with filtering | JSON array |
| `/metrics` | GET | Prometheus format metrics | Plain text |

### Logs API

**Get filtered logs** (`/api/logs`):
```bash
# All logs (last 100)
curl "http://localhost:8080/api/logs"

# Filter by site
curl "http://localhost:8080/api/logs?site=site1&limit=50"

# Filter by success status
curl "http://localhost:8080/api/logs?success=false&limit=20"
```

**Response format:**
```json
{
  "logs": [
    {
      "id": 1234,
      "timestamp": "2024-01-15T10:30:00Z",
      "site_id": "site1",
      "site_name": "Google DNS",
      "target": "primary",
      "ip": "8.8.8.8",
      "success": true,
      "latency": 15.3,
      "error": ""
    }
  ],
  "total": 1,
  "filters": {
    "site": "site1",
    "success": "",
    "limit": 100
  }
}
```

### Example Responses

**Site Status Overview** (`/api/sites`):
```json
{
  "sites": [
    {
      "id": "site-001",
      "name": "Main Office Berlin",
      "location": "Berlin, DE",
      "primary_ip": "192.168.1.1",
      "secondary_ip": "192.168.1.2",
      "interval": 30,
      "enabled": true,
      "status": {
        "site_id": "site-001",
        "primary_online": true,
        "secondary_online": true,
        "both_online": true,
        "primary_latency": 15.3,
        "secondary_latency": 14.8,
        "last_check": "2024-01-15T10:30:00Z"
      }
    }
  ],
  "total": 1,
  "timestamp": "2024-01-15T10:30:05Z"
}
```

**Serverguard Status** (`/api/sites/site-001/status`):
```
success  (HTTP 200)
failure  (HTTP 503)
```

## Web Dashboard

### Features

- **Tab Navigation**: Switch between Dashboard and Logs views
- **Live Updates**: Auto-refresh every 5 seconds (pausable)
- **Responsive Design**: Works on desktop and mobile
- **Professional Icons**: Clean SVG icons instead of emojis
- **Real-time Filtering**: Filter logs by site, status, limit
- **Interactive Modals**: Site details with click-through

### Dashboard Tab

- Site status overview with visual indicators
- Real-time metrics (total sites, online/offline counts)
- Quick action buttons for API endpoints
- Service uptime and check statistics

### Logs Tab

- Ring-buffer log storage (1000 entries max)
- Real-time ping result logging
- Advanced filtering options
- Auto-refresh toggle (5-second intervals)
- Detailed error messages and latency data

## Monitoring Integration

### Telegraf Configuration

Add to your `telegraf.conf`:
```toml
[[inputs.prometheus]]
  urls = ["http://sitewatch:8080/metrics"]
  metric_version = 2
```

### Serverguard Integration

Configure health checks in Serverguard:
```yaml
checks:
  - name: "Berlin Site Health"
    type: "web"
    url: "http://sitewatch:8080/api/sites/site-001/status"
    expect: "success"
    interval: 60s
    timeout: 10s
```

### Available Metrics

- `ping_checks_total{site_id, line_type, success}` - Total ping checks
- `ping_latency_histogram{site_id, line_type}` - Latency distribution
- `site_status{site_id, line_type}` - Line status (1=online, 0=offline)
- `site_both_lines_online{site_id}` - Combined status (1=both online)
- `site_info{site_id, name, location}` - Site metadata

## Environment Variables

SiteWatch supports configuration via environment variables, which take precedence over config file values. This is especially useful for Docker deployments and sensitive values like authentication secrets.

### Available Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| **Server** | | | |
| `SITEWATCH_SERVER_HOST` | Server bind address | `0.0.0.0` | `127.0.0.1` |
| `SITEWATCH_SERVER_PORT` | Server port | `8080` | `3000` |
| `SITEWATCH_SERVER_READ_TIMEOUT` | Request read timeout | `10s` | `30s` |
| `SITEWATCH_SERVER_WRITE_TIMEOUT` | Response write timeout | `10s` | `30s` |
| **Storage** | | | |
| `SITEWATCH_STORAGE_TYPE` | Storage backend | `memory` | `sqlite` |
| `SITEWATCH_STORAGE_SQLITE_PATH` | SQLite database path | `data/ping_monitor.db` | `/data/sitewatch.db` |
| `SITEWATCH_STORAGE_MAX_MEMORY_LOGS` | Max logs in memory | `1000` | `5000` |
| **Metrics** | | | |
| `SITEWATCH_METRICS_ENABLED` | Enable Prometheus metrics | `true` | `false` |
| `SITEWATCH_METRICS_PATH` | Metrics endpoint path | `/metrics` | `/prometheus` |
| **Authentication** | | | |
| `SITEWATCH_AUTH_ENABLED` | Enable authentication | `false` | `true` |
| `SITEWATCH_AUTH_UI_SECRET` | UI session secret | - | Generated secret |
| `SITEWATCH_AUTH_UI_EXPIRES_HOURS` | Session expiry hours | `24` | `72` |
| `SITEWATCH_AUTH_API_TOKEN` | Single API token | - | `sw_abc123...` |
| `SITEWATCH_AUTH_API_TOKEN_PERMISSIONS` | Token permissions | `read` | `metrics,read` |
| **Config Paths** | | | |
| `SITEWATCH_CONFIG_PATH` | Config file path | `configs/config.yaml` | `/etc/sitewatch/config.yaml` |
| `SITEWATCH_SITES_PATH` | Sites file path | `configs/sites.yaml` | `/etc/sitewatch/sites.yaml` |

### Docker Deployment with Environment Variables

1. **Create a `.env` file** (copy from `.env.example`):
```bash
cp .env.example .env
# Edit .env with your values
```

2. **Set authentication secrets**:
```bash
# Generate UI secret
echo "SITEWATCH_AUTH_UI_SECRET=$(go run tools/token-gen/main.go ui-secret | grep 'Secret:' | cut -d' ' -f2)" >> .env

# Generate API token
echo "SITEWATCH_AUTH_API_TOKEN=$(go run tools/token-gen/main.go generate | grep 'Token:' | cut -d' ' -f2)" >> .env
```

3. **Run with Docker Compose**:
```bash
# Production with .env file
docker-compose -f deployments/docker/docker-compose.prod.yml up -d

# Or override specific values
SITEWATCH_AUTH_ENABLED=true SITEWATCH_SERVER_PORT=9090 docker-compose up -d
```

### Priority Order

Configuration values are loaded in the following priority (highest to lowest):
1. **Environment variables** - Override everything
2. **YAML config files** - Base configuration
3. **Default values** - Built-in defaults

### Security Best Practices

- **Never commit `.env` files** to version control
- **Use environment variables** for secrets (UI secret, API tokens)
- **Use read-only mounts** for config files in production
- **Generate strong secrets** using the provided tools
- **Rotate tokens regularly** in production environments

## Configuration

### Site Configuration Modes

SiteWatch supports two monitoring modes per site:

#### Dual-Line Mode
- **Configuration**: Both `primary_ip` and `secondary_ip` specified
- **Success Criteria**: Both IPs must be reachable
- **Use Case**: Active/Active network setups, redundant connections
- **Status**: "Online" when both lines work, "Degraded" when one fails

#### Single-Line Mode  
- **Configuration**: Only `primary_ip` specified, `secondary_ip` omitted
- **Success Criteria**: Only primary IP must be reachable
- **Use Case**: Single internet connection, remote sites, simple setups
- **Status**: "Online" when primary works, "Offline" when it fails

#### IP Address vs Hostnames
- **IP Addresses**: Direct ping to IP (e.g., `192.168.1.1`)
- **Hostnames**: DNS resolution + ping (e.g., `site.company.com`)
- **Mixed**: You can combine both in the same configuration

### configs/config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 10s
  write_timeout: 10s

ping:
  default_interval: 30s
  timeout: 5s
  packet_size: 32

metrics:
  enabled: true
  path: "/metrics"

storage:
  type: "memory"  # In-memory storage for logs
```

### configs/sites.yaml

```yaml
sites:
  # Dual-Line Site (both IPs must be online)
  - id: "site1"
    name: "Google DNS"
    location: "Global"
    primary_ip: "8.8.8.8"
    secondary_ip: "8.8.4.4"
    interval: 30
    enabled: true
    
  # Single-Line Site (only primary IP required)
  - id: "site2"
    name: "Cloudflare DNS"
    location: "Global"
    primary_ip: "1.1.1.1"
    # secondary_ip omitted = single-line mode
    interval: 60
    enabled: true
    
  # Mixed hostname/IP configuration
  - id: "site3"
    name: "Corporate Site"
    location: "Berlin, DE"
    primary_ip: "site-wan1.company.com"
    secondary_ip: "site-wan2.company.com"
    interval: 30
    enabled: true
```

## Development

### Available Commands

```bash
# Development
make dev          # Docker development environment with live-reload
make run          # Local development (requires Go)
make dev-logs     # Show development logs

# Building
make build        # Build binary locally
make docker-build # Build production Docker image

# Testing
make test         # Run tests
make health       # Check application health
make sites        # Show current sites status

# Utilities
make clean        # Clean build artifacts
```

### Project Structure

```
sitewatch/
├── main.go                    # Main application entry point
├── Makefile                   # Build and development commands
├── configs/                   # Configuration files
│   ├── config.yaml            # Service configuration
│   ├── sites.yaml             # Site definitions
│   ├── config.example.yaml    # Configuration template
│   └── sites.example.yaml     # Sites configuration template
├── deployments/               # Deployment configuration
│   └── docker/               # Docker deployment files
│       ├── Dockerfile         # Production container
│       ├── Dockerfile.dev     # Development container
│       ├── docker-compose.dev.yml   # Development environment
│       └── docker-compose.prod.yml  # Production environment
├── cmd/                       # Application commands
│   └── server/
│       └── server.go          # HTTP server setup
├── internal/                  # Internal application code
│   ├── config/                # Configuration loading
│   ├── handlers/              # HTTP handlers (API, UI, metrics)
│   ├── models/                # Data models and types
│   ├── services/              # Business logic services
│   │   ├── ping/              # Ping service and worker
│   │   └── stats/             # Statistics calculations
│   └── storage/               # Storage backends (memory, SQLite)
├── web/                       # Web assets and templates
│   ├── templates/             # HTML templates
│   │   ├── pages/            # Main pages
│   │   │   ├── dashboard.html # Main dashboard
│   │   │   └── logs.html      # Logs page
│   │   ├── fragments/        # HTMX fragments
│   │   │   ├── sites.html     # Sites grid fragment
│   │   │   ├── overview.html  # Status overview fragment
│   │   │   ├── details.html   # Site details modal
│   │   │   └── logs-table.html # Logs table fragment
│   │   └── components/       # Reusable components
│   │       └── enhanced-fragment.html # Enhanced site details
│   └── static/               # Static assets
│       └── js/
│           └── sitewatch-core.js # Core JavaScript functionality
├── data/                      # SQLite database files (if used)
├── bin/                       # Build output directory
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
└── README.md
```

## Production Deployment

### Docker Compose Production

1. **Prepare configuration**
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   cp configs/sites.example.yaml configs/sites.yaml
   # Edit configuration files as needed
   ```

2. **Deploy**
   ```bash
   docker-compose -f deployments/docker/docker-compose.prod.yml up -d
   ```

3. **Verify deployment**
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/sites
   # Access web dashboard: http://localhost:8080
   ```

### Standalone Binary

1. **Build binary**
   ```bash
   make build
   ```

2. **Run**
    ```bash
    ./bin/sitewatch
    ```

### Integration with Existing Infrastructure

The service integrates seamlessly with existing monitoring stacks:

- **Telegraf** collects metrics from `/metrics` endpoint
- **InfluxDB** stores time-series data via Telegraf
- **Grafana** visualizes metrics from InfluxDB
- **Serverguard** monitors site health via `/status` endpoints
- **Web Dashboard** provides real-time monitoring and log analysis

### Quick Reference

| Command | Description |
|---------|-------------|
| `make dev` | Start development environment |
| `make docker-run` | Start production container |
| `docker-compose -f deployments/docker/docker-compose.prod.yml ps` | Check container status |
| `docker-compose -f deployments/docker/docker-compose.prod.yml logs -f` | View logs |
| `docker-compose -f deployments/docker/docker-compose.prod.yml down` | Stop services |
| `curl http://localhost:8080/metrics` | Test Prometheus endpoint |
| `curl http://localhost:8080/api/sites/site1/status` | Test Serverguard endpoint |
| `curl "http://localhost:8080/api/logs?site=site1"` | Get site logs |