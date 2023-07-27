package main

import (
    "encoding/json"
    "errors"
	"crypto/subtle"
    "github.com/auth0/go-jwt-middleware"
    "github.com/form3tech-oss/jwt-go"
    "net/http"
	"os"
	"log"
	"time"
)

var auth0api Auth0Cred
var jwtMiddleware *jwtmiddleware.JWTMiddleware

type Jwks struct {
    Keys[] JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
    Kty string `json:"kty"`
    Kid string `json:"kid"`
    Use string `json:"use"`
    N string `json:"n"`
    E string `json:"e"`
    X5c[] string `json:"x5c"`
}

type Auth0Cred struct {
    Audience string
    Issuer string
	ClientID string
	MgmtAccessToken string
}


func VerifyAudience(m jwt.MapClaims, cmp string, req bool) bool {
	var aud []string
	switch v := m["aud"].(type) {
	case string:
		aud = append(aud, v)
	case []string:
		aud = v
	case []interface{}:
		for _, a := range v {
			vs, ok := a.(string)
			if !ok {
				return false
			}
			aud = append(aud, vs)
			break;
		}
	}
	return verifyAud(aud, cmp, req)
}

func verifyAud(aud []string, cmp string, required bool) bool {
	if len(aud) == 0 {
		return !required
	}
	// use a var here to keep constant time compare when looping over a number of claims
	result := false

	var stringClaims string
	for _, a := range aud {
		if subtle.ConstantTimeCompare([]byte(a), []byte(cmp)) != 0 {
			result = true
		}
		stringClaims = stringClaims + a
	}

	// case where "" is sent in one or many aud claims
	if len(stringClaims) == 0 {
		return !required
	}

	return result
}

func getPemCert(token *jwt.Token) (string, error) {
    cert := ""
    resp, err := http.Get(auth0api.Issuer + ".well-known/jwks.json")

    if err != nil {
        return cert, err
    }
    defer resp.Body.Close()

    var jwks = Jwks{}
    err = json.NewDecoder(resp.Body).Decode(&jwks)

    if err != nil {
        return cert, err
    }

    for k, _ := range jwks.Keys {
        if token.Header["kid"] == jwks.Keys[k].Kid {
            cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
        }
    }

    if cert == "" {
        err := errors.New("Unable to find appropriate key.")
        return cert, err
    }

    return cert, nil
}

func getSubFromJWTToken(tokenString string) string {
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return nil, nil // Since the token is already verified, we don't need to re-verify it here.
	})
	claims, _ := token.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	return sub
}


func InitAuthorization(){
	plan, _ := os.ReadFile("../auth0api.json")
	json.Unmarshal(plan, &auth0api)

	jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {

			aud := auth0api.Audience
			checkAud := VerifyAudience(token.Claims.(jwt.MapClaims), aud, false)
			if !checkAud {
				return token, errors.New("Invalid audience.")
			}

			iss := auth0api.Issuer
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("Invalid issuer.")
			}

			cert, err := getPemCert(token)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
	})
}

func verifyAuth0JWT(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		aud := auth0api.Audience
		checkAud := VerifyAudience(token.Claims.(jwt.MapClaims), aud, false)
		if !checkAud {
			return token, errors.New("Invalid audience.")
		}

		iss := auth0api.Issuer
		checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
		if !checkIss {
			return token, errors.New("Invalid issuer.")
		}

		cert, err := getPemCert(token)
		if err != nil {
			panic(err.Error())
		}

		result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		return result, nil
	})

	if err != nil {
		log.Println("Error parsing token:", err)
		return false
	}

	if !token.Valid {
		log.Println("Invalid token")
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		expirationTime := time.Unix(int64(claims["exp"].(float64)), 0)
		if time.Now().After(expirationTime) {
			log.Println("Token has expired")
			return false
		}
	}

	return true
}