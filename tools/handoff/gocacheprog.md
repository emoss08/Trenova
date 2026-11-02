## Shared build cache (GOCACHEPROG)

### What CI does today

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

### What GOCACHEPROG changes

`GOCACHEPROG` (stable since Go 1.24) makes the go command ask a helper process for each action
ID and output ID instead of reading `GOCACHE`. The helper keeps a local disk cache and fills it
from, and writes through to, a remote store. A job fetches only the objects its build needs,
uploads only the ones it produced, and every job, branch and tag set shares one store, because
the cache is content-addressed: the tag set and `-race` are part of each action ID, so the six
keys become unnecessary.

### Implementations

| Implementation | Store | Notes |
|---|---|---|
| [`richardartoul/gobuildcache`](https://pkg.go.dev/github.com/richardartoul/gobuildcache) | S3 (Express One Zone recommended), S3-compatible (Tigris, R2), GCS | Built for CI; maintained (v1.5.0 adds GCS Anywhere Cache); read-only mode for untrusted PRs; no eviction of its own, so a bucket lifecycle rule expires old objects. **Recommended.** |
| [`tailscale/go-cache-plugin`](https://pkg.go.dev/github.com/tailscale/go-cache-plugin/cmd/go-cache-plugin) | S3 | Runs as a long-lived daemon and can also serve a module proxy; used on Tailscale's own builders; releases are infrequent (v0.0.11, 2024). Good when the cache runs on a shared builder host. |
| [`vearutop/gocacheprog`](https://pkg.go.dev/github.com/vearutop/gocacheprog) | its own server | Has GitHub Actions init/done steps, but needs a server we would run. |
| [`jakebailey/gocachez`](https://pkg.go.dev/github.com/jakebailey/gocachez), [`sgrankin/gocachez`](https://pkg.go.dev/github.com/sgrankin/gocachez) | local disk | zstd compression and automatic eviction of a local cache; no remote store, so no sharing. |
| `bradfitz/go-tool-cache` | S3 / HTTP | The original reference implementation of the protocol; superseded by the above. |

### Recommendation

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

### Worktrees and developer machines

Without `-trimpath` (which Trenova does not set globally, because it changes debug paths), the
go command puts the package's absolute directory into the action ID of every package outside
`GOROOT` (`cmd/go/internal/work/exec.go`, `"dir %s"`). So:

- CI hits fully, because every runner checks out to the same path.
- Worktrees on one machine already share `~/.cache/go-build` for the standard library and every
  module from the shared module cache; they miss only on Trenova's own packages, whose paths
  differ. A remote cache adds nothing there.
- A developer's machine would hit the remote cache only for the standard library and third-party
  modules (about a third of a cold `cmd/api` build's compiler CPU, roughly 330 of 1,050
  CPU-seconds), and only with the same Go version, `GOMODCACHE` path and flags as CI. That is
  worth doing read-only with the same helper once CI uses it, not as a reason to adopt it.
