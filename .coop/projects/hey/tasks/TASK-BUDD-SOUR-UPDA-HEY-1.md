---
id: TASK-BUDD-SOUR-UPDA-HEY-1
short_id: 7252052a6387
title: "buddy source update: hey update <id> for source bundles"
type: feature
status: done
created: 2026-07-15
updated: 2026-07-15
aliases: []
priority: p2
track: modules
acceptance:
  - updateBundle routes Kind=source to buddySourceInstall; re-fetch hey.json;
    reinstall when manifest version OR platform-binary sha changed; no-op when
    unchanged (already up to date); shim unchanged (version-agnostic);
    install+update tests green
tests_required: []
---
