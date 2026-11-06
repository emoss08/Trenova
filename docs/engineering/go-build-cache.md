# Go build cache

CI, developer machines and cloud sessions share one Go build cache: the R2 bucket
`trenova-go-build-cache`, reached through `GOCACHEPROG` and
[`gobuildcache`](https://pkg.go.dev/github.com/richardartoul/gobuildcache) (pinned in
`services/tms/scripts/gocacheprog/version`). The go command asks the helper for each action ID;
the helper answers from a local directory, then from the bucket, and in write mode uploads what
the build produced (LZ4, about 2.5x on Go objects). Everything is content-addressed, so every job,
branch, tag set and checkout path shares one store.

## Pieces

- **`scripts/gocacheprog/gocacheprog.sh`** is what `GOCACHEPROG` points at. It reads the endpoint
  and keys from a credentials file (`GO_REMOTE_CACHE_CREDENTIALS`, default
  `~/.config/trenova/go-remote-cache.env`) or, in a cloud session, the environment's `R2_*`
  variables; drops every inherited `AWS_*` variable; sets the gobuildcache and AWS SDK variables in
  its own environment and execs gobuildcache. It is read-only unless the file says
  `GO_REMOTE_CACHE_WRITE=true`. If it has no credentials or a signed `HEAD` on the bucket fails
  (remembered for ten minutes), it runs as a local disk cache and says so on stderr, because
  gobuildcache exits on an unreachable bucket and takes the go command with it.
- **`.github/actions/setup-go`** takes `r2-endpoint`, `r2-access-key-id` and
  `r2-secret-access-key`. With all three it installs gobuildcache (cached per version), writes the
  credentials file under `$RUNNER_TEMP`, and exports `GOCACHEPROG` and `GODEBUG=goindex=0`.
  Without them (forks, Dependabot) the job builds with a cold local `GOCACHE`. Every caller passes
  the write pair on pushes to `master` and the read pair otherwise.
- **`.github/workflows/go-build-cache.yml`** runs `task build-watch-targets` (cmd/api, cmd/worker,
  cmd/cli with `WATCH_BUILD`'s flags and cgo off) for linux/amd64 and darwin/arm64 on every push to
  `master`. CI's test jobs build with other flags, so without this job no developer build would
  hit.
- **Taskfile.** `GO_REMOTE_CACHE=on` (default off) sets `GOCACHEPROG`, `GODEBUG=goindex=0`, and
  `-p=$GO_REMOTE_CACHE_JOBS` (32) on the watch builds. The watch tasks and `build-cli` (so `run`,
  `run-worker` and the CLI tasks) all build with `WATCH_BUILD` and `CGO_ENABLED=0`.
  `task go-remote-cache-install` installs the pinned helper; `task go-remote-cache-clear` empties
  its local directory (`~/.cache/trenova/gobuildcache`), which gobuildcache never trims.

## Turning it on

Developer machine: `task go-remote-cache-install`, put a **read** token in
`~/.config/trenova/go-remote-cache.env` (mode 600):

```
R2_BUILD_CACHE_ENDPOINT=https://<account>.r2.cloudflarestorage.com
R2_BUILD_CACHE_ACCESS_KEY_ID=...
R2_BUILD_CACHE_SECRET_ACCESS_KEY=...
```

then `GO_REMOTE_CACHE=on task run-watch` (or export `GO_REMOTE_CACHE=on`). Cloud sessions need
only the install task: the wrapper uses the environment's `R2_ACCESS_KEY_ID`,
`R2_SECRET_ACCESS_KEY` and `R2_S3_API_ENDPOINT`, read-only. Give the cloud environment a
read-only build-cache token under those names rather than the app's storage keys.

A hit needs the same Go release, flags, `GOOS`/`GOARCH` and cgo setting as `master`'s build, so a
change to `WATCH_BUILD` or the Go version misses until the next `master` push refills the store.
`-trimpath` keeps the checkout path out of every action ID, and with cgo off a linux runner's
darwin/arm64 build has a Mac's keys. gocut builds through `-toolexec`, and the compiler's tool ID
then comes from `gocut compile -V=full`: a developer running gocut hits only if gocut passes that
line through unchanged.

## Why the two settings

- **`GODEBUG=goindex=0`.** The go command stores its module index in the build cache, keyed by
  each package's absolute path and file times. Under a cache program that is one remote miss per
  first-party package (about 2,700 for cmd/api) on every new checkout, before any compile.
  Turning the index off changes no action ID.
- **`-p 32`.** The go command looks up only as many actions at once as it runs, and a remote hit
  costs a round trip (about 100 ms p50 from a cloud session), so at the default `-p` (one per
  core) a warm store is read at round-trip speed. CI leaves `-p` alone: in `go test` it also sets
  how many test binaries run at once.

## Measurements

2026-11-06, a cloud session (4 cores, 16 GB), Go 1.27.1, the cmd/api watch build, cgo off:

| Build | Wall | Compiled | Transferred |
|---|---|---|---|
| Cold, writing to R2 (no cache anywhere) | 400 s | 3,978 | 1.35 GB up (3.36 GB raw) |
| Other checkout path, empty local dir, read-only, `-p 4`, index on | 484 s | 1 | 1.35 GB down |
| Same, `-p 4`, `goindex=0` | 267 s | 1 | 1.34 GB down |
| Same, `-p 16`, `goindex=0` | 71 s | 1 | 1.34 GB down |
| Same, `-p 32`, `goindex=0` | 43 s | 1 | 1.34 GB down |
| Warm local dir from another path, index on / `goindex=0` | 158 s / 9 s | 0 | 3 MB / 0 |
| S4 (exported func added to `shipment/additionalcharge.go`), `-p 4` / `-p 32` | 76 s / 67 s | 791 | 0 |

The one compile is the main package, whose VCS stamp differed; with the same commit the binaries
were byte-identical. `GO_REMOTE_CACHE=on task build-cli` from an empty local directory took 45 s.
The S4 rebuild at `-p 32` peaked at 3 GB of memory. One cmd/api build is 15,102 objects and
1.45 GB in the bucket.

## Trust

Anything that can write the store can put compiled code into every build that reads it.

- Only pushes to `master` write. Pull requests, other events, developer machines and cloud
  sessions read, and the wrapper enforces read-only from the credentials file, whatever the job
  or shell environment says. Give everything but `master` an R2 token with read permission only.
- Repository secrets are readable by any workflow run on a branch of this repository, so someone
  who can push a branch could change a workflow to use the write pair. To close that, move the
  write pair to an environment restricted to `master` and add `environment:` to the jobs that
  pass it.
- The go command passes its environment to every test binary, so the keys and
  `AWS_ENDPOINT_URL_S3` exist only in the helper's process; the job environment holds only the
  path of the credentials file.
- Release images (`deploy/Dockerfile.tms`, `deploy/Dockerfile.operator`) set an empty
  `GOCACHEPROG` and compile everything themselves.

## The bucket

A lifecycle rule expires objects 14 days after they were written and aborts multipart uploads
after a day. R2 has no last-access expiry, so a hot object also expires after 14 days and the next
`master` build writes it again. R2 has no egress fees; storage is $0.015 per GB-month, writes
$4.50 per million and reads $0.36 per million, a few dollars a month at this size.
