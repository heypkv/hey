---
id: IDEA-HEY-KERN-LOCA-PROV-1
title: "hey kernel: local provisioner/daemon for browser apps (hey up)"
created: 2026-07-12
aliases: []
author: pkvsi
status: captured
tags: []
source: manual
linked_tasks: []
short_id: 21f9f3b7b153
---

`hey up` starts hey as a long-running loopback daemon that provisions and
serves native capabilities to allowed web origins only (heypkv.ai, kitsy.ai) —
an Ansible/kernel layer for offline-first browser apps. Validated need:
GSTN's emSigner is this exact pattern done badly and universally hated.

Full brief with v0 capability set and security invariants:
[docs/hey-kernel-brief.md](../../../../docs/hey-kernel-brief.md)

Key invariants (non-negotiable): localhost is NOT a security boundary —
strict Origin allowlist + per-origin pairing tokens + DNS-rebinding defenses
+ Chrome Private Network Access preflights; hard per-origin data namespaces;
small versioned capability API. Baseline: migrate browser apps sql.js →
sqlite-wasm/OPFS regardless; hey up is the upgrade tier.
