---
id: TASK-KEEP-IDEM-VAUL-SETU-1
short_id: 6a7c2a6b2e11
title: "keeper: idempotent vault setup + clear passphrase error"
type: feature
status: done
created: 2026-07-15
updated: 2026-07-15
aliases: []
priority: p2
track: modules
acceptance:
  - ensureProject uses real .cnos marker (was .cnosrc.yml, never created); vault
    create tolerant of already-exists; missing-passphrase failure surfaces
    CNOS_SECRET_PASSPHRASE_HEY hint instead of exit status 1
tests_required: []
---
