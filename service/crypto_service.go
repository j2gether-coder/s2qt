package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const (
	defaultCryptoVersion   = 1
	defaultPinLength       = 6
	defaultPBKDF2IterCount = 120000
	defaultKDFName         = "pbkdf2-sha256"
	defaultSaltSize        = 32
	defaultKeySize         = 32 // AES-256

	pinLockoutPermanent = 15
)

// PinLockoutStatus describes the current lockout state returned to UI.
type PinLockoutStatus struct {
	FailedCount     int   `json:"failedCount"`
	LockedUntilUnix int64 `json:"lockedUntilUnix"`
	RemainingSecs   int64 `json:"remainingSecs"`
	Permanent       bool  `json:"permanent"`
}

type SecurityConfig struct {
	Version         int    `json:"version"`
	PinEnabled      bool   `json:"pin_enabled"`
	PinLength       int    `json:"pin_length"`
	PinHash         string `json:"pin_hash"`
	Salt            string `json:"salt"`
	KDF             string `json:"kdf"`
	Iterations      int    `json:"iterations"`
	FailedCount     int    `json:"failed_count"`
	LockedUntilUnix int64  `json:"locked_until_unix"`
	PermanentLock   bool   `json:"permanent_lock"`
}

type CryptoService struct {
	securityFile string
	config       *SecurityConfig
}

func NewCryptoService(securityFile string) (*CryptoService, error) {
	if stringsTrim(securityFile) == "" {
		return nil, fmt.Errorf("security file path is empty")
	}

	s := &CryptoService{
		securityFile: securityFile,
	}

	if err := s.loadOrCreateConfig(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *CryptoService) IsPinEnabled() bool {
	if s == nil || s.config == nil {
		return false
	}
	return s.config.PinEnabled
}

func (s *CryptoService) GetPinLength() int {
	if s == nil || s.config == nil || (s.config.PinLength != 4 && s.config.PinLength != 6) {
		return defaultPinLength
	}
	return s.config.PinLength
}

func (s *CryptoService) GetSecurityConfig() SecurityConfig {
	if s == nil || s.config == nil {
		return SecurityConfig{
			Version:    defaultCryptoVersion,
			PinEnabled: false,
			PinLength:  defaultPinLength,
			KDF:        defaultKDFName,
			Iterations: defaultPBKDF2IterCount,
		}
	}
	return *s.config
}

// SetupPin initializes or replaces the PIN metadata.
// NOTE:
// Changing PIN changes the derived encryption key.
// Existing encrypted secret values may become undecryptable unless they are re-encrypted.
func (s *CryptoService) SetupPin(pin string) error {
	if s == nil {
		return fmt.Errorf("crypto service is nil")
	}

	if err := s.validatePinFormat(pin); err != nil {
		return err
	}

	salt, err := generateRandomSalt(defaultSaltSize)
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	hash, err := s.derivePinHash(pin, salt, s.config.Iterations)
	if err != nil {
		return fmt.Errorf("failed to derive pin hash: %w", err)
	}

	s.config.PinEnabled = true
	s.config.PinHash = hash
	s.config.Salt = base64.StdEncoding.EncodeToString(salt)
	s.config.FailedCount = 0
	s.config.LockedUntilUnix = 0
	s.config.PermanentLock = false

	return s.saveConfig()
}

// ChangePin verifies old pin first, then replaces the security config.
// NOTE:
// Existing encrypted secret values may need to be re-entered after changing PIN.
func (s *CryptoService) ChangePin(oldPin, newPin string) error {
	ok, err := s.VerifyPin(oldPin)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("current pin is invalid")
	}
	return s.SetupPin(newPin)
}

