package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("jwt: invalid or expired token")

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims is a generic payload map, e.g. {"sub": userID, "email": "..."}.
// "exp" (unix seconds) is added automatically by GenerateJWT.
type Claims map[string]interface{}

// GenerateJWT creates a signed HS256 JSON Web Token valid for ttl.
func GenerateJWT(claims Claims, secret string, ttl time.Duration) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, _ := json.Marshal(header)

	payload := Claims{}
	for k, v := range claims {
		payload[k] = v
	}
	payload["exp"] = time.Now().Add(ttl).Unix()
	payload["iat"] = time.Now().Unix()
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerPart + "." + payloadPart

	sig := sign(signingInput, secret)
	return signingInput + "." + sig, nil
}

// ParseJWT verifies signature and expiry, and returns the claims.
func ParseJWT(token, secret string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	expected := sign(signingInput, secret)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return nil, ErrInvalidToken
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if expRaw, ok := claims["exp"]; ok {
		exp, ok := expRaw.(float64)
		if !ok || time.Now().Unix() > int64(exp) {
			return nil, ErrInvalidToken
		}
	}
	return claims, nil
}

func sign(input, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
