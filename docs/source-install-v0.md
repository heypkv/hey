# Source install — `hey.source.v1` (v0)

A tool can be installed straight from its own repo, without publishing a release
bundle, by checking a small manifest (`hey.json`) into the repo root. buddy reads
it and installs a native executable — hey never learns the tool's internals.

```
hey keeper auth --name gh-kyive --token-file tok.txt   # once, for a private repo
hey buddy install kyive/boss --cred gh-kyive
boss <args>                # runs directly (PATH shim)
hey runner run boss <args> # equivalent
```

## How it works

1. buddy fetches `hey.json` from the repo via the GitHub **contents API** with the
   keeper token (`Accept: application/vnd.github.raw`). This reads a single file
   from a private repo without cloning the whole monorepo.
2. It picks the `prebuilt` entry for the running platform (`<os>/<arch>`), fetches
   that checked-in binary the same way, and verifies its `sha256` if given.
3. It installs the binary to `~/.hey/apps/<id>/<version>/<id>[.exe]`, records a
   `source` bundle in `meta.json`, and writes a PATH shim next to the `hey`
   binary so `<id> …` works. The shim delegates to `hey runner run <id>`, which
   resolves the current installed version — so it never goes stale on update.

## Manifest

```json
{
  "hey_manifest": "hey.source.v1",
  "id": "boss",
  "version": "0.1.0-alpha.1",
  "prebuilt": {
    "windows/amd64": { "path": "bin/boss.exe", "sha256": "0ca0…d987" },
    "darwin/arm64":  { "path": "bin/boss-darwin-arm64" }
  },
  "build": {
    "toolchain": "go",
    "min_version": "1.26",
    "command": "go build -o {out} ./cli/boss",
    "out": "bin/boss{exe}"
  },
  "launch": { "args": [] }
}
```

- **`prebuilt`** — the working path today: a native executable checked into the
  repo, per platform. `sha256` is optional but recommended.
- **`build`** — declares how to build from source when a toolchain is present.
  **Declared in v1; buddy's build execution ships in a later version.** hey never
  builds on its own: only when the manifest says how *and* the toolchain is
  present *and* the user opts in. (Note: a workspace tool whose `go.work`
  references sibling repos — like boss — won't build from a bare clone; for those,
  the prebuilt is the intended path.)

`{exe}` expands to `.exe` on Windows and empty elsewhere.
