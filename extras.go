package identifyorg

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// --- Phone verification (OTP by SMS) ---

// PhoneVerification is the result of PhoneVerifyStart / PhoneVerifyCheck.
type PhoneVerification struct {
	ID        string  `json:"id"`
	Phone     string  `json:"phone"`
	Status    string  `json:"status"`
	Verified  bool    `json:"verified"`
	ExpiresAt string  `json:"expires_at"`
	Reference string  `json:"reference"`
	Cost      float64 `json:"cost"`
	IsTest    bool    `json:"is_test"`
}

// PhoneVerifyStart sends a code to phone (local 0803... or +234... both work).
// template must contain {code} ({minutes} is also substituted); pass "" for the
// default wording. reference is your own id, echoed back. Test keys return the
// same shape without sending a message.
func (c *Client) PhoneVerifyStart(phone, template, reference string) (*PhoneVerification, error) {
	body := map[string]any{"phone": phone}
	if template != "" {
		body["template"] = template
	}
	if reference != "" {
		body["reference"] = reference
	}
	var out PhoneVerification
	err := c.do(http.MethodPost, "/v1/phone/verify/start", nil, body, "", &out)
	return &out, err
}

// PhoneVerifyCheck is free: it checks the code your user typed.
func (c *Client) PhoneVerifyCheck(id, code string) (*PhoneVerification, error) {
	var out PhoneVerification
	err := c.do(http.MethodPost, "/v1/phone/verify/check", nil, map[string]string{"id": id, "code": code}, "", &out)
	return &out, err
}

func (c *Client) PhoneVerifications(limit int) ([]map[string]any, error) {
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/phone/verifications", url.Values{"limit": {strconv.Itoa(limit)}}, nil, "", &out)
	return out, err
}

// --- SMS ---

func (c *Client) SendSMS(to, message string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/sms/send", nil, map[string]string{"to": to, "message": message}, "", &out)
	return out, err
}

// SendBulkSMS is billed per delivered recipient, not per attempt.
func (c *Client) SendBulkSMS(to []string, message string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/sms/send-bulk", nil, map[string]any{"to": to, "message": message}, "", &out)
	return out, err
}

// --- Bill payments ---

func (c *Client) BuyAirtime(network, phone string, amount float64, idempotencyKey string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/bills/airtime", nil, map[string]any{"network": network, "phone": phone, "amount": amount}, idempotencyKey, &out)
	return out, err
}

func (c *Client) BuyData(network, phone, variationCode, idempotencyKey string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/bills/data", nil, map[string]string{"network": network, "phone": phone, "variation_code": variationCode}, idempotencyKey, &out)
	return out, err
}

// PayElectricity: meterType is "prepaid" or "postpaid".
func (c *Client) PayElectricity(disco, meterNumber, meterType string, amount float64, idempotencyKey string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/bills/electricity", nil, map[string]any{"disco": disco, "meter_number": meterNumber, "meter_type": meterType, "amount": amount}, idempotencyKey, &out)
	return out, err
}

func (c *Client) PayTV(provider, smartcardNumber, variationCode, idempotencyKey string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/bills/tv", nil, map[string]string{"provider": provider, "smartcard_number": smartcardNumber, "variation_code": variationCode}, idempotencyKey, &out)
	return out, err
}

func (c *Client) billGet(path string, q url.Values) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, path, q, nil, "", &out)
	return out, err
}

func (c *Client) BillNetworks() (map[string]any, error) { return c.billGet("/v1/bills/networks", nil) }
func (c *Client) BillDiscos() (map[string]any, error)   { return c.billGet("/v1/bills/discos", nil) }
func (c *Client) BillTvProviders() (map[string]any, error) {
	return c.billGet("/v1/bills/tv-providers", nil)
}

// BillOffers lists offers running now, for example a percentage off airtime.
func (c *Client) BillOffers() (map[string]any, error) { return c.billGet("/v1/bills/offers", nil) }

