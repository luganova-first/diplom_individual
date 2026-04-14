package userauth

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"log"
	"net/http"
	"time"
)

// Claims — структура утверждений, которая включает стандартные утверждения
// и одно пользовательское — UserLogin
type Claims struct {
	jwt.RegisteredClaims
	UserLogin string
}

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "supersecretkey"

func GetUserID(tokenString string) string {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SECRET_KEY), nil
		})
	if err != nil {
		log.Println(err)
		return ""
	}

	if !token.Valid {
		log.Println("Token is not valid")
		return ""
	}

	return claims.UserLogin
}

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userLogin string) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		// собственное утверждение
		UserLogin: userLogin,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func GetUserCookie(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string
		userLogin := ""

		cookie, err := r.Cookie("jwt")
		if err != nil {
			if err != http.ErrNoCookie {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			tokenString = cookie.Value
			userLogin = GetUserID(tokenString)
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "userLogin", userLogin)
		r = r.WithContext(ctx)

		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	})
}
