package compliance

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-barcode-webapp/internal/models"
)

// DigitalSignatureManager handles digital signatures for documents
type DigitalSignatureManager struct {
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	keyPath     string
	companyName string
}

// SignedDocument represents a digitally signed document
type SignedDocument struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	DocumentType    string    `json:"document_type" gorm:"not null;index"`
	DocumentID      string    `json:"document_id" gorm:"not null;index"`
	DocumentHash    string    `json:"document_hash" gorm:"not null"`
	Signature       string    `json:"signature" gorm:"type:text;not null"`
	SignatureHash   string    `json:"signature_hash" gorm:"not null"`
	SignedAt        time.Time `json:"signed_at" gorm:"not null"`
	SignedBy        string    `json:"signed_by" gorm:"not null"`
	CompanyName     string    `json:"company_name" gorm:"not null"`
	SigningMethod   string    `json:"signing_method" gorm:"not null"` // RSA-SHA256
	PublicKeyHash   string    `json:"public_key_hash" gorm:"not null"`
	IsValid         bool      `json:"is_valid" gorm:"default:true"`
	CertificatePath string    `json:"certificate_path"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SignedDocument) TableName() string {
	return "signed_documents"
}

// DocumentIntegrity represents document integrity information
type DocumentIntegrity struct {
	DocumentHash    string    `json:"document_hash"`
	Signature       string    `json:"signature"`
	SignedAt        time.Time `json:"signed_at"`
	SignedBy        string    `json:"signed_by"`
	IsValid         bool      `json:"is_valid"`
	VerifiedAt      time.Time `json:"verified_at"`
	CompanyName     string    `json:"company_name"`
	SigningMethod   string    `json:"signing_method"`
}

// NewDigitalSignatureManager creates a new digital signature manager
func NewDigitalSignatureManager(keyPath, companyName string) (*DigitalSignatureManager, error) {
	dsm := &DigitalSignatureManager{
		keyPath:     keyPath,
		companyName: companyName,
	}

	// Ensure key directory exists
	if err := os.MkdirAll(keyPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Load or generate key pair
	if err := dsm.loadOrGenerateKeys(); err != nil {
		return nil, fmt.Errorf("failed to load or generate keys: %w", err)
	}

	return dsm, nil
}

// loadOrGenerateKeys loads existing keys or generates new ones
func (dsm *DigitalSignatureManager) loadOrGenerateKeys() error {
	privateKeyPath := filepath.Join(dsm.keyPath, "private_key.pem")
	publicKeyPath := filepath.Join(dsm.keyPath, "public_key.pem")

	// Check if keys exist
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		// Generate new key pair
		return dsm.generateKeyPair()
	}

	// Load existing keys
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse private key
	privateBlock, _ := pem.Decode(privateKeyData)
	if privateBlock == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Parse public key
	publicBlock, _ := pem.Decode(publicKeyData)
	if publicBlock == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	dsm.privateKey = privateKey
	dsm.publicKey = publicKey

	return nil
}

// generateKeyPair generates a new RSA key pair
func (dsm *DigitalSignatureManager) generateKeyPair() error {
	// Generate 2048-bit RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	dsm.privateKey = privateKey
	dsm.publicKey = &privateKey.PublicKey

	// Save private key
	privateKeyPath := filepath.Join(dsm.keyPath, "private_key.pem")
	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	// Save public key
	publicKeyPath := filepath.Join(dsm.keyPath, "public_key.pem")
	publicKeyFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(dsm.publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to encode public key: %w", err)
	}

	return nil
}

// SignInvoice digitally signs an invoice
func (dsm *DigitalSignatureManager) SignInvoice(invoice *models.Invoice, signedBy string) (*SignedDocument, error) {
	// Serialize invoice for signing
	invoiceData, err := json.Marshal(invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize invoice: %w", err)
	}

	// Calculate document hash
	documentHash := dsm.calculateHash(invoiceData)

	// Create signature data
	signatureData := fmt.Sprintf("%s:%s:%s:%s:%s",
		"invoice",
		fmt.Sprintf("%d", invoice.InvoiceID),
		documentHash,
		signedBy,
		time.Now().Format(time.RFC3339),
	)

	// Sign the data
	signature, err := dsm.signData([]byte(signatureData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	// Calculate public key hash for verification
	publicKeyHash := dsm.getPublicKeyHash()

	// Create signed document record
	signedDoc := &SignedDocument{
		DocumentType:  "invoice",
		DocumentID:    fmt.Sprintf("%d", invoice.InvoiceID),
		DocumentHash:  documentHash,
		Signature:     signature,
		SignatureHash: dsm.calculateHash([]byte(signature)),
		SignedAt:      time.Now(),
		SignedBy:      signedBy,
		CompanyName:   dsm.companyName,
		SigningMethod: "RSA-SHA256",
		PublicKeyHash: publicKeyHash,
		IsValid:       true,
	}

	return signedDoc, nil
}

// SignDocument digitally signs any document
func (dsm *DigitalSignatureManager) SignDocument(documentType, documentID string, data interface{}, signedBy string) (*SignedDocument, error) {
	// Serialize document for signing
	documentData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize document: %w", err)
	}

	// Calculate document hash
	documentHash := dsm.calculateHash(documentData)

	// Create signature data
	signatureData := fmt.Sprintf("%s:%s:%s:%s:%s",
		documentType,
		documentID,
		documentHash,
		signedBy,
		time.Now().Format(time.RFC3339),
	)

	// Sign the data
	signature, err := dsm.signData([]byte(signatureData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	// Calculate public key hash for verification
	publicKeyHash := dsm.getPublicKeyHash()

	// Create signed document record
	signedDoc := &SignedDocument{
		DocumentType:  documentType,
		DocumentID:    documentID,
		DocumentHash:  documentHash,
		Signature:     signature,
		SignatureHash: dsm.calculateHash([]byte(signature)),
		SignedAt:      time.Now(),
		SignedBy:      signedBy,
		CompanyName:   dsm.companyName,
		SigningMethod: "RSA-SHA256",
		PublicKeyHash: publicKeyHash,
		IsValid:       true,
	}

	return signedDoc, nil
}

// VerifySignature verifies the digital signature of a document
func (dsm *DigitalSignatureManager) VerifySignature(signedDoc *SignedDocument, documentData []byte) (bool, error) {
	// Calculate current document hash
	currentHash := dsm.calculateHash(documentData)
	
	// Check if document hash matches
	if currentHash != signedDoc.DocumentHash {
		return false, fmt.Errorf("document hash mismatch: document has been modified")
	}

	// Recreate signature data
	signatureData := fmt.Sprintf("%s:%s:%s:%s:%s",
		signedDoc.DocumentType,
		signedDoc.DocumentID,
		signedDoc.DocumentHash,
		signedDoc.SignedBy,
		signedDoc.SignedAt.Format(time.RFC3339),
	)

	// Verify signature
	return dsm.verifySignature([]byte(signatureData), signedDoc.Signature)
}

// GetDocumentIntegrity gets the integrity information for a document
func (dsm *DigitalSignatureManager) GetDocumentIntegrity(documentType, documentID string) (*DocumentIntegrity, error) {
	// This would typically query the database for the signed document
	// For now, returning a mock response
	return &DocumentIntegrity{
		DocumentHash:  "mock_hash",
		Signature:     "mock_signature",
		SignedAt:      time.Now(),
		SignedBy:      "system",
		IsValid:       true,
		VerifiedAt:    time.Now(),
		CompanyName:   dsm.companyName,
		SigningMethod: "RSA-SHA256",
	}, nil
}

// signData signs data using RSA-SHA256
func (dsm *DigitalSignatureManager) signData(data []byte) (string, error) {
	// Hash the data
	hash := sha256.Sum256(data)

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, dsm.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign data: %w", err)
	}

	// Encode to base64
	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifySignature verifies a signature using RSA-SHA256
func (dsm *DigitalSignatureManager) verifySignature(data []byte, signature string) (bool, error) {
	// Decode signature from base64
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the data
	hash := sha256.Sum256(data)

	// Verify the signature
	err = rsa.VerifyPKCS1v15(dsm.publicKey, crypto.SHA256, hash[:], signatureBytes)
	return err == nil, err
}

// calculateHash calculates SHA256 hash of data
func (dsm *DigitalSignatureManager) calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// getPublicKeyHash calculates hash of public key for verification
func (dsm *DigitalSignatureManager) getPublicKeyHash() string {
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(dsm.publicKey)
	return dsm.calculateHash(publicKeyBytes)
}

// ExportPublicKey exports the public key for external verification
func (dsm *DigitalSignatureManager) ExportPublicKey() (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(dsm.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	return string(pem.EncodeToMemory(publicKeyPEM)), nil
}

