package auth

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    Sub   string   `json:"sub"`
    Email string   `json:"email"`
    Name  string   `json:"name"`
    Roles []string `json:"roles"`
    jwt.RegisteredClaims
}

func Sign(secret string, c Claims, ttl time.Duration) (string, error) {
    now := time.Now()
    c.RegisteredClaims = jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now)}
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
    return token.SignedString([]byte(secret))
}

func Parse(secret, token string) (*Claims, error) {
    t, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
    if err != nil {
        return nil, err
    }
    if cl, ok := t.Claims.(*Claims); ok && t.Valid {
        return cl, nil
    }
    return nil, jwt.ErrTokenInvalidClaims
}

