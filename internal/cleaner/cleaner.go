package cleaner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	awsclient "github.com/miztch/llrm/internal/aws"
)

// lambdaClient is the interface Cleaner requires from the AWS client.
type lambdaClient interface {
	ListAllLayerVersions(ctx context.Context) ([]awsclient.LayerVersion, error)
	ListFunctionLayerARNs(ctx context.Context) (map[string]struct{}, error)
	DeleteLayerVersion(ctx context.Context, layerName string, version int64) error
}

// Options holds the filtering parameters for cleanup.
type Options struct {
	KeepVersions int
	Name         string
	Filter       string
}

// Candidate is a layer version selected for deletion.
type Candidate struct {
	awsclient.LayerVersion
	Reasons []string `json:"reasons,omitempty"`
}

// Cleaner orchestrates the scan and delete workflow.
type Cleaner struct {
	client lambdaClient
	opts   Options
}

// New creates a Cleaner with the given client and options.
func New(client *awsclient.Client, opts Options) *Cleaner {
	return &Cleaner{client: client, opts: opts}
}

func (c *Cleaner) applyFilters(versions []awsclient.LayerVersion) []awsclient.LayerVersion {
	if c.opts.Name != "" {
		filtered := versions[:0]
		for _, v := range versions {
			if v.LayerName == c.opts.Name {
				filtered = append(filtered, v)
			}
		}
		versions = filtered
	}
	if c.opts.Filter != "" {
		filtered := versions[:0]
		for _, v := range versions {
			if strings.Contains(v.LayerName, c.opts.Filter) {
				filtered = append(filtered, v)
			}
		}
		versions = filtered
	}
	return versions
}

// Scan returns unattached layer versions that match the cleanup criteria.
func (c *Cleaner) Scan(ctx context.Context) ([]Candidate, error) {
	allVersions, err := c.client.ListAllLayerVersions(ctx)
	if err != nil {
		return nil, err
	}
	allVersions = c.applyFilters(allVersions)

	usedARNs, err := c.client.ListFunctionLayerARNs(ctx)
	if err != nil {
		return nil, err
	}

	byLayer := make(map[string][]awsclient.LayerVersion)
	for _, v := range allVersions {
		byLayer[v.LayerName] = append(byLayer[v.LayerName], v)
	}
	for name := range byLayer {
		versions := byLayer[name]
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Version > versions[j].Version
		})
		byLayer[name] = versions
	}

	candidates := []Candidate{}
	for _, versions := range byLayer {
		for idx, v := range versions {
			if _, inUse := usedARNs[v.ARN]; inUse {
				continue
			}
			if c.opts.KeepVersions > 0 && idx < c.opts.KeepVersions {
				continue
			}
			var reasons []string
			if c.opts.KeepVersions > 0 {
				reasons = append(reasons, fmt.Sprintf("older than keep-versions=%d", c.opts.KeepVersions))
			}
			candidates = append(candidates, Candidate{LayerVersion: v, Reasons: reasons})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].LayerName != candidates[j].LayerName {
			return candidates[i].LayerName < candidates[j].LayerName
		}
		return candidates[i].Version > candidates[j].Version
	})

	return candidates, nil
}

// VersionStatus is a layer version with its attachment status.
type VersionStatus struct {
	awsclient.LayerVersion
	Attached bool `json:"attached"`
}

// ListAll returns all layer versions with their attachment status.
func (c *Cleaner) ListAll(ctx context.Context) ([]VersionStatus, error) {
	allVersions, err := c.client.ListAllLayerVersions(ctx)
	if err != nil {
		return nil, err
	}

	allVersions = c.applyFilters(allVersions)

	usedARNs, err := c.client.ListFunctionLayerARNs(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]VersionStatus, len(allVersions))
	for i, v := range allVersions {
		_, attached := usedARNs[v.ARN]
		result[i] = VersionStatus{LayerVersion: v, Attached: attached}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LayerName != result[j].LayerName {
			return result[i].LayerName < result[j].LayerName
		}
		return result[i].Version > result[j].Version
	})
	return result, nil
}

// Delete removes the given candidates, calling onResult after each attempt.
func (c *Cleaner) Delete(ctx context.Context, targets []Candidate, onResult func(Candidate, error)) {
	for _, t := range targets {
		onResult(t, c.client.DeleteLayerVersion(ctx, t.LayerName, t.Version))
	}
}
