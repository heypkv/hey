---
id: TASK-PACK-MANI-V0-SPEC-AR-1
short_id: 4a257768754e
title: Pack manifest v0 spec + archive-exec driver
type: feature
status: done
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: services
acceptance:
  - docs/pack-manifest-v0.md defines the schema (versions, per-platform
    artifacts+sha256, init/start/ready/conn/stop templates)
  - "internal/svc driver: download/verify/extract via existing fetch pipeline,
    templated init-once, start/stop/health lifecycle"
  - unit tests with a fake pack; no postgres specifics in driver code
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
