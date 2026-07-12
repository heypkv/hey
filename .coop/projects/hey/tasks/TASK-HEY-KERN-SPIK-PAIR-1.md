---
id: TASK-HEY-KERN-SPIK-PAIR-1
short_id: 4f6731c305f2
title: "hey kernel spike: pairing + origin-gated info + db handoff"
type: feature
status: todo
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: services
acceptance:
  - flag-gated hey up serves GET /hey/info to one allowlisted origin with
    pairing token
  - browser page on allowed origin obtains a provisioned postgres conn; foreign
    origin and no-token requests rejected
  - findings written to docs/kernel-spike-findings.md
tests_required: []
origin:
  authority_refs:
    - docs/hey-kernel-brief.md
  derived_refs: []
---
