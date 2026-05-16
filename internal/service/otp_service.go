package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

type OTPService interface {
	GenerateAndSendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email string, otp string) error
}

type otpService struct {
	redisClient *redis.Client
	emailSvc    EmailService
}

func NewOTPService(redisClient *redis.Client, emailSvc EmailService) OTPService {
	return &otpService{redisClient: redisClient, emailSvc: emailSvc}
}

func (s *otpService) GenerateAndSendOTP(ctx context.Context, email string) error {

	rateKey := fmt.Sprintf("ratelimit:otp:%s", email)
	count, err := s.redisClient.Get(ctx, rateKey).Int()
	if err == nil && count >= 3 {
		return errors.New("too many OTP requests, please wait")
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	otp := fmt.Sprintf("%06d", n.Int64()+100000)

	otpKey := fmt.Sprintf("otp:%s", email)
	err = s.redisClient.Set(ctx, otpKey, otp, 5*time.Minute).Err()
	if err != nil {
		return err
	}

	if err == redis.Nil || count == 0 {
		s.redisClient.Set(ctx, rateKey, 1, 15*time.Minute).Err()
	} else {
		s.redisClient.Incr(ctx, rateKey).Err()
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
	s.redisClient.Del(ctx, fmt.Sprintf("ratelimit:otp:%s", email))

	return nil
}