// BillDataPlans lists bundles for a network. duration may be empty, or daily, weekly, 14-days, monthly, 2-months, yearly, other.
func (c *Client) BillDataPlans(network, duration string) ([]map[string]any, error) {
	q := url.Values{"network": {network}}
	if duration != "" {
		q.Set("duration", duration)
	}
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/bills/data-plans", q, nil, "", &out)
	return out, err
}

func (c *Client) BillDataDurations(network string) (map[string]any, error) {
	return c.billGet("/v1/bills/data-durations", url.Values{"network": {network}})
}

func (c *Client) BillTvPlans(provider string) ([]map[string]any, error) {
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/bills/tv-plans", url.Values{"provider": {provider}}, nil, "", &out)
	return out, err
}

// BillQuote returns what you will pay. kind is "airtime" or "electricity"; network or disco is optional.
func (c *Client) BillQuote(kind string, amount float64, network, disco string) (map[string]any, error) {
	q := url.Values{"type": {kind}, "amount": {strconv.FormatFloat(amount, 'f', -1, 64)}}
	if network != "" {
		q.Set("network", network)
	}
	if disco != "" {
		q.Set("disco", disco)
	}
	return c.billGet("/v1/bills/quote", q)
}

func (c *Client) VerifyMeter(disco, meterNumber, meterType string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/bills/verify-meter", nil, map[string]string{"disco": disco, "meter_number": meterNumber, "meter_type": meterType}, "", &out)
	return out, err
}

func (c *Client) VerifySmartcard(provider, smartcardNumber string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/bills/verify-smartcard", nil, map[string]string{"provider": provider, "smartcard_number": smartcardNumber}, "", &out)
	return out, err
}

// --- Webhooks ---

func (c *Client) ListWebhooks() ([]map[string]any, error) {
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/webhooks", nil, nil, "", &out)
	return out, err
}

// WebhookEvents lists the event names you can subscribe to.
func (c *Client) WebhookEvents() ([]string, error) {
	var out struct {
		Events []string `json:"events"`
	}
	err := c.do(http.MethodGet, "/v1/webhooks/events", nil, nil, "", &out)
	return out.Events, err
}

// CreateWebhook returns the endpoint including its signing secret, which is
// shown only once. Pass nil events to subscribe to all.
func (c *Client) CreateWebhook(endpointURL string, events []string, description string) (map[string]any, error) {
	body := map[string]any{"url": endpointURL}
	if events != nil {
		body["events"] = events
	}
	if description != "" {
		body["description"] = description
	}
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/webhooks", nil, body, "", &out)
	return out, err
}

// UpdateWebhook sends only the fields you pass (url, events, description, is_active).
func (c *Client) UpdateWebhook(endpointID string, fields map[string]any) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPatch, "/v1/webhooks/"+endpointID, nil, fields, "", &out)
	return out, err
}

func (c *Client) DeleteWebhook(endpointID string) error {
	return c.do(http.MethodDelete, "/v1/webhooks/"+endpointID, nil, nil, "", nil)
}

func (c *Client) RotateWebhookSecret(endpointID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/webhooks/"+endpointID+"/rotate-secret", nil, nil, "", &out)
	return out, err
}

func (c *Client) TestWebhook(endpointID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/webhooks/"+endpointID+"/test", nil, nil, "", &out)
	return out, err
}

func (c *Client) WebhookDeliveries(endpointID string, limit int) ([]map[string]any, error) {
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/webhooks/"+endpointID+"/deliveries", url.Values{"limit": {strconv.Itoa(limit)}}, nil, "", &out)
	return out, err
}

// AllWebhookDeliveries lists every event we sent you, newest first. Filters: event, status, endpoint_id, limit, offset.
func (c *Client) AllWebhookDeliveries(query url.Values) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/v1/webhooks/deliveries", query, nil, "", &out)
	return out, err
}

// ResendWebhookDelivery sends the same event again as a new delivery.
func (c *Client) ResendWebhookDelivery(deliveryID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/webhooks/deliveries/"+deliveryID+"/resend", nil, nil, "", &out)
	return out, err
}

