package service

import "fmt"

type SecurityService struct {
	cryptoSvc *CryptoService
}

func NewSecurityService(cryptoSvc *CryptoService) *SecurityService {
	return &SecurityService{
		cryptoSvc: cryptoSvc,
	}
}

func (s *SecurityService) IsPinEnabled() (bool, error) {
	if s == nil || s.cryptoSvc == nil {
		return false, fmt.Errorf("crypto service is nil")
	}
	return s.cryptoSvc.IsPinEnabled(), nil
}

func (s *SecurityService) SetupPin(pin string) error {
	if s == nil || s.cryptoSvc == nil {
		return fmt.Errorf("crypto service is nil")
	}
	return s.cryptoSvc.SetupPin(pin)
}

func (s *SecurityService) ChangePin(oldPin, newPin string) error {
	if s == nil || s.cryptoSvc == nil {
		return fmt.Errorf("crypto service is nil")
	}
	return s.cryptoSvc.ChangePin(oldPin, newPin)
}

func (s *SecurityService) VerifyPin(pin string) (bool, error) {
	if s == nil || s.cryptoSvc == nil {
		return false, fmt.Errorf("crypto service is nil")
	}
	return s.cryptoSvc.VerifyPin(pin)
}

func (s *SecurityService) ClearPin() error {
	if s == nil || s.cryptoSvc == nil {
		return fmt.Errorf("crypto service is nil")
	}
	return s.cryptoSvc.ClearPin()
}

func (s *SecurityService) GetPinLength() (int, error) {
	if s == nil || s.cryptoSvc == nil {
		return 6, fmt.Errorf("crypto service is nil")
	}
	return s.cryptoSvc.GetPinLength(), nil
}
