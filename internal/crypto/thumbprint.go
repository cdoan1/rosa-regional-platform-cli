package crypto

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// GetOIDCThumbprint fetches the TLS certificate from the OIDC issuer URL
// and calculates the SHA-1 fingerprint of the root certificate.
// This is required by AWS IAM when creating an OIDC provider.
//
// This function mimics Terraform's data.tls_certificate behavior:
// https://registry.terraform.io/providers/hashicorp/tls/latest/docs/data-sources/certificate
func GetOIDCThumbprint(ctx context.Context, issuerURL string) (string, error) {
	// Parse the URL to extract the host
	parsedURL, err := url.Parse(issuerURL)
	if err != nil {
		return "", fmt.Errorf("invalid OIDC issuer URL: %w", err)
	}

	// Ensure the URL uses HTTPS
	if parsedURL.Scheme != "https" {
		return "", fmt.Errorf("OIDC issuer URL must use HTTPS, got: %s", parsedURL.Scheme)
	}

	host := parsedURL.Host
	if !strings.Contains(host, ":") {
		// Add default HTTPS port if not specified
		host = host + ":443"
	}

	// Create a TLS connection to fetch the certificate chain
	dialer := &tls.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return "", fmt.Errorf("failed to connect to OIDC issuer: %w", err)
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "", fmt.Errorf("expected TLS connection, got %T", conn)
	}

	// Get the certificate chain
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return "", fmt.Errorf("no certificates found in TLS connection")
	}

	// Get the root certificate (last in the chain)
	// AWS IAM requires the thumbprint of the root CA certificate
	rootCert := state.PeerCertificates[len(state.PeerCertificates)-1]

	// Calculate SHA-1 fingerprint of the DER-encoded certificate
	// AWS IAM uses SHA-1 for OIDC provider thumbprints
	hash := sha1.Sum(rootCert.Raw)
	thumbprint := hex.EncodeToString(hash[:])

	return thumbprint, nil
}
