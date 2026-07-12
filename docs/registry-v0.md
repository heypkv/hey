# hey registry format v0

A registry maps app names to fetchable release artifacts. The default
registry is embedded in the hey binary (`internal/registry/default.json`).
Overrides use the identical format, so a hosted registry (e.g.
`https://heypkv.ai/hey/registry.json`) can replace the embedded one without
any code change.

## Resolution precedence

1. `--registry <path|https-url>` flag
2. `HEY_REGISTRY` environment variable
3. `~/.hey/registry.json`
4. embedded default

Registries do not merge; the first one found wins entirely. URL overrides
must be `https://`.

## Schema

```json
{
  "hey_registry": 0,
  "apps": {
    "guten": {
      "description": "guten templating engine CLI (kitsy.ai)",
      "source": {
        "type": "github-release",
        "repo": "kitsyai/guten",
        "tag_prefix": "cli/",
        "asset_template": "guten_{version}_{os}_{arch}.{ext}",
        "checksums_asset": "checksums.txt",
        "binary": "guten"
      },
      "ui_commands": []
    }
  }
}
```

- `hey_registry` — format version. hey rejects registries newer than it
  understands ("update hey"); unknown fields are ignored.
- App names must not collide with hey's reserved subcommands (`run`,
  `install`, `update`, `ls`, `ps`, `stop`, `which`, `cache`, `version`,
  `help`) and must not contain `/ \ @` or spaces.
- `source.type` — only `github-release` today; other types are rejected
  per-app with a clear message (the seam for future source kinds).
- `tag_prefix` — for monorepos whose release tags are prefixed (guten tags
  `cli/v0.2.7`). The release tag is always `tag_prefix + "v" + version`.
- `asset_template` — placeholders `{version}` (no leading `v`), `{os}`
  (GOOS), `{arch}` (GOARCH), `{ext}` (`zip` on windows, `tar.gz` elsewhere —
  the goreleaser `format_overrides` convention).
- `checksums_asset` — the goreleaser SHA-256 checksums file
  (`<sha256>  <filename>` lines). Verification is mandatory; there is no
  skip flag.
- `binary` — the executable's base name inside the archive (`.exe` is
  appended automatically on Windows).
- `ui_commands` — subcommands that follow the UI contract
  (docs/app-contract-v0.md). Everything else is plain passthrough.

## Version resolution

hey never uses GitHub's `/releases/latest` (blind to tag prefixes). It lists
releases, filters by `tag_prefix + "v"`, skips drafts and prereleases, and
picks the semver maximum. Results are cached for 24h in
`~/.hey/state/resolve.json`; `hey update` bypasses the cache. Pinned versions
(`hey app@1.2.3`) and already-cached binaries make zero API calls.

## Reserved for later

`minisign_pubkey` (release signature verification) and `homepage` are
documented but unused in v0. Checksums protect download integrity, not
publisher authenticity — signatures are the v1 roadmap item.
