package conchhorse

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/concourse/fly/rc"
	"github.com/concourse/go-concourse/concourse"
	"golang.org/x/oauth2"
)

func NewClient(host, username, password string) (concourse.Client, error) {
	target, err := rc.NewUnauthenticatedTarget(
		"target",
		host,
		"main",
		false,
		"",
		true)

	if err != nil {
		return nil, err
	}

	tokenType, tokenValue, err := passwordGrant(target.Client(), username, password)
	if err != nil {
		return nil, err
	}

	token := &rc.TargetToken{
		Type:  tokenType,
		Value: tokenValue,
	}

	httpClient := defaultHttpClient(token, true, nil)
	return concourse.NewClient(host, httpClient, true), nil
}

func passwordGrant(client concourse.Client, username, password string) (string, string, error) {

	oauth2Config := oauth2.Config{
		ClientID:     "fly",
		ClientSecret: "Zmx5",
		Endpoint:     oauth2.Endpoint{TokenURL: client.URL() + "/sky/token"},
		Scopes:       []string{"openid", "profile", "email", "federated:id", "groups"},
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client.HTTPClient())

	token, err := oauth2Config.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return "", "", err
	}

	return token.TokenType, token.AccessToken, nil
}

func defaultHttpClient(token *rc.TargetToken, insecure bool, caCertPool *x509.CertPool) *http.Client {
	var oAuthToken *oauth2.Token
	if token != nil {
		oAuthToken = &oauth2.Token{
			TokenType:   token.Type,
			AccessToken: token.Value,
		}
	}

	transport := transport(insecure, caCertPool)

	if token != nil {
		transport = &oauth2.Transport{
			Source: oauth2.StaticTokenSource(oAuthToken),
			Base:   transport,
		}
	}

	return &http.Client{Transport: transport}
}

func transport(insecure bool, caCertPool *x509.CertPool) http.RoundTripper {
	var transport http.RoundTripper

	transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
			RootCAs:            caCertPool,
		},
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		Proxy: http.ProxyFromEnvironment,
	}

	return transport
}
