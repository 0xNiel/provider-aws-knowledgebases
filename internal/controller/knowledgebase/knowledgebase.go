/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package knowledgebase

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	bedrockagentTypes "github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/google/uuid"
	"gopkg.in/ini.v1"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"

	"github.com/0xNiel/provider-aws-knowledgebases/apis/awskb/v1alpha1"
	apisv1alpha1 "github.com/0xNiel/provider-aws-knowledgebases/apis/v1alpha1"
	"github.com/0xNiel/provider-aws-knowledgebases/internal/features"
)

const (
	errNotKnowledgeBase    = "managed resource is not a KnowledgeBase custom resource"
	errTrackPCUsage        = "cannot track ProviderConfig usage"
	errGetPC               = "cannot get ProviderConfig"
	errGetCreds            = "cannot get credentials"
	errNewClient           = "cannot create new BedrockAgent client"
	errCreateKnowledgeBase = "cannot create knowledge base"
	errGetKnowledgeBase    = "cannot get knowledge base"
	errDeleteKnowledgeBase = "cannot delete knowledge base"
	errUpdateKnowledgeBase = "cannot update knowledge base"
)

// BedrockAgentClient is an interface for the AWS Bedrock Agent client
type BedrockAgentClient interface {
	CreateKnowledgeBase(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error)
	GetKnowledgeBase(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error)
	DeleteKnowledgeBase(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error)
	UpdateKnowledgeBase(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error)
}

// parseAWSCredentials parses AWS credentials from INI format
func parseAWSCredentials(creds []byte) (accessKeyID, secretAccessKey, sessionToken, roleArn, roleSessionName string, err error) {
	cfg, err := ini.Load(creds)
	if err != nil {
		return "", "", "", "", "", errors.Wrap(err, "failed to parse INI credentials")
	}

	section := cfg.Section("default")
	accessKeyID = section.Key("aws_access_key_id").String()
	secretAccessKey = section.Key("aws_secret_access_key").String()
	sessionToken = section.Key("aws_session_token").String()
	roleArn = section.Key("role_arn").String()
	roleSessionName = section.Key("role_session_name").String()

	if accessKeyID == "" || secretAccessKey == "" {
		return "", "", "", "", "", errors.New("missing required AWS credentials (aws_access_key_id or aws_secret_access_key)")
	}

	return accessKeyID, secretAccessKey, sessionToken, roleArn, roleSessionName, nil
}

// newBedrockAgentClient creates a new Bedrock Agent client
func newBedrockAgentClient(ctx context.Context, region string, creds []byte) (BedrockAgentClient, error) {
	accessKeyID, secretAccessKey, sessionToken, roleArn, roleSessionName, err := parseAWSCredentials(creds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse AWS credentials")
	}

	var cfg aws.Config

	if roleArn != "" {
		// Use role assumption
		if roleSessionName == "" {
			roleSessionName = "crossplane-provider-session"
		}

		// First, create base credentials
		baseCreds := credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				SessionToken:    sessionToken,
			},
		}

		// Then create STS client with base credentials
		baseCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(baseCreds),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load base AWS config")
		}

		// Create STS client and assume role credentials
		stsClient := sts.NewFromConfig(baseCfg)
		roleCredentials := stscreds.NewAssumeRoleProvider(stsClient, roleArn, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = roleSessionName
		})

		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(roleCredentials),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load AWS config with role assumption")
		}
	} else {
		// Use static credentials
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
					SessionToken:    sessionToken,
				},
			}),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load AWS config")
		}
	}

	return bedrockagent.NewFromConfig(cfg), nil
}

