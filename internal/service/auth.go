package service

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/repository"
)

type AuthService struct {
	users  repository.UserRepository
	secret string
}

func NewAuthService(users repository.UserRepository, secret string) *AuthService {
	return &AuthService{users: users, secret: secret}
}

func (s *AuthService) Login(email, password string) (string, map[string]interface{}, error) {
	u, err := s.users.FindByEmail(email)
	if err != nil {
		return "", nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return "", nil, err
	}

	claims := jwt.MapClaims{
		"uid":   u.ID,
		"role":  string(u.Role),
		"email": u.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", nil, err
	}
	payload := map[string]interface{}{
		"id": u.ID, "name": u.Name, "email": u.Email, "role": u.Role,
	}
	return signed, payload, nil
}
