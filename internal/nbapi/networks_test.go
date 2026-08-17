package nbapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListNetworksSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"1","name":"zulu"},{"id":"2","name":"Default"}]`)
	})

	networks, err := client.ListNetworks(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListNetworks: %v", err)
	}
	if len(networks) != 2 || networks[0].Name != "Default" || networks[1].Name != "zulu" {
		t.Errorf("order = %v, want [Default zulu]", networks)
	}
}

func TestGetNetworkPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetNetwork(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetNetwork: %v", err)
	}
	if gotPath != "/api/networks/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreateNetworkSendsNameAndDescription(t *testing.T) {
	var gotMethod string
	var gotBody NetworkRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"n1","name":"hq"}`)
	})

	_, err := client.CreateNetwork(context.Background(), "tok", NetworkRequest{Name: "hq", Description: "office"})
	if err != nil {
		t.Fatalf("CreateNetwork: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotBody.Name != "hq" || gotBody.Description != "office" {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestUpdateNetworkPuts(t *testing.T) {
	var gotMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = io.WriteString(w, `{"id":"n1","name":"renamed"}`)
	})

	network, err := client.UpdateNetwork(context.Background(), "tok", "n1", NetworkRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateNetwork: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if network.Name != "renamed" {
		t.Errorf("Name = %q, want renamed", network.Name)
	}
}

func TestDeleteNetworkPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteNetwork(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteNetwork: %v", err)
	}
	if gotPath != "/api/networks/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestListNetworkResourcesSortsByNameAtNetworkPath(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `[{"id":"1","name":"zulu"},{"id":"2","name":"Default"}]`)
	})

	resources, err := client.ListNetworkResources(context.Background(), "tok", "n1")
	if err != nil {
		t.Fatalf("ListNetworkResources: %v", err)
	}
	if gotPath != "/api/networks/n1/resources" {
		t.Errorf("path = %q, want /api/networks/n1/resources", gotPath)
	}
	if len(resources) != 2 || resources[0].Name != "Default" || resources[1].Name != "zulu" {
		t.Errorf("order = %v, want [Default zulu]", resources)
	}
}

func TestGetNetworkResourceReturnsResolvedGroups(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"r1","name":"db","address":"10.0.0.5/32","type":"host",
			"enabled":true,"groups":[{"id":"g1","name":"backend"}]}`)
	})

	resource, err := client.GetNetworkResource(context.Background(), "tok", "n1", "r1")
	if err != nil {
		t.Fatalf("GetNetworkResource: %v", err)
	}
	if len(resource.Groups) != 1 || resource.Groups[0].Name != "backend" {
		t.Errorf("Groups = %+v, want resolved [backend]", resource.Groups)
	}
	if resource.Type != "host" {
		t.Errorf("Type = %q, want host", resource.Type)
	}
}

func TestCreateNetworkResourceSendsAddressAndGroups(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody NetworkResourceRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"r1","name":"db"}`)
	})

	_, err := client.CreateNetworkResource(context.Background(), "tok", "n1", NetworkResourceRequest{
		Name: "db", Address: "10.0.0.5", Enabled: true, Groups: []string{"g1"},
	})
	if err != nil {
		t.Fatalf("CreateNetworkResource: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/networks/n1/resources" {
		t.Errorf("method/path = %s %s, want POST /api/networks/n1/resources", gotMethod, gotPath)
	}
	if gotBody.Address != "10.0.0.5" || len(gotBody.Groups) != 1 || gotBody.Groups[0] != "g1" {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestUpdateNetworkResourcePuts(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = io.WriteString(w, `{"id":"r1","name":"db-renamed"}`)
	})

	_, err := client.UpdateNetworkResource(context.Background(), "tok", "n1", "r1", NetworkResourceRequest{
		Name: "db-renamed", Address: "10.0.0.5",
	})
	if err != nil {
		t.Fatalf("UpdateNetworkResource: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/networks/n1/resources/r1" {
		t.Errorf("method/path = %s %s, want PUT /api/networks/n1/resources/r1", gotMethod, gotPath)
	}
}

func TestDeleteNetworkResourcePathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteNetworkResource(context.Background(), "tok", "n1", "a/../b"); err != nil {
		t.Fatalf("DeleteNetworkResource: %v", err)
	}
	if gotPath != "/api/networks/n1/resources/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestListNetworkRoutersAtNetworkPath(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `[{"id":"rt1","peer":"p1","metric":9999,"masquerade":true,"enabled":true}]`)
	})

	routers, err := client.ListNetworkRouters(context.Background(), "tok", "n1")
	if err != nil {
		t.Fatalf("ListNetworkRouters: %v", err)
	}
	if gotPath != "/api/networks/n1/routers" {
		t.Errorf("path = %q, want /api/networks/n1/routers", gotPath)
	}
	if len(routers) != 1 || routers[0].Peer != "p1" {
		t.Errorf("routers = %+v, want one router with peer p1", routers)
	}
}

