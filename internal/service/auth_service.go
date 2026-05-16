package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"github.com/prast13/bayaraman/internal/repository"
	"github.com/prast13/bayaraman/pkg/hash"
	"github.com/prast13/bayaraman/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*model.User, error)
	VerifyEmail(ctx context.Context, email string, otp string) error
	Login(ctx context.Context, req LoginRequest, jwtSecret string) (*LoginResponse, error)
	Refresh(ctx context.Context, refreshTokenString string, ip string, ua string, jwtSecret string) (*LoginResponse, error)
	Logout(ctx context.Context, refreshTokenString string) error
}

type authService struct {
	userRepo     repository.UserRepository
	sessionRepo  repository.SessionRepository
	auditLogRepo repository.AuditLogRepository
	otpSvc       OTPService
	redisClient  *redis.Client
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	auditLogRepo repository.AuditLogRepository,
	otpSvc OTPService,
	redisClient *redis.Client,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		auditLogRepo: auditLogRepo,
		otpSvc:       otpSvc,
		redisClient:  redisClient,
	}
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*model.User, error) {
	existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         "buyer", // Default
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.otpSvc.GenerateAndSendOTP(ctx, user.Email)

	return user, nil
}

func (s *authService) VerifyEmail(ctx context.Context, email string, otp string) error {
	err := s.otpSvc.VerifyOTP(ctx, email, otp)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return errors.New("user not found")
	}

	user.IsEmailVerified = true
	return s.userRepo.Update(ctx, user)
}

func (s *authService) Login(ctx context.Context, req LoginRequest, jwtSecret string) (*LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !user.IsEmailVerified {
		return nil, errors.New("please verify your email first")
	}

	match, err := hash.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, errors.New("invalid email or password")
	}

	accessToken, refreshToken, err := jwt.GenerateTokens(user.ID, user.Role, jwtSecret)
	if err != nil {
		return nil, err
	}

	session := &model.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
	}
	err = s.sessionRepo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	sessionKey := fmt.Sprintf("session:%s", session.ID.String())
	s.redisClient.Set(ctx, sessionKey, user.ID.String(), 7*24*time.Hour)

	_ = s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:    user.ID,
		Action:    "LOGIN_SUCCESS",
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	})

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: session.ID.String() + "|" + refreshToken,
	}, nil
}

func (s *authService) Refresh(ctx context.Context, refreshTokenString string, ip string, ua string, jwtSecret string) (*LoginResponse, error) {
	parts := strings.Split(refreshTokenString, "|")
	if len(parts) != 2 {
		return nil, errors.New("invalid refresh token format")
	}
	sessionIDStr, tokenStr := parts[0], parts[1]

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return nil, errors.New("invalid session id")
	}

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, errors.New("session not found")
	}

	// 1. Token Reuse Detection (Security: Refresh Token Rotation)
	if session.IsRevoked {
		// DANGER: Attempted reuse of a revoked token! Revoke ALL sessions for this user.
		_ = s.sessionRepo.RevokeAllUserSessions(ctx, session.UserID)
		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID:    session.UserID,
			Action:    "TOKEN_THEFT_DETECTED",
			IPAddress: ip,
			UserAgent: ua,
		})
		return nil, errors.New("security alert: token reuse detected, all sessions revoked")
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("refresh token expired")
	}

	if session.RefreshToken != tokenStr {
		return nil, errors.New("invalid refresh token")
	}

	// Revoke the old session
	session.IsRevoked = true
	s.sessionRepo.Update(ctx, session)

	// Remove old session from Redis cache
	sessionKey := fmt.Sprintf("session:%s", session.ID.String())
	s.redisClient.Del(ctx, sessionKey)

	// Generate new tokens
	user, _ := s.userRepo.FindByID(ctx, session.UserID)
	accessToken, newRefreshToken, err := jwt.GenerateTokens(user.ID, user.Role, jwtSecret)
	if err != nil {
		return nil, err
	}

	// Create new session
	newSession := &model.Session{
		UserID:       user.ID,
		RefreshToken: newRefreshToken,
		IPAddress:    ip,
		UserAgent:    ua,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
	}
	err = s.sessionRepo.Create(ctx, newSession)
	if err != nil {
		return nil, err
	}

	// Cache new session to Redis
	newSessionKey := fmt.Sprintf("session:%s", newSession.ID.String())
	s.redisClient.Set(ctx, newSessionKey, user.ID.String(), 7*24*time.Hour)

	_ = s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:    user.ID,
		Action:    "REFRESH_TOKEN",
		IPAddress: ip,
		UserAgent: ua,
	})

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newSession.ID.String() + "|" + newRefreshToken,
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshTokenString string) error {
	parts := strings.Split(refreshTokenString, "|")
	if len(parts) != 2 {
		return errors.New("invalid refresh token format")
	}
	sessionIDStr := parts[0]

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return errors.New("invalid session id")
	}

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return errors.New("session not found")
	}

	session.IsRevoked = true
	s.sessionRepo.Update(ctx, session)

	sessionKey := fmt.Sprintf("session:%s", session.ID.String())
	s.redisClient.Del(ctx, sessionKey)

	_ = s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID:    session.UserID,
		Action:    "LOGOUT",
	})

	return nil
}
