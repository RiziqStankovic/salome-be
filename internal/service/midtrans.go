package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"salome-be/internal/config"
	"strings"
	"time"
)

type MidtransService struct {
	serverKey string
	client    *http.Client
}

type MidtransTransactionStatus struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	PaymentType       string `json:"payment_type"`
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	GrossAmount       string `json:"gross_amount"`
	Currency          string `json:"currency"`
	PaymentCode       string `json:"payment_code"`
	RedirectURL       string `json:"redirect_url"`
	MerchantID        string `json:"merchant_id"`
	VaNumbers         []struct {
		Bank     string `json:"bank"`
		VaNumber string `json:"va_number"`
	} `json:"va_numbers"`
	Actions []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"actions"`
}

// Payment Link API Response Structure
type MidtransPaymentLinkResponse struct {
	ID              int    `json:"id"`
	PaymentLinkID   string `json:"payment_link_id"`
	OrderID         string `json:"order_id"`
	GrossAmount     int    `json:"gross_amount"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
	ExpiryTime      string `json:"expiry_time"`
	DynamicAmount   any    `json:"dynamic_amount"`
	PaymentSettings any    `json:"payment_settings"`
	Purchases       []struct {
		ID             int    `json:"id"`
		SnapToken      string `json:"snap_token"`
		OrderID        string `json:"order_id"`
		PaymentStatus  string `json:"payment_status"`
		PaymentMethod  string `json:"payment_method"`
		TransactionID  string `json:"transaction_id"`
		Acquirer       string `json:"acquirer"`
		AmountValue    int    `json:"amount_value"`
		AmountCurrency string `json:"amount_currency"`
		ExpiryTime     string `json:"expiry_time"`
		CreatedAt      string `json:"createdAt"`
		UpdatedAt      string `json:"updatedAt"`
		PaymentLinkID  int    `json:"PaymentLinkId"`
		PaymentLinkID2 int    `json:"payment_link_id"`
	} `json:"purchases"`
}

type MidtransResponse struct {
	StatusCode        string `json:"status_code"`
	StatusMessage     string `json:"status_message"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	PaymentType       string `json:"payment_type"`
	TransactionTime   string `json:"transaction_time"`
	GrossAmount       string `json:"gross_amount"`
	Currency          string `json:"currency"`
	FraudStatus       string `json:"fraud_status"`
	Actions           []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"actions"`
}

