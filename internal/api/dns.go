// DNS bundles three independent sub-features, mirroring
// internal/handlers/dns.go: Nameserver Groups (custom resolvers for the
// overlay), Settings (a single account-wide object, no list/create/delete),
// and Zones (custom DNS records this account serves itself, with Records as
// an independent REST sub-resource per zone).
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// maxNameserverGroupNameChars mirrors nbdns.MaxGroupNameChar server-side —
// checked here too so an over-length name is rejected immediately rather
// than after a round trip.
const maxNameserverGroupNameChars = 40

type nameserverGroupsResponse struct {
	Groups []nbapi.NameserverGroup `json:"groups"`
	Total  int                     `json:"total"`
}

func (s *Server) listNameserverGroups(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	groups, err := s.api.ListNameserverGroups(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, nameserverGroupsResponse{Groups: groups, Total: len(groups)})
}

func (s *Server) getNameserverGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	group, err := s.api.GetNameserverGroup(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, group)
}

type nameserverGroupRequest struct {
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Nameservers          []nbapi.Nameserver `json:"nameservers"`
	Enabled              bool               `json:"enabled"`
	Groups               []string           `json:"groups"`
	Primary              bool               `json:"primary"`
	Domains              []string           `json:"domains"`
	SearchDomainsEnabled bool               `json:"search_domains_enabled"`
}

// validateNameserverGroup mirrors internal/handlers/dns.go's
// parseNameserverGroupForm. The htmx form collects up to 3 nameservers from
// fixed named slots because HTML forms need fixed field names; JSON has no
// such constraint, so this reads a plain array and enforces the same
// server-side cap (nbdns validateNSList) explicitly instead.
func validateNameserverGroup(req nameserverGroupRequest) (nbapi.NameserverGroupRequest, string) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nbapi.NameserverGroupRequest{}, "Name cannot be empty."
	}
	if utf8.RuneCountInString(name) > maxNameserverGroupNameChars {
		return nbapi.NameserverGroupRequest{}, "Name must be 40 characters or fewer."
	}

	if len(req.Nameservers) == 0 {
		return nbapi.NameserverGroupRequest{}, "At least one nameserver is required."
	}
	if len(req.Nameservers) > 3 {
		return nbapi.NameserverGroupRequest{}, "At most 3 nameservers are allowed."
	}
	nameservers := make([]nbapi.Nameserver, len(req.Nameservers))
	for i, ns := range req.Nameservers {
		if ns.Port < 1 || ns.Port > 65535 {
			return nbapi.NameserverGroupRequest{}, "Each nameserver's port must be a number between 1 and 65535."
		}
		nameservers[i] = nbapi.Nameserver{IP: ns.IP, NSType: "udp", Port: ns.Port}
	}

	if len(req.Groups) == 0 {
		return nbapi.NameserverGroupRequest{}, "Select at least one distribution group."
	}

	if req.Primary && len(req.Domains) > 0 {
		return nbapi.NameserverGroupRequest{}, "A primary nameserver group resolves every domain and cannot also list specific domains."
	}
	if !req.Primary && len(req.Domains) == 0 {
		return nbapi.NameserverGroupRequest{}, "Choose Primary, or list at least one domain for this group to resolve."
	}
	if req.Primary && req.SearchDomainsEnabled {
		return nbapi.NameserverGroupRequest{}, "Search domains cannot be enabled for a primary nameserver group."
	}

	return nbapi.NameserverGroupRequest{
		Name:                 name,
		Description:          strings.TrimSpace(req.Description),
		Nameservers:          nameservers,
		Enabled:              req.Enabled,
		Groups:               req.Groups,
		Primary:              req.Primary,
		Domains:              req.Domains,
		SearchDomainsEnabled: req.SearchDomainsEnabled,
	}, ""
}

