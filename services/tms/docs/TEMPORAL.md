# Temporal Workflow Engine Setup Guide

## Overview

Trenova uses [Temporal](https://temporal.io) as its workflow orchestration engine for handling critical background tasks, particularly audit logging and other asynchronous operations. Temporal provides durability, reliability, and scalability for our workflow executions.

## Quick Start (Development)

### 1. Start Temporal Server

For local development, use Temporal's development server:

```bash
temporal server start-dev
```

This starts Temporal on the default port `7233` with a web UI available at <http://localhost:8233>

### 2. Start the Worker Process

The worker process polls for and executes workflows:

```bash
make run-worker
```

### 3. Start the API Server

```bash
make run
```

## Production Setup

### Recommended: Use Temporal Cloud

For production environments, we **strongly recommend** using [Temporal Cloud](https://temporal.io/cloud) instead of self-hosting. Benefits include:

- **Managed Infrastructure**: No operational overhead
- **High Availability**: Built-in redundancy and failover
- **Automatic Scaling**: Handles traffic spikes seamlessly
- **Security**: Enterprise-grade security and compliance
- **Support**: Direct support from Temporal experts
- **Cost-Effective**: Pay-as-you-go pricing without infrastructure costs

#### Temporal Cloud Configuration

1. Sign up for [Temporal Cloud](https://temporal.io/cloud)
2. Create a namespace for your application
3. Update your configuration:

```yaml
temporal:
  hostPort: "your-namespace.tmprl.cloud:7233"
  security:
    enableEncryption: true
    encryptionKeyID: "production"
    enableCompression: true
    compressionThreshold: 1024
```

### Self-Hosted Production (If Required)

If you must self-host Temporal, follow the [official production deployment guide](https://docs.temporal.io/self-hosted-guide/production-deployment).

Key requirements:

- PostgreSQL or Cassandra for persistence
- Elasticsearch for visibility (optional but recommended)
- Proper resource allocation and monitoring
- High availability setup with multiple replicas

## Configuration

### Environment Variables

#### Encryption Key (Required for Production)

Temporal supports payload encryption to protect sensitive data. Set the encryption key:

```bash
# Default encryption key (used when encryptionKeyID is not specified)
export TEMPORAL_ENCRYPTION_KEY="your-32-character-encryption-key-here"

# Or use key-specific environment variables
export TEMPORAL_ENCRYPTION_KEY_production="your-production-encryption-key"
export TEMPORAL_ENCRYPTION_KEY_staging="your-staging-encryption-key"
```

**Important**:

- The encryption key should be at least 32 characters long
- Store encryption keys securely (use secret management systems)
- Never commit encryption keys to version control
- Changing the encryption key will make existing encrypted payloads unreadable

### Application Configuration

Configure Temporal in your `config/config.yaml`:

```yaml
temporal:
  hostPort: "localhost:7233"  # Temporal server address
  security:
    enableEncryption: false    # Enable in production
    encryptionKeyID: "default" # Identifies which key to use
    enableCompression: true    # Compress large payloads
    compressionThreshold: 1024 # Compress payloads > 1KB
```

#### Configuration Options

| Field | Description | Production Recommendation |
|-------|-------------|--------------------------|
| `hostPort` | Temporal server address | Use Temporal Cloud endpoint |
| `enableEncryption` | Encrypt workflow payloads | Set to `true` |
| `encryptionKeyID` | Identifies the encryption key | Use environment-specific IDs |
| `enableCompression` | Compress large payloads | Set to `true` |
| `compressionThreshold` | Minimum size for compression (bytes) | 1024 (1KB) |

## Architecture

### Components

1. **API Server**: Submits workflows to Temporal
2. **Worker Process**: Executes workflows and activities
3. **Temporal Server**: Orchestrates and persists workflow state

```text
┌─────────────┐      ┌──────────────┐      ┌────────────┐
│ API Server  │──────▶│   Temporal   │◀─────│   Worker   │
│             │      │    Server    │      │   Process  │
└─────────────┘      └──────────────┘      └────────────┘
     Submit               Orchestrate           Execute
   Workflows              & Persist           Workflows
```

### Task Queues

The application uses the following task queue:

- `audit-queue`: Processes audit log entries asynchronously

## CLI Tools

### Official Temporal CLI

Install the official Temporal CLI for comprehensive management capabilities:

```bash
# Install via Homebrew (macOS/Linux)
brew install temporal

# Or download from GitHub
curl -L https://github.com/temporalio/cli/releases/latest/download/temporal_cli_<version>_<os>_<arch>.tar.gz | tar -xz
```

#### Common CLI Commands

```bash
# Check server connection
temporal operator cluster health

# List workflows
temporal workflow list

# Show workflow execution details
temporal workflow show --workflow-id <id>

# List task queues and workers
temporal task-queue list-partition

# Describe a task queue (shows pollers)
temporal task-queue describe --task-queue audit-queue

# Generate encryption key
openssl rand -base64 32

# Test encryption setup
export TEMPORAL_ENCRYPTION_KEY="your-32-character-key"
temporal workflow execute --task-queue audit-queue --type ProcessAuditBatchWorkflow
```

For more CLI commands, see the [Temporal CLI documentation](https://docs.temporal.io/cli).

## Monitoring

### Temporal Web UI

- **Development**: <http://localhost:8233>
- **Temporal Cloud**: Available in your cloud dashboard

The Web UI provides:

- Workflow execution history
- Workflow state and progress
- Activity details and retries
- Search and filtering capabilities

### Health Checks

Verify the worker is running and connected:

1. Check worker logs:

```bash
make run-worker
# Look for: "Worker goroutine started"
```

1. Check Temporal Web UI:
   - Navigate to "Task Queues" tab
   - Verify `audit-queue` shows active workers

### Metrics

When metrics are enabled, Temporal provides:

- Workflow execution metrics
- Activity execution metrics
- Task queue metrics
- Worker metrics

## Troubleshooting

### Worker Not Polling Task Queue

**Symptoms**:

- "No Workers polling the audit-queue Task Queue" in Temporal UI
- Workflows not executing

**Solutions**:

1. Ensure worker process is running: `make run-worker`
2. Verify Temporal server is accessible
3. Check configuration matches between API and worker
4. Review worker logs for connection errors

### Encryption Key Issues

**Symptoms**:

- "encryption key not found" errors
- "decryption failed" errors

**Solutions**:

1. Ensure `TEMPORAL_ENCRYPTION_KEY` environment variable is set
2. Verify key ID matches configuration
3. Use the same key for encryption and decryption
4. Check key is at least 32 characters

### Connection Issues

**Symptoms**:

- "failed to dial temporal client" errors
- Connection timeout errors

**Solutions**:

1. Verify Temporal server is running
2. Check `hostPort` configuration
3. Ensure network connectivity
4. Verify firewall rules allow connection

## Security Best Practices

1. **Always enable encryption in production**

   ```yaml
   enableEncryption: true
   ```

2. **Use strong encryption keys**
   - Generate using: `openssl rand -base64 32`
   - Store in secret management system
   - Rotate keys periodically

3. **Enable compression for large payloads**
   - Reduces network traffic
   - Improves performance
   - Saves storage costs

4. **Use separate environments**
   - Different namespaces for dev/staging/prod
   - Separate encryption keys per environment
   - Isolated worker pools

5. **Monitor and audit**
   - Enable Temporal metrics
   - Monitor workflow failures
   - Audit workflow executions
   - Set up alerts for critical failures

## Performance Tuning

### Worker Configuration

Adjust worker concurrency in `internal/bootstrap/modules/worker/temporal.go`:

```go
worker.Options{
    MaxConcurrentActivityExecutionSize:     10,  // Concurrent activities
    MaxConcurrentWorkflowTaskExecutionSize: 10,  // Concurrent workflows
    MaxConcurrentWorkflowTaskPollers:       2,   // Workflow pollers
    MaxConcurrentActivityTaskPollers:       2,   // Activity pollers
}
```

### Compression Threshold

Adjust based on your payload sizes:

- Small payloads (<1KB): Set higher threshold or disable
- Large payloads (>10KB): Enable with lower threshold
- Monitor compression ratio to optimize

## Deployment Checklist

### Pre-Production

- [ ] Set up Temporal Cloud account or self-hosted cluster
- [ ] Configure encryption keys in secret management
- [ ] Update configuration with production endpoints
- [ ] Test worker connectivity
- [ ] Verify workflow execution

### Production

- [ ] Enable encryption (`enableEncryption: true`)
- [ ] Set production encryption key
- [ ] Configure monitoring and alerts
- [ ] Deploy worker processes (recommend 2+ for HA)
- [ ] Verify health checks
- [ ] Monitor initial workflow executions

## Support

For Temporal-specific issues:

- [Temporal Documentation](https://docs.temporal.io)
- [Temporal Community Forum](https://community.temporal.io)
- [Temporal Cloud Support](https://temporal.io/support) (for cloud customers)

For Trenova-specific issues:

- Check application logs: `make logs`
- Review worker logs: `make run-worker`
- Open an issue on GitHub
