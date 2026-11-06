# Go build cache

How CI caches Go builds today, what a shared cache (`GOCACHEPROG`) would change, and what it
would and would not do for worktrees and developer machines. Adopting it needs a bucket and
secrets, which is the maintainer's call; this is the recommendation.

## What CI does today

Every Go job runs `.github/actions/setup-go`, which restores and saves the whole `GOCACHE`
directory with `actions/cache` under `gobuild-<os>-<go>-<cache-key>-<sha>`, falling back to the
newest entry with the same prefix. There are six cache keys (`default`, `integration`, `lint`,
`race`, `assay`, `agent-evals-live`), because a build with `-race` or another tag set produces
different objects. This has three costs:

- **Size.** A cold `cmd/api` build alone writes 3.9 GB to `GOCACHE`; `go test ./...` with test
  binaries and coverage writes several times that. Each job saves a full snapshot per commit,
  and GitHub keeps 10 GB per repository by default, so snapshots evict each other and a job
  regularly falls back to an older snapshot or none (a cold build: about 1,050 compiler
  CPU-seconds for `cmd/api` alone on 4 cores, 5 minutes wall).
- **Transfer.** Restore downloads and unpacks the whole snapshot before the first compile, and
  save tars and uploads it again after the job, whatever the job used.
- **No sharing.** A pull request reads only its own and the default branch's entries; the six keys
  never share the objects they have in common (the standard library, every third-party module
  and every package whose flags match), and nothing reaches a developer's machine.

## What GOCACHEPROG changes

`GOCACHEPROG` (stable since Go 1.24) makes the go command ask a helper process for each action
ID and output ID instead of reading `GOCACHE`. The helper keeps a local disk cache and fills it
from, and writes through to, a remote store. A job fetches only the objects its build needs,
uploads only the ones it produced, and every job, branch and tag set shares one store, because
the cache is content-addressed: the tag set and `-race` are part of each action ID, so the six
keys become unnecessary.

## Implementations

| Implementation | Store | Notes |
|---|---|---|
| [`richardartoul/gobuildcache`](https://pkg.go.dev/github.com/richardartoul/gobuildcache) | S3 (Express One Zone recommended), S3-compatible (Tigris, R2), GCS | Built for CI; maintained (v1.5.0 adds GCS Anywhere Cache); read-only mode for untrusted PRs; no eviction of its own, so a bucket lifecycle rule expires old objects. **Recommended.** |
| [`tailscale/go-cache-plugin`](https://pkg.go.dev/github.com/tailscale/go-cache-plugin/cmd/go-cache-plugin) | S3 | Runs as a long-lived daemon and can also serve a module proxy; used on Tailscale's own builders; releases are infrequent (v0.0.11, 2024). Good when the cache runs on a shared builder host. |
| [`vearutop/gocacheprog`](https://pkg.go.dev/github.com/vearutop/gocacheprog) | its own server | Has GitHub Actions init/done steps, but needs a server we would run. |
| [`jakebailey/gocachez`](https://pkg.go.dev/github.com/jakebailey/gocachez), [`sgrankin/gocachez`](https://pkg.go.dev/github.com/sgrankin/gocachez) | local disk | zstd compression and automatic eviction of a local cache; no remote store, so no sharing. |
| `bradfitz/go-tool-cache` | S3 / HTTP | The original reference implementation of the protocol; superseded by the above. |

## Recommendation

Use `gobuildcache` in CI against an S3-compatible bucket with no egress fees (Cloudflare R2 or
Tigris for GitHub-hosted runners; S3 Express One Zone in the runners' region if CI moves to
self-hosted AWS runners):

1. In `setup-go`, install a pinned `gobuildcache`, set `GOCACHEPROG` to it with the bucket and
   credentials from repository secrets, and drop both `actions/cache` steps for `GOCACHE` (keep
   the module cache step, or point `GOPROXY` at a module proxy) and the `cache-key` input with it.
2. Runs on `master` and on pull requests from the repository write; pull requests from forks get
   no secrets and run read-only (or cold).
3. A bucket lifecycle rule deletes objects not written for 14 days. The store then holds the
   working set of every job instead of six snapshots competing for 10 GB.

Expected effect: the restore and save steps (tarring, uploading and downloading multi-GB
snapshots) disappear, a job never falls back to a cold cache because another job's snapshot
evicted its own, and a pull request starts from the objects of every earlier run, not only its
base. The size of the saving depends on how often jobs miss today; measure it by running
`test-tms` on one branch both ways and comparing the setup, build and post-job step times before
switching every workflow.

## Worktrees and developer machines

The Taskfile sets `-trimpath` for every go command, and with it the action ID carries no
directory (`cmd/go/internal/work/exec.go` hashes `dir %s` only without `-trimpath`). A release
compiler's tool ID is its version line (`compile version go1.27.1`, `buildid.go`), not its binary.
So a cache filled by a Taskfile build is valid at any path on any machine with the same Go, flags,
`GOOS`/`GOARCH` and cgo setting. Measured on 2026-11-06 (4 cores, Go 1.27.1, the watch flags): a
cold `cmd/api` build at one path, then the same commit at another path against that cache,
compiled 0 of 3,978 packages, took 7.5 s (the link) and produced a byte-identical binary.

What a developer cache needs on top of CI's:

- **The developers' flags.** CI's test jobs build without `-trimpath` and without the watch
  build's `-gcflags`, so their objects never match a `task run-watch` build. A job on `master`
  has to build the watch targets (`cmd/api`, `cmd/worker`, `cmd/cli`) with `WATCH_BUILD`'s flags.
- **Every developer platform.** `GOOS` and `GOARCH` are in every action ID. A Linux runner can
  fill `darwin/arm64` by cross-compiling, but only if cgo is off on both sides: a package with cgo
  files hashes the C compiler, and `net` and `runtime/cgo` sit under nearly every package. The
  watch tasks would set `CGO_ENABLED=0`, as `deploy/Dockerfile.tms` already does.
- **The bandwidth.** One `cmd/api` watch build is 3.5 GB in 12,962 objects and compresses about
  1.2x, so a cold fill downloads about 3 GB: about 30 s at 1 Gbps, about 4 minutes at 100 Mbps,
  against about 3.5 minutes compiling on 4 cores and less on a laptop with more. A developer
  cache pays on a fast link and after a pull that touched widely imported packages, not
  everywhere. `gobuildcache` also never trims its local directory, so a laptop needs a periodic
  `clear-local`.

## Trust

Anything that can write the store can put compiled code into every build that reads it, so:

- Only jobs on pushes to `master` hold a write token. Pull requests, from the repository or from
  forks, and developer machines use a read-only token (R2 tokens are scoped per bucket and
  permission) or none.
- Release images (`deploy/Dockerfile.*`) never read the shared cache.
- `GOCACHEPROG` points at a small wrapper that sets the endpoint and credentials for
  `gobuildcache` alone. The go command passes its environment to every test binary, so keys in
  the job environment would reach test code, and `AWS_ENDPOINT_URL_S3` there would redirect any
  AWS SDK client a test builds.

`gobuildcache` v1.9.0 has the pieces: `GOBUILDCACHE_READ_ONLY`, LZ4 compression (on by default),
path-style addressing, and the AWS SDK's `AWS_ENDPOINT_URL_S3` for R2's endpoint with region
`auto`. Building it needs Go 1.27.3 or later (its `go.mod`); the binary does not depend on the
toolchain the build uses. R2 has no egress fees, which matters because a cache is mostly reads:
$0.015 per GB-month, $4.50 per million writes and $0.36 per million reads, so the store costs
a few dollars a month at Trenova's size.
