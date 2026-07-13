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

Each registered scope carries the publisher's ed25519 **public key(s)** in hey's
registry — the one piece of publisher-specific trust data hey holds:

```jsonc
"scopes": {
  "heypkv": {
    "manifest_url": "https://cdn.heypkv.ai/hey/{id}/{channel}.json",
    "default_channel": "stable",
    "keys": [
      { "id": "hk1", "ed25519": "<base64 32-byte public key>" }
      // multiple keys allowed for rotation; old ids stay verifiable until retired
    ]
  }
}
```

An attacker who owns the CDN still can't forge a signature without the private
key. Trust flows from the pinned key, never from the host.

## What gets signed, and how

Only the **manifest** is signed. Its per-artifact `sha256` values then cover the
bundles transitively — verify the manifest's authenticity, trust its checksums,
download and hash the artifacts. One signature secures everything.

**Envelope (hey-owned, detached).** Alongside a manifest at `…/main/stable.json`,
the publisher also serves `…/main/stable.json.heysig`:

```jsonc
{
  "hey_sig": 1,
  "key_id": "hk1",
  "alg": "ed25519",
  "sig": "<base64 signature over the exact manifest bytes>"
}
```

**Primitive:** Go stdlib `crypto/ed25519` (audited). hey defines the *envelope
and workflow*, never the math. Signature is over the raw manifest bytes (byte-
exact — hey verifies the bytes it will parse, no canonicalization ambiguity).

**Verification (client, automatic):**
1. resolve scope → fetch manifest bytes + `.heysig`;
2. find the scope key whose `id` matches `key_id` (unknown id → untrusted);
3. `ed25519.Verify(pubkey, manifestBytes, sig)` → fail ⇒ refuse, loudly;
4. only now parse the manifest and trust its checksums; download + hash artifacts.

## Commands

Publishers:
- `hey keygen [--out <dir>]` — generate an ed25519 keypair; print the public key
  + `key_id` (first 8 hex of SHA-256(pubkey)) to paste into a registry scope;
  private key saved 0600, never transmitted.
- `hey sign <manifest.json>` — write `<manifest.json>.heysig`.
- `hey verify <manifest.json> [--key <pub>|--scope <name>]` — offline check.

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

## Rotation, revocation, and the immutability seam

- **Rotation:** multiple `keys` per scope; sign new manifests with the new id,
  keep old ids until every referenced release is re-signed.
- **Revocation (seam):** a scope may later carry a `revoked` key-id list, or hey
  may fetch a signed revocation list. Not in v0.
- **Immutability / transparency (seam):** the endgame for zero-trust is not
  trusting hey's own view either. The `hey_sig` envelope reserves room for a
  transparency-log **inclusion proof** (Merkle path) and manifests carry a
  monotonic version, so hey can later require that a manifest be publicly logged
  (append-only, tamper-evident) — and that log can itself be anchored to an
  immutable ledger/chain. Designed-in now, built later.

## Rollout

1. `hey keygen/sign/verify` + the envelope + registry `keys` schema.
2. Automatic verification on install/run for signed scopes.
3. The untrusted-tier consent gate for direct URLs.
4. (later) revocation, transparency-log proofs.

Backward compatibility during rollout: a scope with no `keys` behaves as today
(checksum-only) so nothing breaks before publishers have keys; once a scope
lists keys, signatures become mandatory for it.
