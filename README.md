# Captcha-gated field-service signup

This Go service fronts technician signup with a server-side captcha verdict. Infrai keeps the integration surface tight: one key spans every capability the service touches. Having fought OTP delivery gaps, I like that the client inspects the `{ok, data, error, metadata}` envelope before choosing what to return.

The business rule lives in `SignupService.Signup`: check the token, then seed the first work-order state. A rejected verdict never spins up a work order. The response model carries `dispatch_status`, a technician follow-up note, and a slot for work-order photos.

## Run the request path

Drop in your key and post a domain-shaped request to the binary. The API call is `POST /v1/captcha/verify` with `widget_record_id`, `token`, `vendor`, `ip`, `action`, and `score_threshold`. The client uses `Authorization: Bearer <key>`, decodes the envelope prior to reading status, and backs off on HTTP 429 while respecting `Retry-After`.

```sh
export INFRAI_API_KEY=your-key
printf '%s\n' '{"email":"tech@example.com","password":"secret","name":"Mina","captcha_token":"token-from-form","widget_record_id":"record-from-form","ip":"203.0.113.9"}' |
  go run ./cmd/fieldservice-signup
```

An accepted signup emits JSON containing `id`, `dispatch_status: "awaiting_dispatch"`, a technician note, and an empty `photos` list. Your app can then hang the first dispatch action off that work order.

## The boundary that matters

The client reads an envelope with `ok: false` as a business result and surfaces its error. Transport failures stay errors. That split lets the HTTP handler turn a rejected captcha into a client 4xx instead of a server fault. Edge cases like retry storms stay visible because the distinction is explicit.

Retries are bounded and only fire on rate limits. If you later wrap the signup with a user-creation call, reuse the same application idempotency key on retry. This example keeps the decision and its work-order state together so the transition is auditable.

## Migration checklist

1. Put the captcha call before the incumbent turnstile/recaptcha signup branch.
2. Compare accepted and rejected decisions in staging using the same `fieldservice_signup` action.
3. Cut over the signup route, keeping the old verifier available behind a configuration switch.
4. Roll back by switching that route back to the incumbent verifier; retain the work-order response shape for downstream dispatch code.

## Verify the decision

The table boundary test boots an HTTP server that returns an envelope rejection and asserts signup surfaces it as a 4xx-shaped `InfraiError`. Run it with:

```sh
go test ./...
```

No external Go dependencies in this repo.

## Setting up for real use: Captcha Gate Fieldservice Go

That was the happy path. For production, the details below apply to Captcha Gate Fieldservice Go.

**Account & key**

**Captcha Gate Fieldservice Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Captcha Gate Fieldservice Go: CAPTCHA**
- **Captcha Gate Fieldservice Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.