func (s *CryptoService) VerifyPin(pin string) (bool, error) {
	if s == nil || s.config == nil {
		return false, fmt.Errorf("crypto service is not initialized")
	}
	if !s.config.PinEnabled {
		return false, fmt.Errorf("pin is not enabled")
	}

	if s.config.PermanentLock {
		return false, fmt.Errorf("PIN이 영구 잠금 상태입니다. 보안 설정 초기화가 필요합니다")
	}

	now := time.Now().Unix()
	if s.config.LockedUntilUnix > now {
		remain := s.config.LockedUntilUnix - now
		return false, fmt.Errorf("PIN 잠금 상태입니다. %d초 후 다시 시도해 주세요", remain)
	}

	if err := s.validatePinFormat(pin); err != nil {
		return false, err
	}

	salt, err := base64.StdEncoding.DecodeString(s.config.Salt)
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := s.derivePinHash(pin, salt, s.config.Iterations)
	if err != nil {
		return false, err
	}

	matched := subtle.ConstantTimeCompare([]byte(hash), []byte(s.config.PinHash)) == 1

	if matched {
		if s.config.FailedCount != 0 || s.config.LockedUntilUnix != 0 {
			s.config.FailedCount = 0
			s.config.LockedUntilUnix = 0
			_ = s.saveConfig()
		}
		return true, nil
	}

	if err := s.registerFailedAttempt(); err != nil {
		return false, err
	}

	if s.config.PermanentLock {
		return false, fmt.Errorf("PIN 실패 횟수가 초과되어 영구 잠금되었습니다. 보안 설정 초기화가 필요합니다")
	}
	if s.config.LockedUntilUnix > time.Now().Unix() {
		remain := s.config.LockedUntilUnix - time.Now().Unix()
		return false, fmt.Errorf("PIN 잠금이 적용되었습니다. %d초 후 다시 시도해 주세요", remain)
	}
	return false, nil
}

// registerFailedAttempt increments failure counter and applies staged lockout.
//
//	5회: 30초, 7회: 5분, 10회: 30분, 15회: 영구 잠금
func (s *CryptoService) registerFailedAttempt() error {
	s.config.FailedCount++

	var delaySec int64
	switch {
	case s.config.FailedCount >= pinLockoutPermanent:
		s.config.PermanentLock = true
	case s.config.FailedCount >= 10:
		delaySec = 30 * 60
	case s.config.FailedCount >= 7:
		delaySec = 5 * 60
	case s.config.FailedCount >= 5:
		delaySec = 30
	}

	if delaySec > 0 {
		s.config.LockedUntilUnix = time.Now().Unix() + delaySec
	}

	return s.saveConfig()
}

// GetPinLockoutStatus exposes the current lockout state for the UI.
func (s *CryptoService) GetPinLockoutStatus() PinLockoutStatus {
	if s == nil || s.config == nil {
		return PinLockoutStatus{}
	}
	status := PinLockoutStatus{
		FailedCount:     s.config.FailedCount,
		LockedUntilUnix: s.config.LockedUntilUnix,
		Permanent:       s.config.PermanentLock,
	}
	now := time.Now().Unix()
	if s.config.LockedUntilUnix > now {
		status.RemainingSecs = s.config.LockedUntilUnix - now
	}
	return status
}

// ResetPinLockout clears the PIN security file and returns a list of secret
// setting keys that must be wiped by the caller (SMTP/LLM passwords become
// undecryptable once the PIN is removed).
func (s *CryptoService) ResetPinLockout() error {
	if s == nil || s.config == nil {
		return fmt.Errorf("crypto service is not initialized")
	}
	s.config.PinEnabled = false
	s.config.PinHash = ""
	s.config.Salt = ""
	s.config.FailedCount = 0
	s.config.LockedUntilUnix = 0
	s.config.PermanentLock = false
	return s.saveConfig()
}

func (s *CryptoService) ClearPin() error {
	if s == nil {
		return fmt.Errorf("crypto service is nil")
	}

	s.config.PinEnabled = false
	s.config.PinHash = ""
	s.config.Salt = ""
	s.config.FailedCount = 0
	s.config.LockedUntilUnix = 0
	s.config.PermanentLock = false
	return s.saveConfig()
}

