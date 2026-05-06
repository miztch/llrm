package cleaner

import (
	"context"
	"testing"

	awsclient "github.com/miztch/llrm/internal/aws"
)

type stubClient struct {
	versions []awsclient.LayerVersion
	usedARNs map[string]struct{}
}

func (s *stubClient) ListAllLayerVersions(_ context.Context) ([]awsclient.LayerVersion, error) {
	return s.versions, nil
}

func (s *stubClient) ListFunctionLayerARNs(_ context.Context) (map[string]struct{}, error) {
	return s.usedARNs, nil
}

func (s *stubClient) DeleteLayerVersion(_ context.Context, _ string, _ int64) error {
	return nil
}

func lv(name string, version int64, arn string) awsclient.LayerVersion {
	return awsclient.LayerVersion{LayerName: name, Version: version, ARN: arn}
}

func TestScan_UnattachedReturned(t *testing.T) {
	client := &stubClient{
		versions: []awsclient.LayerVersion{
			lv("my-layer", 1, "arn:1"),
			lv("my-layer", 2, "arn:2"),
		},
		usedARNs: map[string]struct{}{
			"arn:2": {},
		},
	}
	c := &Cleaner{client: client, opts: Options{}}
	candidates, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].ARN != "arn:1" {
		t.Errorf("got ARN %s, want arn:1", candidates[0].ARN)
	}
}

func TestScan_KeepVersions(t *testing.T) {
	client := &stubClient{
		versions: []awsclient.LayerVersion{
			lv("my-layer", 1, "arn:1"),
			lv("my-layer", 2, "arn:2"),
			lv("my-layer", 3, "arn:3"),
		},
		usedARNs: map[string]struct{}{},
	}
	c := &Cleaner{client: client, opts: Options{KeepVersions: 2}}
	candidates, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// keep versions 3 and 2, delete version 1
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].Version != 1 {
		t.Errorf("got version %d, want 1", candidates[0].Version)
	}
}

func TestScan_FilterByName(t *testing.T) {
	client := &stubClient{
		versions: []awsclient.LayerVersion{
			lv("layer-a", 1, "arn:a1"),
			lv("layer-b", 1, "arn:b1"),
		},
		usedARNs: map[string]struct{}{},
	}
	c := &Cleaner{client: client, opts: Options{Name: "layer-a"}}
	candidates, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].LayerName != "layer-a" {
		t.Errorf("got layer %s, want layer-a", candidates[0].LayerName)
	}
}

func TestScan_FilterSubstring(t *testing.T) {
	client := &stubClient{
		versions: []awsclient.LayerVersion{
			lv("myapp-utils", 1, "arn:1"),
			lv("other-layer", 1, "arn:2"),
		},
		usedARNs: map[string]struct{}{},
	}
	c := &Cleaner{client: client, opts: Options{Filter: "myapp"}}
	candidates, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].LayerName != "myapp-utils" {
		t.Errorf("unexpected candidates: %v", candidates)
	}
}

func TestScan_NoCandidatesWhenAllAttached(t *testing.T) {
	client := &stubClient{
		versions: []awsclient.LayerVersion{
			lv("my-layer", 1, "arn:1"),
		},
		usedARNs: map[string]struct{}{
			"arn:1": {},
		},
	}
	c := &Cleaner{client: client, opts: Options{}}
	candidates, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Errorf("got %d candidates, want 0", len(candidates))
	}
}
