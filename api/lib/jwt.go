package lib

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

var JwtKey string = "DRAWINGAICHATGPT"

func secret(secval string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(secval), nil
	}
}

func ParseToken(tokenss string) (uint, error) {
	token, err := jwt.Parse(tokenss, secret(JwtKey))
	if err != nil {
		return 0, err
	}
	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("cannot convert claim to mapclaim")
	}
	//验证token，如果token被修改过则为false
	if !token.Valid {
		return 0, errors.New("token is invalid")
	}

	if cusid, ok := claim["cus_id"]; ok {
		return uint(cusid.(float64)), nil
	}
	return 0, errors.New("cusid is not in token")
}

func CreateToken(id int) (string, error) {
	tokenExp := time.Now().Add(time.Hour * 24 * 365).Unix()
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["cus_id"] = id
	claims["exp"] = tokenExp

	value, err := token.SignedString([]byte(JwtKey))
	if err != nil {
		return "", err
	}
	RDB.HSet(context.Background(), RedisUserToken, id, fmt.Sprintf("%d", tokenExp))

	return value, nil
}
