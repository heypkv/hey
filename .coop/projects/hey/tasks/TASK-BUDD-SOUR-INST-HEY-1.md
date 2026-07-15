---
id: TASK-BUDD-SOUR-INST-HEY-1
short_id: 3c0c41663a89
title: "buddy source-install: hey.source.v1 in-repo manifest (boss)"
type: feature
status: done
created: 2026-07-15
updated: 2026-07-15
aliases: []
priority: p2
track: modules
acceptance:
  - hey buddy install owner/repo --cred reads repo hey.json via GitHub contents
    API (raw, keeper token), fetches platform prebuilt, verifies sha256,
    installs to apps/<id>/<ver>, records source bundle, PATH shim; boss runs
    directly + hey runner run boss; build semantics declared (exec deferred);
    synthetic + parse tests green
tests_required: []
---
