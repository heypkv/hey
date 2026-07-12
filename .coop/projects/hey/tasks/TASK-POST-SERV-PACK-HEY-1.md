---
id: TASK-POST-SERV-PACK-HEY-1
short_id: bb23ab8197e6
title: postgres service pack + hey svc up
type: feature
status: in_review
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: services
acceptance:
  - hey svc up postgres provisions and starts postgres on 127.0.0.1 with
    generated credentials (svc.json 0600)
  - hey svc conn prints a working connection string; psql connect verified on
    Windows
  - data/ survives stop/start and pack version upgrade
tests_required: []
origin:
  authority_refs:
    - docs/feature-set-v1.md
  derived_refs: []
---
