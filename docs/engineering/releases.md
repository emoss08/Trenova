# Releases

A Trenova release has one version (`0.13.1`): it is what staff pick in the control plane and what an
instance runs. Inside it, each image carries the version it was last **built** at. A release builds
an image only when that image's inputs changed since the previous release; every other image keeps
its digest, and gets the release's tags as aliases of it. A release that changed only the TMS builds
`tms` and `tms-db` in minutes instead of rebuilding imaging for 75, and an instance that takes it
restarts the API and the worker and nothing else.

| Piece | Where |
|---|---|
| the images and what each is built from | `deploy/release/images.yaml` |
| the plan, the manifest, the input check | `tools/release/release.py` (tests: `tools/release/test_release.py`) |
| the workflow | `.github/workflows/release.yml`, one image's build in `.github/actions/release-image` |
| the charts' side | `deploy/helm/trenova/templates/_helpers.tpl` (`trenova.image`, `trenova.tmsVersion`), `release-job.yaml`; `deploy/helm/trenova-operator/templates/_helpers.tpl` |
| the chart test | `tools/helm/check-release-renders.py`, run by `helm.yml` |

## Cutting a release

From an up-to-date default branch (`master`), with a clean tree:

```bash
tools/release.sh 0.13.1
```

It tags `v0.13.1`, pushes the tag and publishes the GitHub release, which starts `release.yml`. A
release is refused unless its commit is on the default branch and its tag is `vX.Y.Z`.

The workflow:

1. **plan** (`release.py plan`) finds the previous release (the newest `vX.Y.Z` tag older than this
   one that the commit contains), reads its `release-manifest.json` from its GitHub release, and
   decides, per image, **build** or **reuse**. The run's summary is a table of every image, the
   decision and why.
2. **build** builds every image the plan builds, at once, one runner each
   (`release.py image` gives each its Dockerfile, context, target, platforms and build arguments).
   **build-late** then builds what is built FROM another image of the release (the export runner,
   from PostgreSQL).
3. **alias** gives every reused image this release's tags with `docker buildx imagetools create`:
   a registry write of seconds, no build.
4. **manifest** reads every image's digest back from the registry, checks reused and pinned ones
   against the plan, and writes and validates `release-manifest.json`.
5. **publish-chart** packages the manifest into both charts, pins the third-party images, signs the
   charts with Cosign (keyless, the workflow at the tag) and pushes them.
6. **publish-manifest** uploads `release-manifest.json` to the GitHub release (replacing it on a
   re-run). It goes last: the control plane lists a release once its manifest is there, and by then
   its charts exist.

The tags every image gets are the version, `major.minor` and `latest`, the last two only while no
newer release has them, so re-running an old release never moves them back.

### When an image is built

An image is built when any of these holds, and reused otherwise:

- a path under its `inputs` changed between the previous release's tag and this one;
- there is no previous release;
- the previous release's manifest has no entry for it, or the digest it names is no longer on the
  registry;
- an image it `dependsOn` is built (the export runner is built whenever PostgreSQL is);
- a re-run's `rebuild` names it, or says `all`.

A reused image keeps the `version` it was built under: after 0.13.1 rebuilt only the TMS, its
manifest says `imaging` is `0.13.0`. `gtc` and `embedding` are not built by a release (GTC by its
own repository, embedding by `embedding-image.yml`): their entries are the tags the chart pins in
`values.yaml`, resolved to digests, and that tag must be on the registry before the release.

**The first release with a manifest.** Releases up to 0.13.0 published no manifest. The release
after one resolves each unchanged image's previous digest from the registry by the previous
release's tag (`<repository>:0.13.0`), and that tag becomes its `version`. PostgreSQL's tag lagged
the releases, so its previous image is the newest published `X.Y.Z` tag that is not newer than the
previous release. `changed` is then computed against those digests.

### Retrying and forcing a rebuild

A run whose jobs failed: open it and use **Re-run failed jobs**. The plan's outputs are kept, so
only the failed builds run again.

To build an image whose inputs did not change (a base image's security fix, a broken push), run the
release again **on its tag**, with `rebuild` naming the images (comma-separated names from
`images.yaml`) or `all`:

```bash
gh workflow run release.yml --ref v0.13.1 -f rebuild=imaging
gh workflow run release.yml --ref v0.13.1 -f rebuild=all
```

`--ref v0.13.1` is required: the chart's signature names `release.yml` at a `refs/tags/v…` ref,
and the operator refuses any other signer, so a run dispatched on a branch is refused by the
workflow before it builds anything. A release event cannot be fired again with inputs, which is
why the dispatch exists. `rebuild: all` also looks every third-party image the charts name up
again (below). The re-run re-publishes the charts and the manifest at the same version; instances
already on that version do not take the new chart (the operator applies a version once), so a
rebuild that must reach instances is a new patch release.

## The manifest

`release-manifest.json` is the contract between a release and everything that installs one: the
control plane's catalogue, a shared server's `update` stage and the Trenova chart. Its shape is fixed
by the control plane's release contract; `release.py` writes and validates it.

```json
{
  "schemaVersion": 1,
  "version": "0.13.1",
  "commit": "<40 lowercase hex>",
  "previous": "0.13.0",
  "migrations": true,
  "images": {
    "tms":      {"repository": "ghcr.io/emoss08/trenova/tms", "version": "0.13.1", "digest": "sha256:…", "changed": true},
    "imaging":  {"repository": "ghcr.io/emoss08/trenova/imaging", "version": "0.13.0", "digest": "sha256:…", "changed": false}
  }
}
```

- `version`, `previous`: `X.Y.Z`; `previous` is absent on the first release.
- `images.<name>.version`: the tag the digest was built under; `digest`: the multi-platform index's
  digest; `changed`: the digest differs from the previous release's digest for that name (true
  when there is no previous release, or no previous digest).
