# Plan: Firefox Remote Debugging Protocol (RDP) server for Plex

## Context

Plex is a single-process Go browser using Gio for rendering. It already builds a real DOM ([internal/dom/](../dom/)), CSSOM ([internal/css/cssom/](../css/cssom/)), styletree ([internal/layouts/styletree/](../layouts/styletree/)) and layout tree, but has no way to inspect that state at runtime — this directory is currently empty. The goal: stand up enough of Firefox's Remote Debugging Protocol that you can launch Plex with `--remote-debugging-port=6000`, point Firefox's `about:debugging` → Setup → Network Location at `127.0.0.1:6000`, click "Inspect" on the discovered Plex target, and see the live DOM tree + computed CSS for each node in Firefox's DevTools inspector pane.

**Locked scope:**
- **Read-only DOM inspector only** — Root, Tab target, Inspector, Walker, PageStyle actors. No console, no DOM mutation, no script/thread debugging.
- **Native RDP wire format** — plain TCP, length-prefixed JSON (`<bytelen>:<json>`), default port 6000. Works with stock Firefox about:debugging, no Firefox prefs to flip.
- **Opt-in** — listener only opens when `--remote-debugging-port=N` is passed (N > 0). Default off.

## Architecture overview

### Package layout

```
internal/devtools/
  server.go             // devtools.Server: listener accept loop, ctx-driven shutdown
  state.go              // BrowserState interface + LiveState (RWMutex over Document/StyledNode/CSSOM)
  ids.go                // per-Connection actor ID generator ("server1.conn0.walker3")
  transport/
    framing.go          // ReadPacket / WritePacket — <bytelen>:<json> codec
    framing_test.go
  conn/
    conn.go             // Connection: net.Conn + actor registry + write mutex
    dispatch.go         // packet -> actor.Handle(method, payload) router
  actors/
    actor.go            // Actor interface { ID(); Handle(method, payload) }
    root.go             // getRoot, listTabs, listAddons, listWorkers, listProcesses, getProcess
    tab.go              // BrowsingContext target form + attach/detach/getTarget
    inspector.go        // getWalker, getPageStyle, getHighlighter (stub)
    walker.go           // documentElement, children, querySelector, parents, siblings, getMutations
    pagestyle.go        // getComputed, getApplied, getLayout (stub)
    nodeform.go         // *dom.Node -> Firefox NodeActor form; nodeType translation table
```

One file per actor in a single `actors` package — actors share `BrowserState` and `Connection` references, so a flat package keeps cross-actor wiring cheap.

### Threading

DOM/styletree become read-only after `renderHtml`. Use a `sync.RWMutex` inside `LiveState` rather than a channel hand-off to the Gio thread. Reads (from devtools goroutines) take `RLock`; the future resize/reflow path takes `Lock` to swap the snapshot. Simpler than message passing and good enough for v1's read-only contract.

### Packet framing

`<bytelen>:<json>` where `bytelen` is decimal ASCII of `len(json)` in **bytes**.

- **Reader**: `bufio.Reader.ReadSlice(':')` → `strconv.Atoi` → reject if `<0` or `>16 MiB` → `io.ReadFull` for exactly N bytes. EOF on prefix = clean close; EOF mid-body = error.
- **Writer**: marshal → write `strconv.Itoa(len(b))`, `':'`, body under a per-Connection write mutex (multiple actors may emit responses concurrently).

### Actor dispatch

Manual `switch` per actor — only ~25 methods total across 5 actors; reflection saves boilerplate but hides typos and the JSON shapes are heterogeneous enough that you'd still write per-method binding.

Each actor implements:
```go
Handle(method string, payload json.RawMessage) (response any, err error)
```

Dispatch loop: decode `{from, to, type, ...}` → look up `to` in `map[string]Actor` → call `Handle(type, fullPacket)` → wrap reply with `{from: actorID, ...}` → frame, write.

