package stalwart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ManagementRequest makes a JMAP-based management request to Stalwart.
// Stalwart's management/configuration API is JMAP-based and accessed through
// the admin credentials. This method is the entry point for administrative
// operations that will be expanded in Chapter 2.
func (c *Client) ManagementRequest(ctx context.Context, methodCalls []any) (json.RawMessage, error) {
	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling management request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("management request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("management request returned status %d: %s", resp.StatusCode, string(data))
	}

	// Parse methodResponses for JMAP-level errors
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing management response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return nil, fmt.Errorf("JMAP error: %v", mrArr[1])
			}

			// Check for notCreated or notUpdated
			if details, ok := mrArr[1].(map[string]any); ok {
				if notCreated, has := details["notCreated"]; has && len(notCreated.(map[string]any)) > 0 {
					return nil, fmt.Errorf("JMAP notCreated: %v", notCreated)
				}
				if notUpdated, has := details["notUpdated"]; has && len(notUpdated.(map[string]any)) > 0 {
					return nil, fmt.Errorf("JMAP notUpdated: %v", notUpdated)
				}
			}
		}
	}

	return json.RawMessage(data), nil
}

// CreateDomain creates a new domain in Stalwart via JMAP x:Domain/set.
func (c *Client) CreateDomain(ctx context.Context, name string) (string, error) {
	// Use a unique key for the create operation to avoid conflicts
	createKey := fmt.Sprintf("domain_%s_%d", name, time.Now().UnixNano())
	
	methodCalls := []any{
		[]any{"x:Domain/set", map[string]any{
			"create": map[string]any{
				createKey: map[string]any{
					"name": name,
				},
			},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshaling CreateDomain request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", fmt.Errorf("CreateDomain request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CreateDomain returned status %d: %s", resp.StatusCode, string(data))
	}

	// Parse methodResponses for JMAP-level errors
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing CreateDomain response: %w", err)
	}

	var createdDomainId string
	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return "", fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if notCreated, has := details["notCreated"]; has && len(notCreated.(map[string]any)) > 0 {
					return "", fmt.Errorf("JMAP notCreated: %v", notCreated)
				}
				if created, has := details["created"]; has {
					if cMap, ok := created.(map[string]any); ok {
						// Use the createKey to find the created domain
						for _, value := range cMap {
							if nameProps, ok := value.(map[string]any); ok {
								if id, ok := nameProps["id"].(string); ok {
									createdDomainId = id
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return createdDomainId, nil
}

// CreateAccount creates a new mail account in Stalwart via JMAP x:Account/set.
func (c *Client) CreateAccount(ctx context.Context, name, domainId, secret, description string) (string, error) {
	// Use a unique key for the create operation to avoid conflicts
	createKey := fmt.Sprintf("account_%s_%d", name, time.Now().UnixNano())
	
	methodCalls := []any{
		[]any{"x:Account/set", map[string]any{
			"create": map[string]any{
				createKey: map[string]any{
					"@type":       "User",
					"name":        name,
					"domainId":    domainId,
					"description": description,
					"credentials": map[string]any{
						"0": map[string]any{
							"@type":  "Password",
							"secret": secret,
						},
					},
				},
			},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshaling CreateAccount request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", fmt.Errorf("CreateAccount request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CreateAccount returned status %d: %s", resp.StatusCode, string(data))
	}

	// Parse methodResponses for JMAP-level errors
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing CreateAccount response: %w", err)
	}

	var createdAccountId string
	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return "", fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if notCreated, has := details["notCreated"]; has && len(notCreated.(map[string]any)) > 0 {
					return "", fmt.Errorf("JMAP notCreated: %v", notCreated)
				}
				if created, has := details["created"]; has {
					if cMap, ok := created.(map[string]any); ok {
						// Use the createKey to find the created account
						for _, value := range cMap {
							if nameProps, ok := value.(map[string]any); ok {
								if id, ok := nameProps["id"].(string); ok {
									createdAccountId = id
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return createdAccountId, nil
}

// CreateAppPassword creates an application password for a Stalwart account via JMAP x:AppPassword/set.
func (c *Client) CreateAppPassword(ctx context.Context, accountId string, description string) (string, error) {
	// Use a unique key for the create operation to avoid conflicts
	createKey := fmt.Sprintf("app_password_%s_%d", accountId, time.Now().UnixNano())
	
	methodCalls := []any{
		[]any{"x:AppPassword/set", map[string]any{
			"accountId": accountId,
			"create": map[string]any{
				createKey: map[string]any{
					"description": description,
				},
			},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshaling CreateAppPassword request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", fmt.Errorf("CreateAppPassword request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CreateAppPassword returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing CreateAppPassword response: %w", err)
	}

	var secret string
	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return "", fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if notCreated, has := details["notCreated"]; has && len(notCreated.(map[string]any)) > 0 {
					return "", fmt.Errorf("JMAP notCreated: %v", notCreated)
				}
				if created, has := details["created"]; has {
					if cMap, ok := created.(map[string]any); ok {
						// Use the createKey to find the created app password
						for _, value := range cMap {
							if nameProps, ok := value.(map[string]any); ok {
								if s, ok := nameProps["secret"].(string); ok {
									secret = s
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if secret == "" {
		return "", fmt.Errorf("app password secret not found in response: %s", string(data))
	}

	return secret, nil
}

// UpdateAccountQuota updates the quota for a Stalwart account via JMAP x:Account/set.
func (c *Client) UpdateAccountQuota(ctx context.Context, accountId string, maxStorageBytes int64) error {
	methodCalls := []any{
		[]any{"x:Account/set", map[string]any{
			"update": map[string]any{
				accountId: map[string]any{
					"maxDiskQuota": maxStorageBytes,
				},
			},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshaling UpdateAccountQuota request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("UpdateAccountQuota request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("UpdateAccountQuota returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing UpdateAccountQuota response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if notUpdated, has := details["notUpdated"]; has && len(notUpdated.(map[string]any)) > 0 {
					return fmt.Errorf("JMAP notUpdated: %v", notUpdated)
				}
			}
		}
	}

	return nil
}

// DomainExists checks if a domain exists in Stalwart by ID.
func (c *Client) DomainExists(ctx context.Context, domainID string) (bool, error) {
	methodCalls := []any{
		[]any{"x:Domain/get", map[string]any{
			"ids": []string{domainID},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("marshaling DomainExists request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return false, fmt.Errorf("DomainExists request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("DomainExists returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("parsing DomainExists response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return false, fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if list, has := details["list"]; has {
					if listArr, ok := list.([]any); ok && len(listArr) > 0 {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// DomainExistsAndMatches checks if a domain exists in Stalwart by ID AND matches the expected name.
func (c *Client) DomainExistsAndMatches(ctx context.Context, domainID, expectedName string) (bool, error) {
	methodCalls := []any{
		[]any{"x:Domain/get", map[string]any{
			"ids": []string{domainID},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("marshaling DomainExistsAndMatches request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return false, fmt.Errorf("DomainExistsAndMatches request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("DomainExistsAndMatches returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("parsing DomainExistsAndMatches response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return false, fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if list, has := details["list"]; has {
					if listArr, ok := list.([]any); ok && len(listArr) > 0 {
						if domain, ok := listArr[0].(map[string]any); ok {
							if name, ok := domain["name"].(string); ok && name == expectedName {
								return true, nil
							}
						}
					}
				}
			}
		}
	}

	return false, nil
}

// FindDomainByName finds a domain ID by name in Stalwart by getting all domains and filtering.
func (c *Client) FindDomainByName(ctx context.Context, name string) (string, error) {
	methodCalls := []any{
		[]any{"x:Domain/get", map[string]any{}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshaling FindDomainByName request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", fmt.Errorf("FindDomainByName request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FindDomainByName returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing FindDomainByName response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return "", fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if list, has := details["list"]; has {
					if listArr, ok := list.([]any); ok {
						// Manually filter by name
						for _, item := range listArr {
							if domain, ok := item.(map[string]any); ok {
								if domainName, ok := domain["name"].(string); ok && domainName == name {
									if id, ok := domain["id"].(string); ok {
										return id, nil
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return "", nil
}

// FindAccountByName finds an account ID by name in Stalwart by getting all accounts and filtering.
func (c *Client) FindAccountByName(ctx context.Context, name string) (string, error) {
	methodCalls := []any{
		[]any{"x:Account/get", map[string]any{}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshaling FindAccountByName request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", fmt.Errorf("FindAccountByName request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FindAccountByName returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing FindAccountByName response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return "", fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if list, has := details["list"]; has {
					if listArr, ok := list.([]any); ok {
						// Manually filter by name
						for _, item := range listArr {
							if account, ok := item.(map[string]any); ok {
								if accountName, ok := account["name"].(string); ok && accountName == name {
									if id, ok := account["id"].(string); ok {
										return id, nil
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return "", nil
}

// AccountExists checks if an account exists in Stalwart by ID.
func (c *Client) AccountExists(ctx context.Context, accountID string) (bool, error) {
	methodCalls := []any{
		[]any{"x:Account/get", map[string]any{
			"ids": []string{accountID},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("marshaling AccountExists request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return false, fmt.Errorf("AccountExists request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("AccountExists returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("parsing AccountExists response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return false, fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if list, has := details["list"]; has {
					if listArr, ok := list.([]any); ok && len(listArr) > 0 {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// AccountExistsAndMatches checks if an account exists in Stalwart by ID AND matches the expected localPart and domainId.
func (c *Client) AccountExistsAndMatches(ctx context.Context, accountID, expectedLocalPart, expectedDomainID string) (bool, error) {
	methodCalls := []any{
		[]any{"x:Account/get", map[string]any{
			"ids": []string{accountID},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("marshaling AccountExistsAndMatches request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return false, fmt.Errorf("AccountExistsAndMatches request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("AccountExistsAndMatches returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("parsing AccountExistsAndMatches response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return false, fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if list, has := details["list"]; has {
					if listArr, ok := list.([]any); ok && len(listArr) > 0 {
						if account, ok := listArr[0].(map[string]any); ok {
							name, nameOk := account["name"].(string)
							domainId, domainOk := account["domainId"].(string)
							if nameOk && domainOk && name == expectedLocalPart && domainId == expectedDomainID {
								return true, nil
							}
						}
					}
				}
			}
		}
	}

	return false, nil
}

// DisableAccount disables a Stalwart account via JMAP x:Account/set.
func (c *Client) DisableAccount(ctx context.Context, accountID string) error {
	methodCalls := []any{
		[]any{"x:Account/set", map[string]any{
			"update": map[string]any{
				accountID: map[string]any{
					"active": false,
				},
			},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshaling DisableAccount request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("DisableAccount request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DisableAccount returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing DisableAccount response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if notUpdated, has := details["notUpdated"]; has && len(notUpdated.(map[string]any)) > 0 {
					return fmt.Errorf("JMAP notUpdated: %v", notUpdated)
				}
			}
		}
	}

	return nil
}

// EnableAccount enables a Stalwart account via JMAP x:Account/set.
func (c *Client) EnableAccount(ctx context.Context, accountID string) error {
	methodCalls := []any{
		[]any{"x:Account/set", map[string]any{
			"update": map[string]any{
				accountID: map[string]any{
					"active": true,
				},
			},
		}, "0"},
	}

	request := map[string]any{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:stalwart:jmap"},
		"methodCalls": methodCalls,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshaling EnableAccount request: %w", err)
	}

	url := c.BaseURL + "/jmap"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("EnableAccount request: %w", err)
	}

	data, err := readAndClose(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("EnableAccount returned status %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing EnableAccount response: %w", err)
	}

	if methodResps, ok := result["methodResponses"].([]any); ok {
		for _, mr := range methodResps {
			mrArr, ok := mr.([]any)
			if !ok || len(mrArr) < 2 {
				continue
			}
			if mrArr[0] == "error" {
				return fmt.Errorf("JMAP error: %v", mrArr[1])
			}
			if details, ok := mrArr[1].(map[string]any); ok {
				if notUpdated, has := details["notUpdated"]; has && len(notUpdated.(map[string]any)) > 0 {
					return fmt.Errorf("JMAP notUpdated: %v", notUpdated)
				}
			}
		}
	}

	return nil
}
