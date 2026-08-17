package nbapi

import "testing"

func sample() []Peer {
	return []Peer{
		{
			ID: "1", Name: "gateway-01", Hostname: "gw01.corp", IP: "100.92.0.1",
			OS: "Linux Ubuntu 22.04", CityName: "Berlin", CountryCode: "DE",
			Groups: []GroupMinimum{{Name: "Servers"}, {Name: "All"}},
		},
		{
			ID: "2", Name: "ada-laptop", Hostname: "ada-mbp", IP: "100.92.0.2",
			OS: "Darwin 14.5", Groups: []GroupMinimum{{Name: "All"}},
		},
	}
}

func TestFilterPeers(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query returns everything", "", []string{"gateway-01", "ada-laptop"}},
		{"whitespace only returns everything", "   ", []string{"gateway-01", "ada-laptop"}},
		{"matches name", "gateway", []string{"gateway-01"}},
		{"is case insensitive", "GATEWAY", []string{"gateway-01"}},
		{"matches hostname", "mbp", []string{"ada-laptop"}},
		{"matches partial ip", "0.2", []string{"ada-laptop"}},
		{"matches os", "darwin", []string{"ada-laptop"}},
		{"matches group name", "servers", []string{"gateway-01"}},
		{"matches city", "berlin", []string{"gateway-01"}},
		{"group shared by both matches both", "all", []string{"gateway-01", "ada-laptop"}},
		{"no match returns nothing", "nonesuch", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterPeers(sample(), tc.query)

			if len(got) != len(tc.want) {
				t.Fatalf("got %d peers, want %d (%v)", len(got), len(tc.want), names(got))
			}
			for i, want := range tc.want {
				if got[i].Name != want {
					t.Errorf("peer[%d] = %q, want %q", i, got[i].Name, want)
				}
			}
		})
	}
}

// An empty field must not match every query via strings.Contains(x, "").
func TestFilterPeersIgnoresEmptyFields(t *testing.T) {
	peers := []Peer{{ID: "1", Name: "only-name"}}

	if got := FilterPeers(peers, "berlin"); len(got) != 0 {
		t.Errorf("expected no match against empty fields, got %v", names(got))
	}
}

func names(peers []Peer) []string {
	out := make([]string, 0, len(peers))
	for _, p := range peers {
		out = append(out, p.Name)
	}
	return out
}
