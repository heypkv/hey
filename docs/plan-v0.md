# hey plan v0 (draft for review)

A **plan** turns an *intent* ("scan my local network", "file GSTR-1") into a
deterministic sequence of tool invocations. This is the orchestrator layer: hey
still does no domain work itself — a plan just says *which tools, in what order,
with what inputs, and when to ask the user*. It's the thing that makes hey feel
like an assistant while staying a lean, auditable runner.

Status: **draft**. Nothing implemented yet — this is the contract to agree on.

## Three layers of authorship (same executor for all)

1. **Pre-authored plans (free/deterministic).** Hand-written recipes hey ships
   or fetches. hey runs a *known* plan step by step. No AI, fully auditable.
2. **AI-composed plans (paid).** When no plan fits, an AI layer selects tools
   from the registry and emits a plan **in this same format**. AI proposes; the
   deterministic, consent-gated executor disposes. Intelligence never bypasses
   the audited path.
3. **User plans.** You can write and run your own `.plan.json`.

## Plan schema (`hey.plan.v1`)

```jsonc
{
  "hey_plan": 1,
  "intent": "scan-local-devices",
  "description": "Find devices on the local network.",
  "inputs": [
    { "name": "subnet", "prompt": "Subnet to scan", "default": "auto" }
  ],
  "steps": [
    {
      "id": "scan",
      "tool": { "system": "nmap" },            // a DETECTED system tool (never auto-installed)
      "sensitive": true,                        // scanning a network needs consent
      "run": ["-sn", "{{ inputs.subnet }}"],
      "capture": "text"                          // text | json | none
    },
    {
      "id": "report",
      "tool": { "app": "@kitsy/netreport" },     // a REGISTRY tool: installed on demand, trust-verified
      "run": ["--from", "{{ steps.scan.output }}"],
      "capture": "json"
    }
  ],
  "output": "report"                             // the plan's result = this step's capture
}
```

### Tool references (the trust boundary)

- `{ "app": "@scope/id" }` — a **registry tool**: resolved, signature-verified,
  and installed on demand through the exact trust pipeline
  ([trust-and-signing-v0.md](trust-and-signing-v0.md)). Fully trusted.
- `{ "system": "nmap" }` — a **detected system tool** already on the machine.
  hey **never auto-installs** system packages (no silent `apt`/`brew`). If it's
  missing, hey says so and stops with an install hint. Using one is always
  `sensitive` → consent required, because hey can't vouch for it.

That split is deliberate: hey vouches for what it installs (verified), and asks
permission for what it merely borrows.

### Templating & data flow

`{{ inputs.<name> }}` and `{{ steps.<id>.output }}` interpolate into a step's
`run` args. Outputs are captured as text or parsed JSON. No shell — args are
passed directly to the tool (no `sh -c`, no injection surface).

## Execution semantics

1. Resolve inputs (prompt for any without a default/flag).
2. For each step in order:
   - resolve the tool (install+verify a registry tool; detect a system tool);
   - if `sensitive`, show exactly what will run and get consent (unless
     `--yes`);
   - run it with templated args, least privilege (never root; no more access
     than the user), capture output;
   - a non-zero exit stops the plan (unless the step is marked `continue`).
3. The plan's result is `output`'s captured value.

`hey do <intent> [--param k=v] [--yes]` runs a plan; `hey plan list` shows
available plans; `hey plan show <intent>` prints what it will do *before*
running (dry-run transparency).

## Where plans come from

A **plan library**, mirroring the app registry: embedded defaults + fetchable
sets from a scope. Seed plans: `scan-local-devices`, `file-gstr`
(`@heypkv/djin gstr1 build …`), `make-invoice` (`@kitsy/guten batch …`).

## Security — plans are powerful, so they're trusted too

A plan chains tools; a malicious plan could chain *trusted* tools destructively.
So a plan is only as trusted as its source:

- Plans from a **signed scope** (same `.heysig` + quorum mechanism as manifests)
  run without a trust prompt.
- A plan from a **direct URL / unsigned source** is untrusted → `--allow-untrusted`
  + a per-sensitive-step consent gate.
- Every `sensitive` step always shows-and-asks, trusted plan or not. Consent is
  never fully delegated for network scans, installs, or filesystem writes
  outside `~/.hey`.

The AI planner (paid) is bound by the same rules: it can only reference tools the
user's registry trusts, and its plan runs through the same consent gates.

## Open questions for review

1. **Plan trust** — reuse the manifest signing/quorum for plans verbatim (my
   recommendation), or a separate mechanism?
2. **System tools** — detect-only with consent (my recommendation), or ever
   allow hey to install a system tool via a package manager behind a big prompt?
3. **CLI verb** — `hey do <intent>` (natural) vs `hey plan run <intent>`?
4. **Scope of v0** — I'd build: the schema, a deterministic executor,
   registry+system tool resolution, consent gates, `hey do/plan list/plan show`,
   and 1–2 real seed plans. AI planning and the fetchable plan library come after.
