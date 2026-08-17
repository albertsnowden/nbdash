package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListNameserverGroups returns every nameserver group on the account.
func (c *Client) ListNameserverGroups(ctx context.Context, token string) ([]NameserverGroup, error) {
	var groups []NameserverGroup
	if err := c.do(ctx, token, http.MethodGet, "/dns/nameservers", nil, nil, &groups); err != nil {
		return nil, err
	}
	sortNameserverGroups(groups)
	return groups, nil
}

// GetNameserverGroup returns a single nameserver group.
func (c *Client) GetNameserverGroup(ctx context.Context, token, id string) (*NameserverGroup, error) {
	var group NameserverGroup
	if err := c.do(ctx, token, http.MethodGet, "/dns/nameservers/"+url.PathEscape(id), nil, nil, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// CreateNameserverGroup creates a nameserver group.
func (c *Client) CreateNameserverGroup(ctx context.Context, token string, req NameserverGroupRequest) (*NameserverGroup, error) {
	var group NameserverGroup
	if err := c.do(ctx, token, http.MethodPost, "/dns/nameservers", nil, req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateNameserverGroup replaces a nameserver group's editable fields —
// req must carry the complete desired state, not just changes, the same
// full-replace-on-PUT discipline every other resource in this API follows.
func (c *Client) UpdateNameserverGroup(ctx context.Context, token, id string, req NameserverGroupRequest) (*NameserverGroup, error) {
	var group NameserverGroup
	if err := c.do(ctx, token, http.MethodPut, "/dns/nameservers/"+url.PathEscape(id), nil, req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// DeleteNameserverGroup removes a nameserver group.
func (c *Client) DeleteNameserverGroup(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/dns/nameservers/"+url.PathEscape(id), nil, nil, nil)
}

func sortNameserverGroups(groups []NameserverGroup) {
	sort.SliceStable(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name)
	})
}

// GetDNSSettings returns the account's single DNS settings object — there is
// exactly one per account, unlike every other resource this client fetches,
// which is why there is no accompanying List/Create/Delete.
func (c *Client) GetDNSSettings(ctx context.Context, token string) (*DNSSettings, error) {
	var settings DNSSettings
	if err := c.do(ctx, token, http.MethodGet, "/dns/settings", nil, nil, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateDNSSettings replaces the account's DNS settings.
func (c *Client) UpdateDNSSettings(ctx context.Context, token string, settings DNSSettings) (*DNSSettings, error) {
	var updated DNSSettings
	if err := c.do(ctx, token, http.MethodPut, "/dns/settings", nil, settings, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// ListZones returns every custom DNS zone on the account.
func (c *Client) ListZones(ctx context.Context, token string) ([]Zone, error) {
	var zones []Zone
	if err := c.do(ctx, token, http.MethodGet, "/dns/zones", nil, nil, &zones); err != nil {
		return nil, err
	}
	sortZones(zones)
	return zones, nil
}

// GetZone returns a single zone, including its records.
func (c *Client) GetZone(ctx context.Context, token, id string) (*Zone, error) {
	var zone Zone
	if err := c.do(ctx, token, http.MethodGet, "/dns/zones/"+url.PathEscape(id), nil, nil, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

// CreateZone creates a zone. req.Domain must not already be in use by
// another zone or by the account's own peer DNS domain (see Zone's doc
// comment).
func (c *Client) CreateZone(ctx context.Context, token string, req ZoneRequest) (*Zone, error) {
	var zone Zone
	if err := c.do(ctx, token, http.MethodPost, "/dns/zones", nil, req, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

// UpdateZone replaces a zone's editable fields — everything except Domain,
// which the server rejects changing (see Zone's doc comment); req.Domain
// must still be sent as whatever the zone's current domain already is.
func (c *Client) UpdateZone(ctx context.Context, token, id string, req ZoneRequest) (*Zone, error) {
	var zone Zone
	if err := c.do(ctx, token, http.MethodPut, "/dns/zones/"+url.PathEscape(id), nil, req, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

// DeleteZone removes a zone and, server-side, every record in it
// (DeleteZone in the manager deletes the zone's records in the same
// transaction before the zone itself).
func (c *Client) DeleteZone(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/dns/zones/"+url.PathEscape(id), nil, nil, nil)
}

func sortZones(zones []Zone) {
	sort.SliceStable(zones, func(i, j int) bool {
		return strings.ToLower(zones[i].Name) < strings.ToLower(zones[j].Name)
	})
}

// ListZoneRecords returns every record in the given zone.
func (c *Client) ListZoneRecords(ctx context.Context, token, zoneID string) ([]DNSRecord, error) {
	var records []DNSRecord
	path := "/dns/zones/" + url.PathEscape(zoneID) + "/records"
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// GetZoneRecord returns a single record.
func (c *Client) GetZoneRecord(ctx context.Context, token, zoneID, recordID string) (*DNSRecord, error) {
	var record DNSRecord
	path := "/dns/zones/" + url.PathEscape(zoneID) + "/records/" + url.PathEscape(recordID)
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateZoneRecord adds a record to a zone.
func (c *Client) CreateZoneRecord(ctx context.Context, token, zoneID string, req DNSRecordRequest) (*DNSRecord, error) {
	var record DNSRecord
	path := "/dns/zones/" + url.PathEscape(zoneID) + "/records"
	if err := c.do(ctx, token, http.MethodPost, path, nil, req, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateZoneRecord replaces a record's editable fields.
func (c *Client) UpdateZoneRecord(ctx context.Context, token, zoneID, recordID string, req DNSRecordRequest) (*DNSRecord, error) {
	var record DNSRecord
	path := "/dns/zones/" + url.PathEscape(zoneID) + "/records/" + url.PathEscape(recordID)
	if err := c.do(ctx, token, http.MethodPut, path, nil, req, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteZoneRecord removes a record from a zone.
func (c *Client) DeleteZoneRecord(ctx context.Context, token, zoneID, recordID string) error {
	path := "/dns/zones/" + url.PathEscape(zoneID) + "/records/" + url.PathEscape(recordID)
	return c.do(ctx, token, http.MethodDelete, path, nil, nil, nil)
}