func (c *Client) RetryWebhookDelivery(deliveryID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/webhooks/deliveries/"+deliveryID+"/retry", nil, nil, "", &out)
	return out, err
}

// --- Verification records ---

func (c *Client) GetVerification(id string) (*VerifyResponse, error) {
	var out VerifyResponse
	err := c.do(http.MethodGet, "/v1/verify/"+id, nil, nil, "", &out)
	return &out, err
}

func (c *Client) VerifyDocumentTamper(filePath, idempotencyKey string) (*VerifyResponse, error) {
	var out VerifyResponse
	err := c.doMultipart("/v1/verify/document-tamper", map[string]string{"file": filePath}, idempotencyKey, &out)
	return &out, err
}

// --- Take payments ---
// Account numbers for your customers, card links, withdrawals and refunds.
// A test key works in test mode with pretend money; a live key moves real money.

func (c *Client) biz(method, path string, q url.Values, body any) (map[string]any, error) {
	var out map[string]any
	err := c.do(method, "/v1/business"+path, q, body, "", &out)
	return out, err
}

func (c *Client) PaymentsOverview() (map[string]any, error) {
	return c.biz(http.MethodGet, "/overview", nil, nil)
}
func (c *Client) PaymentBanks() (map[string]any, error) {
	return c.biz(http.MethodGet, "/banks", nil, nil)
}
func (c *Client) Payouts() (map[string]any, error) {
	return c.biz(http.MethodGet, "/payouts", nil, nil)
}
func (c *Client) Refunds() (map[string]any, error) {
	return c.biz(http.MethodGet, "/refunds", nil, nil)
}

// SaveBusinessProfile: businessType is "registered" or "individual".
func (c *Client) SaveBusinessProfile(businessName, businessType, contactPhone string) (map[string]any, error) {
	return c.biz(http.MethodPut, "/profile", nil, map[string]string{"business_name": businessName, "business_type": businessType, "contact_phone": contactPhone})
}

func (c *Client) SavePayoutAccount(bankCode, accountNumber string) (map[string]any, error) {
	return c.biz(http.MethodPut, "/payout-account", nil, map[string]string{"bank_code": bankCode, "account_number": accountNumber})
}

func (c *Client) RequestPayout(amount float64) (map[string]any, error) {
	return c.biz(http.MethodPost, "/payouts", nil, map[string]any{"amount": amount})
}

// RequestRefund refunds a payment. Pass amount 0 to refund all of it.
func (c *Client) RequestRefund(paymentID string, amount float64, reason string) (map[string]any, error) {
	body := map[string]any{"payment_id": paymentID, "reason": reason}
	if amount > 0 {
		body["amount"] = amount
	}
	return c.biz(http.MethodPost, "/refunds", nil, body)
}

func (c *Client) PaymentEvents(limit int) (map[string]any, error) {
	return c.biz(http.MethodGet, "/events", url.Values{"limit": {strconv.Itoa(limit)}}, nil)
}

// CreateVirtualAccount makes a bank account number for one of your customers.
// The banks need a BVN or NIN for a permanent number; it is used once and never stored.
func (c *Client) CreateVirtualAccount(customerName, customerEmail, reference, bvn, nin string) (map[string]any, error) {
	body := map[string]any{"customer_name": customerName, "customer_email": customerEmail}
	for k, v := range map[string]string{"reference": reference, "bvn": bvn, "nin": nin} {
		if v != "" {
			body[k] = v
		}
	}
	return c.biz(http.MethodPost, "/virtual-accounts", nil, body)
}

func (c *Client) VirtualAccounts(limit int) (map[string]any, error) {
	return c.biz(http.MethodGet, "/virtual-accounts", url.Values{"limit": {strconv.Itoa(limit)}}, nil)
}

// ReceivedPayments lists payments received. virtualAccountID and status may be empty.
func (c *Client) ReceivedPayments(virtualAccountID, status string, limit int) (map[string]any, error) {
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	if virtualAccountID != "" {
		q.Set("virtual_account_id", virtualAccountID)
	}
	if status != "" {
		q.Set("status", status)
	}
	return c.biz(http.MethodGet, "/payments", q, nil)
}

