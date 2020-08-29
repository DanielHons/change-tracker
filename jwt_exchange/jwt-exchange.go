package jwt_exchange

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const bearerPrefix = "Bearer "
const authorization = "Authorization"

type ServiceConfig struct {
	jwks                  *jwksCache
	JwksUrl               string
	jwksRefreshInterval   time.Duration
	ProxyRawUrl           string
	RequestDirector       func(r *http.Request)
	IncomingTokenHeader   TokenHeaderField
	OutgoingTokenHeader   TokenHeaderField
	InternalJwtSecret     string
	InternalTokenAudience string
	BindPort              string
	ClaimsMapper          func(claims jwt.MapClaims) jwt.Claims
}

type TokenHeaderField struct {
	header string
	bearer bool
}

func PlainTokenHeader(header string) TokenHeaderField {
	return TokenHeaderField{
		header: header,
		bearer: false,
	}
}

func BearerTokenHeader(header string) TokenHeaderField {
	return TokenHeaderField{
		header: header,
		bearer: true,
	}
}

type jwksCache struct {
	JwksUrl             string
	jwksMutex           sync.Mutex
	JwksRefreshInterval time.Duration
	lastRefresh         int64
	jwkSet              *jwk.Set
}

func NewJwksCache(jwksUrl string, refreshInterval time.Duration) *jwksCache {
	cache := jwksCache{
		JwksUrl:             jwksUrl,
		JwksRefreshInterval: refreshInterval,
		lastRefresh:         0,
		jwkSet:              nil,
	}
	err := cache.reloadJwks()
	if err != nil {
		log.Fatal("Could not load JWKS to start working:", err)
	}
	return &cache
}

func TokenExchangerConfigFromEnv() ServiceConfig {
	return ServiceConfig{
		ProxyRawUrl:       os.Getenv("TARGET_URL"),
		InternalJwtSecret: os.Getenv("JWT_SECRET"),
		jwks:              NewJwksCache(os.Getenv("JWKS_URL"), 24*time.Second),
		ClaimsMapper:      defaultClaumsMapper,
		RequestDirector:   defaultDirector(os.Getenv("TARGET_URL")),
		// The default header configuration is to search for header "Authorization" with Content "Bearer "+$token
		IncomingTokenHeader: TokenHeaderField{
			header: getEnvOrDefault("TOKEN_HEADER_IN", authorization),
			bearer: true,
		},
		// This configuration can also be written as follows:
		OutgoingTokenHeader: BearerTokenHeader(getEnvOrDefault("TOKEN_HEADER_OUT", authorization)),
		BindPort:            getEnvOrDefault("PORT", "3000"),
	}
}

func defaultClaumsMapper(claims jwt.MapClaims) jwt.Claims {
	return claims
}

func getEnvOrDefault(key string, defaultValue string) string {
	env := os.Getenv(key)
	if len(env) == 0 {
		return defaultValue
	}
	return env
}

func (c *jwksCache) refreshIfRequired() {
	if c.lastRefresh+c.JwksRefreshInterval.Milliseconds()/1000 < time.Now().Unix() {
		go func() {
			// this can wait till the request was handled
			defer c.jwksMutex.Unlock()
			_ = c.reloadJwks() //fire and forget
		}()
	}
}

func (c *jwksCache) reloadJwks() error {
	log.Println("Refreshing jwk set from " + c.JwksUrl)
	set, err := jwk.FetchHTTP(c.JwksUrl)
	if err != nil {
		log.Println("ERROR fetching jwk set from " + c.JwksUrl)
		return err
	}
	c.jwkSet = set
	c.lastRefresh = time.Now().Unix()
	log.Println("jwk set refreshedL")
	return nil
}

func (c *jwksCache) getKey(token *jwt.Token) (interface{}, error) {
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	if key := c.jwkSet.LookupKeyID(keyID); len(key) == 1 {
		var k interface{}
		err := key[0].Raw(&k)
		return k, err
	}

	return nil, fmt.Errorf("unable to find key %q", keyID)
}

func (c *jwksCache) checkToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, c.getKey)
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(jwt.MapClaims)
	return claims, nil
}

func (config *ServiceConfig) ProxyHandler() func(w http.ResponseWriter, r *http.Request) {
	proxy := &httputil.ReverseProxy{Director: config.RequestDirector}
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract claims from
		token := extractTokenFromIncomingRequest(r, config.IncomingTokenHeader)
		claims, err := config.jwks.checkToken(token)
		if err != nil {
			log.Println("Authentication failed:", err)
			w.WriteHeader(401)
			w.Write([]byte(err.Error()))
			return
		}
		newToken, err := config.createInternalToken(claims, config.InternalJwtSecret)
		if err != nil {
			log.Println("Could not create the internal token:", err)
			w.WriteHeader(401)
			return
		}

		setInternalTokenToRequest(r, newToken, config.OutgoingTokenHeader)
		proxy.ServeHTTP(w, r)
	}
}

func defaultDirector(rawUrl string) func(req *http.Request) {
	targetUrl := rawUrl
	origin, _ := url.Parse(targetUrl)
	director := func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", origin.Host)
		req.URL.Scheme = extractProtocol(targetUrl)
		req.URL.Host = origin.Host
	}
	return director
}

func extractProtocol(targetUrl string) string {
	return strings.Split(targetUrl, "://")[0]
}

func setInternalTokenToRequest(r *http.Request, newToken string, field TokenHeaderField) {
	if field.bearer {
		newToken = "Bearer " + newToken
	}
	r.Header.Set(field.header, newToken)
}

func extractTokenFromIncomingRequest(r *http.Request, field TokenHeaderField) string {
	tokenHeader := r.Header.Get(field.header)
	if field.bearer {
		tokenHeader = strings.TrimPrefix(tokenHeader, bearerPrefix)
	}
	r.Header.Set(field.header, "") // Do not forward
	return tokenHeader
}

func (config *ServiceConfig) createInternalToken(claims jwt.MapClaims, secret string) (string, error) {
	internalToken, signingErr := jwt.NewWithClaims(jwt.SigningMethodHS256, config.ClaimsMapper(claims)).SignedString([]byte(secret))
	if signingErr != nil {
		return "", signingErr
	}
	return internalToken, nil
}