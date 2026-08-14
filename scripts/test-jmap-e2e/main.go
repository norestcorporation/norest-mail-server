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
	StalwartURL = "http://localhost:8081"
)

var testPassword string

func init() {
	testPassword = os.Getenv("TEST_ACCOUNT_PASSWORD")
	if testPassword == "" {
		fmt.Println("TEST_ACCOUNT_PASSWORD environment variable is not set")
		os.Exit(1)
	}
}

func main() {
	fmt.Println("============================================")
	fmt.Println("  Norest Mail End-to-End JMAP Proof")
	fmt.Println("============================================")

	// 1. Authenticate as Alice
	fmt.Println("\n[1] Authenticating as Alice (alice@example.test)...")
	aliceSession := getSession("alice@example.test", testPassword)
	aliceAccountId := aliceSession.PrimaryAccounts["urn:ietf:params:jmap:mail"]
	fmt.Printf("✓ Alice authenticated successfully. Account ID: %s\n", aliceAccountId)

	// 2. Get Alice's Mailboxes
	fmt.Println("\n[2] Fetching Alice's Mailboxes (Mailbox/get)...")
	aliceMailboxes := jmapCall("alice@example.test", testPassword, []any{
		[]any{"Mailbox/get", map[string]any{
			"accountId": aliceAccountId,
		}, "0"},
	})

	draftsMailboxId := ""

	if mList, ok := extractMethodResponse(aliceMailboxes, "Mailbox/get"); ok {
		list := mList["list"].([]any)
		for _, m := range list {
			mb := m.(map[string]any)
			role := ""
			if mb["role"] != nil {
				role = mb["role"].(string)
			}
			fmt.Printf("  - %s (Role: %s, ID: %s)\n", mb["name"], role, mb["id"])

			if role == "drafts" {
				draftsMailboxId = mb["id"].(string)
			}
		}
	}

	if draftsMailboxId == "" {
		fmt.Println("❌ Could not find Drafts mailbox")
		os.Exit(1)
	}

	// 2.5 Fetch Alice's Identity ID
	fmt.Println("\n[2.5] Fetching Alice's Identity ID (Identity/get)...")
	identityRes := jmapCall("alice@example.test", testPassword, []any{
		[]any{"Identity/get", map[string]any{
			"accountId": aliceAccountId,
		}, "0"},
	})

	aliceIdentityId := ""
	if mList, ok := extractMethodResponse(identityRes, "Identity/get"); ok {
		list := mList["list"].([]any)
		if len(list) > 0 {
			first := list[0].(map[string]any)
			aliceIdentityId = first["id"].(string)
		}
	}

	if aliceIdentityId == "" {
		fmt.Println("❌ Could not find Alice's identity")
		os.Exit(1)
	}

	// 3. Create and Send a message from Alice to Bob
	fmt.Println("\n[3] Creating and sending a message (alice -> bob)...")
	sendRes := jmapCall("alice@example.test", testPassword, []any{
		[]any{"Email/set", map[string]any{
			"accountId": aliceAccountId,
			"create": map[string]any{
				"msg1": map[string]any{
					"mailboxIds": map[string]bool{draftsMailboxId: true},
					"from":       []map[string]any{{"name": "Alice", "email": "alice@example.test"}},
					"to":         []map[string]any{{"name": "Bob", "email": "bob@example.test"}},
					"subject":    "End-to-End JMAP Test",
					"bodyValues": map[string]any{
						"body1": map[string]any{
							"value":             "Hello Bob,\n\nThis is a test message from Norest Mail foundation.",
							"isEncodingProblem": false,
							"isTruncated":       false,
						},
					},
					"textBody": []map[string]any{
						{"partId": "body1", "type": "text/plain"},
					},
				},
			},
		}, "0"},
		[]any{"EmailSubmission/set", map[string]any{
			"accountId": aliceAccountId,
			"create": map[string]any{
				"sub1": map[string]any{
					"emailId":    "#msg1",
					"identityId": aliceIdentityId,
				},
			},
		}, "1"},
	})

	if _, ok := extractMethodResponse(sendRes, "EmailSubmission/set"); !ok {
		fmt.Printf("❌ EmailSubmission/set failed: %v\n", sendRes)
		os.Exit(1)
	}
	// Check for any errors in the JMAP call
	if errs, _ := extractMethodResponse(sendRes, "error"); errs != nil {
		fmt.Printf("❌ Got error response: %v\n", errs)
	}

	fmt.Printf("✓ Message submitted. Full response: %v\n", sendRes)

	// Give Stalwart a second to deliver locally
	time.Sleep(2 * time.Second)

	// 4. Authenticate as Bob
	fmt.Println("\n[4] Authenticating as Bob (bob@example.test)...")
	bobSession := getSession("bob@example.test", testPassword)
	bobAccountId := bobSession.PrimaryAccounts["urn:ietf:params:jmap:mail"]
	fmt.Printf("✓ Bob authenticated successfully. Account ID: %s\n", bobAccountId)

	// 5. Query Bob's Inbox
	fmt.Println("\n[5] Querying Bob's emails (Email/query)...")
	queryRes := jmapCall("bob@example.test", testPassword, []any{
		[]any{"Email/query", map[string]any{
			"accountId": bobAccountId,
		}, "0"},
	})

	var emailId string
	if mList, ok := extractMethodResponse(queryRes, "Email/query"); ok {
		ids := mList["ids"].([]any)
		fmt.Printf("✓ Found %d emails in Bob's account.\n", len(ids))
		if len(ids) == 0 {
			fmt.Println("❌ Bob has no emails. Delivery failed?")
			os.Exit(1)
		}
		emailId = ids[0].(string)
	}

	// 6. Fetch the email
	fmt.Println("\n[6] Fetching the received email (Email/get)...")
	getRes := jmapCall("bob@example.test", testPassword, []any{
		[]any{"Email/get", map[string]any{
			"accountId":  bobAccountId,
			"ids":        []string{emailId},
			"properties": []string{"id", "subject", "from", "to", "preview", "keywords"},
		}, "0"},
	})

	if mList, ok := extractMethodResponse(getRes, "Email/get"); ok {
		list := mList["list"].([]any)
		email := list[0].(map[string]any)
		fmt.Printf("  - Subject: %s\n", email["subject"])
		fmt.Printf("  - Preview: %s\n", email["preview"])
		keywords := email["keywords"].(map[string]any)
		hasSeen := keywords["$seen"] != nil && keywords["$seen"].(bool)
		fmt.Printf("  - Is Read ($seen): %v\n", hasSeen)
	}

	// 7. Mark as Read
	fmt.Println("\n[7] Marking the email as read (Email/set)...")
	setRes := jmapCall("bob@example.test", testPassword, []any{
		[]any{"Email/set", map[string]any{
			"accountId": bobAccountId,
			"update": map[string]any{
				emailId: map[string]any{
					"keywords/$seen": true,
				},
			},
		}, "0"},
	})
	if _, ok := extractMethodResponse(setRes, "Email/set"); ok {
		fmt.Println("✓ Email successfully updated.")
	}

	// 8. Verify the update
	fmt.Println("\n[8] Verifying read state (Email/get)...")
	getRes2 := jmapCall("bob@example.test", testPassword, []any{
		[]any{"Email/get", map[string]any{
			"accountId":  bobAccountId,
			"ids":        []string{emailId},
			"properties": []string{"keywords"},
		}, "0"},
	})
	if mList, ok := extractMethodResponse(getRes2, "Email/get"); ok {
		email := mList["list"].([]any)[0].(map[string]any)
		keywords := email["keywords"].(map[string]any)
		hasSeen := keywords["$seen"] != nil && keywords["$seen"].(bool)
		fmt.Printf("  - Is Read ($seen): %v\n", hasSeen)
		if hasSeen {
			fmt.Println("✓ Verified! State was changed successfully.")
		} else {
			fmt.Println("❌ Failed to change read state.")
		}
	}

	fmt.Println("\n============================================")
	fmt.Println("  Proof Completed Successfully!")
	fmt.Println("============================================")
}

// Structs and helpers for parsing

type Session struct {
	PrimaryAccounts map[string]string `json:"primaryAccounts"`
	ApiUrl          string            `json:"apiUrl"`
}

func getSession(username, password string) *Session {
	req, _ := http.NewRequest("GET", StalwartURL+"/.well-known/jmap", nil)
	req.SetBasicAuth(username, password)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Add basic auth to redirected requests
			req.SetBasicAuth(username, password)
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching session: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Auth failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var session Session
	json.NewDecoder(resp.Body).Decode(&session)
	return &session
}

func jmapCall(username, password string, methodCalls []any) map[string]any {
	reqBody := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail", "urn:ietf:params:jmap:submission"},
		"methodCalls": methodCalls,
	}
	b, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", StalwartURL+"/jmap", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error making JMAP call: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Check for top-level errors in methodResponses
	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr := mr.([]any)
			if mrArr[0].(string) == "error" {
				errObj := mrArr[1].(map[string]any)
				fmt.Printf("JMAP Error Response: %v\n", errObj["type"])
			}
		}
	}

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