// CreateCardCharge returns a checkout link to send your customer.
func (c *Client) CreateCardCharge(amount float64, customerEmail, customerName, description string) (map[string]any, error) {
	return c.biz(http.MethodPost, "/charges", nil, map[string]any{"amount": amount, "customer_email": customerEmail, "customer_name": customerName, "description": description})
}

// SimulatePayment is for test keys only.
func (c *Client) SimulatePayment(virtualAccountID string, amount float64) (map[string]any, error) {
	return c.biz(http.MethodPost, "/test/simulate-payment", nil, map[string]any{"virtual_account_id": virtualAccountID, "amount": amount})
}

func (c *Client) SimulateCardPayment(reference string) (map[string]any, error) {
	return c.biz(http.MethodPost, "/test/simulate-card-payment", nil, map[string]string{"reference": reference})
}

// --- WhatsApp for platforms ---
// A workspace (one number) for every client you serve. A test key works on test
// workspaces: pretend linking, no real number, never billed.

func (c *Client) wa(method, path string, q url.Values, body any) (map[string]any, error) {
	var out map[string]any
	err := c.do(method, "/v1/wa"+path, q, body, "", &out)
	return out, err
}

// CreateWhatsAppWorkspace makes one workspace per client business. Fields: label (required),
// external_id, ai_enabled, profile, brand, mode.
func (c *Client) CreateWhatsAppWorkspace(fields map[string]any) (map[string]any, error) {
	return c.wa(http.MethodPost, "/channels", nil, fields)
}

func (c *Client) WhatsAppWorkspaces(externalID string) ([]map[string]any, error) {
	q := url.Values{}
	if externalID != "" {
		q.Set("external_id", externalID)
	}
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/wa/channels", q, nil, "", &out)
	return out, err
}

// GetWhatsAppWorkspace returns live status, plus "qr" (text to draw as a QR code) while waiting to be scanned.
func (c *Client) GetWhatsAppWorkspace(id string) (map[string]any, error) {
	return c.wa(http.MethodGet, "/channels/"+id, nil, nil)
}

// UpdateWhatsAppWorkspace changes label, ai_enabled, mode or brand.
func (c *Client) UpdateWhatsAppWorkspace(id string, fields map[string]any) (map[string]any, error) {
	return c.wa(http.MethodPatch, "/channels/"+id, nil, fields)
}

func (c *Client) DeleteWhatsAppWorkspace(id string) error {
	_, err := c.wa(http.MethodDelete, "/channels/"+id, nil, nil)
	return err
}

// ConnectWhatsApp starts linking and returns connect_url, the hosted page to send your client to.
func (c *Client) ConnectWhatsApp(id string) (map[string]any, error) {
	return c.wa(http.MethodPost, "/channels/"+id+"/connect", nil, map[string]any{})
}

func (c *Client) DisconnectWhatsApp(id string) (map[string]any, error) {
	return c.wa(http.MethodPost, "/channels/"+id+"/disconnect", nil, map[string]any{})
}

func (c *Client) SetWhatsAppProfile(id string, profile map[string]any) (map[string]any, error) {
	return c.wa(http.MethodPut, "/channels/"+id+"/profile", nil, profile)
}

func (c *Client) SetWhatsAppRules(id string, rules []map[string]any) (map[string]any, error) {
	return c.wa(http.MethodPut, "/channels/"+id+"/rules", nil, map[string]any{"rules": rules})
}

// SetWhatsAppWebsite sets where customer messages are sent, signed. It returns the signing secret.
func (c *Client) SetWhatsAppWebsite(id, websiteURL, fallbackMessage string) (map[string]any, error) {
	body := map[string]any{"url": websiteURL}
	if fallbackMessage != "" {
		body["fallback_message"] = fallbackMessage
	}
	return c.wa(http.MethodPut, "/channels/"+id+"/website", nil, body)
}

