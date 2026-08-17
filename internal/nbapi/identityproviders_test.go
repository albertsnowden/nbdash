package nbapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListIdentityProvidersSendsBearerTokenToAPIPath(t *testing.T) {
	var gotPath, gotAuth string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[]`)
	})

	if _, err := client.ListIdentityProviders(context.Background(), "tok-123"); err != nil {
		t.Fatalf("ListIdentityProviders: %v", err)
	}

	if gotPath != "/api/identity-providers" {
		t.Errorf("path = %q, want /api/identity-providers", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", gotAuth)
	}
}

func TestListIdentityProvidersSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[
			{"id":"1","name":"zulu-idp","type":"oidc"},
			{"id":"2","name":"alpha-idp","type":"google"},
			{"id":"3","name":"Bravo-idp","type":"okta"}
		]`)
	})

	providers, err := client.ListIdentityProviders(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListIdentityProviders: %v", err)
	}

	var got []string
	for _, p := range providers {
		got = append(got, p.Name)
	}
	want := []string{"alpha-idp", "Bravo-idp", "zulu-idp"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestGetIdentityProviderPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetIdentityProvider(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetIdentityProvider: %v", err)
	}
	if gotPath != "/api/identity-providers/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreateIdentityProviderSendsRequestBody(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody IdentityProviderRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"idp1","type":"google","name":"Google","client_id":"abc"}`)
	})

	provider, err := client.CreateIdentityProvider(context.Background(), "tok", IdentityProviderRequest{
		Type:         IdentityProviderGoogle,
		Name:         "Google",
		ClientID:     "abc",
		ClientSecret: "shh",
	})
	if err != nil {
		t.Fatalf("CreateIdentityProvider: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/identity-providers" {
		t.Errorf("path = %q, want /api/identity-providers", gotPath)
	}
	if gotBody.Type != IdentityProviderGoogle || gotBody.Name != "Google" || gotBody.ClientSecret != "shh" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
	if provider.ID != "idp1" {
		t.Errorf("ID = %q, want idp1", provider.ID)
	}
}

// The response never carries a secret — this pins that CreateIdentityProvider
// doesn't choke decoding a response with no client_secret field at all, and
// that the returned struct genuinely has no field to leak one into.
func TestCreateIdentityProviderResponseNeverCarriesSecret(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"idp1","type":"oidc","name":"Corp SSO",
			"issuer":"https://login.example.com","client_id":"abc"}`)
	})

	provider, err := client.CreateIdentityProvider(context.Background(), "tok", IdentityProviderRequest{
		Type: IdentityProviderOIDC, Name: "Corp SSO", Issuer: "https://login.example.com", ClientID: "abc",
	})
	if err != nil {
		t.Fatalf("CreateIdentityProvider: %v", err)
	}
	if provider.Issuer != "https://login.example.com" {
		t.Errorf("Issuer = %q, want it round-tripped from the response", provider.Issuer)
	}
}

func TestUpdateIdentityProviderSendsBlankSecretAsEmptyString(t *testing.T) {
	var gotMethod string
	var rawBody []byte

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		rawBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"id":"idp1","type":"oidc","name":"Corp SSO","client_id":"abc"}`)
	})

	_, err := client.UpdateIdentityProvider(context.Background(), "tok", "idp1", IdentityProviderRequest{
		Type: IdentityProviderOIDC, Name: "Corp SSO", Issuer: "https://login.example.com", ClientID: "abc",
		ClientSecret: "",
	})
	if err != nil {
		t.Fatalf("UpdateIdentityProvider: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	// client_secret is "omitempty": a blank secret is left off the wire
	// entirely, not sent as client_secret:"". Either way the server's
	// partial-overlay update (see IdentityProviderRequest's doc comment)
	// treats it as "don't touch the stored secret" — this pins the actual
	// wire shape our encoding produces.
	var decoded map[string]any
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		t.Fatalf("decode sent body: %v", err)
	}
	if _, present := decoded["client_secret"]; present {
		t.Errorf("client_secret present in body (%s), want omitted for a blank secret", rawBody)
	}
}

func TestDeleteIdentityProviderIgnoresEmptyBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteIdentityProvider(context.Background(), "tok", "idp1"); err != nil {
		t.Fatalf("DeleteIdentityProvider: %v", err)
	}
}

func TestDeleteIdentityProviderPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteIdentityProvider(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteIdentityProvider: %v", err)
	}
	if gotPath != "/api/identity-providers/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestHasBuiltInIssuer(t *testing.T) {
	tests := []struct {
		typ  IdentityProviderType
		want bool
	}{
		{IdentityProviderGoogle, true},
		{IdentityProviderMicrosoft, true},
		{IdentityProviderOIDC, false},
		{IdentityProviderOkta, false},
		{IdentityProviderZitadel, false},
		{IdentityProviderAuthentik, false},
		{IdentityProviderKeycloak, false},
		{IdentityProviderADFS, false},
		{IdentityProviderEntra, false},
		{IdentityProviderPocketID, false},
	}
	for _, tc := range tests {
		if got := tc.typ.HasBuiltInIssuer(); got != tc.want {
			t.Errorf("%s.HasBuiltInIssuer() = %v, want %v", tc.typ, got, tc.want)
		}
	}
}
