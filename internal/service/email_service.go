package service

import (
	"log"
)

type EmailService interface {
	SendOTP(email string, otp string) error
}

type stubEmailService struct{}

func NewEmailService() EmailService {
	return &stubEmailService{}
}

func (s *stubEmailService) SendOTP(email string, otp string) error {
	log.Printf("========================================================\n")
	log.Printf("📧 [STUB EMAIL] TO: %s\n", email)
	log.Printf("🔐 OTP CODE: %s\n", otp)
	log.Printf("========================================================\n")
	return nil
}
