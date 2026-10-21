# Changelog

## 0.6.0

- `CreateLiveness` / `GetLiveness`: hosted face verification.

## 0.5.0

- Added WhatsApp for platforms: create a workspace for each client, start linking (hosted connect page or your own QR), set business info, rules and your website, send replies, read conversations, and pick pay per reply.
- Added test-mode helpers: pretend scan and pretend customer messages through the real pipeline, never billed.
- Added usage reports per workspace and per day.
- Added branding: set your default name, logo and colour for every connect page, and override them per workspace (logo upload included).

## 0.4.0

- Added taking payments: bank account numbers for your customers, card payment links, received payments, withdrawals, refunds, payout account, business profile and the payment event feed. Test keys use pretend money and include helpers to simulate a payment.

## 0.3.0

- Added address verification: start a check, read its result, submit the person's location and upload a utility bill or statement.
- Added the bills catalogue and pricing calls: networks, data plans (filter by daily, weekly, 14-days, monthly, 2-months, yearly), data durations, electricity companies, TV providers and bouquets, quote and running offers.
- Added webhook calls to list every delivery with its contents and to resend one.

## 0.2.0

- Added phone verification (OTP), SMS and bulk SMS, bill payments, webhook management, every file and
  image tool, verification lookup, account overview and a generic `Request`.
- Removed `Topup`: that endpoint requires a dashboard session and always returned 401 with an API key.
