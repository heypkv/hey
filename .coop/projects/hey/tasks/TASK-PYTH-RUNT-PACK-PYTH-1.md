---
id: TASK-PYTH-RUNT-PACK-PYTH-1
short_id: 2db0de05c493
title: python runtime pack (python-build-standalone)
type: feature
status: todo
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: services
acceptance:
  - hey run python@3.12 script.py works offline after first fetch,
    checksum-verified
  - runtime packs reuse the artifact pipeline; no service lifecycle
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
