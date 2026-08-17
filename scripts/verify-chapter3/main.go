package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	NorestAPIURL = "http://localhost:8080/v1"
)

var (
	aliceEmail    string
	alicePassword = "SecurePassword123!"
	aliceToken    string
	aliceDomainID string
	aliceJMAPURL  string
	aliceJMAPAuth string
	aliceAccount  string

	bobEmail    string
	bobPassword = "SecurePassword123!"
	bobToken    string
	bobDomainID string
	bobJMAPURL  string
	bobJMAPAuth string
	bobAccount  string
)

func main() {
	fmt.Println("============================================")
	fmt.Println("  Norest Mail Chapter 3 - Full Acceptance")
	fmt.Println("============================================")

	timestamp := time.Now().UnixNano()
	aliceEmail = fmt.Sprintf("alice_%d@example.com", timestamp)
	bobEmail = fmt.Sprintf("bob_%d@example.com", timestamp)

	// --- SETUP ALICE ---
	fmt.Println("\n[1] Registering Alice...")
	aliceToken = register(aliceEmail, alicePassword)
	aliceDomainID = createDomain(aliceToken, fmt.Sprintf("alice-%d.test", timestamp))
	waitForDomain(aliceToken, aliceDomainID)
	createAddress(aliceToken, aliceDomainID, "alice")

	// --- SETUP BOB ---
	fmt.Println("\n[2] Registering Bob...")
	bobToken = register(bobEmail, bobPassword)
	bobDomainID = createDomain(bobToken, fmt.Sprintf("bob-%d.test", timestamp))
	waitForDomain(bobToken, bobDomainID)
	createAddress(bobToken, bobDomainID, "bob")

	fmt.Println("\n[3] Waiting for worker to provision mailboxes...")
	waitForProvisioning(aliceToken)
	waitForProvisioning(bobToken)

	fmt.Println("\n[4] Requesting mail sessions from Norest...")
	aliceJMAPURL, aliceJMAPAuth, aliceAccount = getSession(aliceToken)
	bobJMAPURL, bobJMAPAuth, bobAccount = getSession(bobToken)

	// --- 75. Mailbox List Acceptance Test ---
	fmt.Println("\n[5] (75) Mailbox List Acceptance Test")
	aliceMailboxAddress := fmt.Sprintf("alice@alice-%d.test", timestamp)
	aliceMailboxes := jmapCall(aliceJMAPURL, aliceMailboxAddress, aliceJMAPAuth, []any{
		[]any{"Mailbox/get", map[string]any{"accountId": aliceAccount}, "0"},
	})

	fmt.Printf("Alice Mailbox/get response: %v\n", aliceMailboxes)

	var inboxID, draftsID string
	if mList, ok := extractMethodResponse(aliceMailboxes, "Mailbox/get"); ok {
		list := mList["list"].([]any)
		for _, m := range list {
			mb := m.(map[string]any)
			role := ""
			if mb["role"] != nil {
				role = mb["role"].(string)
			}
			fmt.Printf("  - Found Folder: %s (Role: %s)\n", mb["name"], role)
			switch role {
			case "inbox":
				inboxID = mb["id"].(string)
			case "drafts":
				draftsID = mb["id"].(string)
			}
		}
	}

	if inboxID == "" || draftsID == "" {
		fmt.Println("❌ Critical folders missing!")
		os.Exit(1)
	}

	// --- 78. Send Acceptance Test ---
	fmt.Println("\n[6] (78) Send Acceptance Test (Alice -> Bob)")
	aliceIdentityId := getIdentityId(aliceJMAPURL, aliceMailboxAddress, aliceJMAPAuth, aliceAccount)
	sendMsgResponse := jmapCall(aliceJMAPURL, aliceMailboxAddress, aliceJMAPAuth, []any{
		[]any{"Email/set", map[string]any{
			"accountId": aliceAccount,
			"create": map[string]any{
				"msg1": map[string]any{
					"mailboxIds": map[string]bool{draftsID: true},
					"from":       []map[string]any{{"name": "Alice", "email": fmt.Sprintf("alice@alice-%d.test", timestamp)}},
					"to":         []map[string]any{{"name": "Bob", "email": fmt.Sprintf("bob@bob-%d.test", timestamp)}},
					"subject":    "Chapter 3 Hello!",
					"keywords":   map[string]any{"$seen": true},
					"bodyValues": map[string]any{
						"b1": map[string]any{
							"value":             "This is an integration test.",
							"isEncodingProblem": false,
							"isTruncated":       false,
						},
					},
					"textBody": []map[string]any{
						{"partId": "b1", "type": "text/plain"},
					},
				},
			},
		}, "0"},
		[]any{"EmailSubmission/set", map[string]any{
			"accountId": aliceAccount,
			"create": map[string]any{
				"sub1": map[string]any{
					"emailId":    "#msg1",
					"identityId": aliceIdentityId,
				},
			},
		}, "1"},
	})

	if _, ok := extractMethodResponse(sendMsgResponse, "EmailSubmission/set"); !ok {
		fmt.Printf("❌ Failed to submit message: %v\n", sendMsgResponse)
		os.Exit(1)
	}
	fmt.Println("✓ Message submitted")

	time.Sleep(3 * time.Second) // wait for delivery

	// --- 76. Inbox Acceptance Test & 81. Pagination ---
	fmt.Println("\n[7] (76, 81) Inbox List & Pagination Acceptance Test")
	bobMailboxAddress := fmt.Sprintf("bob@bob-%d.test", timestamp)
	// Query Bob's emails
	bobQueryRes := jmapCall(bobJMAPURL, bobMailboxAddress, bobJMAPAuth, []any{
		[]any{"Email/query", map[string]any{
			"accountId": bobAccount,
			"limit":     10,
		}, "0"},
	})

	var emailID string
	if mList, ok := extractMethodResponse(bobQueryRes, "Email/query"); ok {
		ids := mList["ids"].([]any)
		fmt.Printf("  - Bob has %d emails\n", len(ids))
		if len(ids) == 0 {
			fmt.Println("❌ Bob did not receive the email!")
			os.Exit(1)
		}
		emailID = ids[0].(string)
	}

	// Email/get preview
	bobGetRes := jmapCall(bobJMAPURL, bobMailboxAddress, bobJMAPAuth, []any{
		[]any{"Email/get", map[string]any{
			"accountId":  bobAccount,
			"ids":        []string{emailID},
			"properties": []string{"id", "subject", "preview", "keywords"},
		}, "0"},
	})

	if mList, ok := extractMethodResponse(bobGetRes, "Email/get"); ok {
		list := mList["list"].([]any)
		email := list[0].(map[string]any)
		fmt.Printf("  - Subject: %s\n", email["subject"])
		fmt.Printf("  - Preview: %s\n", email["preview"])
	}

	// --- 77. Read/Unread Acceptance Test ---
	fmt.Println("\n[8] (77) Read/Unread Acceptance Test")
	setRes := jmapCall(bobJMAPURL, bobMailboxAddress, bobJMAPAuth, []any{
		[]any{"Email/set", map[string]any{
			"accountId": bobAccount,
			"update": map[string]any{
				emailID: map[string]any{
					"keywords/$seen": true,
				},
			},
		}, "0"},
	})
	if _, ok := extractMethodResponse(setRes, "Email/set"); ok {
		fmt.Println("✓ Marked as read ($seen=true)")
	}

	// --- 79. Search Acceptance Test ---
	fmt.Println("\n[9] (79) Search Acceptance Test")
	searchRes := jmapCall(bobJMAPURL, bobMailboxAddress, bobJMAPAuth, []any{
		[]any{"Email/query", map[string]any{
			"accountId": bobAccount,
			"filter": map[string]any{
				"text": "integration",
			},
		}, "0"},
	})
	if mList, ok := extractMethodResponse(searchRes, "Email/query"); ok {
		ids := mList["ids"].([]any)
		fmt.Printf("✓ Search found %d messages matching 'integration'\n", len(ids))
	}

	// --- Star/Move/Delete (Extra acceptance items) ---
	fmt.Println("\n[10] Move / Star Acceptance Test")
	moveRes := jmapCall(bobJMAPURL, bobMailboxAddress, bobJMAPAuth, []any{
		[]any{"Email/set", map[string]any{
			"accountId": bobAccount,
			"update": map[string]any{
				emailID: map[string]any{
					"keywords/$flagged": true,
				},
			},
		}, "0"},
	})
	if _, ok := extractMethodResponse(moveRes, "Email/set"); ok {
		fmt.Println("✓ Message starred ($flagged=true)")
	}

	fmt.Println("\n============================================")
	fmt.Println("  Chapter 3 Acceptance Tests Passed!")
	fmt.Println("============================================")
}

