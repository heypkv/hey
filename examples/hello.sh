#!/bin/bash
# hey deploy demo "app" — a stand-in bundle that proves the
# manifest -> download -> verify -> install -> launch path end to end.
# Real apps (e.g. an Electron .app) ship the same way; this is just tiny.
msg="hey installed & launched @demo/hello — manifest -> verify -> run works!"
printf '%s\n' "$msg"
printf '[%s] %s\n' "$(date)" "$msg" >> "$HOME/hey-demo-ran.txt"
# Visible proof on macOS; harmless no-op elsewhere.
if command -v osascript >/dev/null 2>&1; then
  osascript -e "display dialog \"$msg\" buttons {\"Nice\"} default button 1" >/dev/null 2>&1 || true
fi