func NewMidtransService() *MidtransService {
	// Coba ambil dari environment variable dulu
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	if serverKey == "" {
		// Jika tidak ada, ambil dari config
		appConfig := config.GetConfig()
		if appConfig.Midtrans.ServerKey != "" {
			serverKey = appConfig.Midtrans.ServerKey
		} else {
			// Fallback ke hardcoded value
			serverKey = "SB-Mid-server-i1dcC7yEZ88t9ojPby0wej3n"
		}
	}

	fmt.Printf("🔑 [MIDTRANS DEBUG] Using server key: %s\n", serverKey[:20]+"...")

	return &MidtransService{
		serverKey: serverKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetTransactionStatus mengambil status transaksi dari Midtrans menggunakan Payment Link API
func (m *MidtransService) GetTransactionStatus(paymentLinkID string) (*MidtransTransactionStatus, error) {
	return m.checkPaymentLinkStatus(paymentLinkID)
}

func (m *MidtransService) checkPaymentLinkStatus(paymentLinkID string) (*MidtransTransactionStatus, error) {
	url := fmt.Sprintf("https://api.sandbox.midtrans.com/v1/payment-links/%s", paymentLinkID)

	fmt.Printf("🔍 [MIDTRANS DEBUG] Checking payment link status for ID: %s\n", paymentLinkID)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Request URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error creating request: %v\n", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Set authorization header
	auth := base64.StdEncoding.EncodeToString([]byte(m.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)

	fmt.Printf("🔍 [MIDTRANS DEBUG] Request headers: Authorization=Basic %s\n", auth[:10]+"...")

	resp, err := m.client.Do(req)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error reading response body: %v\n", err)
		return nil, err
	}

	fmt.Printf("🔍 [MIDTRANS DEBUG] Response status: %d\n", resp.StatusCode)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ [MIDTRANS DEBUG] API error - Status: %d, Body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("midtrans API error: %s", string(body))
	}

	var paymentLinkResp MidtransPaymentLinkResponse
	err = json.Unmarshal(body, &paymentLinkResp)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error unmarshaling response: %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ [MIDTRANS DEBUG] Payment link response parsed successfully\n")
	fmt.Printf("✅ [MIDTRANS DEBUG] Payment Link ID: %s\n", paymentLinkResp.PaymentLinkID)
	fmt.Printf("✅ [MIDTRANS DEBUG] Order ID: %s\n", paymentLinkResp.OrderID)
	fmt.Printf("✅ [MIDTRANS DEBUG] Status: %s\n", paymentLinkResp.Status)
	fmt.Printf("✅ [MIDTRANS DEBUG] Number of purchases: %d\n", len(paymentLinkResp.Purchases))

	// Cari purchase dengan status apapun (SETTLEMENT, EXPIRE, dll)
	var foundPurchase *MidtransPaymentLinkResponse
	var purchaseStatus string
	var purchaseMethod string
	var purchaseOrderID string
	var purchaseTransactionID string
	var purchaseAmount int
	var purchaseCurrency string
	var purchaseTime string

	for i, purchase := range paymentLinkResp.Purchases {
		fmt.Printf("✅ [MIDTRANS DEBUG] Purchase %d: OrderID=%s, Status=%s, Method=%s\n",
			i+1, purchase.OrderID, purchase.PaymentStatus, purchase.PaymentMethod)

		// Ambil purchase terbaru (biasanya hanya ada 1)
		foundPurchase = &paymentLinkResp
		purchaseStatus = purchase.PaymentStatus
		purchaseMethod = purchase.PaymentMethod
		purchaseOrderID = purchase.OrderID
		purchaseTransactionID = purchase.TransactionID
		purchaseAmount = purchase.AmountValue
		purchaseCurrency = purchase.AmountCurrency
		purchaseTime = purchase.UpdatedAt
		fmt.Printf("✅ [MIDTRANS DEBUG] Found purchase with status: %s\n", purchaseStatus)
		break
	}

	// Jika ada purchase, return status berdasarkan purchase status
	if foundPurchase != nil {
		// Map status dari Midtrans ke format yang diharapkan
		var mappedStatus string
		switch purchaseStatus {
		case "SETTLEMENT":
			mappedStatus = "settlement" // Akan di-mapping ke "completed" di handler
		case "EXPIRE":
			mappedStatus = "expire"
		case "DENY":
			mappedStatus = "deny"
		case "CANCEL":
			mappedStatus = "cancel"
		case "CREATED":
			mappedStatus = "failed" // CREATED dengan method INIT = failed
		default:
			mappedStatus = "pending"
		}

		fmt.Printf("✅ [MIDTRANS DEBUG] Status mapping: %s -> %s\n", purchaseStatus, mappedStatus)
		fmt.Printf("✅ [MIDTRANS DEBUG] Payment method: %s\n", purchaseMethod)

		// Log khusus untuk status CREATED
		if purchaseStatus == "CREATED" {
			fmt.Printf("⚠️ [MIDTRANS DEBUG] CREATED status detected with method %s - marking as failed\n", purchaseMethod)
		}

		return &MidtransTransactionStatus{
			TransactionID:     purchaseTransactionID,
			OrderID:           purchaseOrderID,
			PaymentType:       purchaseMethod,
			TransactionTime:   purchaseTime,
			TransactionStatus: mappedStatus,
			GrossAmount:       fmt.Sprintf("%d", purchaseAmount),
			Currency:          purchaseCurrency,
		}, nil
	}

	// Jika tidak ada purchases, berarti transaksi belum pernah dibayar atau gagal
	fmt.Printf("⚠️ [MIDTRANS DEBUG] No purchases found, marking as failed\n")
	return &MidtransTransactionStatus{
		OrderID:           paymentLinkResp.OrderID,
		TransactionStatus: "failed",
		GrossAmount:       fmt.Sprintf("%d", paymentLinkResp.GrossAmount),
		Currency:          paymentLinkResp.Currency,
	}, nil

}

func (m *MidtransService) checkTransactionStatus(orderID string) (*MidtransTransactionStatus, error) {
	url := fmt.Sprintf("https://api.sandbox.midtrans.com/v2/%s/status", orderID)

	fmt.Printf("🔍 [MIDTRANS DEBUG] Checking transaction status for order ID: %s\n", orderID)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Request URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Failed to create request: %v\n", err)
		return nil, err
	}

	// Set authorization header
	auth := base64.StdEncoding.EncodeToString([]byte(m.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("🔍 [MIDTRANS DEBUG] Request headers: Authorization=Basic %s\n", auth[:10]+"...")

	resp, err := m.client.Do(req)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Request failed: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Failed to read response body: %v\n", err)
		return nil, err
	}

	fmt.Printf("🔍 [MIDTRANS DEBUG] Response status: %d\n", resp.StatusCode)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ [MIDTRANS DEBUG] API error - Status: %d, Body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("midtrans API error: %s", string(body))
	}

	var result MidtransTransactionStatus
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Failed to unmarshal response: %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ [MIDTRANS DEBUG] Successfully parsed response:\n")
	fmt.Printf("   - Order ID: %s\n", result.OrderID)
	fmt.Printf("   - Transaction Status: %s\n", result.TransactionStatus)
	fmt.Printf("   - Payment Type: %s\n", result.PaymentType)
	fmt.Printf("   - Gross Amount: %s\n", result.GrossAmount)
	fmt.Printf("   - Transaction Time: %s\n", result.TransactionTime)

	return &result, nil
}

