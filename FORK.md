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
| `pkg/api/request.go` | `MessagesRequest.System` accepts a string **or** content blocks |
| `pkg/api/response.go` | `MessagesContentBlockStart` — `content_block_start` always carries `"text":""` |
| `pkg/communication/response_builder.go` | the two `content_block_start` construction sites |
| `pkg/tests/messages_wire_compat_test.go` | **new** — both wire fixes |
| `pkg/common/env_aliases.go` | **new** — `SIM_<FLAG_NAME>` alias for every flag |
| `pkg/common/parser.go` | one call site, before `config.validate()` |
| `pkg/common/env_aliases_test.go` | **new** — precedence, empty value, parse error, toggle twin |
| `docs/configuration.md` | the alias rules, and the precedence bullet they change |
| `FORK.md` | this file |

The `pkg/api` and `response_builder.go` rows are **bug fixes, not DIAL specifics** — they
belong upstream and should be sent there; the fork carries them only until that lands.

The **environment aliases** are not DIAL-specific either, and upstream already has the idea
(`SIM_MODEL`, `PYTHONHASHSEED`): `SIM_<FLAG_NAME>` generalises it so a Kubernetes deployment can
be configured without an argument string. Worth offering upstream. Until then the fork carries
it; see [docs/configuration.md](docs/configuration.md#environment-variable-aliases). Both were found by putting a real DIAL
adapter in front of the simulator:

- Anthropic allows `system` as a string or an array of content blocks. The simulator accepted
  only a string, so `ai-dial-adapter-bedrock` (which emits blocks) got
  `400 cannot unmarshal array into Go struct field MessagesRequest.system of type string`.
- `content_block_start` serialised as `{"type":"text"}` because the `Text` field carried
  `omitempty`. The real API always sends `"text":""`, and a client that accumulates text deltas
  onto the block crashes without it — the adapter died with
  `unsupported operand type(s) for +=: 'NoneType' and 'str'`.

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