// SendWhatsApp replies to a customer who messaged this number. Fields: to, text, media, session ("start" or "end").
func (c *Client) SendWhatsApp(id string, fields map[string]any) (map[string]any, error) {
	return c.wa(http.MethodPost, "/channels/"+id+"/send", nil, fields)
}

func (c *Client) ReplyToWhatsAppConversation(conversationID, body string) (map[string]any, error) {
	return c.wa(http.MethodPost, "/conversations/"+conversationID+"/reply", nil, map[string]any{"body": body})
}

// TestWhatsAppAssistant asks the assistant a question. Nothing is sent or billed.
func (c *Client) TestWhatsAppAssistant(id, message string) (map[string]any, error) {
	return c.wa(http.MethodPost, "/channels/"+id+"/test", nil, map[string]any{"message": message, "history": []any{}})
}

// TestWhatsAppScan pretends the QR code was scanned (test workspaces only).
func (c *Client) TestWhatsAppScan(id string) (map[string]any, error) {
	return c.wa(http.MethodPost, "/channels/"+id+"/test/scan", nil, map[string]any{})
}

// TestWhatsAppInbound sends a pretend customer message through your rules, your website and the assistant (test workspaces only).
func (c *Client) TestWhatsAppInbound(id, text, fromPhone string) (map[string]any, error) {
	body := map[string]any{"text": text}
	if fromPhone != "" {
		body["from_phone"] = fromPhone
	}
	return c.wa(http.MethodPost, "/channels/"+id+"/test/inbound", nil, body)
}

func (c *Client) WhatsAppPlans() (map[string]any, error) {
	return c.wa(http.MethodGet, "/plans", nil, nil)
}
func (c *Client) WhatsAppSubscription() (map[string]any, error) {
	return c.wa(http.MethodGet, "/subscription", nil, nil)
}
func (c *Client) UseWhatsAppPayPerReply() (map[string]any, error) {
	return c.wa(http.MethodPost, "/subscription", nil, map[string]any{"tier": "payg"})
}

// WhatsAppUsage reports replies and cost per workspace and per day. workspaceID may be empty.
func (c *Client) WhatsAppUsage(days int, workspaceID string) (map[string]any, error) {
	q := url.Values{"days": {strconv.Itoa(days)}}
	if workspaceID != "" {
		q.Set("channel_id", workspaceID)
	}
	return c.wa(http.MethodGet, "/usage", q, nil)
}

func (c *Client) GetWhatsAppBrand() (map[string]any, error) {
	return c.wa(http.MethodGet, "/brand", nil, nil)
}

// SetWhatsAppBrand sets your default look for every connect page. Empty fields are cleared.
func (c *Client) SetWhatsAppBrand(name, logoURL, color string) (map[string]any, error) {
	body := map[string]any{"name": name, "logo_url": logoURL}
	if color != "" {
		body["color"] = color
	}
	return c.wa(http.MethodPut, "/brand", nil, body)
}

// UploadWhatsAppLogo uploads a PNG, JPEG, WebP or SVG up to 5 MB. Pass a workspace id for one workspace only.
func (c *Client) UploadWhatsAppLogo(filePath, workspaceID string) (map[string]any, error) {
	path := "/v1/wa/brand/logo"
	if workspaceID != "" {
		path = "/v1/wa/channels/" + workspaceID + "/brand/logo"
	}
	var out map[string]any
	err := c.doMultipart(path, map[string]string{"file": filePath}, "", &out)
	return out, err
}

// --- Address verification ---

// VerifyAddress starts an address check. Send the person confirm_url from the result so they can share their location.
func (c *Client) VerifyAddress(address, fullName, country string, radiusM int, reference, idempotencyKey string) (map[string]any, error) {
	body := map[string]any{"address": address, "radius_m": radiusM}
	if fullName != "" {
		body["full_name"] = fullName
	}
	if country != "" {
		body["country"] = country
	}
	if reference != "" {
		body["reference"] = reference
	}
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/verify/address", nil, body, idempotencyKey, &out)
	return out, err
}

func (c *Client) GetAddressVerification(id string) (map[string]any, error) {
	return c.billGet("/v1/verify/address/"+id, nil)
}

