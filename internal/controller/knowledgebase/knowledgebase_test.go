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
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	bedrockagentTypes "github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/0xNiel/provider-aws-knowledgebases/apis/awskb/v1alpha1"
)

// MockBedrockAgentClient is a mock implementation of the BedrockAgentClient interface
type MockBedrockAgentClient struct {
	MockCreateKnowledgeBase func(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error)
	MockGetKnowledgeBase    func(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error)
	MockDeleteKnowledgeBase func(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error)
	MockUpdateKnowledgeBase func(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error)
}

func (m *MockBedrockAgentClient) CreateKnowledgeBase(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error) {
	return m.MockCreateKnowledgeBase(ctx, params, optFns...)
}

func (m *MockBedrockAgentClient) GetKnowledgeBase(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error) {
	return m.MockGetKnowledgeBase(ctx, params, optFns...)
}

func (m *MockBedrockAgentClient) DeleteKnowledgeBase(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error) {
	return m.MockDeleteKnowledgeBase(ctx, params, optFns...)
}

func (m *MockBedrockAgentClient) UpdateKnowledgeBase(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error) {
	return m.MockUpdateKnowledgeBase(ctx, params, optFns...)
}

// Helper functions for creating test resources
func knowledgeBase(m ...func(*v1alpha1.KnowledgeBase)) *v1alpha1.KnowledgeBase {
	cr := &v1alpha1.KnowledgeBase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KnowledgeBase",
			APIVersion: "awskb.template.crossplane.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-kb",
		},
		Spec: v1alpha1.KnowledgeBaseSpec{
			ForProvider: v1alpha1.KnowledgeBaseParameters{
				Region:  "us-east-1",
				Name:    aws.String("test-knowledge-base"),
				RoleArn: aws.String("arn:aws:iam::123456789012:role/test-role"),
				KnowledgeBaseConfiguration: &v1alpha1.KnowledgeBaseConfiguration{
					Type: "VECTOR",
					VectorKnowledgeBaseConfiguration: &v1alpha1.VectorKnowledgeBaseConfiguration{
						EmbeddingModelArn: aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1"),
					},
				},
				StorageConfiguration: &v1alpha1.StorageConfiguration{
					Type: "S3_VECTORS",
					S3VectorsConfiguration: &v1alpha1.S3VectorsConfiguration{
						IndexArn:        aws.String("arn:aws:bedrock:us-east-1:123456789012:index/test-index"),
						VectorBucketArn: aws.String("arn:aws:s3:::test-bucket"),
					},
				},
			},
		},
	}

	for _, f := range m {
		f(cr)
	}

	return cr
}

func withExternalName(name string) func(*v1alpha1.KnowledgeBase) {
	return func(cr *v1alpha1.KnowledgeBase) {
		meta.SetExternalName(cr, name)
	}
}

