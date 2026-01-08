# Raven Alert Service

A high-performance Golang microservice for dynamic API health monitoring with JSONPath rule evaluation, webhook alerts, exponential backoff retry, and circuit breaker pattern.

## Features

- **Dynamic Configuration**: Health check configurations stored in MongoDB with on-the-fly updates
- **JSONPath Evaluation**: Flexible rule engine using JSONPath expressions to evaluate API responses
- **Custom Webhook Alerts**: HTTP POST notifications with retry logic and exponential backoff
- **High Concurrency**: Leverages Go's goroutines and channels for scalability
- **Authentication Support**: Basic Auth and Bearer Token for monitored APIs
- **Execution History**: Complete audit trail of all health check executions and alerts
- **RESTful API**: Standard HTTP endpoints for configuration and execution management
- **Circuit Breaker**: Production-ready resilience pattern for webhook failures
- **Async Execution**: Support for both synchronous and asynchronous health check execution
- **Cron Scheduling**: Automated health check execution with standard cron expressions
- **Distributed Locking**: MongoDB-based distributed locks for horizontal scaling in Kubernetes

## Technology Stack

- **Go**: 1.23.4
- **MongoDB**: Document store for configurations and history
- **Dependencies**:
  - `go.mongodb.org/mongo-driver` - MongoDB driver
  - `github.com/google/uuid` - UUID generation
  - `github.com/oliveagle/jsonpath` - JSONPath evaluation
  - `github.com/robfig/cron/v3` - Cron expression parsing and scheduling
  - `golang.org/x/sync` - Enhanced concurrency primitives

## Quick Start

### Prerequisites