func (c *Client) SubmitAddressLocation(id string, latitude, longitude, accuracyM float64) (map[string]any, error) {
	body := map[string]any{"latitude": latitude, "longitude": longitude}
	if accuracyM > 0 {
		body["accuracy_m"] = accuracyM
	}
	var out map[string]any
	err := c.do(http.MethodPost, "/v1/verify/address/"+id+"/location", nil, body, "", &out)
	return out, err
}

// UploadAddressDocument sends a utility bill or bank statement showing the address.
func (c *Client) UploadAddressDocument(id, filePath string) (map[string]any, error) {
	var out map[string]any
	err := c.doMultipart("/v1/verify/address/"+id+"/document", map[string]string{"file": filePath}, "", &out)
	return out, err
}

// --- Account ---

// AccountOverview returns spend, top-ups, API calls and success rate for the
// last `days`, compared with the period before.
func (c *Client) AccountOverview(days int) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/v1/account/overview", url.Values{"days": {strconv.Itoa(days)}}, nil, "", &out)
	return out, err
}

// Request calls any endpoint not wrapped above. path starts with /v1/. body may
// be nil; out may be nil to ignore the response.
func (c *Client) Request(method, path string, query url.Values, body any, out any) error {
	return c.do(method, path, query, body, "", out)
}

// --- File & image tools ---
// Each tool returns the converted file as bytes (PDF, DOCX, PNG, ZIP...).
// Write them to disk yourself with os.WriteFile. OCR returns parsed JSON.

type filePart struct{ field, path string }

