// Copyright 2015 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package acme

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// jwsEncodeJSON signs claimset using provided key and a nonce.
// The result is serialized in JSON format.
// See https://tools.ietf.org/html/rfc7515#section-7.
func jwsEncodeJSON(claimset interface{}, key *rsa.PrivateKey, nonce string) ([]byte, error) {
	jwk := jwkEncode(&key.PublicKey)
	phead := fmt.Sprintf(`{"alg":"RS256","jwk":%s,"nonce":%q}`, jwk, nonce)
	phead = base64.RawURLEncoding.EncodeToString([]byte(phead))
	cs, err := json.Marshal(claimset)
	if err != nil {
		return nil, err
	}
	payload := base64.RawURLEncoding.EncodeToString(cs)
	h := sha256.New()
	h.Write([]byte(phead + "." + payload))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return nil, err
	}
	enc := struct {
		Protected string `json:"protected"`
		Payload   string `json:"payload"`
		Sig       string `json:"signature"`
	}{
		Protected: phead,
		Payload:   payload,
		Sig:       base64.RawURLEncoding.EncodeToString(sig),
	}
	return json.Marshal(&enc)
}

// jwkEncode encodes public part of an RSA key into a JWK.
// The result is also suitable for creating a JWK thumbprint.
func jwkEncode(pub *rsa.PublicKey) string {
	n := pub.N
	e := big.NewInt(int64(pub.E))
	// fields order is important
	// see https://tools.ietf.org/html/rfc7638#section-3.3 for details
	return fmt.Sprintf(`{"e":"%s","kty":"RSA","n":"%s"}`,
		base64.RawURLEncoding.EncodeToString(e.Bytes()),
		base64.RawURLEncoding.EncodeToString(n.Bytes()),
	)
}

// JWKThumbprint creates a JWK thumbprint out of pub
// as specified in https://tools.ietf.org/html/rfc7638.
func JWKThumbprint(pub *rsa.PublicKey) string {
	jwk := jwkEncode(pub)
	h := sha256.New()
	h.Write([]byte(jwk))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
