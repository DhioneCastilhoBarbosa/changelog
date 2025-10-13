package service

import (
	"fmt"
	"os"
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

// lê TTL do access token. Default 2h para manter seu comportamento atual.
func accessTokenTTL() time.Duration {
	if v := os.Getenv("ACCESS_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 2 * time.Hour
}

func (s *AuthService) Login(email, password string) (string, map[string]interface{}, error) {
	u, err := s.users.FindByEmail(email)
	if err != nil {
		return "", nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return "", nil, err
	}

	now := time.Now().UTC()
	ttl := accessTokenTTL()
	exp := now.Add(ttl)

	claims := jwt.MapClaims{
		// seus dados de domínio
		"uid":   u.ID,
		"role":  string(u.Role),
		"email": u.Email,

		// padrões JWT
		"sub": fmt.Sprint(u.ID),
		"iat": now.Unix(),          // issued at
		"nbf": now.Add(-5 * time.Second).Unix(), // leeway anti-skew
		"exp": exp.Unix(),          // expiration real, controlada por env
		"iss": "firmware-changelog",// opcional, para auditoria
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(s.secret))
	if err != nil {
		return "", nil, err
	}

	// mantém payload atual sem quebrar o frontend
	payload := map[string]interface{}{
		"id": u.ID, "name": u.Name, "email": u.Email, "role": u.Role,
		// se quiser facilitar o cliente a sincronizar, pode expor exp também:
		// "exp": exp.Unix(),
	}
	return signed, payload, nil
}