func TestGetNetworkRouterReturnsRawPeerGroupIDs(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"rt1","peer_groups":["g1","g2"],"metric":100,"masquerade":false,"enabled":true}`)
	})

	router, err := client.GetNetworkRouter(context.Background(), "tok", "n1", "rt1")
	if err != nil {
		t.Fatalf("GetNetworkRouter: %v", err)
	}
	// Unlike NetworkResource.Groups, the server does not resolve these to
	// names — this pins that so a future server change surfaces as a test
	// failure here rather than a silent UI regression.
	if len(router.PeerGroups) != 2 || router.PeerGroups[0] != "g1" || router.PeerGroups[1] != "g2" {
		t.Errorf("PeerGroups = %v, want raw IDs [g1 g2]", router.PeerGroups)
	}
}

func TestCreateNetworkRouterSendsPeerGroups(t *testing.T) {
	var gotBody NetworkRouterRequest
	var gotRaw string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotRaw = string(raw)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = io.WriteString(w, `{"id":"rt1"}`)
	})

	_, err := client.CreateNetworkRouter(context.Background(), "tok", "n1", NetworkRouterRequest{
		PeerGroups: []string{"g1"}, Metric: 9999, Masquerade: true, Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateNetworkRouter: %v", err)
	}
	if len(gotBody.PeerGroups) != 1 || gotBody.PeerGroups[0] != "g1" {
		t.Errorf("PeerGroups = %v, want [g1]", gotBody.PeerGroups)
	}
	// Peer is omitempty and unset here — it must not appear in the request at
	// all, since sending "peer":"" alongside a non-empty peer_groups is the
	// exact shape the server's mutual-exclusivity check exists to reject if
	// it were ever sent non-omitted for both.
	if strings.Contains(gotRaw, `"peer"`) {
		t.Errorf("request body = %s, must omit peer when only peer_groups is set", gotRaw)
	}
}

func TestUpdateNetworkRouterSendsPeer(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody NetworkRouterRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"rt1"}`)
	})

	_, err := client.UpdateNetworkRouter(context.Background(), "tok", "n1", "rt1", NetworkRouterRequest{
		Peer: "p1", Metric: 100, Masquerade: false, Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateNetworkRouter: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/networks/n1/routers/rt1" {
		t.Errorf("method/path = %s %s, want PUT /api/networks/n1/routers/rt1", gotMethod, gotPath)
	}
	if gotBody.Peer != "p1" {
		t.Errorf("Peer = %q, want p1", gotBody.Peer)
	}
}

func TestDeleteNetworkRouterPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteNetworkRouter(context.Background(), "tok", "n1", "a/../b"); err != nil {
		t.Fatalf("DeleteNetworkRouter: %v", err)
	}
	if gotPath != "/api/networks/n1/routers/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}
