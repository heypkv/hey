# hey feature set v1 — from runner to provisioner

Context: [north-star.md](north-star.md). hey v0.1 ships fetch/verify/run for
apps. v1 adds **services** (local infrastructure provisioning) and
**runtimes** (embedded interpreters), keeping the core lean: lifecycle logic
lives in a handful of Go *drivers*; everything specific to postgres or python
lives in data-only *packs*.

## 1. hey services (`hey svc`) — the provisioner

Turn the user's machine into a local cloud: real PostgreSQL, cache, object
storage — downloaded from official upstreams, checksum-verified, managed like
UI apps are today.

### CLI surface

```
hey svc up <pack>[@version] [--name <instance>]   # provision + start (idempotent)
hey svc ls                                        # instances: pack, version, port, state, data size
hey svc stop <instance> | start <instance>
hey svc logs <instance> [--tail N]
hey svc conn <instance>                           # print connection string / env exports
hey svc rm <instance> [--purge-data]              # --purge-data requires confirm
```

### Pack manifest v0 (data, not code)

A pack describes one service using a driver the hey binary already ships.
Packs live in the registry (embedded defaults + hosted later), same trust
pipeline as apps: https-only, pinned SHA-256 per platform artifact.

```jsonc
{
  "pack": "postgres",
  "driver": "archive-exec",              // the generic service driver
  "versions": {
    "16.4.0": {
      "artifacts": {                     // per {os}_{arch}: url + sha256
        "windows_amd64": { "url": "https://...zip", "sha256": "..." },
        "linux_amd64":   { "url": "https://...tar.gz", "sha256": "..." }
      }
    }
  },
  "init":  [ "{bin}/initdb -D {data} -U {user} --pwfile {pwfile} -E UTF8" ],
  "start": "{bin}/postgres -D {data} -p {port} -k \"\"",
  "ready": { "tcp": "127.0.0.1:{port}" },
  "conn":  "postgresql://{user}:{password}@127.0.0.1:{port}/postgres",
  "stop":  { "signal": "term", "grace_seconds": 20 }
}
```

Drivers in the hey binary (v1 needs only the first):
- **archive-exec** — download/verify/extract archive, run templated
  init-once commands, start/stop a foreground process, health-check it.
- (later) **container-less composites** — packs that depend on other packs.

### Candidate packs (v1 picks two)

| Pack | Upstream | Notes |
|---|---|---|
| **postgres** | official/EDB binary archives | the flagship; MSME apps standardize on it |
| **python** | astral python-build-standalone | relocatable, checksummed — ideal (see §2) |
| garnet or valkey | Microsoft Garnet / Valkey | redis-compatible cache; Garnet has native Windows builds |
| minio | MinIO single binary | S3-compatible object storage, trivially fits the model |

### Instance layout & invariants

```
~/.hey/svc/<instance>/
  bin/            verified binaries (per pack version)
  data/           the user's data — NEVER touched by upgrades or cache clean
  logs/service.log
  svc.json        pack, version, port, generated credentials (0600), state
```

- Ports allocated from a local range, recorded in `svc.json`; stable per
  instance across restarts.
- Credentials generated at init, never defaulted, file mode 0600.
- Services bind 127.0.0.1 only — enforced by pack review, not user config.
- `hey svc` state is separate from app state; `hey ps` gains a services
  section.

## 2. hey runtimes — embedded interpreters

`hey run python@3.12 script.py` — fetch a relocatable Python (or Node)
runtime as a pack, verify, cache, exec with args passed through. Enables:
automation scripts for MSME workflows, app extension hooks, data
import/export jobs — without "install Python" ever appearing in a doc.

Runtimes are just packs with `"kind": "runtime"` and no long-running
lifecycle — the existing passthrough runner handles them once the artifact
pipeline knows how to fetch them.

## 3. hey kernel (`hey up`) — bridge to web surfaces

Spec'd in [hey-kernel-brief.md](hey-kernel-brief.md). v1 scope here is only
the **spike**: pairing-token flow + origin-gated `GET /hey/info` +
provisioned-postgres handoff to one allowed origin, behind a flag. Full
kernel ships after djin proves the DSC story.

## 4. Already-filed distribution work

Signing (minisign + Authenticode), winget/scoop/brew, self-update — see the
`distribution` track. These harden the trust chain §1 depends on.

## Sequencing

1. **svc-core**: pack manifest v0 + archive-exec driver + `up/ls/stop/logs/conn/rm` (postgres pack as the proving target)
2. **python runtime pack** (unlocks scripting stories, exercises the runtime kind)
3. **second service pack** (garnet or minio — proves pluggability, no driver changes allowed)
4. **kernel spike** (flag-gated)