// tool POSTs multipart to /v1/tools/<slug>. A field name may repeat to send
// several files. It returns the body and the response content type.
func (c *Client) tool(slug string, files []filePart, fields map[string]string) ([]byte, string, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for _, fp := range files {
		f, err := os.Open(fp.path)
		if err != nil {
			return nil, "", err
		}
		part, err := w.CreateFormFile(fp.field, filepath.Base(fp.path))
		if err == nil {
			_, err = io.Copy(part, f)
		}
		f.Close()
		if err != nil {
			return nil, "", err
		}
	}
	for k, v := range fields {
		if v != "" {
			_ = w.WriteField(k, v)
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/tools/"+slug, body)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("X-IdentifyOrg-Key", c.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.finishFile(req)
}

func (c *Client) finishFile(req *http.Request) ([]byte, string, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		var env errorEnvelope
		_ = json.Unmarshal(raw, &env)
		return nil, "", &APIError{Status: resp.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
	}
	return raw, resp.Header.Get("Content-Type"), nil
}

func (c *Client) simpleTool(slug, path string, fields map[string]string) ([]byte, error) {
	raw, _, err := c.tool(slug, []filePart{{"file", path}}, fields)
	return raw, err
}

func (c *Client) PDFToDOCX(path string) ([]byte, error) {
	return c.simpleTool("pdf-to-docx", path, nil)
}
func (c *Client) DOCXToPDF(path string) ([]byte, error) {
	return c.simpleTool("docx-to-pdf", path, nil)
}

// PDFMerge merges PDFs in the order given.
func (c *Client) PDFMerge(paths []string) ([]byte, error) {
	parts := make([]filePart, 0, len(paths))
	for _, p := range paths {
		parts = append(parts, filePart{"files", p})
	}
	raw, _, err := c.tool("pdf-merge", parts, nil)
	return raw, err
}

// PDFSplit extracts pages, e.g. "1-3,7,9-". password is only for locked PDFs.
func (c *Client) PDFSplit(path, pages, password string) ([]byte, error) {
	return c.simpleTool("pdf-split", path, map[string]string{"pages": pages, "password": password})
}

func (c *Client) PDFCompress(path, password string) ([]byte, error) {
	return c.simpleTool("pdf-compress", path, map[string]string{"password": password})
}

func (c *Client) PDFWatermark(path, text string, opacity float64, password string) ([]byte, error) {
	f := map[string]string{"text": text, "password": password}
	if opacity > 0 {
		f["opacity"] = strconv.FormatFloat(opacity, 'f', -1, 64)
	}
	return c.simpleTool("pdf-watermark", path, f)
}

func (c *Client) PDFProtect(path, newPassword, password string) ([]byte, error) {
	return c.simpleTool("pdf-protect", path, map[string]string{"new_password": newPassword, "password": password})
}

func (c *Client) PDFUnlock(path, password string) ([]byte, error) {
	return c.simpleTool("pdf-unlock", path, map[string]string{"password": password})
}

// OCR returns the parsed JSON response containing the recognised text.
func (c *Client) OCR(path, password string) (map[string]any, error) {
	raw, err := c.simpleTool("ocr", path, map[string]string{"password": password})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveBackground returns a transparent PNG.
func (c *Client) RemoveBackground(path string) ([]byte, error) {
	return c.simpleTool("remove-background", path, nil)
}

// ImageResize: pass 0 for width or height to keep the ratio; "" for the defaults.
func (c *Client) ImageResize(path string, width, height int, mode, outputFormat string) ([]byte, error) {
	f := map[string]string{"mode": mode, "output_format": outputFormat}
	if width > 0 {
		f["width"] = strconv.Itoa(width)
	}
	if height > 0 {
		f["height"] = strconv.Itoa(height)
	}
	return c.simpleTool("image-resize", path, f)
}

func (c *Client) ImageConvert(path, targetFormat string) ([]byte, error) {
	return c.simpleTool("image-convert", path, map[string]string{"target_format": targetFormat})
}

func (c *Client) ImageCompress(path string, quality int) ([]byte, error) {
	f := map[string]string{}
	if quality > 0 {
		f["quality"] = strconv.Itoa(quality)
	}
	return c.simpleTool("image-compress", path, f)
}

func (c *Client) AudioConvert(path, targetFormat string) ([]byte, error) {
	return c.simpleTool("audio-convert", path, map[string]string{"target_format": targetFormat})
}

// AppIcons: one square image in, a ZIP of every iOS, Android and web icon size out.
func (c *Client) AppIcons(path string) ([]byte, error) { return c.simpleTool("app-icons", path, nil) }

// StoreAssets: one image in, a ZIP of every Play Store / App Store size out.
func (c *Client) StoreAssets(path, suite string) ([]byte, error) {
	return c.simpleTool("store-assets", path, map[string]string{"suite": suite})
}

// StoreAssetsBatchStart starts a full-submission batch. slots maps a slot name
// (ios_iphone_6_3, ios_ipad_13, ios_watch, android_phone, android_icon, ...) to
// file paths. Poll StoreAssetsBatchStatus, then StoreAssetsBatchDownload.
func (c *Client) StoreAssetsBatchStart(slots map[string][]string) (map[string]any, error) {
	var parts []filePart
	for slot, paths := range slots {
		for _, p := range paths {
			parts = append(parts, filePart{slot, p})
		}
	}
	raw, _, err := c.tool("store-assets/start", parts, nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	return out, json.Unmarshal(raw, &out)
}

func (c *Client) StoreAssetsBatchStatus(jobID string) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/v1/tools/store-assets/"+jobID, nil, nil, "", &out)
	return out, err
}

func (c *Client) StoreAssetsBatchDownload(jobID string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/v1/tools/store-assets/"+jobID+"/download", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-IdentifyOrg-Key", c.APIKey)
	raw, _, err := c.finishFile(req)
	return raw, err
}

// GenerateLogo returns a PNG. Pass "" / 0 for the defaults.
func (c *Client) GenerateLogo(companyName, style, primaryColor, secondaryColor string, size int) ([]byte, error) {
	body := map[string]any{"company_name": companyName}
	if style != "" {
		body["style"] = style
	}
	if primaryColor != "" {
		body["primary_color"] = primaryColor
	}
	if secondaryColor != "" {
		body["secondary_color"] = secondaryColor
	}
	if size > 0 {
		body["size"] = size
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/tools/generate-logo", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-IdentifyOrg-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	raw, _, err := c.finishFile(req)
	return raw, err
}