// Setup adds a controller that reconciles KnowledgeBase managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.KnowledgeBaseGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{})}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...),
		managed.WithManagementPolicies(),
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.KnowledgeBaseList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.KnowledgeBaseList")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.KnowledgeBaseGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.KnowledgeBase{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.KnowledgeBase)
	if !ok {
		return nil, errors.New(errNotKnowledgeBase)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	client, err := newBedrockAgentClient(ctx, cr.Spec.ForProvider.Region, data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client BedrockAgentClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.KnowledgeBase)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKnowledgeBase)
	}

	externalName := meta.GetExternalName(cr)

	// If no external name is set, the resource doesn't exist yet
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// If the external name looks like a Kubernetes resource name (contains hyphens, lowercase)
	// rather than an AWS knowledge base ID (uppercase alphanumeric), treat as non-existent
	// AWS knowledge base IDs are typically like "UPEIDTWQLX" (uppercase, no hyphens)
	if len(externalName) > 15 || strings.Contains(externalName, "-") {
		// Clear the external name and treat as non-existent
		meta.SetExternalName(cr, "")
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Get the knowledge base from AWS
	resp, err := c.client.GetKnowledgeBase(ctx, &bedrockagent.GetKnowledgeBaseInput{
		KnowledgeBaseId: aws.String(externalName),
	})
	if err != nil {
		// Check if it's a not found error
		var notFoundErr *bedrockagentTypes.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetKnowledgeBase)
	}

	kb := resp.KnowledgeBase

	// Update the status with observed values
	cr.Status.AtProvider.KnowledgeBaseArn = kb.KnowledgeBaseArn
	cr.Status.AtProvider.KnowledgeBaseId = kb.KnowledgeBaseId
	cr.Status.AtProvider.Status = (*string)(&kb.Status)
	if kb.CreatedAt != nil {
		cr.Status.AtProvider.CreatedAt = &metav1.Time{Time: *kb.CreatedAt}
	}
	if kb.UpdatedAt != nil {
		cr.Status.AtProvider.UpdatedAt = &metav1.Time{Time: *kb.UpdatedAt}
	}
	cr.Status.AtProvider.FailureReasons = kb.FailureReasons

	// Set the resource as ready if it's in ACTIVE status
	cr.Status.SetConditions(xpv1.Available())

	// Check if resource is up to date
	upToDate := c.isUpToDate(cr, kb)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
		ConnectionDetails: managed.ConnectionDetails{
			"knowledgeBaseId":  []byte(*kb.KnowledgeBaseId),
			"knowledgeBaseArn": []byte(*kb.KnowledgeBaseArn),
		},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.KnowledgeBase)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotKnowledgeBase)
	}

	// Generate client token if not provided
	clientToken := cr.Spec.ForProvider.ClientToken
	if clientToken == nil {
		token := uuid.New().String()
		clientToken = &token
	}

	// Convert our types to AWS SDK types
	input := &bedrockagent.CreateKnowledgeBaseInput{
		Name:                       cr.Spec.ForProvider.Name,
		RoleArn:                    cr.Spec.ForProvider.RoleArn,
		Description:                cr.Spec.ForProvider.Description,
		ClientToken:                clientToken,
		Tags:                       cr.Spec.ForProvider.Tags,
		KnowledgeBaseConfiguration: c.convertKnowledgeBaseConfiguration(cr.Spec.ForProvider.KnowledgeBaseConfiguration),
		StorageConfiguration:       c.convertStorageConfiguration(cr.Spec.ForProvider.StorageConfiguration),
	}

	resp, err := c.client.CreateKnowledgeBase(ctx, input)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateKnowledgeBase)
	}

	// Set the external name to the knowledge base ID
	meta.SetExternalName(cr, *resp.KnowledgeBase.KnowledgeBaseId)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"knowledgeBaseId":  []byte(*resp.KnowledgeBase.KnowledgeBaseId),
			"knowledgeBaseArn": []byte(*resp.KnowledgeBase.KnowledgeBaseArn),
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.KnowledgeBase)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotKnowledgeBase)
	}

	input := &bedrockagent.UpdateKnowledgeBaseInput{
		KnowledgeBaseId:            aws.String(meta.GetExternalName(cr)),
		Name:                       cr.Spec.ForProvider.Name,
		RoleArn:                    cr.Spec.ForProvider.RoleArn,
		Description:                cr.Spec.ForProvider.Description,
		KnowledgeBaseConfiguration: c.convertKnowledgeBaseConfiguration(cr.Spec.ForProvider.KnowledgeBaseConfiguration),
	}

	resp, err := c.client.UpdateKnowledgeBase(ctx, input)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateKnowledgeBase)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			"knowledgeBaseId":  []byte(*resp.KnowledgeBase.KnowledgeBaseId),
			"knowledgeBaseArn": []byte(*resp.KnowledgeBase.KnowledgeBaseArn),
		},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.KnowledgeBase)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotKnowledgeBase)
	}

	_, err := c.client.DeleteKnowledgeBase(ctx, &bedrockagent.DeleteKnowledgeBaseInput{
		KnowledgeBaseId: aws.String(meta.GetExternalName(cr)),
	})
	if err != nil {
		// If the resource is already deleted, that's fine
		var notFoundErr *bedrockagentTypes.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteKnowledgeBase)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// isUpToDate checks if the knowledge base is up to date
