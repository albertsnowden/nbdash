package nbapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListNameserverGroupsSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"1","name":"zulu"},{"id":"2","name":"Cloudflare"}]`)
	})

	groups, err := client.ListNameserverGroups(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListNameserverGroups: %v", err)
	}
	if len(groups) != 2 || groups[0].Name != "Cloudflare" || groups[1].Name != "zulu" {
		t.Errorf("order = %v, want [Cloudflare zulu]", groups)
	}
}

func TestGetNameserverGroupPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetNameserverGroup(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetNameserverGroup: %v", err)
	}
	if gotPath != "/api/dns/nameservers/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreateNameserverGroupSendsFields(t *testing.T) {
	var gotMethod string
	var gotBody NameserverGroupRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"ns1","name":"Cloudflare"}`)
	})

	_, err := client.CreateNameserverGroup(context.Background(), "tok", NameserverGroupRequest{
		Name:        "Cloudflare",
		Nameservers: []Nameserver{{IP: "1.1.1.1", NSType: "udp", Port: 53}},
		Enabled:     true,
		Groups:      []string{"g1"},
		Primary:     true,
	})
	if err != nil {
		t.Fatalf("CreateNameserverGroup: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if len(gotBody.Nameservers) != 1 || gotBody.Nameservers[0].IP != "1.1.1.1" {
		t.Errorf("Nameservers = %+v", gotBody.Nameservers)
	}
	if !gotBody.Primary {
		t.Error("Primary = false, want true")
	}
}

func TestUpdateNameserverGroupPuts(t *testing.T) {
	var gotMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = io.WriteString(w, `{"id":"ns1","name":"renamed"}`)
	})

	group, err := client.UpdateNameserverGroup(context.Background(), "tok", "ns1", NameserverGroupRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateNameserverGroup: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if group.Name != "renamed" {
		t.Errorf("Name = %q, want renamed", group.Name)
	}
}

func TestDeleteNameserverGroupPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteNameserverGroup(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteNameserverGroup: %v", err)
	}
	if gotPath != "/api/dns/nameservers/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestGetDNSSettingsSendsBearerToken(t *testing.T) {
	var gotPath, gotAuth string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"disabled_management_groups":["g1"]}`)
	})

	settings, err := client.GetDNSSettings(context.Background(), "tok-123")
	if err != nil {
		t.Fatalf("GetDNSSettings: %v", err)
	}
	if gotPath != "/api/dns/settings" {
		t.Errorf("path = %q, want /api/dns/settings", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if len(settings.DisabledManagementGroups) != 1 || settings.DisabledManagementGroups[0] != "g1" {
		t.Errorf("DisabledManagementGroups = %v, want [g1]", settings.DisabledManagementGroups)
	}
}

func TestUpdateDNSSettingsPuts(t *testing.T) {
	var gotMethod string
	var gotBody DNSSettings
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"disabled_management_groups":["g1","g2"]}`)
	})

	_, err := client.UpdateDNSSettings(context.Background(), "tok", DNSSettings{DisabledManagementGroups: []string{"g1", "g2"}})
	if err != nil {
		t.Fatalf("UpdateDNSSettings: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if len(gotBody.DisabledManagementGroups) != 2 {
		t.Errorf("DisabledManagementGroups = %v, want 2 entries", gotBody.DisabledManagementGroups)
	}
}

func TestListZonesSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"1","name":"zulu"},{"id":"2","name":"Office"}]`)
	})

	zones, err := client.ListZones(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListZones: %v", err)
	}
	if len(zones) != 2 || zones[0].Name != "Office" || zones[1].Name != "zulu" {
		t.Errorf("order = %v, want [Office zulu]", zones)
	}
}

func TestGetZoneReturnsRecords(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"z1","name":"Office","domain":"example.com",
			"records":[{"id":"r1","name":"www.example.com","type":"A","content":"192.168.1.1","ttl":300}]}`)
	})

	zone, err := client.GetZone(context.Background(), "tok", "z1")
	if err != nil {
		t.Fatalf("GetZone: %v", err)
	}
	if len(zone.Records) != 1 || zone.Records[0].Type != "A" {
		t.Errorf("Records = %+v", zone.Records)
	}
}

func TestGetZonePathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetZone(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetZone: %v", err)
	}
	if gotPath != "/api/dns/zones/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreateZoneSendsDomainAndDistributionGroups(t *testing.T) {
	var gotBody ZoneRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"z1","name":"Office"}`)
	})

	_, err := client.CreateZone(context.Background(), "tok", ZoneRequest{
		Name: "Office", Domain: "example.com", Enabled: true, DistributionGroups: []string{"g1"},
	})
	if err != nil {
		t.Fatalf("CreateZone: %v", err)
	}
	if gotBody.Domain != "example.com" {
		t.Errorf("Domain = %q, want example.com", gotBody.Domain)
	}
	if len(gotBody.DistributionGroups) != 1 || gotBody.DistributionGroups[0] != "g1" {
		t.Errorf("DistributionGroups = %v, want [g1]", gotBody.DistributionGroups)
	}
}

func TestUpdateZonePuts(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = io.WriteString(w, `{"id":"z1","name":"renamed"}`)
	})

	_, err := client.UpdateZone(context.Background(), "tok", "z1", ZoneRequest{Name: "renamed", Domain: "example.com"})
	if err != nil {
		t.Fatalf("UpdateZone: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/dns/zones/z1" {
		t.Errorf("method/path = %s %s, want PUT /api/dns/zones/z1", gotMethod, gotPath)
	}
}

func TestDeleteZonePathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteZone(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteZone: %v", err)
	}
	if gotPath != "/api/dns/zones/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestListZoneRecordsAtZonePath(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `[{"id":"r1","name":"www.example.com","type":"A","content":"192.168.1.1","ttl":300}]`)
	})

	records, err := client.ListZoneRecords(context.Background(), "tok", "z1")
	if err != nil {
		t.Fatalf("ListZoneRecords: %v", err)
	}
	if gotPath != "/api/dns/zones/z1/records" {
		t.Errorf("path = %q, want /api/dns/zones/z1/records", gotPath)
	}
	if len(records) != 1 || records[0].Content != "192.168.1.1" {
		t.Errorf("records = %+v", records)
	}
}

func TestCreateZoneRecordSendsFields(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody DNSRecordRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"r1"}`)
	})

	_, err := client.CreateZoneRecord(context.Background(), "tok", "z1", DNSRecordRequest{
		Name: "www.example.com", Type: "A", Content: "192.168.1.1", TTL: 300,
	})
	if err != nil {
		t.Fatalf("CreateZoneRecord: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/dns/zones/z1/records" {
		t.Errorf("method/path = %s %s, want POST /api/dns/zones/z1/records", gotMethod, gotPath)
	}
	if gotBody.Type != "A" || gotBody.TTL != 300 {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestUpdateZoneRecordPuts(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = io.WriteString(w, `{"id":"r1"}`)
	})

	_, err := client.UpdateZoneRecord(context.Background(), "tok", "z1", "r1", DNSRecordRequest{
		Name: "www.example.com", Type: "AAAA", Content: "::1", TTL: 60,
	})
	if err != nil {
		t.Fatalf("UpdateZoneRecord: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/dns/zones/z1/records/r1" {
		t.Errorf("method/path = %s %s, want PUT /api/dns/zones/z1/records/r1", gotMethod, gotPath)
	}
}

func TestDeleteZoneRecordPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteZoneRecord(context.Background(), "tok", "z1", "a/../b"); err != nil {
		t.Fatalf("DeleteZoneRecord: %v", err)
	}
	if gotPath != "/api/dns/zones/z1/records/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}