// GetPaymentLinkURL mengambil URL payment link dari Midtrans
func (m *MidtransService) GetPaymentLinkURL(orderID string) (string, error) {
	// Cek apakah ada payment link yang tersimpan di database
	// Atau buat payment link baru jika belum ada

	// Untuk sementara, return URL sandbox Midtrans
	// Di production, ini harus disimpan di database saat membuat transaksi
	return fmt.Sprintf("https://app.sandbox.midtrans.com/payment-links/%s", orderID), nil
}

// IsTransactionSettled mengecek apakah transaksi sudah settlement
func (m *MidtransService) IsTransactionSettled(status string) bool {
	return status == "settlement" || status == "capture"
}

// IsTransactionPending mengecek apakah transaksi masih pending
func (m *MidtransService) IsTransactionPending(status string) bool {
	return status == "pending" || status == "challenge"
}

// IsTransactionFailed mengecek apakah transaksi gagal
func (m *MidtransService) IsTransactionFailed(status string) bool {
	return status == "deny" || status == "cancel" || status == "expire" || status == "EXPIRE" || status == "failed"
}

// GetPaymentLinkStatus mengambil status payment link dari Midtrans
func (m *MidtransService) GetPaymentLinkStatus(paymentLinkID string) (*MidtransPaymentLinkResponse, error) {
	url := fmt.Sprintf("https://api.sandbox.midtrans.com/v1/payment-links/%s", paymentLinkID)

	fmt.Printf("🔍 [MIDTRANS DEBUG] Getting payment link status for ID: %s\n", paymentLinkID)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Request URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error creating request: %v\n", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Set authorization header
	auth := base64.StdEncoding.EncodeToString([]byte(m.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)

	fmt.Printf("🔍 [MIDTRANS DEBUG] Request headers: Authorization=Basic %s\n", auth[:10]+"...")

	resp, err := m.client.Do(req)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error reading response body: %v\n", err)
		return nil, err
	}

	fmt.Printf("🔍 [MIDTRANS DEBUG] Response status: %d\n", resp.StatusCode)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ [MIDTRANS DEBUG] API error - Status: %d, Body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("midtrans API error: %s", string(body))
	}

	var paymentLinkResp MidtransPaymentLinkResponse
	err = json.Unmarshal(body, &paymentLinkResp)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error unmarshaling response: %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ [MIDTRANS DEBUG] Payment link response parsed successfully\n")
	fmt.Printf("✅ [MIDTRANS DEBUG] Payment Link ID: %s\n", paymentLinkResp.PaymentLinkID)
	fmt.Printf("✅ [MIDTRANS DEBUG] Order ID: %s\n", paymentLinkResp.OrderID)
	fmt.Printf("✅ [MIDTRANS DEBUG] Status: %s\n", paymentLinkResp.Status)
	fmt.Printf("✅ [MIDTRANS DEBUG] Gross Amount: %d\n", paymentLinkResp.GrossAmount)
	fmt.Printf("✅ [MIDTRANS DEBUG] Currency: %s\n", paymentLinkResp.Currency)
	fmt.Printf("✅ [MIDTRANS DEBUG] Expiry Time: %s\n", paymentLinkResp.ExpiryTime)
	fmt.Printf("✅ [MIDTRANS DEBUG] Number of purchases: %d\n", len(paymentLinkResp.Purchases))

	return &paymentLinkResp, nil
}

