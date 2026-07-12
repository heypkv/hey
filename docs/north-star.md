# North star — the heypkv/kitsy ecosystem

**Modern IT infrastructure, cost-free, on hardware you already own — with a
growth ladder to the cloud when your business earns it.**

Indian MSMEs and small businesses should get the experience of a modern cloud
stack — apps, databases, background services, document workflows, compliance
tooling — without a monthly bill, without vendor lock-in, and without needing
an IT department. The trade they make is the honest one: they bring their own
CPU, memory, and storage. When they outgrow local, heypkv.ai / kitsy.ai
surfaces move the same apps and the same data up the ladder to hosted cloud —
subscriptions fund the growth path, not the entry.

## The pieces

| Layer | What | Status |
|---|---|---|
| **hey** (runner) | Fetch, verify, run, and manage single-binary apps. The only thing a user ever installs by hand. | shipped v0.1 |
| **hey services** (provisioner) | Provision real infrastructure locally — PostgreSQL, cache, object storage, embedded runtimes (Python) — as pluggable, checksum-verified service packs. Your machine becomes a local cloud. | planned (feature-set-v1.md) |
| **hey kernel** (`hey up`) | Loopback daemon exposing local capabilities to *allowed web origins*: provisioned databases, document signing, filesystem blobs. The emSigner pattern done right. | idea (hey-kernel-brief.md) |
| **apps** | guten (documents), djin (compliance), heypkv business apps (accounting, billing, inventory). Go single binaries with embedded web UIs, distributed through hey. | guten shipped; djin next |
| **surfaces** | heypkv.ai / kitsy.ai web apps. Offline-first in the browser (sqlite-wasm/OPFS baseline); upgrade seamlessly when the local kernel is present; move to hosted cloud on subscription. | existing + evolving |

## Principles (every component honors these)

1. **Offline-first, local-first.** Everything works with zero network after
   install. The cloud is an upgrade, never a requirement.
2. **Single static binaries.** No Node, no Java, no installers, no admin
   rights. Windows-first, always cross-platform (linux/darwin, amd64/arm64).
3. **Verify everything.** HTTPS-only downloads, mandatory SHA-256 against
   release checksums, signatures on the roadmap. No unverified execution.
4. **Loopback is not trust.** Local servers bind 127.0.0.1 only; anything a
   browser can reach is origin-gated and pairing-token-guarded.
5. **The user owns the data.** Portable, inspectable formats: SQLite files,
   `pg_dump`-able databases, plain JSON/CSV exports. Leaving must be easy —
   that's why staying is credible.
6. **Small versioned contracts.** App contract v0, registry v0, pack
   manifest v0 — components integrate through documented, versioned seams,
   never through private knowledge.
7. **Lean core, pluggable everything.** hey stays a thin kernel; apps,
   service packs, and runtimes are data-described plugins fetched on demand.

## The growth ladder

```
rung 0  browser only        sqlite-wasm/OPFS, works on any machine, zero install
rung 1  + hey               apps run natively, UIs in browser, all local
rung 2  + hey services      real postgres/cache/runtimes on your hardware
rung 3  + hey kernel        web surfaces use your local infrastructure
rung 4  + kitsy/heypkv cloud sync, backup, multi-device, AI tasks (subscription)
rung 5  hosted               same apps, our hardware, full cloud (subscription)
```

Every rung is a superset; no rung invalidates data or workflows from the rung
below. Migration up (and down) is a product feature, not a support ticket.
