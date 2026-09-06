# Fork delta

This fork adds one thing to `llm-d/llm-d-inference-sim`: the ingress paths DIAL Core forwards
on, behind an off-by-default flag. Everything else is upstream.

Keep this list short. **Anything that needs an upstream line to change goes upstream as a PR
instead of into this fork** — that is what keeps rebasing onto a new tag a 15-minute job.

## The delta

| File | Change |
|---|---|
| `pkg/communication/dial_paths.go` | **new** — `registerDialRoutes`, the alias table |
| `pkg/communication/http.go` | one call site, after the `/v1/embeddings` registration |
| `pkg/common/config.go` | `DialPaths` field |
| `pkg/common/parser.go` | `--dial-paths` flag |
| `pkg/tests/dial_paths_test.go` | **new** — paths served with the flag, 404 without it |
| `FORK.md` | this file |

Four added lines in upstream files, two new files.

## Why

DIAL Core appends the inbound ingress path to a deployment's `interfaces.<type>.base_url`
(`DeploymentEndpointUtil.resolveRequestUri`), so a target addressed through the `interfaces`
map must serve the DIAL path, not the API's own:

| Interface | DIAL ingress path | Simulator's own path |
|---|---|---|
| `anthropicMessages` | `/anthropic/v1/messages` | `/v1/messages` |
| `openaiResponses` | `/openai/v1/responses` | `/v1/responses` |
| `openaiChatCompletions` | `/openai/deployments/{model}/chat/completions` | `/v1/chat/completions` |
| `openaiEmbeddings` | `/openai/deployments/{model}/embeddings` | `/v1/embeddings` |

Without the aliases the simulator is reachable only through Core's legacy complete-url fields
(`endpoint`, `responsesEndpoint`). Those cannot express `anthropicMessages` at all — Core's
`resolveLegacyEndpoint` returns null for it — and the single `endpoint` field serves the whole
deployments-POST family, so chat completions and embeddings cannot both be right at once.

With `--dial-paths`, one `interfaces.base_url` serves every interface, which is how DIAL
addresses an adapter.

## Usage

```bash
llm-d-inference-sim --model llm-d-sim --port 8000 --dial-paths
```

DIAL Core model entity:

```json
{
  "type": "chat",
  "overrideName": "llm-d-sim",
  "baseUrl": "https://<sim-host>",
  "interfaces": {
    "openaiChatCompletions": {},
    "openaiResponses": {},
    "anthropicMessages": {}
  },
  "userRoles": ["default"],
  "upstreams": [{ "id": "sim-1" }]
}
```

`upstreams[].id` is required — Core answers `503 Upstream is missing required id` on the
responses and messages paths for an upstream without one, while chat completions tolerates it.

## Upstreaming

The DIAL paths themselves do not belong upstream. The generic form does: a route-alias or
path-prefix flag that lets any caller map extra paths onto the existing handlers. If upstream
takes that, this fork disappears and we go back to stock images.

## Rebasing

```bash
git fetch upstream --tags
git rebase upstream/<tag>
```

Expect conflicts only in `pkg/communication/http.go` (the call site), `pkg/common/config.go`
and `pkg/common/parser.go` — all three are one-line additions next to
`enable-request-id-headers` / `/v1/embeddings`. Re-run `make test` afterwards; the added test
fails if a handler is renamed or a route moves.