func (s *Server) createNameserverGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req nameserverGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateNameserverGroup(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	created, err := s.api.CreateNameserverGroup(r.Context(), token, nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateNameserverGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req nameserverGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateNameserverGroup(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	updated, err := s.api.UpdateNameserverGroup(r.Context(), token, r.PathValue("id"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteNameserverGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteNameserverGroup(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getDNSSettings(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	settings, err := s.api.GetDNSSettings(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, settings)
}

func (s *Server) updateDNSSettings(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var settings nbapi.DNSSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if settings.DisabledManagementGroups == nil {
		settings.DisabledManagementGroups = []string{}
	}

	updated, err := s.api.UpdateDNSSettings(r.Context(), token, settings)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

type zonesResponse struct {
	Zones []nbapi.Zone `json:"zones"`
	Total int          `json:"total"`
}

func (s *Server) listZones(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	zones, err := s.api.ListZones(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, zonesResponse{Zones: zones, Total: len(zones)})
}

func (s *Server) getZone(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	zone, err := s.api.GetZone(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, zone)
}

type zoneRequest struct {
	Name               string   `json:"name"`
	Domain             string   `json:"domain"`
	Enabled            bool     `json:"enabled"`
	EnableSearchDomain bool     `json:"enable_search_domain"`
	DistributionGroups []string `json:"distribution_groups"`
}

// validateZone mirrors internal/handlers/dns.go's parseZoneForm. Domain
// format is not validated here beyond non-empty — the server's format and
// uniqueness checks are the authority, and their rejection surfaces through
// the normal error envelope like any other API error.
func validateZone(req zoneRequest) (nbapi.ZoneRequest, string) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nbapi.ZoneRequest{}, "Name cannot be empty."
	}
	if len(name) > 255 {
		return nbapi.ZoneRequest{}, "Name must be 255 characters or fewer."
	}
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return nbapi.ZoneRequest{}, "Domain cannot be empty."
	}
	if len(req.DistributionGroups) == 0 {
		return nbapi.ZoneRequest{}, "Select at least one distribution group."
	}
	return nbapi.ZoneRequest{
		Name:               name,
		Domain:             domain,
		Enabled:            req.Enabled,
		EnableSearchDomain: req.EnableSearchDomain,
		DistributionGroups: req.DistributionGroups,
	}, ""
}

func (s *Server) createZone(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req zoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateZone(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	created, err := s.api.CreateZone(r.Context(), token, nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

// updateZoneMeta carries the zone's existing Domain forward — the server
// rejects a write that changes it (see nbapi.Zone's doc comment), so the
// request's own Domain field, if different, would just be a wasted
// round trip to find that out; this fetches the current zone and always
// resends its real domain.
func (s *Server) updateZoneMeta(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	var req zoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	current, err := s.api.GetZone(r.Context(), token, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	req.Domain = current.Domain

	nbReq, msg := validateZone(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	updated, err := s.api.UpdateZone(r.Context(), token, id, nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteZone(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteZone(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type recordRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// validateRecord mirrors internal/handlers/dns.go's parseRecordForm.
// Content's format (IPv4/IPv6/domain, depending on Type) is not validated
// here beyond non-empty — the server validates it per type and that
// rejection surfaces through the normal error envelope.
func validateRecord(req recordRequest) (nbapi.DNSRecordRequest, string) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nbapi.DNSRecordRequest{}, "Name cannot be empty."
	}
	switch req.Type {
	case "A", "AAAA", "CNAME":
	default:
		return nbapi.DNSRecordRequest{}, "Type must be A, AAAA or CNAME."
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nbapi.DNSRecordRequest{}, "Content cannot be empty."
	}
	if req.TTL < 0 {
		return nbapi.DNSRecordRequest{}, "TTL must be a non-negative number."
	}
	return nbapi.DNSRecordRequest{Name: name, Type: req.Type, Content: content, TTL: req.TTL}, ""
}

func (s *Server) createRecord(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req recordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateRecord(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	created, err := s.api.CreateZoneRecord(r.Context(), token, r.PathValue("id"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateRecord(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req recordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateRecord(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	updated, err := s.api.UpdateZoneRecord(r.Context(), token, r.PathValue("id"), r.PathValue("recordID"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteRecord(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteZoneRecord(r.Context(), token, r.PathValue("id"), r.PathValue("recordID")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