func TestObserve(t *testing.T) {
	type fields struct {
		client BedrockAgentClient
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	now := time.Now()
	kbId := "test-kb-id"
	kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/test-kb-id"

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"NotKnowledgeBase": {
			reason: "Should return error if managed resource is not a KnowledgeBase",
			fields: fields{
				client: &MockBedrockAgentClient{},
			},
			args: args{
				ctx: context.Background(),
				mg:  &struct{ resource.Managed }{}, // Wrong type for test
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotKnowledgeBase),
			},
		},
		"NoExternalName": {
			reason: "Should return ResourceExists=false when no external name is set",
			fields: fields{
				client: &MockBedrockAgentClient{},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
			},
		},
		"ResourceNotFound": {
			reason: "Should return ResourceExists=false when AWS returns ResourceNotFoundException",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockGetKnowledgeBase: func(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error) {
						return nil, &bedrockagentTypes.ResourceNotFoundException{
							Message: aws.String("Knowledge base not found"),
						}
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
			},
		},
		"GetKnowledgeBaseError": {
			reason: "Should return error when GetKnowledgeBase fails with non-404 error",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockGetKnowledgeBase: func(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error) {
						return nil, errors.New("some aws error")
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errors.New("some aws error"), errGetKnowledgeBase),
			},
		},
		"ResourceExists": {
			reason: "Should return ResourceExists=true and update status when knowledge base exists",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockGetKnowledgeBase: func(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error) {
						return &bedrockagent.GetKnowledgeBaseOutput{
							KnowledgeBase: &bedrockagentTypes.KnowledgeBase{
								KnowledgeBaseId:  aws.String(kbId),
								KnowledgeBaseArn: aws.String(kbArn),
								Name:             aws.String("test-knowledge-base"),
								Status:           bedrockagentTypes.KnowledgeBaseStatusActive,
								CreatedAt:        &now,
								UpdatedAt:        &now,
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
					ConnectionDetails: managed.ConnectionDetails{
						"knowledgeBaseId":  []byte(kbId),
						"knowledgeBaseArn": []byte(kbArn),
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got, test.EquateConditions()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	type fields struct {
		client BedrockAgentClient
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	kbId := "test-kb-id"
	kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/test-kb-id"

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"NotKnowledgeBase": {
			reason: "Should return error if managed resource is not a KnowledgeBase",
			fields: fields{
				client: &MockBedrockAgentClient{},
			},
			args: args{
				ctx: context.Background(),
				mg:  &struct{ resource.Managed }{}, // Wrong type for test
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotKnowledgeBase),
			},
		},
		"CreateKnowledgeBaseError": {
			reason: "Should return error when CreateKnowledgeBase fails",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockCreateKnowledgeBase: func(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error) {
						return nil, errors.New("create failed")
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New("create failed"), errCreateKnowledgeBase),
			},
		},
		"CreateSuccess": {
			reason: "Should create knowledge base successfully",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockCreateKnowledgeBase: func(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error) {
						return &bedrockagent.CreateKnowledgeBaseOutput{
							KnowledgeBase: &bedrockagentTypes.KnowledgeBase{
								KnowledgeBaseId:  aws.String(kbId),
								KnowledgeBaseArn: aws.String(kbArn),
								Name:             params.Name,
								Status:           bedrockagentTypes.KnowledgeBaseStatusCreating,
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(),
			},
			want: want{
				o: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						"knowledgeBaseId":  []byte(kbId),
						"knowledgeBaseArn": []byte(kbArn),
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Create(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Create(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Create(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	type fields struct {
		client BedrockAgentClient
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalUpdate
		err error
	}

	kbId := "test-kb-id"
	kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/test-kb-id"

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"NotKnowledgeBase": {
			reason: "Should return error if managed resource is not a KnowledgeBase",
			fields: fields{
				client: &MockBedrockAgentClient{},
			},
			args: args{
				ctx: context.Background(),
				mg:  &struct{ resource.Managed }{}, // Wrong type for test
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errNotKnowledgeBase),
			},
		},
		"UpdateKnowledgeBaseError": {
			reason: "Should return error when UpdateKnowledgeBase fails",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockUpdateKnowledgeBase: func(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error) {
						return nil, errors.New("update failed")
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errors.New("update failed"), errUpdateKnowledgeBase),
			},
		},
		"UpdateSuccess": {
			reason: "Should update knowledge base successfully",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockUpdateKnowledgeBase: func(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error) {
						return &bedrockagent.UpdateKnowledgeBaseOutput{
							KnowledgeBase: &bedrockagentTypes.KnowledgeBase{
								KnowledgeBaseId:  aws.String(kbId),
								KnowledgeBaseArn: aws.String(kbArn),
								Name:             params.Name,
								Status:           bedrockagentTypes.KnowledgeBaseStatusUpdating,
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o: managed.ExternalUpdate{
					ConnectionDetails: managed.ConnectionDetails{
						"knowledgeBaseId":  []byte(kbId),
						"knowledgeBaseArn": []byte(kbArn),
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Update(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Update(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type fields struct {
		client BedrockAgentClient
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalDelete
		err error
	}

	kbId := "test-kb-id"

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"NotKnowledgeBase": {
			reason: "Should return error if managed resource is not a KnowledgeBase",
			fields: fields{
				client: &MockBedrockAgentClient{},
			},
			args: args{
				ctx: context.Background(),
				mg:  &struct{ resource.Managed }{}, // Wrong type for test
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.New(errNotKnowledgeBase),
			},
		},
		"DeleteKnowledgeBaseError": {
			reason: "Should return error when DeleteKnowledgeBase fails with non-404 error",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockDeleteKnowledgeBase: func(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error) {
						return nil, errors.New("delete failed")
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errors.New("delete failed"), errDeleteKnowledgeBase),
			},
		},
		"ResourceNotFound": {
			reason: "Should succeed when resource is already deleted (404 error)",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockDeleteKnowledgeBase: func(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error) {
						return nil, &bedrockagentTypes.ResourceNotFoundException{
							Message: aws.String("Knowledge base not found"),
						}
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o: managed.ExternalDelete{},
			},
		},
		"DeleteSuccess": {
			reason: "Should delete knowledge base successfully",
			fields: fields{
				client: &MockBedrockAgentClient{
					MockDeleteKnowledgeBase: func(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error) {
						return &bedrockagent.DeleteKnowledgeBaseOutput{
							KnowledgeBaseId: aws.String(kbId),
							Status:          bedrockagentTypes.KnowledgeBaseStatusDeleting,
						}, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
				mg:  knowledgeBase(withExternalName(kbId)),
			},
			want: want{
				o: managed.ExternalDelete{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestConvertKnowledgeBaseConfiguration(t *testing.T) {
	e := &external{}

	tests := []struct {
		name   string
		input  *v1alpha1.KnowledgeBaseConfiguration
		expect *bedrockagentTypes.KnowledgeBaseConfiguration
	}{
		{
			name:   "nil input",
			input:  nil,
			expect: nil,
		},
		{
			name: "vector configuration",
			input: &v1alpha1.KnowledgeBaseConfiguration{
				Type: "VECTOR",
				VectorKnowledgeBaseConfiguration: &v1alpha1.VectorKnowledgeBaseConfiguration{
					EmbeddingModelArn: aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1"),
					EmbeddingModelConfiguration: &v1alpha1.EmbeddingModelConfiguration{
						BedrockEmbeddingModelConfiguration: &v1alpha1.BedrockEmbeddingModelConfiguration{
							Dimensions:        aws.Int32(1536),
							EmbeddingDataType: aws.String("FLOAT32"),
						},
					},
				},
			},
			expect: &bedrockagentTypes.KnowledgeBaseConfiguration{
				Type: bedrockagentTypes.KnowledgeBaseTypeVector,
				VectorKnowledgeBaseConfiguration: &bedrockagentTypes.VectorKnowledgeBaseConfiguration{
					EmbeddingModelArn: aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1"),
					EmbeddingModelConfiguration: &bedrockagentTypes.EmbeddingModelConfiguration{
						BedrockEmbeddingModelConfiguration: &bedrockagentTypes.BedrockEmbeddingModelConfiguration{
							Dimensions:        aws.Int32(1536),
							EmbeddingDataType: bedrockagentTypes.EmbeddingDataTypeFloat32,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.convertKnowledgeBaseConfiguration(tt.input)
			if diff := cmp.Diff(tt.expect, got, cmpopts.IgnoreUnexported(bedrockagentTypes.KnowledgeBaseConfiguration{}, bedrockagentTypes.VectorKnowledgeBaseConfiguration{}, bedrockagentTypes.EmbeddingModelConfiguration{}, bedrockagentTypes.BedrockEmbeddingModelConfiguration{})); diff != "" {
				t.Errorf("convertKnowledgeBaseConfiguration() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConvertStorageConfiguration(t *testing.T) {
	e := &external{}

	tests := []struct {
		name   string
		input  *v1alpha1.StorageConfiguration
		expect *bedrockagentTypes.StorageConfiguration
	}{
		{
			name:   "nil input",
			input:  nil,
			expect: nil,
		},
		{
			name: "S3 vectors configuration",
			input: &v1alpha1.StorageConfiguration{
				Type: "S3_VECTORS",
				S3VectorsConfiguration: &v1alpha1.S3VectorsConfiguration{
					IndexArn:        aws.String("arn:aws:bedrock:us-east-1:123456789012:index/test-index"),
					IndexName:       aws.String("test-index"),
					VectorBucketArn: aws.String("arn:aws:s3:::test-bucket"),
				},
			},
			expect: &bedrockagentTypes.StorageConfiguration{
				Type: bedrockagentTypes.KnowledgeBaseStorageTypeS3Vectors,
				S3VectorsConfiguration: &bedrockagentTypes.S3VectorsConfiguration{
					IndexArn:        aws.String("arn:aws:bedrock:us-east-1:123456789012:index/test-index"),
					IndexName:       aws.String("test-index"),
					VectorBucketArn: aws.String("arn:aws:s3:::test-bucket"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.convertStorageConfiguration(tt.input)
			if diff := cmp.Diff(tt.expect, got, cmpopts.IgnoreUnexported(bedrockagentTypes.StorageConfiguration{}, bedrockagentTypes.S3VectorsConfiguration{}, bedrockagentTypes.OpenSearchServerlessConfiguration{}, bedrockagentTypes.OpenSearchServerlessFieldMapping{}, bedrockagentTypes.PineconeConfiguration{}, bedrockagentTypes.PineconeFieldMapping{})); diff != "" {
				t.Errorf("convertStorageConfiguration() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	e := &external{}

	tests := []struct {
		name   string
		cr     *v1alpha1.KnowledgeBase
		kb     *bedrockagentTypes.KnowledgeBase
		expect bool
	}{
		{
			name: "up to date",
			cr: knowledgeBase(func(kb *v1alpha1.KnowledgeBase) {
				kb.Spec.ForProvider.Name = aws.String("test-kb")
				kb.Spec.ForProvider.Description = aws.String("test description")
				kb.Spec.ForProvider.RoleArn = aws.String("arn:aws:iam::123456789012:role/test-role")
			}),
			kb: &bedrockagentTypes.KnowledgeBase{
				Name:        aws.String("test-kb"),
				Description: aws.String("test description"),
				RoleArn:     aws.String("arn:aws:iam::123456789012:role/test-role"),
			},
			expect: true,
		},
		{
			name: "name mismatch",
			cr: knowledgeBase(func(kb *v1alpha1.KnowledgeBase) {
				kb.Spec.ForProvider.Name = aws.String("test-kb")
			}),
			kb: &bedrockagentTypes.KnowledgeBase{
				Name: aws.String("different-name"),
			},
			expect: false,
		},
		{
			name: "description mismatch",
			cr: knowledgeBase(func(kb *v1alpha1.KnowledgeBase) {
				kb.Spec.ForProvider.Description = aws.String("test description")
			}),
			kb: &bedrockagentTypes.KnowledgeBase{
				Description: aws.String("different description"),
			},
			expect: false,
		},
		{
			name: "role arn mismatch",
			cr: knowledgeBase(func(kb *v1alpha1.KnowledgeBase) {
				kb.Spec.ForProvider.RoleArn = aws.String("arn:aws:iam::123456789012:role/test-role")
			}),
			kb: &bedrockagentTypes.KnowledgeBase{
				RoleArn: aws.String("arn:aws:iam::123456789012:role/different-role"),
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.isUpToDate(tt.cr, tt.kb)
			if got != tt.expect {
				t.Errorf("isUpToDate() = %v, want %v", got, tt.expect)
			}
		})
	}
}
