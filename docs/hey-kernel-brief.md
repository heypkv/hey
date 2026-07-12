# hey kernel (`hey up`) — planning brief

Status: idea (IDEA-HEY-KERN-LOCA-PROV-1). Not scheduled; guten ui and djin
GSTR-1 come first.

## Problem

heypkv.ai / kitsy.ai business apps (accounting, billing, inventory for
Indian MSMEs) are offline-first in the browser. Browser storage (sql.js
today) caps the experience: in-memory DBs with manual persistence, no
native filesystem, no background work, and no access to hardware like DSC
USB tokens.

## Idea

`hey up` runs hey as a long-running loopback daemon — a provisioner/kernel
for browser apps. The web app detects it and upgrades itself:

1. App probes `http://127.0.0.1:<well-known-port>/hey/info`.
2. If absent, UI offers: "Better offline experience? Install hey and run
   `hey up`."
3. If present, the origin pairs with the daemon and asks it to provision
   capabilities (e.g. a real SQLite DB); the app then talks to local
   native services instead of in-browser fallbacks.

Validation from the field: GSTN's emSigner is exactly this pattern
(localhost daemon bridging browser → native crypto) executed terribly.
The pattern is proven-necessary; the bar is low; the ecosystem fit
(djin + DSC signing) is exact.

## v0 capability set

- **storage** — provision origin-scoped SQLite databases; query over
  HTTP/WebSocket. Browser baseline remains sqlite-wasm + OPFS (that
  migration from sql.js is worth doing independently of hey up).
- **info/discovery** — version, capabilities, pairing state.

## Later capabilities

- **crypto bridge** — USB-token DSC signing via the OS certificate store;
  the one thing browsers cannot do, and djin's killer feature.
- **fs blobs** — origin-scoped document storage (invoices, filings).
- **sync** — background sync to kitsy/heypkv cloud with user's
  subscription.

## Security invariants (design-in from day one, non-negotiable)

1. **localhost is not a security boundary.** Any website can attempt
   requests against 127.0.0.1 (drive-by probing, DNS rebinding). Required:
   strict `Origin` allowlist (heypkv.ai, kitsy.ai), per-origin pairing
   tokens issued through an explicit user consent step, Host-header
   validation, and Chrome Private Network Access preflight support
   (`Access-Control-Request-Private-Network`).
2. **Hard per-origin namespaces.** One origin never sees another origin's
   databases, blobs, or keys. Namespace = exact origin string.
3. **Small versioned capability API**, spec'd like the app contract
   (`docs/app-contract-v0.md`) with an explicit version handshake.
4. Loopback bind only; no LAN exposure, ever.

## Fit with existing hey

Reuses: home layout (`~/.hey`), state tracking, contract discipline,
loopback-only rules, the installer + self-update story. `hey up` becomes a
reserved subcommand (registry validation already forbids app-name
collisions; add `up` to the reserved list before v1 of the kernel).
