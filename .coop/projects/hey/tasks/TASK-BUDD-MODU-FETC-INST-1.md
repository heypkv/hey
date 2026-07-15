---
id: TASK-BUDD-MODU-FETC-INST-1
short_id: be33580aa119
title: "buddy module: fetch/install native bundles + authenticated clone"
type: feature
status: done
created: 2026-07-15
updated: 2026-07-15
aliases: []
priority: p2
track: modules
acceptance:
  - hey buddy install <ref> --cred (private bundle via keeper, Bearer auth
    threaded through deploy fetch); hey buddy clone <repo> --cred [--build]
    (authenticated git clone, opt-in build only); never builds by default; auth
    header tested; real clone smoke green
tests_required: []
---
