package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// LayerVersion represents a single version of a Lambda Layer.
type LayerVersion struct {
	LayerName     string   `json:"layerName"`
	Version       int64    `json:"version"`
	Description   string   `json:"description,omitempty"`
	CreatedDate   string   `json:"createdDate"`
	ARN           string   `json:"arn"`
	Runtimes      []string `json:"runtimes,omitempty"`
	Architectures []string `json:"architectures,omitempty"`
}

// Client wraps the AWS Lambda client.
type Client struct {
	lambda *lambda.Client
}

// NewClient creates a new AWS client using the default credential chain.
func NewClient(ctx context.Context, region string) (*Client, error) {
	opts := []func(*config.LoadOptions) error{}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &Client{lambda: lambda.NewFromConfig(cfg)}, nil
}

// ListAllLayerVersions returns all versions for every Lambda Layer in the account/region.
func (c *Client) ListAllLayerVersions(ctx context.Context) ([]LayerVersion, error) {
	var result []LayerVersion

	var layerMarker *string
	for {
		layersOut, err := c.lambda.ListLayers(ctx, &lambda.ListLayersInput{
			Marker: layerMarker,
		})
		if err != nil {
			return nil, fmt.Errorf("list layers: %w", err)
		}

		for _, layer := range layersOut.Layers {
			layerName := aws.ToString(layer.LayerName)

			var versionMarker *string
			for {
				versionsOut, err := c.lambda.ListLayerVersions(ctx, &lambda.ListLayerVersionsInput{
					LayerName: aws.String(layerName),
					Marker:    versionMarker,
				})
				if err != nil {
					return nil, fmt.Errorf("list layer versions for %s: %w", layerName, err)
				}
				for _, v := range versionsOut.LayerVersions {
					runtimes := make([]string, len(v.CompatibleRuntimes))
					for i, r := range v.CompatibleRuntimes {
						runtimes[i] = string(r)
					}
					archs := make([]string, len(v.CompatibleArchitectures))
					for i, a := range v.CompatibleArchitectures {
						archs[i] = string(a)
					}
					result = append(result, LayerVersion{
						LayerName:     layerName,
						Version:       v.Version,
						Description:   aws.ToString(v.Description),
						CreatedDate:   aws.ToString(v.CreatedDate),
						ARN:           aws.ToString(v.LayerVersionArn),
						Runtimes:      runtimes,
						Architectures: archs,
					})
				}
				if versionsOut.NextMarker == nil {
					break
				}
				versionMarker = versionsOut.NextMarker
			}
		}

		if layersOut.NextMarker == nil {
			break
		}
		layerMarker = layersOut.NextMarker
	}

	return result, nil
}

// ListFunctionLayerARNs returns the set of Layer Version ARNs currently attached to any function.
func (c *Client) ListFunctionLayerARNs(ctx context.Context) (map[string]struct{}, error) {
	used := make(map[string]struct{})

	var marker *string
	for {
		out, err := c.lambda.ListFunctions(ctx, &lambda.ListFunctionsInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("list functions: %w", err)
		}
		for _, fn := range out.Functions {
			for _, l := range fn.Layers {
				used[aws.ToString(l.Arn)] = struct{}{}
			}
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
	return used, nil
}

// DeleteLayerVersion deletes a specific version of a Lambda Layer.
func (c *Client) DeleteLayerVersion(ctx context.Context, layerName string, version int64) error {
	_, err := c.lambda.DeleteLayerVersion(ctx, &lambda.DeleteLayerVersionInput{
		LayerName:     aws.String(layerName),
		VersionNumber: &version,
	})
	if err != nil {
		return fmt.Errorf("delete layer %s v%d: %w", layerName, version, err)
	}
	return nil
}
