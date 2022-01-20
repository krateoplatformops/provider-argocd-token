package accounts

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
)

// Login do a login with username and password credentials and returns the auth token.
func Login(opts *TokenProviderOptions, user, pass string) (string, error) {
	cli, err := NewTokenProvider(opts)
	if err != nil {
		return "", err
	}

	return cli.CreateSession(user, pass)
}

// GenerateToken generate a token for the account with the specified name.
// expiresIn specify the duration before the token will expire; by default: no expiration.
func GenerateToken(opts *TokenProviderOptions, name string, expiresIn int64) (string, error) {
	cli, err := NewTokenProvider(opts)
	if err != nil {
		return "", err
	}

	return cli.CreateTokenForAccount(name)
}

// TokenProviderOptions hold address, security, and other settings for the API client.
type TokenProviderOptions struct {
	ServerAddr string
	UserAgent  string
	AuthToken  string
}

// TokenProvider defines an interface for interaction with an Argo CD server.
type TokenProvider interface {
	CreateSession(username, password string) (string, error)
	CreateTokenForAccount(name string) (string, error)
}

// NewTokenProvider creates a new ArgoCD token provider from a set of config options.
func NewTokenProvider(opts *TokenProviderOptions) (TokenProvider, error) {
	var res tokenProvider

	if opts.UserAgent != "" {
		res.userAgent = opts.UserAgent
	}

	if opts.ServerAddr != "" {
		res.serverAddr = opts.ServerAddr
	}
	// Make sure we got the server address and auth token from somewhere
	if res.serverAddr == "" {
		return nil, errors.New("unspecified server address for Argo CD")
	}

	res.httpClient = &http.Client{}
	res.httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &res, nil
}

type tokenProvider struct {
	serverAddr string
	userAgent  string
	authToken  string
	httpClient *http.Client
}

func (tp tokenProvider) CreateSession(user, pass string) (string, error) {
	data := map[string]string{
		"username": user,
		"password": pass,
	}

	bin, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/api/v1/session", tp.serverAddr)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bin))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := tp.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	//debug(httputil.DumpResponse(res, true))

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create argocd session request failed: %s", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response map[string]string
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	return response["token"], nil
}

func (tp tokenProvider) CreateTokenForAccount(name string) (string, error) {
	/*
		data := map[string]string{}

		bin, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
	*/
	url := fmt.Sprintf("%s/api/v1/account/%s/token", tp.serverAddr, name)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tp.authToken))

	debug(httputil.DumpRequestOut(req, true))

	res, err := tp.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	debug(httputil.DumpResponse(res, true))

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create argocd account token request failed: %s", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response map[string]string
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	return response["token"], nil
}

func debug(data []byte, err error) {
	if err == nil {
		fmt.Printf("%s\n\n", data)
	} else {
		log.Fatalf("%s\n\n", err)
	}
}
