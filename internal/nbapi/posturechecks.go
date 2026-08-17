package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// MinVersionCheck and MinKernelVersionCheck are the two version-comparison
// shapes the spec reuses across NB/OS checks — a plain semver-ish string for
// most platforms, a kernel/build string for Linux and Windows.
type MinVersionCheck struct {
	MinVersion string `json:"min_version"`
}

type MinKernelVersionCheck struct {
	MinKernelVersion string `json:"min_kernel_version"`
}

// OSVersionCheck is per-platform, and every field is optional: a posture
// check may only care about, say, Linux kernel version and leave the rest
// unset.
type OSVersionCheck struct {
	Android *MinVersionCheck       `json:"android,omitempty"`
	Darwin  *MinVersionCheck       `json:"darwin,omitempty"`
	IOS     *MinVersionCheck       `json:"ios,omitempty"`
	Linux   *MinKernelVersionCheck `json:"linux,omitempty"`
	Windows *MinKernelVersionCheck `json:"windows,omitempty"`
}

// Location is a country (required) plus an optional city, used by
// GeoLocationCheck.
type Location struct {
	CountryCode string `json:"country_code"`
	CityName    string `json:"city_name,omitempty"`
}

// GeoLocationCheck allows or denies peers based on their resolved location.
type GeoLocationCheck struct {
	Locations []Location `json:"locations"`
	Action    string     `json:"action"` // "allow" or "deny"
}

// PeerNetworkRangeCheck allows or denies peers whose local or public IP
// falls inside any of Ranges (CIDR notation).
type PeerNetworkRangeCheck struct {
	Ranges []string `json:"ranges"`
	Action string   `json:"action"` // "allow" or "deny"
}

// Process names a required executable by its per-platform path. At least
// one path is expected to be set, but the spec doesn't require all three.
type Process struct {
	LinuxPath   string `json:"linux_path,omitempty"`
	MacPath     string `json:"mac_path,omitempty"`
	WindowsPath string `json:"windows_path,omitempty"`
}

type ProcessCheck struct {
	Processes []Process `json:"processes"`
}

// Checks is the set of individual checks a PostureCheck runs — every field
// is optional (a posture check might only set one or two), which is why
// they're all pointers/nil-able here rather than zero-valued structs: an
// absent nb_version_check must round-trip as absent, not as a
// min_version:"" that the server would reject.
type Checks struct {
	NBVersionCheck        *MinVersionCheck       `json:"nb_version_check,omitempty"`
	OSVersionCheck        *OSVersionCheck        `json:"os_version_check,omitempty"`
	GeoLocationCheck      *GeoLocationCheck      `json:"geo_location_check,omitempty"`
	PeerNetworkRangeCheck *PeerNetworkRangeCheck `json:"peer_network_range_check,omitempty"`
	ProcessCheck          *ProcessCheck          `json:"process_check,omitempty"`
}

type PostureCheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Checks      Checks `json:"checks"`
}

type PostureCheckRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Checks      Checks `json:"checks"`
}

func (c *Client) ListPostureChecks(ctx context.Context, token string) ([]PostureCheck, error) {
	var checks []PostureCheck
	if err := c.do(ctx, token, http.MethodGet, "/posture-checks", nil, nil, &checks); err != nil {
		return nil, err
	}
	sort.SliceStable(checks, func(i, j int) bool {
		return strings.ToLower(checks[i].Name) < strings.ToLower(checks[j].Name)
	})
	return checks, nil
}

func (c *Client) GetPostureCheck(ctx context.Context, token, id string) (*PostureCheck, error) {
	var check PostureCheck
	if err := c.do(ctx, token, http.MethodGet, "/posture-checks/"+url.PathEscape(id), nil, nil, &check); err != nil {
		return nil, err
	}
	return &check, nil
}

func (c *Client) CreatePostureCheck(ctx context.Context, token string, req PostureCheckRequest) (*PostureCheck, error) {
	var check PostureCheck
	if err := c.do(ctx, token, http.MethodPost, "/posture-checks", nil, req, &check); err != nil {
		return nil, err
	}
	return &check, nil
}

func (c *Client) UpdatePostureCheck(ctx context.Context, token, id string, req PostureCheckRequest) (*PostureCheck, error) {
	var check PostureCheck
	if err := c.do(ctx, token, http.MethodPut, "/posture-checks/"+url.PathEscape(id), nil, req, &check); err != nil {
		return nil, err
	}
	return &check, nil
}

func (c *Client) DeletePostureCheck(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/posture-checks/"+url.PathEscape(id), nil, nil, nil)
}
