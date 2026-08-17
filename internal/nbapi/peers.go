package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListPeers returns every peer on the account.
//
// The API's own ?name= and ?ip= filters are not used: search here spans name,
// hostname, IP, OS and group names at once, which those two parameters cannot
// express. Accounts are small enough that the whole list is cheap to fetch —
// the Next.js dashboard loads it in full and filters in the browser for the
// same reason. Revisit if peer counts grow into the thousands.
func (c *Client) ListPeers(ctx context.Context, token string) ([]Peer, error) {
	var peers []Peer
	if err := c.do(ctx, token, http.MethodGet, "/peers", nil, nil, &peers); err != nil {
		return nil, err
	}
	sortPeers(peers)
	return peers, nil
}

// GetPeer returns a single peer.
func (c *Client) GetPeer(ctx context.Context, token, id string) (*Peer, error) {
	var peer Peer
	if err := c.do(ctx, token, http.MethodGet, "/peers/"+url.PathEscape(id), nil, nil, &peer); err != nil {
		return nil, err
	}
	return &peer, nil
}

// UpdatePeer writes the editable fields of a peer and returns the stored
// result. req must carry the peer's full editable set — see PeerRequest.
func (c *Client) UpdatePeer(ctx context.Context, token, id string, req PeerRequest) (*Peer, error) {
	var peer Peer
	if err := c.do(ctx, token, http.MethodPut, "/peers/"+url.PathEscape(id), nil, req, &peer); err != nil {
		return nil, err
	}
	return &peer, nil
}

// DeletePeer removes a peer from the network.
func (c *Client) DeletePeer(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/peers/"+url.PathEscape(id), nil, nil, nil)
}

// FilterPeers narrows peers to those matching a free-text query, checking the
// fields a user is likely to paste in: name, hostname, address, OS and groups.
// An empty query returns the input untouched.
func FilterPeers(peers []Peer, query string) []Peer {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return peers
	}

	matched := make([]Peer, 0, len(peers))
	for _, p := range peers {
		if peerMatches(p, query) {
			matched = append(matched, p)
		}
	}
	return matched
}

func peerMatches(p Peer, query string) bool {
	fields := []string{p.Name, p.Hostname, p.IP, p.IPv6, p.DNSLabel, p.OS, p.Version, p.CityName, p.CountryCode}
	for _, f := range fields {
		if f != "" && strings.Contains(strings.ToLower(f), query) {
			return true
		}
	}
	for _, g := range p.Groups {
		if strings.Contains(strings.ToLower(g.Name), query) {
			return true
		}
	}
	return false
}

// sortPeers puts connected peers first, then orders by name, so the peers a
// user is most likely to act on sit at the top.
func sortPeers(peers []Peer) {
	sort.SliceStable(peers, func(i, j int) bool {
		if peers[i].Connected != peers[j].Connected {
			return peers[i].Connected
		}
		return strings.ToLower(peers[i].Name) < strings.ToLower(peers[j].Name)
	})
}
