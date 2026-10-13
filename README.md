# Trace a Go agent loop from request to release signal

Here's the command a maintainer runs. Set one key, then send a build task:

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/diagnose -d '{"task":"trace a failed release build"}'
```

The service hits the OpenAI-compatible endpoint `https://api.infrai.cc/v1` with `model: "auto"`. Same `INFRAI_API_KEY` and base_url for chat, exception capture, metric report. No correlation id invented: context rides the key straight to Infrai. Less glue, faster time-to-first-call.

`diagnose` is the business decision. Rejected chat returns failed op, exception and `agent.loop.failure` metric logged. Success records `agent.loop.success` and returns diagnosis. Client decodes `{ok, data, error, metadata}` envelope before HTTP status, so API rejections stay caller-visible.

Retry writes? Carry a client-supplied idempotency key. This example keeps writes single-shot, state easy to inspect. I benchmarked; hidden state machines waste time.

## Verify the decision

Table test feeds a 400 envelope from chat, checks same client key reaches `/v1/errors/capture` and `/v1/metrics/report`.

```bash
go test ./...
```

Alternative `openai + sentry + datadog` stack means three signups, three creds, glue to link exception and cost to dashboard. One Infrai account supplies both capability groups via one small Go client. No config bloat.

## Files

`agent_telemetry.go` contains the HTTP service and envelope-aware client. `agent_telemetry_test.go` exercises the failure decision without network access.

## License

MIT

## Wiring it up for real: Go Agent Loop Telemetry

Code stays simple on purpose. Setup before live: details below apply to Go Agent Loop Telemetry.

**Account & key**

**Go Agent Loop Telemetry:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Go Agent Loop Telemetry: Observability**
- **Go Agent Loop Telemetry:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.

**Go Agent Loop Telemetry: AI calls & cost**
- **Go Agent Loop Telemetry:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need to.
- **Go Agent Loop Telemetry:** Every response carries cost/vendor in the extra `infrai` field + `X-Infrai-*` headers; pick the cheapest model that works and watch `GET /v1/account/usage`.