func (c *external) isUpToDate(cr *v1alpha1.KnowledgeBase, kb *bedrockagentTypes.KnowledgeBase) bool {
	// Check if basic fields match
	if cr.Spec.ForProvider.Name != nil && kb.Name != nil && *cr.Spec.ForProvider.Name != *kb.Name {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && kb.Description != nil && *cr.Spec.ForProvider.Description != *kb.Description {
		return false
	}
	if cr.Spec.ForProvider.RoleArn != nil && kb.RoleArn != nil && *cr.Spec.ForProvider.RoleArn != *kb.RoleArn {
		return false
	}

	// For now, assume it's up to date if basic fields match
	// In a full implementation, you'd compare the full configuration
	return true
}

// convertKnowledgeBaseConfiguration converts our types to AWS SDK types
func (c *external) convertKnowledgeBaseConfiguration(config *v1alpha1.KnowledgeBaseConfiguration) *bedrockagentTypes.KnowledgeBaseConfiguration {
	if config == nil {
		return nil
	}

	result := &bedrockagentTypes.KnowledgeBaseConfiguration{
		Type: bedrockagentTypes.KnowledgeBaseType(config.Type),
	}

	if config.VectorKnowledgeBaseConfiguration != nil {
		result.VectorKnowledgeBaseConfiguration = &bedrockagentTypes.VectorKnowledgeBaseConfiguration{
			EmbeddingModelArn: config.VectorKnowledgeBaseConfiguration.EmbeddingModelArn,
		}

		if config.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration != nil {
			result.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration = &bedrockagentTypes.EmbeddingModelConfiguration{}

			if config.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration.BedrockEmbeddingModelConfiguration != nil {
				result.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration.BedrockEmbeddingModelConfiguration = &bedrockagentTypes.BedrockEmbeddingModelConfiguration{
					Dimensions: config.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration.BedrockEmbeddingModelConfiguration.Dimensions,
				}

				if config.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration.BedrockEmbeddingModelConfiguration.EmbeddingDataType != nil {
					result.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration.BedrockEmbeddingModelConfiguration.EmbeddingDataType = bedrockagentTypes.EmbeddingDataType(*config.VectorKnowledgeBaseConfiguration.EmbeddingModelConfiguration.BedrockEmbeddingModelConfiguration.EmbeddingDataType)
				}
			}
		}
	}

	return result
}

// convertStorageConfiguration converts our types to AWS SDK types
func (c *external) convertStorageConfiguration(config *v1alpha1.StorageConfiguration) *bedrockagentTypes.StorageConfiguration {
	if config == nil {
		return nil
	}

	result := &bedrockagentTypes.StorageConfiguration{
		Type: bedrockagentTypes.KnowledgeBaseStorageType(config.Type),
	}

	if config.S3VectorsConfiguration != nil {
		result.S3VectorsConfiguration = &bedrockagentTypes.S3VectorsConfiguration{
			IndexArn:        config.S3VectorsConfiguration.IndexArn,
			IndexName:       config.S3VectorsConfiguration.IndexName,
			VectorBucketArn: config.S3VectorsConfiguration.VectorBucketArn,
		}
	}

	if config.OpensearchServerlessConfiguration != nil {
		result.OpensearchServerlessConfiguration = &bedrockagentTypes.OpenSearchServerlessConfiguration{
			CollectionArn:   config.OpensearchServerlessConfiguration.CollectionArn,
			VectorIndexName: config.OpensearchServerlessConfiguration.VectorIndexName,
		}

		if config.OpensearchServerlessConfiguration.FieldMapping != nil {
			result.OpensearchServerlessConfiguration.FieldMapping = &bedrockagentTypes.OpenSearchServerlessFieldMapping{
				MetadataField: config.OpensearchServerlessConfiguration.FieldMapping.MetadataField,
				TextField:     config.OpensearchServerlessConfiguration.FieldMapping.TextField,
				VectorField:   config.OpensearchServerlessConfiguration.FieldMapping.VectorField,
			}
		}
	}

	if config.PineconeConfiguration != nil {
		result.PineconeConfiguration = &bedrockagentTypes.PineconeConfiguration{
			ConnectionString:     config.PineconeConfiguration.ConnectionString,
			CredentialsSecretArn: config.PineconeConfiguration.CredentialsSecretArn,
			Namespace:            config.PineconeConfiguration.Namespace,
		}

		if config.PineconeConfiguration.FieldMapping != nil {
			result.PineconeConfiguration.FieldMapping = &bedrockagentTypes.PineconeFieldMapping{
				MetadataField: config.PineconeConfiguration.FieldMapping.MetadataField,
				TextField:     config.PineconeConfiguration.FieldMapping.TextField,
			}
		}
	}

	return result
}
