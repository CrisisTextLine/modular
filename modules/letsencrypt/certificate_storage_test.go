package letsencrypt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCertificateStorage_SaveAndLoad tests the basic save and load functionality
func TestCertificateStorage_SaveAndLoad(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "letsencrypt_storage_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage, err := newCertificateStorage(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, storage)

	// Create test certificate
	domain := "example.com"
	certResource := createTestCertificate(t, domain)

	// Save certificate
	err = storage.SaveCertificate(domain, certResource)
	require.NoError(t, err)

	// Verify files were created
	domainDir := filepath.Join(tempDir, sanitizeDomain(domain))
	assert.DirExists(t, domainDir)
	assert.FileExists(t, filepath.Join(domainDir, "cert.pem"))
	assert.FileExists(t, filepath.Join(domainDir, "key.pem"))
	assert.FileExists(t, filepath.Join(domainDir, "metadata.txt"))

	// Load certificate
	cert, err := storage.LoadCertificate(domain)
	require.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotEmpty(t, cert.Certificate)
	assert.NotEmpty(t, cert.PrivateKey)
}

// TestCertificateStorage_ListCertificates tests listing stored certificates
func TestCertificateStorage_ListCertificates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "letsencrypt_list_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage, err := newCertificateStorage(tempDir)
	require.NoError(t, err)

	domains := []string{"example.com", "test.org", "demo.net"}

	// Save certificates for multiple domains
	for _, domain := range domains {
		certResource := createTestCertificate(t, domain)
		err := storage.SaveCertificate(domain, certResource)
		require.NoError(t, err)
	}

	// List certificates
	listedDomains, err := storage.ListCertificates()
	require.NoError(t, err)
	assert.Len(t, listedDomains, 3)

	// Verify all domains are present
	for _, domain := range domains {
		assert.Contains(t, listedDomains, domain)
	}
}

// TestCertificateStorage_ExpiryCheck tests certificate expiry detection
func TestCertificateStorage_ExpiryCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "letsencrypt_expiry_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage, err := newCertificateStorage(tempDir)
	require.NoError(t, err)

	// Create certificate expiring in 10 days
	domain := "expiring-soon.com"
	certResource := createTestCertificateWithExpiry(t, domain, time.Now().Add(10*24*time.Hour))

	err = storage.SaveCertificate(domain, certResource)
	require.NoError(t, err)

	// Check if expiring soon (within 30 days)
	expiringSoon, err := storage.IsCertificateExpiringSoon(domain, 30)
	require.NoError(t, err)
	assert.True(t, expiringSoon, "Certificate should be expiring soon")

	// Check if expiring soon (within 5 days)
	expiringSoon, err = storage.IsCertificateExpiringSoon(domain, 5)
	require.NoError(t, err)
	assert.False(t, expiringSoon, "Certificate should not be expiring within 5 days")
}

// TestCertificateStorage_Integration tests integration with LetsEncryptModule
func TestCertificateStorage_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "letsencrypt_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create module with storage configuration
	config := &LetsEncryptConfig{
		Email:       "test@example.com",
		Domains:     []string{"integration-test.com"},
		StoragePath: tempDir,
		UseStaging:  true,
	}

	module, err := New(config)
	require.NoError(t, err)
	assert.NotNil(t, module)

	// Manually initialize storage (normally done in Init)
	storage, err := newCertificateStorage(config.StoragePath)
	require.NoError(t, err)
	module.storage = storage
	module.logger = &testLogger{}

	// Pre-populate storage with a certificate
	domain := "integration-test.com"
	certResource := createTestCertificate(t, domain)
	err = storage.SaveCertificate(domain, certResource)
	require.NoError(t, err)

	// Test GetCertificateForDomain loads from storage
	cert, err := module.GetCertificateForDomain(domain)
	require.NoError(t, err)
	assert.NotNil(t, cert)

	// Verify certificate is now in memory
	module.certMutex.RLock()
	_, inMemory := module.certificates[domain]
	module.certMutex.RUnlock()
	assert.True(t, inMemory, "Certificate should be cached in memory after loading")
}

// TestCertificateStorage_WildcardDomains tests wildcard domain handling
func TestCertificateStorage_WildcardDomains(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "letsencrypt_wildcard_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage, err := newCertificateStorage(tempDir)
	require.NoError(t, err)

	// Save wildcard certificate
	wildcardDomain := "*.example.com"
	certResource := createTestCertificate(t, wildcardDomain)
	err = storage.SaveCertificate(wildcardDomain, certResource)
	require.NoError(t, err)

	// Load wildcard certificate
	cert, err := storage.LoadCertificate(wildcardDomain)
	require.NoError(t, err)
	assert.NotNil(t, cert)

	// Verify domain sanitization worked
	domainDir := filepath.Join(tempDir, sanitizeDomain(wildcardDomain))
	assert.DirExists(t, domainDir)
	assert.Contains(t, domainDir, "*_example_com")
}

