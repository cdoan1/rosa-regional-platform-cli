package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
)

// RSAKeyPair represents an RSA key pair with JWKS-compatible public key info
type RSAKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	// JWKS fields
	N   string // Base64URL-encoded modulus
	E   string // Base64URL-encoded exponent
	KID string // Key ID (SHA256 hash of modulus)
}

// GenerateRSAKeyPair generates a new 2048-bit RSA key pair for OIDC/JWKS usage
func GenerateRSAKeyPair() (*RSAKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	publicKey := &privateKey.PublicKey
	n := publicKey.N
	e := publicKey.E

	// Convert modulus (N) to base64url
	nBytes := n.Bytes()
	nBase64 := base64.RawURLEncoding.EncodeToString(nBytes)

	// Convert exponent (E) to base64url
	eBytes := big.NewInt(int64(e)).Bytes()
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	// Generate key ID as SHA256 hash of modulus (first 16 chars)
	hash := sha256.Sum256(n.Bytes())
	kid := fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes = 16 hex chars

	return &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		N:          nBase64,
		E:          eBase64,
		KID:        kid,
	}, nil
}

// JWK returns the public key in JWK (JSON Web Key) format
func (k *RSAKeyPair) JWK() map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"kid": k.KID,
		"n":   k.N,
		"e":   k.E,
		"alg": "RS256",
	}
}

// PrivateKeyToPEM converts an RSA private key to PEM format
func PrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	return privateKeyPEM
}
