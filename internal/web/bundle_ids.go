package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// BundleIDCapabilityOption describes one option inside a capability setting.
type BundleIDCapabilityOption struct {
	Key              string `json:"key"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Enabled          *bool  `json:"enabled,omitempty"`
	EnabledByDefault *bool  `json:"enabledByDefault,omitempty"`
	SupportsWildcard *bool  `json:"supportsWildcard,omitempty"`
}

// BundleIDCapabilitySetting describes an App Store Connect bundle ID capability setting.
type BundleIDCapabilitySetting struct {
	Key              string                     `json:"key"`
	Name             string                     `json:"name,omitempty"`
	Description      string                     `json:"description,omitempty"`
	EnabledByDefault *bool                      `json:"enabledByDefault,omitempty"`
	Visible          *bool                      `json:"visible,omitempty"`
	AllowedInstances string                     `json:"allowedInstances,omitempty"`
	MinInstances     *int                       `json:"minInstances,omitempty"`
	Options          []BundleIDCapabilityOption `json:"options,omitempty"`
}

// AppClipBundleIDCapabilitySyncRequest updates an App Clip Bundle ID capability set
// through Apple's web-session bundleIds patch payload.
type AppClipBundleIDCapabilitySyncRequest struct {
	BundleID       string
	ParentBundleID string
	Capability     string
	Enabled        bool
	Settings       []BundleIDCapabilitySetting
}

// AppClipBundleIDCapabilitySyncResult summarizes the private capability sync.
type AppClipBundleIDCapabilitySyncResult struct {
	BundleID       string `json:"bundleId"`
	ParentBundleID string `json:"parentBundleId"`
	Capability     string `json:"capability"`
	Enabled        bool   `json:"enabled"`
}

type webBundleIDResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name       string `json:"name"`
			Identifier string `json:"identifier"`
			SeedID     string `json:"seedId,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
}

type webBundleIDPatchRequest struct {
	Data struct {
		ID            string `json:"id"`
		Type          string `json:"type"`
		Attributes    any    `json:"attributes"`
		Relationships struct {
			BundleIDCapabilities struct {
				Data []webBundleIDCapabilityRelationship `json:"data"`
			} `json:"bundleIdCapabilities"`
		} `json:"relationships"`
	} `json:"data"`
}

type webBundleIDCapabilityRelationship struct {
	Type          string `json:"type"`
	Attributes    any    `json:"attributes"`
	Relationships struct {
		Capability struct {
			Data relationshipData `json:"data"`
		} `json:"capability"`
		ParentBundleID struct {
			Data relationshipData `json:"data"`
		} `json:"parentBundleId"`
	} `json:"relationships"`
}

func normalizeAppClipBundleIDCapabilitySyncRequest(req AppClipBundleIDCapabilitySyncRequest) (AppClipBundleIDCapabilitySyncRequest, error) {
	req.BundleID = strings.TrimSpace(req.BundleID)
	req.ParentBundleID = strings.TrimSpace(req.ParentBundleID)
	req.Capability = strings.ToUpper(strings.TrimSpace(req.Capability))
	if req.BundleID == "" {
		return req, fmt.Errorf("bundle id is required")
	}
	if req.ParentBundleID == "" {
		return req, fmt.Errorf("parent bundle id is required")
	}
	if req.Capability == "" {
		return req, fmt.Errorf("capability is required")
	}
	return req, nil
}

// SyncAppClipBundleIDCapability patches a bundle ID capability relationship with
// the parentBundleId relationship required by App Clip targets.
func (c *Client) SyncAppClipBundleIDCapability(ctx context.Context, req AppClipBundleIDCapabilitySyncRequest) (*AppClipBundleIDCapabilitySyncResult, error) {
	req, err := normalizeAppClipBundleIDCapabilitySyncRequest(req)
	if err != nil {
		return nil, err
	}

	body, err := c.doIrisV1Request(ctx, http.MethodGet, fmt.Sprintf("/bundleIds/%s?include=bundleIdCapabilities", req.BundleID), nil)
	if err != nil {
		return nil, err
	}
	var current webBundleIDResponse
	if err := json.Unmarshal(body, &current); err != nil {
		return nil, fmt.Errorf("failed to parse bundle id response: %w", err)
	}

	payload := buildAppClipBundleIDCapabilityPatchRequest(current, req)
	if _, err := c.doIrisV1Request(ctx, http.MethodPatch, fmt.Sprintf("/bundleIds/%s", req.BundleID), payload); err != nil {
		return nil, err
	}

	return &AppClipBundleIDCapabilitySyncResult{
		BundleID:       req.BundleID,
		ParentBundleID: req.ParentBundleID,
		Capability:     req.Capability,
		Enabled:        req.Enabled,
	}, nil
}

func buildAppClipBundleIDCapabilityPatchRequest(current webBundleIDResponse, req AppClipBundleIDCapabilitySyncRequest) webBundleIDPatchRequest {
	var payload webBundleIDPatchRequest
	payload.Data.ID = req.BundleID
	payload.Data.Type = "bundleIds"
	payload.Data.Attributes = struct {
		Name       string `json:"name"`
		Identifier string `json:"identifier"`
		SeedID     string `json:"seedId,omitempty"`
	}{
		Name:       current.Data.Attributes.Name,
		Identifier: current.Data.Attributes.Identifier,
		SeedID:     current.Data.Attributes.SeedID,
	}

	capability := webBundleIDCapabilityRelationship{
		Type: "bundleIdCapabilities",
		Attributes: struct {
			Enabled  bool                        `json:"enabled"`
			Settings []BundleIDCapabilitySetting `json:"settings"`
		}{
			Enabled:  req.Enabled,
			Settings: req.Settings,
		},
	}
	capability.Relationships.Capability.Data = relationshipData{Type: "capabilities", ID: req.Capability}
	capability.Relationships.ParentBundleID.Data = relationshipData{Type: "bundleIds", ID: req.ParentBundleID}
	payload.Data.Relationships.BundleIDCapabilities.Data = []webBundleIDCapabilityRelationship{capability}
	return payload
}