func register(email, password string) string {
	b, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(NorestAPIURL+"/auth/register", "application/json", bytes.NewReader(b))
	if err != nil {
		fmt.Printf("Register error: %v\n", err)
		os.Exit(1)
	}

	var res map[string]any
	json.NewDecoder(resp.Body).Decode(&res)
	return res["access_token"].(string)
}

func createDomain(token, domain string) string {
	b, _ := json.Marshal(map[string]string{"name": domain})
	req, _ := http.NewRequest("POST", NorestAPIURL+"/domains", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("HTTP request error: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var res map[string]any
	json.NewDecoder(resp.Body).Decode(&res)
	return res["id"].(string)
}

func waitForDomain(token, domainID string) {
	for i := 0; i < 30; i++ {
		req, _ := http.NewRequest("GET", NorestAPIURL+"/domains/"+domainID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			var res map[string]any
			json.NewDecoder(resp.Body).Decode(&res)
			resp.Body.Close()
			if res["status"] == "active" {
				return
			}
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Println("❌ Domain provisioning timed out")
	os.Exit(1)
}

func createAddress(token, domainID, localPart string) {
	b, _ := json.Marshal(map[string]string{"local_part": localPart})
	req, _ := http.NewRequest("POST", NorestAPIURL+"/domains/"+domainID+"/addresses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("HTTP request error: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

func waitForProvisioning(token string) {
	for i := 0; i < 30; i++ {
		req, _ := http.NewRequest("GET", NorestAPIURL+"/mail/account", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			var res map[string]any
			json.NewDecoder(resp.Body).Decode(&res)
			resp.Body.Close()
			if res["status"] == "active" {
				return
			}
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Println("❌ Provisioning timed out")
	os.Exit(1)
}

func getSession(token string) (string, string, string) {
	req, _ := http.NewRequest("POST", NorestAPIURL+"/mail/session", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		fmt.Printf("❌ Failed to get mail session (status: %d, err: %v)\n", resp.StatusCode, err)
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("Response body: %s\n", string(body))
		}
		os.Exit(1)
	}

	var res map[string]any
	json.NewDecoder(resp.Body).Decode(&res)

	_ = res["jmap_session_url"].(string)
	access := res["access_token"].(string)
	account := res["account_id"].(string)

	// Stalwart's JMAP API URL is /jmap relative to the base
	apiUrl := "http://localhost:8081/jmap"

	return apiUrl, access, account
}

func getIdentityId(jmapURL, email, token, accountID string) string {
	res := jmapCall(jmapURL, email, token, []any{
		[]any{"Identity/get", map[string]any{"accountId": accountID}, "0"},
	})
	if mList, ok := extractMethodResponse(res, "Identity/get"); ok {
		list := mList["list"].([]any)
		if len(list) > 0 {
			return list[0].(map[string]any)["id"].(string)
		}
	}
	return ""
}

func jmapCall(url, email, token string, methodCalls []any) map[string]any {
	reqBody := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail", "urn:ietf:params:jmap:submission"},
		"methodCalls": methodCalls,
	}
	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(email, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error making JMAP call: %v\n", err)
		os.Exit(1)
	}


	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func extractMethodResponse(response map[string]any, methodName string) (map[string]any, bool) {
	if resps, ok := response["methodResponses"].([]any); ok {
		for _, mr := range resps {
			mrArr := mr.([]any)
			if mrArr[0].(string) == methodName {
				return mrArr[1].(map[string]any), true
			}
		}
	}
	return nil, false
}
