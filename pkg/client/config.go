package client

import (
	"context"
	"fmt"
	"strconv"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

func (c *Client) AddConfig(config *v1alpha1.K8sGPT) error {
	client := rpc.NewServerConfigServiceClient(c.Conn)
	req := &schemav1.AddConfigRequest{}
	// If multiple caches are configured we pick S3
	// which emulates the behaviour of K8sGPT cli
	if config.Spec.RemoteCache != nil {
		if config.Spec.RemoteCache.S3 != nil {
			// validate inputs
			if config.Spec.RemoteCache.S3.BucketName == "" {
				return fmt.Errorf("s3 bucket name is required")
			}
			if config.Spec.RemoteCache.S3.Region == "" {
				return fmt.Errorf("s3 region is required")
			}
			req.Cache = &schemav1.Cache{
				CacheType: &schemav1.Cache_S3Cache{
					S3Cache: &schemav1.S3Cache{
						BucketName: config.Spec.RemoteCache.S3.BucketName,
						Region:     config.Spec.RemoteCache.S3.Region,
					},
				},
			}
		} else if config.Spec.RemoteCache.Azure != nil {
			if config.Spec.RemoteCache.Azure.StorageAccount == "" {
				return fmt.Errorf("azure storage account is required")
			}
			if config.Spec.RemoteCache.Azure.ContainerName == "" {
				return fmt.Errorf("azure container name is required")
			}
			req.Cache = &schemav1.Cache{
				CacheType: &schemav1.Cache_AzureCache{
					AzureCache: &schemav1.AzureCache{
						StorageAccount: config.Spec.RemoteCache.Azure.StorageAccount,
						ContainerName:  config.Spec.RemoteCache.Azure.ContainerName,
					},
				},
			}
		} else if config.Spec.RemoteCache.GCS != nil {
			if config.Spec.RemoteCache.GCS.BucketName == "" {
				return fmt.Errorf("gcs bucket name is required")
			}
			if config.Spec.RemoteCache.GCS.Region == "" {
				return fmt.Errorf("gcs region is required")
			}
			if config.Spec.RemoteCache.GCS.ProjectId == "" {
				return fmt.Errorf("gcs project id is required")
			}
			req.Cache = &schemav1.Cache{
				CacheType: &schemav1.Cache_GcsCache{
					GcsCache: &schemav1.GCSCache{
						BucketName: config.Spec.RemoteCache.GCS.BucketName,
						Region:     config.Spec.RemoteCache.GCS.Region,
						ProjectId:  config.Spec.RemoteCache.GCS.ProjectId,
					},
				},
			}
		} else if config.Spec.RemoteCache.Interplex != nil {
			if config.Spec.RemoteCache.Interplex.Endpoint == "" {
				return fmt.Errorf("interplex endpoint is required")
			}
			req.Cache = &schemav1.Cache{
				CacheType: &schemav1.Cache_InterplexCache{
					InterplexCache: &schemav1.InterplexCache{
						Endpoint: config.Spec.RemoteCache.Interplex.Endpoint,
					},
				},
			}
		}
	}
	if config.Spec.CustomAnalyzers != nil {
		for _, customAnalyzer := range config.Spec.CustomAnalyzers {
			req.CustomAnalyzers = append(req.CustomAnalyzers, &schemav1.CustomAnalyzer{
				Name: customAnalyzer.Name,
				Connection: &schemav1.Connection{
					Url:  customAnalyzer.Connection.Url,
					Port: strconv.Itoa(customAnalyzer.Connection.Port),
				},
			})
		}
	}
	_, err := client.AddConfig(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to call AddConfig RPC: %v", err)
	}

	return nil
}

func (c *Client) RemoveConfig(config *v1alpha1.K8sGPT) error {
	client := rpc.NewServerConfigServiceClient(c.Conn)
	req := &schemav1.RemoveConfigRequest{
		Cache:           &schemav1.Cache{},
		Integrations:    nil,
		CustomAnalyzers: make([]*schemav1.CustomAnalyzer, 0),
	}

	_, err := client.RemoveConfig(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to call RemoveConfig RPC: %v", err)
	}

	return nil
}
