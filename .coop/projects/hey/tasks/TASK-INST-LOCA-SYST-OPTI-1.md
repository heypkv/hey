---
id: TASK-INST-LOCA-SYST-OPTI-1
short_id: 3f0b27deae49
title: "Install location: --system option + least-privilege runtime elevation"
type: feature
status: done
created: 2026-07-13
updated: 2026-07-13
aliases: []
priority: p2
track: trust
acceptance:
  - installer defaults per-user (no sudo); --system installs to a system dir
    (may elevate); document that privileged runtime ops request elevation
    per-action, not by running hey as root
tests_required: []
---
