---
id: TASK-SVC-CLI-LS-STOP-STAR-1
short_id: cdbb0a5def65
title: "svc CLI: ls/stop/start/logs/rm"
type: feature
status: todo
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: services
acceptance:
  - hey svc ls shows instance, pack, version, port, state
  - rm refuses to delete data/ without --purge-data + confirmation
  - stop is graceful (pack stop spec) with force fallback
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
