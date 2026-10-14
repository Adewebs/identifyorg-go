# github.com/Adewebs/identifyorg-go

Server-side Go SDK for IdentifyOrg. Zero third-party dependencies
(stdlib `net/http` + `encoding/json` only).

```bash
go get github.com/Adewebs/identifyorg-go
```

```go
package main

import (
	"fmt"
	"os"

	identifyorg "github.com/Adewebs/identifyorg-go"
)

func main() {
	client := identifyorg.NewClient(os.Getenv("IDENTIFYORG_SECRET_KEY"), "")

	result, err := client.VerifyBVN("22212345678", "Ada", "Okafor", "order-123")
	if err != nil {
		var apiErr *identifyorg.APIError
		if errors.As(err, &apiErr) {
			fmt.Println("IdentifyOrg error:", apiErr.Code, apiErr.Message)
		}
		return
	}
	fmt.Println(result.Status, result.Cost)

	// Mint a realtime token for YOUR user, then hand it to the browser/mobile
	// client (IdentifyOrg JS/React Native/Flutter/Kotlin SDK) — never expose
	// your secret key to a client.
	token, _ := client.StreamingToken("video", userID, displayName, "", "publisher")
	fmt.Println(token.Token, token.URL, token.RoomName)

	// Website Audit — first standard-tier audit free per account, then
	// billed. "deep" runs in the background: returns immediately with
	// Status "pending", poll GetAudit until "ready"/"failed".
	audit, _ := client.RunAudit("https://example.com", "standard", "")
	audit, _ = client.GetAudit(audit.ID)
	fmt.Println(*audit.Score, *audit.Grade, audit.Findings)
	pdfBytes, _ := client.GetAuditPDF(audit.ID)
	os.WriteFile("report.pdf", pdfBytes, 0644)

	// AI media detection — self-hosted, no third-party AI-detection API.
	// Image/text: first scan free per account, then billed. Video: always billed.
	imgScan, _ := client.ScanImage("photo.jpg")
	textScan, _ := client.ScanText("... at least ~800 characters ...")
	videoScan, _ := client.ScanVideo("clip.mp4", 12)
	fmt.Println(*textScan.Verdict, textScan.Signals)
}
```

## What's new in 0.2.0

```go
started, _ := client.PhoneVerifyStart("08031234567", "", "signup-42")
res, _ := client.PhoneVerifyCheck(started.ID, "418209") // res.Verified

docx, _ := client.PDFToDOCX("report.pdf") // []byte
os.WriteFile("out.docx", docx, 0o644)

client.SendSMS("08031234567", "Your order has shipped")
client.CreateWebhook("https://example.com/hooks/identifyorg", nil, "orders")
overview, _ := client.AccountOverview(30)
```

Also: bills (`BuyAirtime`, `BuyData`, `PayElectricity`, `PayTV`, `VerifyMeter`, `VerifySmartcard`), every
file and image tool (`PDFMerge`, `PDFSplit`, `OCR`, `RemoveBackground`, `StoreAssets`, `AppIcons`, ...),
`GetVerification`, `VerifyDocumentTamper` and a generic `Request` method. `Topup` was removed: that
endpoint needs a dashboard login, so it always returned 401 with an API key.

## Notes

- Requires Go 1.21+ (uses the `any` alias).
- `Client.HTTPClient` is exported so you can swap in a custom
  `*http.Client` (timeouts, retries, tracing) if needed.
- This SDK is server-side only — never embed your secret key
  (`vl_*_sk_...`) in a mobile app or web frontend. Mint realtime tokens on
  your backend with `StreamingToken`/`ChatToken` and pass the result to a
  client SDK instead.

## Address verification and bill prices

```go
check, _ := client.VerifyAddress("12 Adeola Odeku Street, Victoria Island, Lagos", "Ada Obi", "NG", 150, "", "")
fmt.Println(check["confirm_url"]) // send this to the person
result, _ := client.GetAddressVerification(check["id"].(string))

plans, _ := client.BillDataPlans("mtn", "monthly")
price, _ := client.BillQuote("airtime", 500, "mtn", "")
```

## Take payments

```go
client.SaveBusinessProfile("Ada Stores", "individual", "08031234567")
acct, _ := client.CreateVirtualAccount("Tunde Bello", "tunde@example.com", "order-1001", "", "")
charge, _ := client.CreateCardCharge(5000, "tunde@example.com", "Tunde Bello", "") // send him the checkout link
paid, _ := client.ReceivedPayments("", "success", 50)
```
