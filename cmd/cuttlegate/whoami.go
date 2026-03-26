package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func cmdWhoami(g *globalFlags) error {
	tok, err := LoadToken()
	if err != nil {
		return fmt.Errorf("loading token: %w", err)
	}
	if tok == nil {
		return fmt.Errorf("not logged in — run 'cuttlegate login'")
	}

	// Decode the ID token (or access token) claims without verification.
	// This is just for display; the server does the real validation.
	tokenStr := tok.IDToken
	if tokenStr == "" {
		tokenStr = tok.AccessToken
	}

	claims, err := decodeJWTClaims(tokenStr)
	if err != nil {
		return fmt.Errorf("decoding token: %w", err)
	}

	if g.JSON {
		data, _ := json.MarshalIndent(claims, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display human-readable info.
	if sub, ok := claims["sub"].(string); ok {
		fmt.Printf("Subject:  %s\n", sub)
	}
	if email, ok := claims["email"].(string); ok {
		fmt.Printf("Email:    %s\n", email)
	}
	if name, ok := claims["name"].(string); ok {
		fmt.Printf("Name:     %s\n", name)
	}
	fmt.Printf("Issuer:   %s\n", tok.Issuer)
	if tok.Expired() {
		fmt.Printf("Status:   expired\n")
	} else {
		fmt.Printf("Expires:  %s\n", tok.ExpiresAt.Local().Format("2006-01-02 15:04:05"))
	}
	return nil
}

func decodeJWTClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}