Actor IDs are scoped per Connection. `root` is fixed; everything else is `server1.conn<N>.<kind><M>`.

## Critical detail: nodeType translation

Plex's `IsNode()` returns non-spec values that Firefox's inspector will mis-render. Translate inside `actors/nodeform.go`:

| Plex type | `IsNode()` | DOM spec / Firefox |
|---|---|---|
| `*Document` ([document.go:18](../dom/document.go#L18)) | 0 | 9 |
| `*Text` ([text.go:19](../dom/text.go#L19)) | 2 | 3 |
| `*Comment` | 3 | 8 |
| `*Element` ([element.go:37](../dom/element.go#L37)) | 4 | 1 |
| `*DocumentType` | (per file) | 10 |
| `*ShadowRoot` ([shadow_root.go:59](../dom/shadow_root.go#L59)) | 11 | 11 |
| `*TemplateElement` | (per file) | 1 |

One helper `domNodeType(n dom.Node) int` keeps the table in one place.

## NodeActor form fields

Each node sent to Firefox needs:
`actor`, `baseURI`, `parent` (parent actorID or null), `nodeType` (translated), `nodeName` (uppercase for HTML elements), `nodeValue` (text data only), `namespaceURI`, `attrs` (`[{name, value, namespace}]`), `numChildren`, `displayName` (lowercase tag), `isDocumentElement` (only for `<html>`).

Walker maintains two parallel maps so PageStyle can find the right `StyledNode` from a node actor ID:
- `map[string]dom.Node` — actorID → node
- `map[dom.Node]*styletree.StyledNode` — node → styled node

## Sequence: Firefox first connect

```
Firefox -> Plex: TCP connect
Plex    -> Firefox: {from:"root", applicationType:"browser", traits:{networkMonitor:false}}
Firefox -> Plex:   {to:"root", type:"getRoot"}
Plex    -> Firefox: {from:"root", selected:0, deviceActor:null, ...}
Firefox -> Plex:   {to:"root", type:"listTabs"}
Plex    -> Firefox: {from:"root", tabs:[<tabDescriptor>], selected:0}
Firefox -> Plex:   {to:"<tabDesc>", type:"getTarget"}
Plex    -> Firefox: {from:"<tabDesc>", frame:{actor:"tab1", url:"about:plex", title:"Plex",
                       inspectorActor:"inspector1", styleSheetsActor:"sheets1", ...}}
Firefox -> Plex:   {to:"tab1", type:"attach"}             -> {type:"tabAttached", threadActor:null}
Firefox -> Plex:   {to:"inspector1", type:"getWalker"}    -> {walker:{actor:"walker1", root:<NodeForm>}}
Firefox -> Plex:   {to:"inspector1", type:"getPageStyle"} -> {pageStyle:{actor:"pageStyle1"}}
Firefox -> Plex:   {to:"walker1", type:"children", node:"node1", maxNodes:100, whatToShow:-1}
Plex    -> Firefox: {nodes:[...NodeForms...], hasFirst:true, hasLast:true}
Firefox -> Plex:   {to:"pageStyle1", type:"getComputed", node:"node42"}
Plex    -> Firefox: {computed:{"background-color":{value:"#ff0000",matched:true}, ...}}
```

Walker's `getMutations` returns `{mutations:[]}` — keeps Firefox happy without any push notifications. `getHighlighter` and `getLayout` can return empty/stub responses for v1.

## Changes to existing files

### [internal/core/render.go](../core/render.go) — return all four artifacts, not just the layout box

Current signature returns only `*layouts.Box`. Add a `Rendered` struct so the devtools server can read the DOM/styletree/CSSOM:

```go
type Rendered struct {
    Document *dom.Document
    Style    *styletree.StyledNode
    CSSOM    *cssom.Cssom
    Layout   *layouts.Box
}
func renderHtml(html, style string, width, height float64) *Rendered { ... }
```

### [internal/core/app.go](../core/app.go) — accept a nil-safe BrowserState

`StartPlex` gains a `state devtools.BrowserState` parameter. After `renderHtml`, if `state != nil` call `state.Replace(r.Document, r.Style, r.CSSOM, "about:plex", "Plex")`. Use `r.Layout` where `tree` is referenced today (line 78).

### [cmd/plex/main.go](../../cmd/plex/main.go) — new flag, conditional server boot

```go
var rdpPort = flag.Int("remote-debugging-port", 0, "Firefox RDP listener port (0=off)")
...
var state devtools.BrowserState
if *rdpPort > 0 {
    live := devtools.NewLiveState()
    srv := devtools.NewServer(logger, live, *rdpPort)
    go srv.ListenAndServe(ctx)
    state = live
}
go func() { ... core.StartPlex(ctx, logger, window, state) ... }()
```

Shutdown: `app.DestroyEvent` returns from `StartPlex`; cancel the root ctx; `Server.ListenAndServe` selects on `<-ctx.Done()` and closes the listener (unblocking `Accept`); each `Connection` closes its `net.Conn`.

## Reusable code

- [internal/dom/](../dom/) — Node/ElementNode interfaces already give everything the Walker needs (`Children()`, `Parent()`, `Attributes()`, `Tag()`, `Namespace()`).
- [internal/layouts/styletree/style_tree.go:37](../layouts/styletree/style_tree.go#L37) — `StyledNode.SpecifiedValues` (a `PropertyMap`) is exactly the source for `PageStyle.getComputed`. Iterate the map, stringify each `*cssom.Declaration.Value` with the existing CSS value formatters, emit `{"prop":{value, matched:true}}`.
- [internal/log/](../log/) — slog wrapper; reuse for structured server logs (connection accept, packet errors).
- `internal/utils.Option[T]` is already used across the codebase; use it for the new `LiveState` accessors where a value may be absent.

## Testing strategy

- `internal/devtools/transport/framing_test.go` — round-trip random bodies; reject prefix without `:`; reject oversize (`999999999999:`); EOF mid-body.
- `internal/devtools/actors/root_test.go` — drive a `Connection` over `io.Pipe`, send `getRoot`, assert `applicationType=="browser"`.
- `internal/devtools/actors/walker_test.go` — feed a fixture `*dom.Document`, call `documentElement` then `children`, assert `nodeType` translation (Element → 1, Text → 3, Document → 9).
- `internal/devtools/rdp_smoke_test.go` — `net.Listen` on `:0`, drive the full connect → getRoot → listTabs → getTarget → getWalker → children sequence end-to-end against a real TCP socket.

## End-to-end verification

1. `go run ./cmd/plex --remote-debugging-port=6000`
2. Firefox → `about:debugging` → "Setup" tab → add Network Location `127.0.0.1:6000` → Connect.
3. Click `127.0.0.1:6000` in the left sidebar; the "Plex" target appears under **Tabs**.
4. Click **Inspect**; DevTools opens. The Inspector pane shows the `<html><body><div class="a">...` tree from the demo HTML in [internal/core/app.go:15-50](../core/app.go#L15-L50).
5. Click `.a`; the **Computed** pane shows `background-color: #ff0000`.
6. Walk to `.g`; computed `background-color` is `#800080`.

If Firefox shows "Connection refused" → listener isn't bound (flag typo / port collision).
If the tab appears but inspector shows nothing → almost certainly the `nodeType` translation table is off — Firefox silently hides nodes whose `nodeType` it can't interpret.

## Out of scope (and what would extend later)

Deliberately excluded from v1: console actor (no JS engine yet), thread/script actors, network monitor, mutation observers (returning empty `getMutations` keeps Firefox happy without push), CSS editing / `setRuleText`, box-model highlighter overlay, multi-tab/multi-window, source maps, screenshot actor.

Future extensions, in order of likely value:
1. Once [internal/script/runtime/](../script/runtime/) lands (WASM engine), add a `webconsoleActor` that bridges `slog` output → Firefox console, then a `threadActor` for breakpoints.
2. Add a DOM-mutation event bus so `getMutations` can report live tree changes; today's response is `{mutations:[]}`.
3. Implement `getLayout` properly once layout tree exposes box rects on demand — gives Firefox's box-model panel real numbers.
4. Wire `netMonitor` once a network/HTTP layer exists.

## References

All canonical Mozilla docs. Read in this order while implementing.

### Core protocol spec
- [Remote Debugging Protocol](https://firefox-source-docs.mozilla.org/devtools/backend/protocol.html) — main document. Packet framing, JSON message shape (`{to, type, from}`), actors, conversations, request/reply ordering, error packets, bulk data. **Start here.**
- [Architecture overview](https://firefox-source-docs.mozilla.org/devtools/architecture-overview.html) — client/server split, transports.
- [Client API](https://firefox-source-docs.mozilla.org/devtools/backend/client-api.html) — what a client sends. Reading it in reverse tells the server what to answer.

### Actor model (server-side)
- [How actors are organized](https://firefox-source-docs.mozilla.org/devtools/backend/actor-hierarchy.html) — Root → global vs. target-scoped. Lifetime tree (close parent → close children).
- [Writing an Actor (protocol.js)](https://firefox-source-docs.mozilla.org/devtools/backend/protocol.js.html) — Mozilla's internal Actor DSL. We aren't using protocol.js (Go side), but this is the best reference for the *shape* of request/response types per actor and which `RetVal` keys Firefox expects.
- [How to register an actor](https://firefox-source-docs.mozilla.org/devtools/backend/actor-registration.html) — only relevant for add-ons, but worth skimming for Root actor discovery semantics.
- [Actor Best Practices](https://firefox-source-docs.mozilla.org/devtools/backend/actor-best-practices.html) — naming, lifetimes, when to emit unsolicited events.
- [Backward Compatibility](https://firefox-source-docs.mozilla.org/devtools/backend/backward-compatibility.html) — how Firefox negotiates older protocol versions.

### Source-of-truth (when docs are vague)

The docs are incomplete in places — `mozilla-central` is the final word for exact method shapes.

- [searchfox: devtools/server/actors/](https://searchfox.org/mozilla-central/source/devtools/server/actors) — every actor implementation. Files relevant to this plan:
  - `root.js` — Root actor (`getRoot`, `listTabs`, `listProcesses`, `listAddons`, …)
  - `descriptors/tab.js` and `targets/window-global.js` — tab/target actor forms
  - `inspector.js` — Inspector entry point
  - `inspector/walker.js` — Walker (DOM tree)
  - `inspector/node.js` — NodeActor form fields (exact field list Firefox expects)
  - `page-style.js` — PageStyle (`getComputed`, `getApplied`, `getLayout`)
- [searchfox: devtools/shared/specs/](https://searchfox.org/mozilla-central/source/devtools/shared/specs) — *shared* request/response type definitions consumed by both client and server. Closest thing to a machine-readable schema. Relevant files: `inspector.js`, `node.js`, `walker.js`, `page-style.js`, `root.js`.

### Transport / framing

Packet format (`<bytelen>:<json>`) is described in the [Remote Debugging Protocol](https://firefox-source-docs.mozilla.org/devtools/backend/protocol.html) under "Packets" and implemented in:

- [searchfox: devtools/shared/transport/transport.js](https://searchfox.org/mozilla-central/source/devtools/shared/transport/transport.js) — Mozilla's reference codec. If our Go codec disagrees, this file wins.

### Do NOT confuse with these

- [Remote Protocols index](https://firefox-source-docs.mozilla.org/remote/index.html) — Firefox's *other* protocols (CDP shim, WebDriver BiDi). **Not** what we're implementing.
- [MozillaWiki: Remote Debugging Protocol](https://wiki.mozilla.org/Remote_Debugging_Protocol) — historical / partially obsolete. Background only.
