# hey deployment manifest v0 (`hey.deploy.v1`)

This is the contract **hey dictates** for installing and running an application.
hey knows nothing about any product. It knows an app only through a manifest at
a URL. Anyone — heypkv, a third party, you — can publish a conforming manifest
and hey will install, launch, verify, and clean up. There is no product logic in
hey; there is no hey logic in the product.

## How hey resolves an app

`hey install <ref>` / `hey run <ref>` resolves `<ref>` to a **manifest URL**:

- A direct URL: `hey install https://example.com/app/beta.json` — used as-is.
- A scoped id: `hey install @heypkv/main` — the registry maps the **scope** to a
  manifest URL template. That template is the *only* producer-specific data hey
  holds — one line, no product knowledge:

  ```json
  { "scopes": { "heypkv": { "manifest_url": "https://cdn.heypkv.ai/hey/{id}/{channel}.json" } } }
  ```

  `hey install @heypkv/main --channel beta` → GET
  `https://cdn.heypkv.ai/hey/main/beta.json`.

Channels (`stable`, `beta`, `alpha`, `nightly`, …) are opaque names to hey — a
manifest URL segment, nothing more. Default channel is configurable per scope
(else `stable`).

## Manifest schema

```jsonc
{
  "hey_deploy": 1,
  "id": "main",
  "name": "HEYPKV Main",
  "version": "0.1.0-beta.1",
  "channel": "beta",
  "artifacts": [
    {
      "platform": "macos",          // macos | windows | linux | android | ios
      "arch": "arm64",              // arm64 | x64 | universal  (omit for mobile/link)
      "kind": "archive",            // archive | appimage | binary | installer | package | link
      "format": "zip",              // archive only: zip | tar.gz
      "url": "https://cdn.heypkv.ai/assets/apps/main/releases/0.1.0-beta.1/macos/arm64/heypkv-main.zip",
      "sha256": "…64 hex…",         // REQUIRED for every downloadable artifact
      "size": 128472913,            // bytes, optional but recommended
      "launch": { "exec": "HEYPKV Main.app", "args": [] },
      "interface": "window"         // window | hey-contract
    }
  ]
}
```

### `kind` — tells hey the install/launch mechanics (never what the app is)

| kind | hey does |
| --- | --- |
| `archive` | download → verify sha256 → extract (zip-slip/symlink-safe) into the install dir → launch `launch.exec` |
| `appimage` | download → verify → `chmod +x` → run |
| `binary` | download → verify → `chmod +x` → run directly |
| `installer` | download → verify → hand the native installer (`.dmg`/`.exe`/`.pkg`) to the OS; hey does not manage its lifecycle afterward |
| `package` | a device-installable package (e.g. `.apk`). Not a desktop install; consumed by `hey mobile push` (§ nearby devices) |
| `link` | `hey` opens the URL (TestFlight, a store page). No download |

hey picks the artifact whose `platform`+`arch` matches the current machine. If
none matches for a desktop install, it errors clearly.

### `launch` — for `archive` / `binary`

`exec` is the entry to run **relative to the extracted/install root**, resolved
per platform by hey: `open "<exec>"` for a macOS `.app`, execute the `.exe` on
Windows, exec the file on Linux. `args` are appended.

### `interface`

- `window` — a self-windowing GUI app (Electron/native). hey launches it and
  returns; no port handshake. This is what desktop bundles use.
- `hey-contract` — the app speaks [app-contract-v0.md](app-contract-v0.md)
  (prints the `{"hey":1,…}` port handshake, serves `/healthz`). hey tracks the
  port, health-checks, opens the browser. This is what the Go single-binary UI
  apps use. Both models coexist under one manifest schema.

## Install locations & lifecycle

- default: `~/.hey/apps/<id>/<version>/` (cached, reusable).
- `hey run --temp <ref>`: extract to a throwaway dir, launch, **delete on exit**
  (npx-style ephemeral).
- `hey install --location <path> <ref>` / `hey run --location <path>`: install to
  a caller-chosen directory (persistent, outside `~/.hey`).

`hey ls` / `hey which` / `hey cache clean` extend to cover installed apps;
`hey ps` / `hey stop` continue to track `hey-contract` apps.

## Nearby devices (mobile `package` / `link` artifacts)

For `package` (apk) artifacts, hey can push a build to a physically-attached or
same-network device for testing (until a hey mobile client exists):

- `hey mobile devices` — list reachable devices (Android via `adb`; hey may fetch
  a portable adb).
- `hey mobile push <ref> [--device <id>] [--channel <c>]` — resolve the manifest,
  download+verify the `android`/`package` artifact, `adb -s <device> install` it.
- iOS prerelease uses a `link` artifact (TestFlight); `hey open <ref>` opens it.
  (Direct `.ipa` push to a device needs Apple provisioning and is out of scope.)

## Security

Same posture as the rest of hey: HTTPS-only manifest + artifact URLs; **sha256
mandatory** on every downloadable artifact (no skip flag); zip-slip/symlink-safe
extraction (reuses `internal/fetch`); a documented seam for artifact signatures
(the manifest may later carry a `signature` per artifact). hey refuses to launch
an unverified artifact.

## What a producer must publish (the whole integration surface)

To make `hey install @you/<id>` work, a producer publishes, per app + channel:

1. **One platform artifact per target** at an immutable URL, and its **sha256**.
2. **One `hey.deploy.v1` manifest** at a stable (short-TTL) URL naming those
   artifacts.
3. **One registry line** mapping the scope to the manifest URL template.

That is the entire contract. hey never imports producer code; the producer never
imports hey. The manifest is the only interface.
```