func (s *CryptoService) EncryptString(pin, plain string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("crypto service is nil")
	}
	if !s.config.PinEnabled {
		return "", fmt.Errorf("pin is not enabled")
	}

	key, err := s.deriveKey(pin)
	if err != nil {
		return "", err
	}

	return encryptToString([]byte(plain), key)
}

func (s *CryptoService) DecryptString(pin, encodedCipherText string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("crypto service is nil")
	}
	if !s.config.PinEnabled {
		return "", fmt.Errorf("pin is not enabled")
	}

	key, err := s.deriveKey(pin)
	if err != nil {
		return "", err
	}

	plain, err := decryptFromString(encodedCipherText, key)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func (s *CryptoService) loadOrCreateConfig() error {
	dir := filepath.Dir(s.securityFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create security directory: %w", err)
	}

	if _, err := os.Stat(s.securityFile); err == nil {
		return s.loadConfig()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat security file: %w", err)
	}

	s.config = &SecurityConfig{
		Version:    defaultCryptoVersion,
		PinEnabled: false,
		PinLength:  defaultPinLength,
		PinHash:    "",
		Salt:       "",
		KDF:        defaultKDFName,
		Iterations: defaultPBKDF2IterCount,
	}

	return s.saveConfig()
}

func (s *CryptoService) loadConfig() error {
	b, err := os.ReadFile(s.securityFile)
	if err != nil {
		return fmt.Errorf("failed to read security file: %w", err)
	}

	var cfg SecurityConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("failed to parse security file: %w", err)
	}

	// 기본값 보정
	if cfg.Version == 0 {
		cfg.Version = defaultCryptoVersion
	}
	if cfg.PinLength != 4 && cfg.PinLength != 6 {
		cfg.PinLength = defaultPinLength
	}
	if cfg.KDF == "" {
		cfg.KDF = defaultKDFName
	}
	if cfg.Iterations <= 0 {
		cfg.Iterations = defaultPBKDF2IterCount
	}

	s.config = &cfg
	return nil
}

func (s *CryptoService) saveConfig() error {
	if s == nil || s.config == nil {
		return fmt.Errorf("security config is nil")
	}

	b, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal security config: %w", err)
	}

	if err := os.WriteFile(s.securityFile, b, 0o600); err != nil {
		return fmt.Errorf("failed to write security file: %w", err)
	}

	return nil
}

func (s *CryptoService) deriveKey(pin string) ([]byte, error) {
	ok, err := s.VerifyPin(pin)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("pin verification failed")
	}

	salt, err := base64.StdEncoding.DecodeString(s.config.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	key := pbkdf2.Key([]byte(pin), salt, s.config.Iterations, defaultKeySize, sha256.New)
	return key, nil
}

func (s *CryptoService) derivePinHash(pin string, salt []byte, iter int) (string, error) {
	key := pbkdf2.Key([]byte(pin), salt, iter, defaultKeySize, sha256.New)
	sum := sha256.Sum256(key)
	return base64.StdEncoding.EncodeToString(sum[:]), nil
}

func (s *CryptoService) validatePinFormat(pin string) error {
	length := s.GetPinLength()

	if len(pin) != length {
		return fmt.Errorf("pin length must be %d", length)
	}

	for _, ch := range pin {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("pin must contain digits only")
		}
	}

	return nil
}

func generateRandomSalt(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid salt size")
	}
	salt := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func encryptToString(plainText []byte, key []byte) (string, error) {
	encrypted, err := encrypt(plainText, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptFromString(encodedCipherText string, key []byte) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(encodedCipherText)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	return decrypt(cipherText, key)
}

// encrypt uses AES-256-GCM and returns nonce||ciphertext.
func encrypt(plainText []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid aes key length: %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plainText, nil), nil
}

func decrypt(cipherText []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid aes key length: %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("cipher text is too short")
	}

	nonce := cipherText[:nonceSize]
	actualCipherText := cipherText[nonceSize:]

	return gcm.Open(nil, nonce, actualCipherText, nil)
}
