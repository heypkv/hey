---
id: TASK-UX-HEY-FLAG-AFTE-APP-1
short_id: 9d43de97a496
title: "UX: hey flags after app name silently pass to the app"
type: feature
status: todo
created: 2026-07-12
updated: 2026-07-12
aliases: []
priority: p2
track: services
acceptance:
  - hey djin ui --no-browser either works (hey consumes its known flags wherever
    placed) or fails with a clear message pointing to flags-before-app usage
  - app exiting on an unknown appended flag surfaces a helpful hint, not just
    'exited before handshake'
tests_required: []
---