// TestCertificateStorage_LoadNonExistent tests loading non-existent certificates
func TestCertificateStorage_LoadNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "letsencrypt_nonexistent_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage, err := newCertificateStorage(tempDir)
	require.NoError(t, err)

	// Try to load non-existent certificate
	cert, err := storage.LoadCertificate("nonexistent.com")
	assert.Error(t, err)
	assert.Nil(t, cert)
	assert.Contains(t, err.Error(), "no certificate found")
}

// TestSanitizeDomain tests domain sanitization functions
func TestSanitizeDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example_com"},
		{"*.example.com", "*_example_com"},
		{"sub.domain.example.com", "sub_domain_example_com"},
		{"no-dots", "no-dots"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := sanitizeDomain(test.input)
			assert.Equal(t, test.expected, result)

			// Test round-trip
			original := desanitizeDomain(result)
			assert.Equal(t, test.input, original)
		})
	}
}

// Helper function to create a test certificate
func createTestCertificate(t *testing.T, domain string) *certificate.Resource {
	return createTestCertificateWithExpiry(t, domain, time.Now().Add(90*24*time.Hour))
}

// Helper function to create a test certificate with specific expiry
func createTestCertificateWithExpiry(t *testing.T, domain string, notAfter time.Time) *certificate.Resource {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test Organization"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     notAfter,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     []string{domain},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return &certificate.Resource{
		Domain:            domain,
		Certificate:       certPEM,
		PrivateKey:        keyPEM,
		IssuerCertificate: nil, // Optional for testing
	}
}

// TestLetsEncryptModule_CertificateStorageLifecycle tests the full certificate storage lifecycle
func TestLetsEncryptModule_CertificateStorageLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "letsencrypt_lifecycle_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create and initialize module
	config := &LetsEncryptConfig{
		Email:           "lifecycle@example.com",
		Domains:         []string{"lifecycle.test.com"},
		StoragePath:     tempDir,
		UseStaging:      true,
		RenewBeforeDays: 30,
	}

	module, err := New(config)
	require.NoError(t, err)

	// Initialize storage manually (normally done in Init)
	storage, err := newCertificateStorage(tempDir)
	require.NoError(t, err)
	module.storage = storage
	module.logger = &testLogger{}

	// Test 1: Load existing certificates on startup
	domain := "lifecycle.test.com"
	certResource := createTestCertificate(t, domain)
	err = storage.SaveCertificate(domain, certResource)
	require.NoError(t, err)

	err = module.loadExistingCertificates()
	require.NoError(t, err)

	// Verify certificate was loaded into memory
	module.certMutex.RLock()
	_, found := module.certificates[domain]
	module.certMutex.RUnlock()
	assert.True(t, found, "Certificate should be loaded into memory")

	// Test 2: GetCertificateForDomain returns stored certificate
	cert, err := module.GetCertificateForDomain(domain)
	require.NoError(t, err)
	assert.NotNil(t, cert)

	// Test 3: Certificate expiry checking integrates with storage
	expiringSoonCert := createTestCertificateWithExpiry(t, "expiring.test.com", time.Now().Add(15*24*time.Hour))
	err = storage.SaveCertificate("expiring.test.com", expiringSoonCert)
	require.NoError(t, err)

	// Simulate renewal check (without actual ACME calls)
	var renewalNeeded bool
	_ = context.Background() // Context would be used in real renewal scenario

	// This would normally trigger renewal, but we'll just check the logic
	if storage != nil {
		expiring, err := storage.IsCertificateExpiringSoon("expiring.test.com", config.RenewBeforeDays)
		require.NoError(t, err)
		renewalNeeded = expiring
	}

	assert.True(t, renewalNeeded, "Certificate should be flagged for renewal")

	// Test 4: Wildcard certificate handling
	wildcardDomain := "*.wildcard-test.com"
	wildcardCert := createTestCertificate(t, wildcardDomain)
	err = storage.SaveCertificate(wildcardDomain, wildcardCert)
	require.NoError(t, err)

	// Clear memory to force storage load
	module.certMutex.Lock()
	delete(module.certificates, wildcardDomain)
	module.certMutex.Unlock()

	// Request certificate for subdomain - should load wildcard from storage
	cert, err = module.GetCertificateForDomain("sub.wildcard-test.com")
	require.NoError(t, err)
	assert.NotNil(t, cert)
}
