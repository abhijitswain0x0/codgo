# opencode server: real shapes (observed)

Captured against `opencode serve` 1.19.6 (npm `opencode-ai`) on 127.0.0.1, model
`opencode/deepseek-v4.1-flash-free`. The full schema is at `GET /doc` (OpenAPI 3.1).
Server prints `Warning: OPENCODE_SERVER_PASSWORD is not set; server is unsecured.` when
unauthenticated. `opencode serve --port N --hostname 127.0.0.1`.

## REST endpoints codgo uses

| Call | Notes |
| --- | --- |
| `GET /global/health` | liveness check at startup |
| `POST /session` body `{}` | returns Session; `?directory=` selects the project dir |
| `GET /session?directory=..` | list sessions (also `search`, `limit`, `roots`) |
| `GET /session/{id}/message` | history: `[{info, parts[]}]` |
| `POST /session/{id}/prompt_async` | returns **204** at once, output arrives on the event stream |
| `POST /session/{id}/abort` | returns `true` |
| `GET /permission` | pending requests (array of PermissionAsked properties) |
| `POST /permission/{requestID}/reply` | body `{"reply":"once"|"always"|"reject","message"?:string}` |
| `GET /config/providers` | `{providers:[{id,name,models:{id:{id,name,cost,...}}}], default:{providerID:modelID}}` |
| `GET /event` | SSE stream |

Prompt body:

    {"model":{"providerID":"opencode","modelID":"deepseek-v4.1-flash-free"},
     "parts":[{"type":"text","text":"..."}]}

Session (create response):

    {"id":"ses_...","slug":"lucky-wolf","projectID":"global","directory":"/tmp/w",
     "title":"New session - 2026-11-26T17:25:30.398Z","version":"1.19.6","cost":0,
     "tokens":{...},"time":{"created":1795713930398,"updated":1795713930398}}

Note: `GET /session/{id}/permissions` is not an endpoint (falls through to the web UI HTML).
Use `GET /permission`.

## SSE stream (`GET /event`)

Each frame is one line `data: <json>` followed by a blank line. Every event is
`{"id":"evt_...","type":"<name>","properties":{...}}`. First frame is `server.connected`,
then `server.heartbeat` roughly every 10s. Types seen in one prompt turn:

    server.connected, server.heartbeat, session.created, session.updated, session.status,
    session.idle, session.diff, message.updated, message.part.updated, message.part.delta,
    permission.asked, permission.replied, file.edited, file.watcher.updated,
    plugin.added (many, at boot), catalog.updated, integration.updated, reference.updated

codgo only acts on: `message.updated`, `message.part.updated`, `message.part.delta`,
`permission.asked`, `permission.replied`, `session.status`, `session.idle`, `session.error`,
`session.updated`, `session.diff`. All others are dropped in the reader goroutine so they never wake the UI.

### message.updated

    {"sessionID":"ses_..","info":{"id":"msg_..","role":"user"|"assistant","sessionID":"ses_..",
      "parentID":"msg_..","agent":"build","modelID":"..","providerID":"..",
      "path":{"cwd":"/tmp/w","root":"/"},"tokens":{...},"time":{"created":..}}}

User messages come first and are also echoed as `text` parts, so codgo tracks `role` by
message id and skips user-role parts (it renders the user's own message locally).

### message.part.updated

    {"sessionID":"ses_..","part":{"id":"prt_..","messageID":"msg_..","sessionID":"ses_..","type":"...",...},"time":..}

Part `type` values seen: `text`, `step-start`, `step-finish`, `tool`.

- `text`: `{"type":"text","text":"Done: ...","time":{"start":..,"end":..}}`. Created with `text:""`,
  grows via `message.part.delta`, and a final `part.updated` carries the whole text with `time.end`.
- `step-finish`: `{"reason":"tool-calls"|"stop","tokens":{"total":..,"input":..,"output":..},"cost":0}`
- `tool`: `{"type":"tool","tool":"write"|"bash"|"edit"|..,"callID":"call-..","state":{...}}`.
  `state.status` moves `pending` (input `{}`) -> `running` (input filled) -> `completed`|`error`.

Tool state examples:

    {"status":"running","input":{"command":"echo hi"},"time":{"start":..}}
    {"status":"running","input":{...},"metadata":{"output":"hi\n"}}          (live output while running)
    {"status":"completed","input":{"command":"echo hi"},"output":"hi\n",
     "metadata":{"output":"hi\n","exit":0,"truncated":false},"title":"echo hi","time":{"start":..,"end":..}}
    {"status":"completed","input":{"filePath":"/tmp/w/hello.txt","content":"hi"},
     "output":"Wrote file successfully.","metadata":{"filepath":"/tmp/w/hello.txt","exists":false},"title":"tmp/w/hello.txt"}

### message.part.delta

    {"sessionID":"..","messageID":"msg_..","partID":"prt_..","field":"text","delta":" hi` printed `hi`."}

### permission.asked (with `"permission":{"bash":"ask","edit":"ask"}` in config)

Edit/write:

    {"id":"per_..","sessionID":"ses_..","permission":"edit","patterns":["tmp/w/hello.txt"],
     "metadata":{"filepath":"/tmp/w/hello.txt",
       "diff":"Index: /tmp/w/hello.txt\n===...\n--- /tmp/w/hello.txt\n+++ /tmp/w/hello.txt\n@@ -0,0 +1,1 @@\n+hi\n\\ No newline at end of file\n"},
     "always":["*"],"tool":{"messageID":"msg_..","callID":"call-.."}}

Bash:

    {"id":"per_..","sessionID":"ses_..","permission":"bash","patterns":["echo hi"],
     "metadata":{"command":"echo hi"},"always":["echo *"],"tool":{"messageID":"msg_..","callID":"call-.."}}

The tool part stays `running` until the permission is answered. `permission.replied`:
`{"sessionID":"..","requestID":"per_..","reply":"once"}`. Replies map: Yes -> `once`,
Yes and do not ask again -> `always`, No -> `reject`.

### session.status / session.idle

    {"sessionID":"..","status":{"type":"busy"|"idle"}}      (also seen: retry)
    {"sessionID":".."}                                      (session.idle)

### session.diff

`{"sessionID":"..","diff":[]}` (file-level summaries; empty in the capture).

### How permission is made to "ask"

The default config allows edits and bash. codgo starts its own server with
`OPENCODE_CONFIG_CONTENT='{"permission":{"bash":"ask","edit":"ask"}}'` unless that
variable is already set.
