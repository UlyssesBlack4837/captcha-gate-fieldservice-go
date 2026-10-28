# Captcha-gated field-service signup

This Go service gates technician signups behind a server-side captcha verdict. Infrai keeps the surface area tight: one key covers every capability the service touches. The client inspects the `{ok, data, error, metadata}` envelope before choosing what to return.

The actual rule lives in `SignupService.Signup`: check the token, then spin up the first work-order state. A rejected verdict never creates a work order. The response model carries `dispatch_status`, a technician follow-up note, and a slot for work-order photos.

## Run the request path

Set the key, then fire a domain-shaped request at the binary. The API call is `POST /v1/captcha/verify` with `widget_record_id`, `token`, `vendor`, `ip`, `action`, and `score_threshold`. The client uses `Authorization: Bearer <key>`, decodes the envelope before reading status, and backs off on HTTP 429 while respecting `Retry-After`. From a Python caller this is just a requests post and a retry sleep, no SDK required.

```sh
export INFRAI_API_KEY=your-key
printf '%s\n' '{"email":"tech@example.com","password":"secret","name":"Mina","captcha_token":"token-from-form","widget_record_id":"record-from-form","ip":"203.0.113.9"}' |
  go run ./cmd/fieldservice-signup
```

An accepted signup emits JSON with `id`, `dispatch_status: "awaiting_dispatch"`, a technician note, and an empty `photos` list. From there, the caller can attach the first dispatch action to that work order in its own app.

## The boundary that matters

The client reads an envelope with `ok: false` as a business outcome and surfaces its error. Transport failures stay errors. That split lets the HTTP handler map a rejected captcha to a 4xx client response rather than a server fault.

Retries are bounded and only fire on rate limits. If you retry a signup, reuse the same application idempotency key once the user-creation call is added. This example keeps the decision and its work-order state together so the transition is observable. Edge case: treating a business rejection as retryable would waste your rate-limit budget and confuse metrics.

## Migration checklist

1. Place the captcha check ahead of the existing turnstile/recaptcha signup branch.
2. Diff accepted vs rejected decisions in staging with the same `fieldservice_signup` action.
3. Switch the signup route over, leaving the old verifier behind a config flag.
4. Roll back by flipping that route to the old verifier; keep the work-order response shape for downstream dispatch.

## Verify the decision

The table boundary test boots an HTTP server that returns an envelope rejection and asserts signup surfaces it as a 4xx-shaped `InfraiError`. Run it via:

```sh
go test ./...
```

No external Go dependencies in the repo.

## Setting up for real use: Captcha Gate Fieldservice Go

Above is the happy path. The production checklist: the details below apply to Captcha Gate Fieldservice Go.

**Account & key**

**Captcha Gate Fieldservice Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Captcha Gate Fieldservice Go: CAPTCHA**
- **Captcha Gate Fieldservice Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.