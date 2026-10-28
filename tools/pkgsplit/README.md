# pkgsplit

Unshipped tooling for splitting a Go package in `services/tms` into smaller packages, as the
small-packages rule in [go.md](../../docs/engineering/backend/go.md#small-packages) asks. Every
step works on the AST, never on regular expressions over source, and each split is one
refactor commit.

Build the tools once (outside `go.work`):

```bash
cd tools/pkgsplit
export PKGSPLIT_WORK=/tmp/pkgsplit PKGSPLIT_BIN=/tmp/pkgsplit/bin
GOWORK=off go build -o $PKGSPLIT_BIN/ ./...
```

| Tool | Does |
|---|---|
| `declgraph <dir>` | prints a package's declarations (`N name file lines`, methods merged into their type) and references between them (`E from to`) |
| `scripts/closure.py <graph> <Root,...>` | the core: everything the roots need transitively, by file |
| `scripts/scc.py <graph> <Root,...>` | reference cycles outside the core that span files; they must share a package |
| `scripts/mixedfiles.py <graph> <Root,...>` | files that are all core, all outside, or mixed |
| `scripts/owners.py <graph> <file>` | which files use each declaration of a file |
| `scripts/plan.py <graph> <map> <core> <mixed files...>` | assigns each declaration in a mixed file to a package and prints the `declsplit` commands |
| `declsplit -dir D -out new.go Name Type Type.Method ...` | moves declarations (a type takes its methods) into a new file in the same package |
| `pkgmove -root services/tms -src <pkg> -map <file-to-package map> [-n]` | moves files into sibling packages, qualifies cross-package references in moved files, rewrites `pkg.X` selectors and imports in every importer, moves `<pkg>mock` mocks with their interfaces; refuses unexported cross-package references, import cycles and methods separated from their type |
| `pkgmove -root services/tms -src <old pkg> -symmap <Symbol newImportPath lines>` | rewrite-only: points `old.X` selectors at new packages (generated `<entity>db` tables after `keel gen`) |
| `scripts/ledger.py <old> <new...>` | appends every moved symbol to `docs/modules/symbol-moves.tsv` and repoints older entries |
| `scripts/textrefs.py <old> <new...>` | rewrites `old.X` in yaml and markdown (`gqlgen.yml`, docs); never the ledger |
| `scripts/mockery_add.py .mockery.yaml <prefix> <pkg...>` | adds new ports packages to `.mockery.yaml` |
| `scripts/fmtchanged.py services/tms <changed list>` | normalizes imports (standard library, then everything else) and runs `golines -m 100` on hand-written changed files |

A split:

1. `declgraph`, then `closure.py` and `mixedfiles.py` with the package's main type as the root.
2. Cut mixed files with `declsplit` (or `plan.py` to plan it). Replace private copies of
   `shared/` helpers instead of exporting them.
3. Write the map (`file package`, one per line; test files go with what they test) and run
   `pkgmove -n` until it is clean. A cycle caused by a core test that reaches moved code is fixed
   by moving the test or making it an external `_test` package.
4. Run `pkgmove`, then `ledger.py`, then `textrefs.py`, then `mockery_add.py` for ports, then add
   new domain packages to `docs/modules/module-map.yml`.
5. For a domain package, run `task keel-gen`, then `pkgmove -symmap` to point `<old>db.X` at each
   new `<pkg>db`.
6. `fmtchanged.py`, commit, push; CI compiles it. Never `go build ./...` locally.
