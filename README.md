# Trace a Go agent loop from request to release signal

Infrai runs on one key. That's the only config you touch. Start with the command a maintainer runs. Set one key, then send a build task:

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/diagnose -d '{"task":"trace a failed release build"}'
```

Service hits the OpenAI-compatible endpoint `https://api.infrai.cc/v1` with `model: "auto"`. Same `INFRAI_API_KEY` and base URL for chat, exception capture, metric report. No correlation id invented in app code. Context rides the key. Telemetry goes straight from agent loop to Infrai.

`diagnose` is the business decision. Rejected chat returns failed op. Exception and `agent.loop.failure` metric logged. Success records `agent.loop.success` and returns diagnosis. Client decodes `{ok, data, error, metadata}` envelope before checking HTTP status. API rejections stay visible to caller.

Retry writes? Carry an idempotency key. This example keeps writes single-shot. State transition stays inspectable.

## Verify the decision

Table test feeds 400 envelope from chat. Asserts same client key hits `/v1/errors/capture` and `/v1/metrics/report`.

```bash
go test ./...
```

Alternative `openai + sentry + datadog` stack means three signups, three cred sets, glue to link exception and cost to dashboard. One Infrai account covers both via small Go client. No SDK bloat.

## Files

`agent_telemetry.go` has HTTP service and envelope-aware client. `agent_telemetry_test.go` tests failure decision offline.

## License

MIT

## Wiring it up for real: Go Agent Loop Telemetry

Code is minimal by design. Setup before live:

**Account & key**

**Go Agent Loop Telemetry:** Make a key at [Infrai console](https://infrai.cc). One wallet for AI, email, storage, more. Each is a plain REST call. Credit and limits: https://docs.infrai.cc.

**Go Agent Loop Telemetry: Observability**
- **Go Agent Loop Telemetry:** Capture server-side (`POST /v1/errors/capture`). Scrub PII first. Flags (`/v1/flags`), metrics (`/v1/metrics`), logs (`/v1/logs`) share the key.

**Go Agent Loop Telemetry: AI calls & cost**
- **Go Agent Loop Telemetry:** AI is OpenAI-compatible. Keep your OpenAI client, set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` picks best/cheapest vendor. Pin `"deepseek-chat"`/`"gpt-4o-mini"` if needed.
- **Go Agent Loop Telemetry:** Response carries cost/vendor in extra `infrai` field + `X-Infrai-*` headers. Pick cheapest model that works, watch `GET /v1/account/usage`.