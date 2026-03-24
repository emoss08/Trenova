<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Trenova Stress Testing Framework (SMACK)

SMACK is a Bash load-testing utility for Trenova API endpoints. It now authenticates with an API key instead of session cookies.

The built-in scenario list is intentionally limited to routes that align with the runtime API-key policy in `services/tms/internal/core/services/apikeyservice/policy.go`.

## Authentication

By default SMACK sends:

```http
Authorization: Bearer <api-key>
```

You can provide the API key in either of these ways:

- `--api-key <key>`
- `SMACK_API_KEY=<key>`
- Interactive secure prompt if stdin is a TTY and no key was supplied

You can override the header shape for compatibility:

- `--auth-header X-API-Key`
- `--auth-scheme none`

That produces:

```http
X-API-Key: <api-key>
```

## Usage

```bash
./smack.sh [OPTIONS] [COMMAND]
```

## Options

- `-h, --help`: Show help.
- `-u, --url URL`: API base URL. Default: `http://localhost:8080`
- `-c, --concurrent N`: Concurrent requests. Default: `50`
- `-t, --total N`: Total requests for `spike`, `ramp`, and `burst`. Default: `500`
- `-d, --duration N`: Duration in seconds for `sustained` and `endurance`. Default: `60`
- `-r, --ramp-time N`: Ramp duration for `ramp`. Default: `10`
- `--api-key KEY`: API key used for authentication
- `--auth-header NAME`: Authentication header override. Default: `Authorization`
- `--auth-scheme SCHEME`: Authentication scheme override. Default: `Bearer`
- `--results-dir PATH`: Output directory for logs and reports. Default: `scripts/smack/results`
- `--hey`: Use `hey` instead of `curl`
- `-m, --monitor`: Enable system monitoring when supported
- `-v, --verbose`: Enable shell tracing

## Commands

- `quick`: Default command. Runs a short suite across key endpoints.
- `full`: Runs all endpoints against all patterns.
- `single ENDPOINT`: Runs all patterns against one endpoint.
- `test PATTERN ENDPOINT`: Runs one pattern against one endpoint.
- `custom`: Interactive endpoint/pattern/parameter selection.

## Test Patterns

- `spike`: Sends requests as fast as possible up to the target concurrency.
- `ramp`: Gradually increases request rate over the configured ramp time.
- `sustained`: Maintains concurrency for a fixed duration.
- `burst`: Sends short bursts separated by pauses.
- `endurance`: Long-running sustained load.

## Endpoints

- `workers_select` -> `api/v1/workers/select-options/`
- `customers_select` -> `api/v1/customers/select-options/`
- `workers_list` -> `api/v1/workers/`
- `customers_list` -> `api/v1/customers/`
- `equipment_types` -> `api/v1/equipment-types/`

## Examples

Quick suite:

```bash
./smack.sh --api-key trv_test.secret quick
```

Single spike test:

```bash
./smack.sh --api-key trv_test.secret -c 100 -t 1000 test spike workers_select
```

Sustained run for 2 minutes:

```bash
./smack.sh --api-key trv_test.secret -d 120 test sustained workers_list
```

Use `hey`:

```bash
./smack.sh --api-key trv_test.secret --hey test spike organizations
```

Use a raw API key header instead of bearer auth:

```bash
./smack.sh --api-key trv_test.secret --auth-header X-API-Key --auth-scheme none test spike organizations
```

Write results somewhere else:

```bash
./smack.sh --api-key trv_test.secret --results-dir /tmp/smack-results quick
```

## Output

Each test run writes a timestamped directory under the results root with:

- `report.txt`: Human-readable summary
- `raw_results.tsv`: Per-request raw metrics
- `errors.log`: Failed request details when present
- `system_metrics.csv`: System metrics when monitoring is enabled

The run also writes a timestamped top-level SMACK log file into the results root.