// CreatePaymentLink membuat payment link baru di Midtrans
func (m *MidtransService) CreatePaymentLink(orderID string, amount int, currency string, expiryMinutes int) (*MidtransPaymentLinkResponse, error) {
	url := "https://api.sandbox.midtrans.com/v1/payment-links"

	fmt.Printf("🔍 [MIDTRANS DEBUG] Creating payment link for order: %s\n", orderID)

	// Prepare request body
	requestBody := map[string]interface{}{
		"order_id":     orderID,
		"gross_amount": amount,
		"currency":     currency,
		"expiry_time":  fmt.Sprintf("%d minutes", expiryMinutes),
		"payment_settings": map[string]interface{}{
			"payment_methods": []string{"credit_card", "bca_va", "bni_va", "bri_va", "echannel", "permata_va", "other_va", "gopay", "kredivo", "shopeepay"},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error marshaling request body: %v\n", err)
		return nil, err
	}

	fmt.Printf("🔍 [MIDTRANS DEBUG] Request body: %s\n", string(jsonBody))

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error creating request: %v\n", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Set authorization header
	auth := base64.StdEncoding.EncodeToString([]byte(m.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)

	fmt.Printf("🔍 [MIDTRANS DEBUG] Request headers: Authorization=Basic %s\n", auth[:10]+"...")

	resp, err := m.client.Do(req)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error reading response body: %v\n", err)
		return nil, err
	}

	fmt.Printf("🔍 [MIDTRANS DEBUG] Response status: %d\n", resp.StatusCode)
	fmt.Printf("🔍 [MIDTRANS DEBUG] Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Printf("❌ [MIDTRANS DEBUG] API error - Status: %d, Body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("midtrans API error: %s", string(body))
	}

	var paymentLinkResp MidtransPaymentLinkResponse
	err = json.Unmarshal(body, &paymentLinkResp)
	if err != nil {
		fmt.Printf("❌ [MIDTRANS DEBUG] Error unmarshaling response: %v\n", err)
		return nil, err
	}

	fmt.Printf("✅ [MIDTRANS DEBUG] Payment link created successfully\n")
	fmt.Printf("✅ [MIDTRANS DEBUG] Payment Link ID: %s\n", paymentLinkResp.PaymentLinkID)
	fmt.Printf("✅ [MIDTRANS DEBUG] Order ID: %s\n", paymentLinkResp.OrderID)
	fmt.Printf("✅ [MIDTRANS DEBUG] Status: %s\n", paymentLinkResp.Status)
	fmt.Printf("✅ [MIDTRANS DEBUG] Gross Amount: %d\n", paymentLinkResp.GrossAmount)
	fmt.Printf("✅ [MIDTRANS DEBUG] Currency: %s\n", paymentLinkResp.Currency)
	fmt.Printf("✅ [MIDTRANS DEBUG] Expiry Time: %s\n", paymentLinkResp.ExpiryTime)

	return &paymentLinkResp, nil
}
