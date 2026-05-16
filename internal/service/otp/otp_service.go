package otp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	emailSvc "github.com/prast13/bayaraman/internal/service/email"
	rateLimiterSvc "github.com/prast13/bayaraman/internal/service/ratelimiter"
	"github.com/redis/go-redis/v9"
)

type OTPService interface {
	GenerateAndSendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email string, otp string) error
}

type otpService struct {
	redisClient *redis.Client
	emailSvc    emailSvc.EmailService
	rateLimiter rateLimiterSvc.RateLimiterService
}

func NewOTPService(redisClient *redis.Client, emailSvc emailSvc.EmailService, rateLimiter rateLimiterSvc.RateLimiterService) OTPService {
	return &otpService{redisClient: redisClient, emailSvc: emailSvc, rateLimiter: rateLimiter}
}

func (s *otpService) GenerateAndSendOTP(ctx context.Context, email string) error {
	err := s.rateLimiter.CheckLimit(ctx, "otp", email, 3, 15*time.Minute)
	if err != nil {
		return errors.New("too many OTP requests, please wait")
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	otp := fmt.Sprintf("%06d", n.Int64()+100000)

	otpKey := fmt.Sprintf("otp:%s", email)
	err = s.redisClient.Set(ctx, otpKey, otp, 5*time.Minute).Err()
	if err != nil {
		return err
	}

	return s.emailSvc.SendOTP(email, otp)
}

func (s *otpService) VerifyOTP(ctx context.Context, email string, inputOTP string) error {
	otpKey := fmt.Sprintf("otp:%s", email)
	storedOTP, err := s.redisClient.Get(ctx, otpKey).Result()
	if err == redis.Nil {
		return errors.New("OTP expired or not found")
	} else if err != nil {
		return err
	}

	if storedOTP != inputOTP {
		return errors.New("invalid OTP")
	}

	s.redisClient.Del(ctx, otpKey)
	s.rateLimiter.ResetLimit(ctx, "otp", email)

	return nil
}
