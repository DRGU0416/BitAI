package lib

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const (
	appstoreUrl        = "https://buy.itunes.apple.com/verifyReceipt"
	appstoreSandboxUrl = "https://sandbox.itunes.apple.com/verifyReceipt"
)

type LastReceiptInfo struct {
	ProductID       string `json:"product_id"`
	TransactionID   string `json:"transaction_id"`
	OriginalTransID string `json:"original_transaction_id"`
	PurchaseDate    string `json:"purchase_date_ms"`
	ExpireAt        string `json:"expires_date_ms"`
	CancelDate      string `json:"cancellation_date_ms"`
	IsTrial         string `json:"is_trial_period"`
	IsDiscount      string `json:"is_in_intro_offer_period"`
}

type ReceiptList []LastReceiptInfo

// func (s ReceiptList) Len() int           { return len(s) }
// func (s ReceiptList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
// func (s ReceiptList) Less(i, j int) bool { return s[i].PurchaseDate > s[j].PurchaseDate }

type AppStoreData struct {
	Status      int    `json:"status"`
	Environment string `json:"environment"`
	Receipt     struct {
		BundleID string      `json:"bundle_id"`
		InApp    ReceiptList `json:"in_app"`
	} `json:"receipt"`
	LatestReceiptInfo  ReceiptList `json:"latest_receipt_info"`
	PendingRenewalInfo []struct {
		AutoRenewStatus string `json:"auto_renew_status"`
	} `json:"pending_renewal_info"`
}

// 验证AppStore内购 通用
func ConfirmAppStorePay(receipt string, sandbox bool, needPassword bool, password string) (*AppStoreData, error) {
	url := appstoreUrl
	if sandbox {
		url = appstoreSandboxUrl
	}

	postBody := make(map[string]string, 2)
	postBody["receipt-data"] = receipt
	if needPassword {
		postBody["password"] = password
	}
	byteMsg, err := json.Marshal(postBody)
	if err != nil {
		return nil, err
	}
	request, _ := http.NewRequest("POST", url, bytes.NewReader(byteMsg))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Content-Length", fmt.Sprintf("%d", len(byteMsg)))

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	respData := AppStoreData{}
	if err = json.Unmarshal(result, &respData); err != nil {
		return nil, err
	}
	return &respData, nil
}
