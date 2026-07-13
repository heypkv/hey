# Publishing a signed bundle (`@heypkv` / `@kitsy`)

How a publisher ships an app that `hey install @scope/id` installs **trusted**.
The full contract is [deployment-manifest-v0.md](deployment-manifest-v0.md) +
[trust-and-signing-v0.md](trust-and-signing-v0.md); this is the checklist.

A working, signed example lives in `examples/`
(`hello.stable.deploy.json` + `.heysig`, pinned in `demo-registry.json`).

## One-time: keys and the scope

1. On a **secure machine**, generate a signing key per trust party:
   ```
   hey keygen
   ```
   The private key is saved 0600 under `~/.hey/keys` — never share or commit it.
   For a quorum (recommended), each independent attestor runs this on their own
   machine.
2. **Pin the public key(s)** in your scope's registry entry and set `threshold`
   (how many must sign). For `@heypkv`/`@kitsy` this is the scope block in the
   hosted registry (or hey's default registry via PR):
   ```jsonc
   "kitsy": {
     "manifest_url": "https://cdn.kitsy.ai/hey/{id}/{channel}.json",
     "default_channel": "stable",
     "threshold": 2,                    // e.g. publisher + one auditor
     "keys": [
       { "id": "<from keygen>", "ed25519": "<from keygen>", "role": "publisher" },
       { "id": "<auditor id>",  "ed25519": "<auditor key>", "role": "auditor" }
     ]
   }
   ```
   A scope with no `keys` stays untrusted (checksum-only); the moment you pin
   keys, signatures become mandatory for it.

## Per release

3. **Build the artifact per platform.** For a desktop (Electron) app this is a
   real native build — macOS builds the `.app`/`.dmg`, Windows the portable
   zip/installer, Linux the AppImage. **This build must run on the target OS or
   in CI** (a macOS runner for the mac artifact); it can't be produced from a
   Windows box. The webapp's `desktop/main` packaging config + CI are set up for
   this; it's the one step gated on real hardware.
4. **Generate the manifest.** Run the producer generator
   (`scripts/generate-hey-manifest.mjs` in the webapp) to emit a `hey.deploy.v1`
   manifest with each artifact's URL, `sha256`, and `launch`/`interface`.
5. **Publish** the immutable artifacts and the manifest to the CDN paths the
   scope's `manifest_url` resolves to.
6. **Sign the published manifest.** Each trust party runs, against the exact
   published bytes:
   ```
   hey sign kitsy-main-stable.json          # appends this key's signature to .heysig
   ```
   Collect a `threshold` quorum, then publish `<manifest>.heysig` next to the
   manifest.

## Result

```
hey install @kitsy/main        # resolve scope -> fetch manifest + .heysig ->
                               # verify quorum against pinned keys -> install -> run
```

hey prints `verified @kitsy — signed by <ids> (N of M required)` and installs
with no `--allow-untrusted`. A tampered manifest, a missing quorum, or an
unpinned signer is refused. Verify locally any time with:

```
hey verify kitsy-main-stable.json --scope kitsy
```
