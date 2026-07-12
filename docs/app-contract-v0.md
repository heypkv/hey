# hey app contract v0

Apps runnable by `hey` are single, self-contained binaries. Ordinary
subcommands need nothing special — hey runs them in the foreground with
inherited stdio and propagates the exit code.

A **UI command** is a long-running subcommand listed in the app's registry
entry under `ui_commands` (e.g. `djin ui`). UI commands must implement this
contract so hey can start, track, and stop them.

## Invocation

hey launches `<binary> <ui-command> [user args...]`, appending `--port 0
--json` unless the user already supplied `--port` or `--json`. hey also sets
the environment variables `HEY=1` and `HEY_CONTRACT=0`.

Apps MUST accept:

- `--port <n>` — TCP port to bind; `0` means ephemeral (OS-assigned).
- `--json` — enables the machine-readable handshake below.

## Handshake

With `--json`, after the HTTP listener is **bound**, the app MUST print
exactly one line to stdout: a single JSON object, at most 4096 bytes,
terminated by `\n`:

```json
{"hey":1,"name":"djin","version":"0.1.0","url":"http://127.0.0.1:52341","pid":4242,"port":52341}
```

- Required fields: `hey` (int, handshake version, currently `1`), `name`,
  `url`. Optional: `version`, `pid`, `port`.
- `url` MUST be loopback http (`http://127.0.0.1:<port>` or
  `http://localhost:<port>`). Never bind non-loopback interfaces under this
  contract.
- The app MUST flush stdout immediately. hey captures stdout in a log file,
  not a tty; runtimes that block-buffer file output (e.g. C stdio, Python)
  must flush explicitly or the handshake never arrives. Go's `os.Stdout` is
  unbuffered and safe.
- Nothing else may be written to stdout before the handshake. After it, all
  logging goes to stderr.

## Health

The app MUST serve `GET {url}/healthz` → `200` (any body) within 30 seconds
of the handshake, and for its whole lifetime. hey uses it for liveness (it is
also the defense against PID reuse).

## Shutdown (optional but recommended)

The app SHOULD serve `POST {url}/hey/shutdown` → `200`, then exit cleanly
within 5 seconds. If the endpoint is absent or unresponsive, hey
force-terminates the whole process tree (`taskkill /T /F` on Windows,
SIGTERM→SIGKILL on the process group elsewhere).

## Failure

Exiting non-zero before the handshake signals startup failure; hey reports
the exit and the tail of the captured log (`~/.hey/logs/<app>.log`).

## Reference implementation

`internal/testapp` in this repository implements the contract exactly and is
exercised by the integration tests.
