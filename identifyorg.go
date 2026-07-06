// Package identifyorg is the IdentifyOrg server-side SDK for Go:
// identity verification, prepaid billing, and realtime call/chat token
// issuance. Server-side only — use your SECRET key
// (io_test_sk_.../io_live_sk_...) here, never in client code. For
// browser/mobile clients, mint a token with Client.StreamingToken or
// Client.ChatToken on your backend and hand the result to the IdentifyOrg
// JS/React Native/Flutter/Kotlin SDK.
package identifyorg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const DefaultBaseURL = "https://api.identifyorg.com"

// APIError is returned for any non-2xx response.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	code := e.Code
	if code == "" {
		code = fmt.Sprintf("%d", e.Status)
	}
	return fmt.Sprintf("identifyorg API error (%s): %s", code, e.Message)
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error"`
}

// Client is a IdentifyOrg API client scoped to one secret API key.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a client. Pass "" for baseURL to use DefaultBaseURL.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		APIKey:     apiKey,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: http.DefaultClient,
	}
}

func (c *Client) do(method, path string, query url.Values, body any, idempotencyKey string, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	fullURL := c.BaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, fullURL, reader)
	if err != nil {
		return err
	}
	req.Header.Set("X-IdentifyOrg-Key", c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		var env errorEnvelope
		_ = json.Unmarshal(raw, &env)
		return &APIError{Status: resp.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// --- Verification ---

type VerifyResponse struct {
	ID               string         `json:"id"`
	Type             string         `json:"type"`
	Status           string         `json:"status"`
	Match            *bool          `json:"match"`
	ConfidenceScore  *int           `json:"confidence_score"`
	Data             map[string]any `json:"data"`
	Cost             float64        `json:"cost"`
	Currency         string         `json:"currency"`
	IsTest           bool           `json:"is_test"`
	CreatedAt        string         `json:"created_at"`
}

func (c *Client) VerifyBVN(bvn, firstName, lastName, idempotencyKey string) (*VerifyResponse, error) {
	var out VerifyResponse
	body := map[string]string{"bvn": bvn, "first_name": firstName, "last_name": lastName}
	err := c.do(http.MethodPost, "/v1/verify/bvn", nil, body, idempotencyKey, &out)
	return &out, err
}

func (c *Client) VerifyNIN(nin, firstName, lastName, idempotencyKey string) (*VerifyResponse, error) {
	var out VerifyResponse
	body := map[string]string{"nin": nin, "first_name": firstName, "last_name": lastName}
	err := c.do(http.MethodPost, "/v1/verify/nin", nil, body, idempotencyKey, &out)
	return &out, err
}

func (c *Client) VerifyFRSC(licenceNumber, firstName, lastName, idempotencyKey string) (*VerifyResponse, error) {
	var out VerifyResponse
	body := map[string]string{"licence_number": licenceNumber, "first_name": firstName, "last_name": lastName}
	err := c.do(http.MethodPost, "/v1/verify/frsc", nil, body, idempotencyKey, &out)
	return &out, err
}

func (c *Client) VerifyCreditScore(bvn, firstName, lastName, idempotencyKey string) (*VerifyResponse, error) {
	var out VerifyResponse
	body := map[string]string{"bvn": bvn, "first_name": firstName, "last_name": lastName}
	err := c.do(http.MethodPost, "/v1/verify/credit-score", nil, body, idempotencyKey, &out)
	return &out, err
}

func (c *Client) VerifyCAC(rcNumber, firstName, lastName, idempotencyKey string) (*VerifyResponse, error) {
	var out VerifyResponse
	body := map[string]string{"rc_number": rcNumber, "first_name": firstName, "last_name": lastName}
	err := c.do(http.MethodPost, "/v1/verify/cac", nil, body, idempotencyKey, &out)
	return &out, err
}

// --- Realtime: calls, streaming, chat ---

type RealtimeToken struct {
	SessionID          string  `json:"session_id"`
	RoomName           string  `json:"room_name"`
	Token              string  `json:"token"`
	URL                string  `json:"url"`
	Type               string  `json:"type"`
	ExpiresInMinutes   int     `json:"expires_in_minutes"`
	PricePerMinute     float64 `json:"price_per_minute"`
	Currency           string  `json:"currency"`
	IsTest             bool    `json:"is_test"`
}

// StreamingToken mints a token for a video/voice/stream session. sessionType
// is "video", "voice", or "stream". Mint this server-side, then hand the
// result to a client SDK (JS/React Native/Flutter/Kotlin) — never expose
// your secret key to a client.
func (c *Client) StreamingToken(sessionType, identity, displayName, roomName, role string) (*RealtimeToken, error) {
	var out RealtimeToken
	body := map[string]string{
		"identity": identity, "display_name": displayName, "room_name": roomName, "role": role,
	}
	err := c.do(http.MethodPost, "/v1/streaming/"+sessionType+"/token", nil, body, "", &out)
	return &out, err
}

func (c *Client) ChatToken(visitorID, visitorName, channelID string) (*RealtimeToken, error) {
	var out RealtimeToken
	body := map[string]string{"visitor_id": visitorID, "visitor_name": visitorName, "channel_id": channelID}
	err := c.do(http.MethodPost, "/v1/chat/token", nil, body, "", &out)
	return &out, err
}

// --- Billing ---

type TopupResponse struct {
	PaymentID         string  `json:"payment_id"`
	PaymentReference  string  `json:"payment_reference"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Status            string  `json:"status"`
	CheckoutURL       string  `json:"checkout_url"`
}

func (c *Client) Topup(amount float64) (*TopupResponse, error) {
	var out TopupResponse
	err := c.do(http.MethodPost, "/v1/payments/topup", nil, map[string]float64{"amount": amount}, "", &out)
	return &out, err
}

type BalanceResponse struct {
	Balance    float64 `json:"balance"`
	Currency   string  `json:"currency"`
	LowBalance bool    `json:"low_balance"`
}

func (c *Client) Balance() (*BalanceResponse, error) {
	var out BalanceResponse
	err := c.do(http.MethodGet, "/v1/balance", nil, nil, "", &out)
	return &out, err
}

// Pricing returns the current price list (no auth required upstream, but
// this client always sends the API key, which is harmless).
func (c *Client) Pricing() ([]map[string]any, error) {
	var out []map[string]any
	err := c.do(http.MethodGet, "/v1/pricing", nil, nil, "", &out)
	return out, err
}

// Usage returns raw JSON (summary + records) — decode into your own struct
// or a map[string]any depending on what you need.
func (c *Client) Usage(query url.Values) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/v1/usage", query, nil, "", &out)
	return out, err
}

func (c *Client) Transactions(query url.Values) (map[string]any, error) {
	var out map[string]any
	err := c.do(http.MethodGet, "/v1/transactions", query, nil, "", &out)
	return out, err
}

// --- Host moderation — call from your backend in response to your own
// host UI (mute/kick/end buttons), never directly from a client. ---

type ParticipantInfo struct {
	Identity    string `json:"identity"`
	Name        string `json:"name"`
	AudioMuted  *bool  `json:"audio_muted"`
	VideoMuted  *bool  `json:"video_muted"`
}

func (c *Client) ListParticipants(sessionID string) ([]ParticipantInfo, error) {
	var out []ParticipantInfo
	err := c.do(http.MethodGet, "/v1/streaming/sessions/"+sessionID+"/participants", nil, nil, "", &out)
	return out, err
}

func (c *Client) moderationAction(sessionID, identity, action string) error {
	path := "/v1/streaming/sessions/" + sessionID + "/participants/" + identity + "/" + action
	return c.do(http.MethodPost, path, nil, map[string]string{}, "", nil)
}

func (c *Client) MuteParticipant(sessionID, identity string) error {
	return c.moderationAction(sessionID, identity, "mute")
}

func (c *Client) UnmuteParticipant(sessionID, identity string) error {
	return c.moderationAction(sessionID, identity, "unmute")
}

func (c *Client) CameraOffParticipant(sessionID, identity string) error {
	return c.moderationAction(sessionID, identity, "camera-off")
}

func (c *Client) CameraOnParticipant(sessionID, identity string) error {
	return c.moderationAction(sessionID, identity, "camera-on")
}

func (c *Client) KickParticipant(sessionID, identity string) error {
	return c.moderationAction(sessionID, identity, "kick")
}

func (c *Client) EndSession(sessionID string) error {
	return c.do(http.MethodPost, "/v1/streaming/sessions/"+sessionID+"/end", nil, map[string]string{}, "", nil)
}

// --- The gifting economy ---

type GiftCatalogItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Emoji      string `json:"emoji"`
	PriceCoins int    `json:"price_coins"`
	IsActive   bool   `json:"is_active"`
}

func (c *Client) CreateGift(name, emoji string, priceCoins int) (*GiftCatalogItem, error) {
	var out GiftCatalogItem
	body := map[string]any{"name": name, "emoji": emoji, "price_coins": priceCoins}
	err := c.do(http.MethodPost, "/v1/streaming/gifts", nil, body, "", &out)
	return &out, err
}

func (c *Client) ListGifts() ([]GiftCatalogItem, error) {
	var out []GiftCatalogItem
	err := c.do(http.MethodGet, "/v1/streaming/gifts", nil, nil, "", &out)
	return out, err
}

func (c *Client) DeleteGift(giftID string) error {
	return c.do(http.MethodDelete, "/v1/streaming/gifts/"+giftID, nil, nil, "", nil)
}

type EndUser struct {
	ExternalID  string `json:"external_id"`
	DisplayName string `json:"display_name"`
	CoinBalance int    `json:"coin_balance"`
}

func (c *Client) CreditCoins(externalID string, amount int, displayName string) (*EndUser, error) {
	var out EndUser
	body := map[string]any{"amount": amount, "display_name": displayName}
	err := c.do(http.MethodPost, "/v1/streaming/viewers/"+externalID+"/coins/credit", nil, body, "", &out)
	return &out, err
}

func (c *Client) GetViewer(externalID string) (*EndUser, error) {
	var out EndUser
	err := c.do(http.MethodGet, "/v1/streaming/viewers/"+externalID, nil, nil, "", &out)
	return &out, err
}

type SendGiftResponse struct {
	GiftTransactionID string `json:"gift_transaction_id"`
	GiftName          string `json:"gift_name"`
	CoinsSpent        int    `json:"coins_spent"`
	FromBalanceAfter  int    `json:"from_balance_after"`
	ToIdentity        string `json:"to_identity"`
}

func (c *Client) SendGift(fromExternalID, toIdentity, giftID, roomName string) (*SendGiftResponse, error) {
	var out SendGiftResponse
	body := map[string]string{
		"from_external_id": fromExternalID,
		"to_identity":      toIdentity,
		"gift_id":          giftID,
		"room_name":        roomName,
	}
	err := c.do(http.MethodPost, "/v1/streaming/gifts/send", nil, body, "", &out)
	return &out, err
}