- `migrations`: `services/tms/schema/migrations/` changed since `previous` (true without one).
- Readers ignore fields they do not know, tolerate image names they do not know, and refuse a
  `schemaVersion` they do not know (the charts `fail` on one).

`release.py validate <file>` checks a manifest by hand.

## No restart without a change

The charts are what keep an unchanged image from restarting.

- **Images.** A Trenova image the packaged manifest names renders as
  `<repository>:<its own version>@<digest>`, so a release that carried a digest forward renders the
  same pod template and Kubernetes leaves the pods alone. The tag is the image's own version, never
  the release's: it changes exactly when the digest does, and the operator's telemetry reads a
  part's version from it. An image the manifest does not name (third-party ones) renders as
  `<repository>:<tag>@<digest>` from `image-digests.yaml`; an override to another repository or
  tag finds no entry and is used as given; an unreleased chart renders tags alone.
- **Third-party digests.** `image-digests.yaml` pins every third-party image a chart names
  (Redis, Meilisearch, Temporal, …). A tag such as `redis:8` floats upstream, so the publish step
  takes the digest the previous release's signed chart pinned for the same `repository:tag`,
  and looks a tag up only when it is new. Moving to a newer upstream image is a change to its tag
  in `values.yaml`, or a `rebuild: all` re-run.
- **The TMS's version.** `app.version` in the TMS configuration is the `tms` image's version
  (`trenova.tmsVersion`), so the configuration, and the checksum on the API's and worker's pods,
  change only with the TMS. The release's version stays visible where it changes nothing: the
  resources' own `app.kubernetes.io/version` and `helm.sh/chart` labels and Helm's records. A
  pod template never names the release.
- **The release Job** (the database's roles, `trenova-db up`, the RLS roles, GTC's publication,
  the bootstrap or import) carries `trenova.app/release-inputs`, a hash of its whole spec: its
  images by digest (`tms-db`'s among them), the configuration's checksum and the checksum of the
  scripts it runs. An upgrade that leaves the hash alone renders the Job that already ran, under
  its name (found with `lookup`), so Helm changes nothing and the migration does not run; a release
  that did not rebuild `tms-db` runs no migration. A new hash, or a Job with the same hash that
  failed, gets a new Job named after the revision, which Helm creates while it deletes the one it
  replaces. A rollback applies the revision's stored Job, which runs if it is not the one there.
  The ordering is unchanged: the API and the worker wait for their role and their schema version
  themselves, so new pods never serve before a migration they need.
  `helm template` has no cluster to look in and always names the Job after the revision. The
  operator reads the Job's name from the deployed release's manifest
  (`release.Deployed.ReleaseJob`), never from the revision.
- **The operator's pod** names no release either. Its version is the
  `trenova.app/operator-version` annotation on its Deployment, read on every tick; while a new pod
  rolls out, the old one still runs the version it moved from
  (`trenova.app/operator-upgrade-from` on its HelmChart). A release that did not rebuild the
  operator moves its chart, and the same process runs the new release
  ([operator.md](operator.md#its-own-version)).

`tools/helm/check-release-renders.py` renders three releases (everything built, only `tms` and
`tms-db`, only `client`) in the default and managed shapes and fails when anything but what was
rebuilt renders differently, when the TMS's configuration changes without the TMS, or when the
release Job would run without a change to what it runs. `helm.yml`'s k3d install also upgrades
twice: once changing the Job (it runs) and once changing nothing (the Job and the API's generation
stay).

## Adding an image

1. Add it to `deploy/release/images.yaml`: repository, Dockerfile, context, target, platforms,
   build arguments and every input. An image whose Dockerfile copies its whole context needs an
   allowlist `<Dockerfile>.dockerignore` (`*`, then `!path` for each thing it reads); its inputs
   are exactly those paths.
2. Run the check and the tests:

   ```bash
   python3 tools/release/release.py check-inputs
   python3 -m unittest discover -s tools/release
   ```

   `check-inputs` (CI, in `helm.yml`) fails when a Dockerfile copies a path its entry does not
   list, a dockerignore lets in a path that is not an input, an input does not exist, or a
   workflow the release runs builds a Dockerfile the list does not name. Never add a
   `docker/build-push-action` step to `release.yml`: the `build` matrix builds every entry.
3. If the chart runs it, name it in `values.yaml` with its repository and an empty tag, and it
   renders from the manifest like the others. Tell the control plane's maintainers the new name and
   the words people are shown for it; readers tolerate a name they do not know until then.

Changing what an image is built from (a new `COPY`, a new directory) is a change to its entry in
the same commit; the check fails otherwise.
