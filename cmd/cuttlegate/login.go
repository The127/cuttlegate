package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultIssuer   = "https://keyline-api.karo.gay/oidc/keyline"
	defaultClientID = "cuttlegate"
	deviceScope     = "openid profile email"
)

// oidcDiscovery holds the subset of fields we need from the OIDC discovery doc.
type oidcDiscovery struct {
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
}

// deviceAuthResponse is the response from the device authorization endpoint.
type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

func cmdLogin(args []string, g *globalFlags) error {
	issuer := defaultIssuer
	clientID := defaultClientID

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--issuer" && i+1 < len(args):
			i++
			issuer = args[i]
		case strings.HasPrefix(args[i], "--issuer="):
			issuer = strings.TrimPrefix(args[i], "--issuer=")
		case args[i] == "--client-id" && i+1 < len(args):
			i++
			clientID = args[i]
		case strings.HasPrefix(args[i], "--client-id="):
			clientID = strings.TrimPrefix(args[i], "--client-id=")
		case args[i] == "--help" || args[i] == "-h":
			fmt.Print("Usage: cuttlegate login [--issuer URL] [--client-id ID]\n")
			return nil
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	// Step 1: discover endpoints.
	disco, err := discover(issuer)
	if err != nil {
		return fmt.Errorf("OIDC discovery: %w", err)
	}
	if disco.DeviceAuthorizationEndpoint == "" {
		return fmt.Errorf("issuer does not support device authorization flow")
	}

	// Step 2: start device authorization.
	dar, err := startDeviceAuth(disco.DeviceAuthorizationEndpoint, clientID)
	if err != nil {
		return fmt.Errorf("device authorization: %w", err)
	}

	// Step 3: tell the user what to do.
	fmt.Printf("To sign in, open this URL in your browser:\n\n")
	fmt.Printf("  %s\n\n", dar.VerificationURI)
	fmt.Printf("Enter code: %s\n\n", dar.UserCode)
	if dar.VerificationURIComplete != "" {
		fmt.Printf("Or open: %s\n\n", dar.VerificationURIComplete)
	}
	fmt.Printf("Waiting for authorization...")

	// Step 4: poll for the token.
	interval := time.Duration(dar.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dar.ExpiresIn) * time.Second)

	for {
		time.Sleep(interval)
		if time.Now().After(deadline) {
			fmt.Println()
			return fmt.Errorf("device code expired — please try again")
		}

		tok, retry, err := pollToken(disco.TokenEndpoint, clientID, dar.DeviceCode)
		if err != nil {
			fmt.Println()
			return err
		}
		if retry == "slow_down" {
			interval += 5 * time.Second
			continue
		}
		if retry == "authorization_pending" {
			fmt.Print(".")
			continue
		}

		// Success.
		fmt.Println(" done!")
		tok.Issuer = issuer
		tok.ClientID = clientID
		if err := tok.Save(); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}
		fmt.Printf("Logged in successfully. Token stored at %s\n", tokenPath())
		return nil
	}
}

func discover(issuer string) (*oidcDiscovery, error) {
	resp, err := http.Get(strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery returned %d: %s", resp.StatusCode, string(body))
	}
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

// startDeviceAuth initiates the device authorization flow.
func startDeviceAuth(endpoint, clientID string) (*deviceAuthResponse, error) {
	form := url.Values{
		"client_id": {clientID},
		"scope":     {deviceScope},
	}
	resp, err := http.PostForm(endpoint, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device auth returned %d: %s", resp.StatusCode, string(body))
	}
	var dar deviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&dar); err != nil {
		return nil, err
	}
	return &dar, nil
}

// DeviceAuthRequest builds the form values for a device authorization request.
// Exported for testing.
func DeviceAuthRequest(clientID string) url.Values {
	return url.Values{
		"client_id": {clientID},
		"scope":     {deviceScope},
	}
}

// pollToken attempts to exchange the device code for a token.
// Returns (token, "", nil) on success, (nil, reason, nil) if should retry,
// or (nil, "", err) on fatal error.
func pollToken(tokenEndpoint, clientID, deviceCode string) (*StoredToken, string, error) {
	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	}
	resp, err := http.PostForm(tokenEndpoint, form)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Check for pending/slow_down errors (returned as 400).
	if resp.StatusCode == http.StatusBadRequest {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp.Error == "authorization_pending" || errResp.Error == "slow_down" {
				return nil, errResp.Error, nil
			}
		}
		return nil, "", fmt.Errorf("token error: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", fmt.Errorf("parsing token response: %w", err)
	}

	tok := &StoredToken{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		IDToken:      raw.IDToken,
		TokenType:    raw.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(raw.ExpiresIn) * time.Second),
	}
	return tok, "", nil
}
