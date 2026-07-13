---
id: TASK-PUBL-SIGN-VERI-PER-1
short_id: eb8a02d3c312
title: Publisher signature verification (per-scope trust anchor)
type: feature
status: todo
created: 2026-07-13
updated: 2026-07-13
aliases: []
priority: p2
track: trust
acceptance:
  - each registry scope carries a publisher public key; manifests are signed;
    hey verifies the manifest signature against the scope key BEFORE trusting
    any artifact/checksum in it
  - fills the existing fetch.Verify SigSpec seam; tampered manifest or wrong key
    is rejected in tests
tests_required: []
---
