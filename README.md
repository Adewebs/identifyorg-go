# github.com/Adewebs/identifyorg/go-sdk

Server-side Go SDK for IdentifyOrg. Zero third-party dependencies
(stdlib `net/http` + `encoding/json` only).

```bash
go get github.com/Adewebs/identifyorg/go-sdk
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
}
```

## Notes

- Requires Go 1.21+ (uses the `any` alias).
- `Client.HTTPClient` is exported so you can swap in a custom
  `*http.Client` (timeouts, retries, tracing) if needed.
- This SDK is server-side only — never embed your secret key
  (`vl_*_sk_...`) in a mobile app or web frontend. Mint realtime tokens on
  your backend with `StreamingToken`/`ChatToken` and pass the result to a
  client SDK instead.
