# hey trust & signing v0 (draft for review)

hey's job is to run other people's code. That makes trust the whole product.
This is the design for how hey decides a bundle is genuinely from the publisher
it claims to be — not just intact, but *authentic*. hey owns the protocol end to
end; it depends on no external signing tool.

Status: **draft**. Nothing here is implemented yet — it's the spec for the
`trust` track (publisher signatures, consent tiers).

## Threat model

hey must stay safe when any of these is hostile:

- the **manifest host** (a compromised CDN, a poisoned mirror, a typo-squatted
  URL) serves a bad manifest with a matching bad checksum;
- the **network** (MITM) tampers in flight;
- an **attacker** publishes their own signed bundle and points a victim's hey at
  it.

Checksums alone lose to the first attack — a sha256 only proves "these bytes
match what *this manifest* claimed." Authenticity requires a signature hey can
check against a key it **already trusts, pinned ahead of time**.

## Trust anchor: per-scope pinned keys

Each registered scope carries the pinned ed25519 **public keys** of its trust
parties, plus a **threshold** — how many independent signatures a manifest needs
to be trusted. This is the judge-and-jury model: no single key is the authority;
a verdict needs a quorum.

```jsonc
"scopes": {
  "heypkv": {
    "manifest_url": "https://cdn.heypkv.ai/hey/{id}/{channel}.json",
    "default_channel": "stable",
    "threshold": 1,                 // v0: 1-of-1. Raise to require an M-of-N quorum.
    "keys": [
      { "id": "hk1", "ed25519": "<base64 32-byte public key>", "role": "publisher" }
      // add independent attestor/auditor keys to distribute trust; a manifest is
      // trusted only when `threshold` distinct listed keys have signed it.
    ]
  }
}
```

An attacker who owns the CDN can't forge a signature without a private key; with
`threshold > 1` they can't forge trust without compromising a *majority* of
independent parties. Trust flows from the pinned quorum, never from the host —
and never from a single person.

## What gets signed, and how

Only the **manifest** is signed. Its per-artifact `sha256` values then cover the
bundles transitively — verify the manifest's authenticity, trust its checksums,
download and hash the artifacts. The manifest's signatures secure everything.

**Envelope (hey-owned, detached).** Alongside a manifest at `…/main/stable.json`,
its `…/main/stable.json.heysig` holds a **list** of signatures — one per trust
party — so the same file grows from a single signer to a quorum without any
format change:

```jsonc
{
  "hey_sig": 1,
  "signatures": [
    { "key_id": "hk1", "alg": "ed25519", "sig": "<base64 over the exact manifest bytes>" }
    // independent parties append their own signatures; hey counts distinct valid ones
  ]
}
```

**Primitive:** Go stdlib `crypto/ed25519` (audited). hey defines the *envelope
and workflow*, never the math. Every signature is over the raw manifest bytes
(byte-exact — hey verifies the bytes it will parse, no canonicalization
ambiguity), so parties can co-sign independently, in any order.

**Verification (client, automatic):**
1. resolve scope → fetch manifest bytes + `.heysig`;
2. for each signature, find the scope key with matching `key_id` and check
   `ed25519.Verify(pubkey, manifestBytes, sig)`;
3. count **distinct valid signer keys**; if it's `< threshold` ⇒ refuse, loudly;
4. only now parse the manifest and trust its checksums; download + hash artifacts.

## Commands

Publishers:
- `hey keygen [--out <dir>]` — generate an ed25519 keypair; print the public key
  + `key_id` (first 8 hex of SHA-256(pubkey)) to paste into a registry scope;
  private key saved 0600, never transmitted.
- `hey sign <manifest.json>` — **append** this key's signature to
  `<manifest.json>.heysig` (co-signing: each party runs it independently), so a
  quorum is assembled without any single party holding all keys.
- `hey verify <manifest.json> [--scope <name>]` — offline check that a quorum
  (`threshold`) of the scope's keys have validly signed.

Clients verify automatically on `install`/`run`; no command needed.

## Trust tiers & consent

| Source | Trust | Behavior |
| --- | --- | --- |
| registered scope, valid signature, pinned key | **trusted** | install without prompt |
| registered scope, bad/absent signature or unknown key | **rejected** | refuse |
| direct manifest URL (`hey run https://…`) | **untrusted** | checksum-only (integrity, not authenticity) → require explicit consent or `--allow-untrusted`, with a loud warning |

Before a trusted install, hey shows: publisher (scope), `key_id`, app id +
version, source URL. Least privilege throughout — no install runs as root; the
bundle gets no more access than the user running hey.

## Distributed trust: no single point, and no rewriting history

The threat isn't only a stolen key — it's *any* single point of trust, including
hey believing its own local view. The trust model is therefore a **layered
chain**, each layer removing a concentration of power. v0 ships layer 0; the
format is shaped so the rest slot in without breaking it.

**Layer 0 — single publisher signature (v0, now).** One pinned key, `threshold:
1`. Honest limitation: it proves authenticity + integrity ("the supply side
signed this, and what you received is byte-for-byte what was signed"), but that
one key *is* a single point of trust. It's the floor, not zero-trust.

**Layer 1 — quorum / M-of-N (judge & jury).** Raise `threshold` and pin
independent attestor keys (publisher + auditors +, later, community/ecosystem
parties). A verdict — "this manifest is trusted" — requires a *majority* of
independent signatures. Compromising one, or even a few, keys no longer forges
trust. Already expressible in the schema above; hey just enforces the count.

**Layer 2 — immutable public log (append-only, then anchored).** A quorum you
can't audit is still trust-me. Signed manifests are recorded in a public,
append-only, tamper-evident log (a Merkle transparency log); the `.heysig`
envelope reserves room for a per-signature **inclusion proof**, and manifests
carry a monotonic version so downgrade/rollback is detectable. hey can then
require that a manifest be *publicly logged* before it's trusted — so a
compromised quorum can't sign something in secret; it's on the record for
everyone, forever. That log can itself be **anchored to an immutable ledger /
blockchain** (periodic Merkle-root commitments), making the history
un-rewritable without trusting any single operator — hey included.

- **Rotation:** multiple `keys` per scope; sign new manifests with new ids, keep
  old ids until every referenced release is re-signed.
- **Revocation:** a scope may carry a `revoked` key-id list (later, itself
  signed/logged), so a compromised key is dropped from the quorum count.

The endgame: trust is a distributed verdict, publicly and immutably recorded —
not a secret held by one person, one key, or one server (again, hey included).

## Rollout

1. `hey keygen/sign/verify` + the envelope + registry `keys`/`threshold` schema
   (layer 0: single signer, and the quorum-shaped format).
2. Automatic verification on install/run: count distinct valid signers ≥
   `threshold` (layer 1 works the moment a scope pins more keys + raises it).
3. The untrusted-tier consent gate for direct URLs.
4. (later, layer 2) transparency-log inclusion proofs, ledger anchoring,
   signed revocation lists.

Backward compatibility: a scope with no `keys` behaves as today (checksum-only)
so nothing breaks before publishers have keys; once a scope lists keys,
`threshold`-of-them signatures become mandatory for it.