- Go 1.23.4 or later
- MongoDB 4.4 or later
- Docker (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/dandantas/raven
cd raven

# Download dependencies
go mod download

# Build the application
go build -o raven-alert ./cmd/server

# Run the service
./raven-alert
```

### Docker

```bash
# Build Docker image
docker build -t raven-alert:latest .

# Run with Docker
docker run -d \
  --name raven-alert \
  -p 8080:8080 \
  -e MONGO_URI="mongodb://localhost:27017/raven_alert" \
  -e LOG_LEVEL="info" \
  raven-alert:latest
```

## Configuration

The service is configured via environment variables:

### MongoDB Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MONGO_URI` | MongoDB connection URI | `mongodb://localhost:27017/raven_alert?authSource=admin` |
| `MONGO_DATABASE` | Database name | `raven_alert` |
| `MONGO_TIMEOUT_SEC` | Connection timeout | `10` |

### HTTP Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | `8080` |
| `HTTP_READ_TIMEOUT_SEC` | Read timeout | `30` |
| `HTTP_WRITE_TIMEOUT_SEC` | Write timeout | `30` |

### Worker Pool Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `WORKER_POOL_SIZE` | Number of worker goroutines | `10` |
| `MAX_CONCURRENT_JOBS` | Job queue buffer size | `1000` |

### Logging Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Log format (json, text) | `json` |

### Timeout Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `DEFAULT_API_TIMEOUT_SEC` | Default timeout for target API calls | `30` |
| `DEFAULT_WEBHOOK_TIMEOUT_SEC` | Default timeout for webhook calls | `10` |

### Scheduler Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SCHEDULER_ENABLED` | Enable/disable cron scheduler | `true` |
| `SCHEDULER_TICK_INTERVAL_SEC` | How often to check for due schedules | `60` |
| `SCHEDULER_LOCK_TTL_SEC` | Lock expiration time (handles pod crashes) | `300` |
| `SCHEDULER_CONCURRENCY` | Max concurrent scheduled executions | `10` |

## API Endpoints

### Health Endpoints

- `GET /health` - Service health status
- `GET /ready` - Service readiness check

### Health Check Configuration

- `POST /api/v1/health-checks` - Create configuration
- `GET /api/v1/health-checks` - List configurations
- `GET /api/v1/health-checks/{id}` - Get configuration
- `PUT /api/v1/health-checks/{id}` - Update configuration
- `DELETE /api/v1/health-checks/{id}` - Delete configuration

### Execution

- `POST /api/v1/health-checks/{id}/execute` - Execute single check
- `POST /api/v1/health-checks/execute-batch` - Execute multiple checks

### History & Alerts

- `GET /api/v1/executions` - List execution history
- `GET /api/v1/executions/{correlation_id}` - Get execution details
- `GET /api/v1/alerts` - List alert logs

## Example Health Check Configuration

### With Cron Scheduling

```json
{
  "name": "Payment API Health Check",
  "description": "Monitors payment gateway availability",
  "enabled": true,
  "schedule": "*/5 * * * *",
  "schedule_enabled": true,
  "target": {
    "url": "https://api.payment.example.com/health",
    "method": "GET",
    "headers": {
      "Accept": "application/json"
    },
    "auth": {
      "type": "bearer",
      "token": "secret-token-here"
    },
    "timeout": 30
  },
  "rules": [
    {
      "name": "Check Status Field",
      "description": "Verify status field equals 'healthy'",
      "expression": "$.status",
      "operator": "eq",
      "expected_value": "healthy",
      "alert_on_match": false
    },
    {
      "name": "Check Response Time",
      "description": "Alert if response time > 5000ms",
      "expression": "$.response_time_ms",
      "operator": "gt",
      "expected_value": 5000,
      "alert_on_match": true
    }
  ],
  "webhook": {
    "url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json"
    },
    "retry_config": {
      "max_attempts": 3,
      "initial_delay_ms": 1000,
      "max_delay_ms": 30000,
      "multiplier": 2
    }
  },
  "metadata": {
    "created_by": "admin@example.com",
    "tags": ["payment", "critical"]
  }
}
```

### Manual Trigger Only (No Scheduling)

```json
{
  "name": "On-Demand API Check",
  "description": "Manual health check without scheduling",
  "enabled": true,
  "schedule_enabled": false,
  "target": {
    "url": "https://api.example.com/health",
    "method": "GET"
  },
  "rules": [...],
  "webhook": {...}
}
```

## Cron Scheduling

Health checks can be configured with standard cron expressions for automated execution. The scheduler runs in each Kubernetes pod and uses distributed locking to prevent duplicate executions across multiple instances.

### Cron Expression Format

Standard cron format with 5 fields:

```
 ┌───────────── minute (0 - 59)
 │ ┌───────────── hour (0 - 23)
 │ │ ┌───────────── day of month (1 - 31)
 │ │ │ ┌───────────── month (1 - 12)
 │ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
 │ │ │ │ │
 * * * * *
```

### Examples

| Expression | Description |
|------------|-------------|
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour |
| `0 */2 * * *` | Every 2 hours |
| `30 9 * * *` | Daily at 9:30 AM |
| `0 9 * * 1-5` | Weekdays at 9:00 AM |
| `*/15 8-17 * * 1-5` | Every 15 minutes during business hours (8am-5pm, Mon-Fri) |

### Distributed Scheduling

The scheduler uses MongoDB-based distributed locking to ensure that:
- Only one pod executes a scheduled health check at a time
- Locks automatically expire after 5 minutes (configurable via `SCHEDULER_LOCK_TTL_SEC`)
- Crashed pods don't leave stale locks
- Horizontal scaling works seamlessly in Kubernetes

## JSONPath Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `$.status eq "healthy"` |
| `ne` | Not equals | `$.error ne null` |
| `gt` | Greater than | `$.response_time gt 1000` |
| `lt` | Less than | `$.cpu_usage lt 80` |
| `gte` | Greater than or equal | `$.count gte 10` |
| `lte` | Less than or equal | `$.memory lte 90` |
| `contains` | String/array contains | `$.message contains "error"` |
| `exists` | Field exists | `$.optional_field exists` |
| `regex` | Regular expression | `$.email regex "^[a-z]+@"` |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP API Layer                          │
│  (net/http - RESTful endpoints for configuration & execution)   │
└────────────────┬────────────────────────────────────────────────┘
                 │
┌────────────────┴────────────────────────────────────────────────┐
│                      Service Orchestrator                       │
│  • Request validation & correlation ID generation               │
│  • Context propagation & timeout management                     │
│  • Structured logging with slog                                 │
└────────────┬──────────────────────┬─────────────────────────────┘
             │                      │
┌────────────┴─────────┐  ┌─────────┴──────────────────────────┐
│  Configuration Mgr   │  │   Health Check Executor            │
│  • CRUD operations   │  │   • Worker pool (configurable)     │
│  • Schema validation │  │   • Concurrent API calls           │
│  • MongoDB access    │  │   • Response capture & timeout     │
└──────────────────────┘  └────────┬───────────────────────────┘
                                   │
                        ┌──────────┴──────────┐
                        │  Rule Evaluator     │
                        │  • JSONPath parsing │
                        │  • Boolean logic    │
                        │  • Type coercion    │
                        └──────────┬──────────┘
                                   │
                        ┌──────────┴───────────────┐
                        │   Alert Dispatcher       │
                        │   • Webhook POST calls   │
                        │   • Retry with backoff   │
                        │   • Circuit breaker      │
                        └──────────────────────────┘
```

## MongoDB Collections

### health_check_configs
Stores health check configurations with validation rules and scheduling information.

### execution_history
Records every health check execution with full request/response details.

### alert_logs
Tracks webhook alert delivery attempts and outcomes.

### schedule_locks
Stores distributed locks for scheduled health check executions (automatic TTL cleanup).

## Performance

| Metric | Target |
|--------|--------|
| Request latency (p50) | <100ms (CRUD) |
| Request latency (p95) | <500ms (CRUD) |
| Execution throughput | 100 checks/sec |
| Concurrent executions | 100 |
| Memory usage | <512 MB |

## Development

```bash
# Run tests
go test ./...

# Run with debug logging
LOG_LEVEL=debug go run ./cmd/server

# Format code
go fmt ./...

# Run linter
golangci-lint run
```
