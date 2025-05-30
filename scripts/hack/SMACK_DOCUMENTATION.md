# Trenova Stress Testing Framework (SMACK) Documentation

**USAGE:**

```bash
./smack.sh [OPTIONS] [COMMAND]
```

**OPTIONS:**

* `-h, --help`: Show the help message.
* `-u, --url URL`: API base URL.
  * Default: `http://localhost:3001`
* `-c, --concurrent N`: Number of concurrent requests.
  * Default: `50`
* `-t, --total N`: Total number of requests for `spike` and `burst` test patterns.
  * Default: `500`
* `-d, --duration N`: Duration in seconds for `sustained` and `endurance` test patterns.
  * Default: `60`
* `-r, --ramp-time N`: Ramp-up time in seconds for the `ramp` test pattern.
  * Default: `10`
* `-m, --monitor`: Enable system monitoring (CPU, memory, load average).
* `-v, --verbose`: Enable verbose logging (uses `set -x`).

**COMMANDS:**

* `test PATTERN ENDPOINT`
  * Runs a specific test `PATTERN` on a specific `ENDPOINT`.
  * The values for `PATTERN` and `ENDPOINT` are detailed below.
* `quick`
  * Runs a quick test suite. This is the default command if no other command is specified.
  * It tests critical endpoints (`workers_select`, `customers_select`, `organizations`) with moderate load using `spike` (25 concurrent, 250 total requests) and `sustained` (25 concurrent, 30 seconds duration) patterns.
* `full`
  * Runs a comprehensive test suite.
  * It tests all defined `ENDPOINTS` with all defined `TEST_PATTERNS` using the default or command-line specified values for concurrent requests, total requests, duration, and ramp-up time.
* `single ENDPOINT`
  * Runs all available test `PATTERNS` on a single specified `ENDPOINT`.
  * Uses default or command-line specified values for test parameters.
* `custom`
  * Starts an interactive custom test configuration mode.
  * You will be prompted to select an endpoint, a test pattern, and then to input parameters like concurrent requests, total requests, or duration based on the chosen pattern.

**TEST PATTERNS:**

* `spike`: Simulates an instant high load.
  * Uses: `--concurrent` (for managing how many are sent *at once* before waiting for that batch) and `--total`.
* `ramp`: Simulates a gradual increase in load over a specified time.
  * Uses: `--concurrent` (for managing active background processes), `--total`, and `--ramp-time`.
* `sustained`: Simulates a constant load for a specified duration.
  * Uses: `--concurrent` and `--duration`.
* `burst`: Simulates intermittent high load, with bursts of requests followed by rest periods.
  * Uses: `--concurrent` (bursts are 1/5th of this value) and `--total`.
* `endurance`: Simulates a long-duration test with a constant number of concurrent users.
  * Uses: `--concurrent` and `--duration`.

**ENDPOINTS:**

The script defines the following API endpoints that can be targeted for tests:

* `workers_select` - `api/v1/workers/select-options`
* `customers_select` - `api/v1/customers/select-options`
* `tractors_select` - `api/v1/tractors/select-options`
* `trailers_select` - `api/v1/trailers/select-options`
* `workers_list` - `api/v1/workers`
* `customers_list` - `api/v1/customers`
* `organizations` - `api/v1/organizations/me`
* `shipments_list` - `api/v1/shipments`
* `equipment_types` - `api/v1/equipment-types`
* `fleet_codes` - `api/v1/fleet-codes`

**EXAMPLES:**

* **Run the quick test suite with default settings:**

    ```bash
    ./smack.sh quick
    ```

    (or simply `./smack.sh` as `quick` is the default)

* **Run a spike test on the `workers_select` endpoint with 100 concurrent requests and a total of 1000 requests:**

    ```bash
    ./smack.sh -c 100 -t 1000 test spike workers_select
    ```

* **Run a sustained load test on `workers_list` for 2 minutes (120 seconds) with the default 50 concurrent requests:**

    ```bash
    ./smack.sh -d 120 test sustained workers_list
    ```

* **Run the full test suite with system monitoring enabled:**

    ```bash
    ./smack.sh --monitor full
    ```

* **Run a ramp test on `customers_list` with 200 concurrent requests, 1000 total requests, and a ramp-up time of 20 seconds:**

    ```bash
    ./smack.sh -c 200 -t 1000 -r 20 test ramp customers_list
    ```

* **Run all test patterns on the `organizations` endpoint:**

    ```bash
    ./smack.sh single organizations
    ```

* **Start an interactive session to define a custom test:**

    ```bash
    ./smack.sh custom
    ```
