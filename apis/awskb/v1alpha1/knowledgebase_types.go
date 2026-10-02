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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// KnowledgeBaseParameters are the configurable fields of a KnowledgeBase.
type KnowledgeBaseParameters struct {
	// Region is the region you'd like your resource to be created in.
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// A name for the knowledge base.
	// +kubebuilder:validation:Required
	Name *string `json:"name"`

	// The Amazon Resource Name (ARN) of the IAM role with permissions to invoke API
	// operations on the knowledge base.
	// +kubebuilder:validation:Required
	RoleArn *string `json:"roleArn"`

	// Contains details about the embeddings model used for the knowledge base.
	// +kubebuilder:validation:Required
	KnowledgeBaseConfiguration *KnowledgeBaseConfiguration `json:"knowledgeBaseConfiguration"`

	// A description of the knowledge base.
	// +optional
	Description *string `json:"description,omitempty"`

	// Contains details about the configuration of the vector database used for the
	// knowledge base.
	// +optional
	StorageConfiguration *StorageConfiguration `json:"storageConfiguration,omitempty"`

	// Specify the key-value pairs for the tags that you want to attach to your
	// knowledge base in this object.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// A unique, case-sensitive identifier to ensure that the API request completes no
	// more than one time. If this token matches a previous request, Amazon Bedrock
	// ignores the request, but does not return an error.
	// +optional
	ClientToken *string `json:"clientToken,omitempty"`
}

// KnowledgeBaseObservation are the observable fields of a KnowledgeBase.
type KnowledgeBaseObservation struct {
	// The Amazon Resource Name (ARN) of the knowledge base.
	KnowledgeBaseArn *string `json:"knowledgeBaseArn,omitempty"`

	// The unique identifier of the knowledge base.
	KnowledgeBaseId *string `json:"knowledgeBaseId,omitempty"`

	// The status of the knowledge base.
	Status *string `json:"status,omitempty"`

	// The time the knowledge base was created.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// The time the knowledge base was last updated.
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// A list of reasons that the API operation on the knowledge base failed.
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// KnowledgeBaseConfiguration contains details about the vector embeddings configuration of the knowledge base.
type KnowledgeBaseConfiguration struct {
	// The type of data that the data source is converted into for the knowledge base.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=VECTOR;KENDRA;SQL
	Type string `json:"type"`

	// Contains details about the model that's used to convert the data source into
	// vector embeddings.
	// +optional
	VectorKnowledgeBaseConfiguration *VectorKnowledgeBaseConfiguration `json:"vectorKnowledgeBaseConfiguration,omitempty"`
}

// VectorKnowledgeBaseConfiguration contains details about the model used to create vector embeddings for the knowledge base.
type VectorKnowledgeBaseConfiguration struct {
	// The Amazon Resource Name (ARN) of the model used to create vector embeddings for the knowledge base.
	// +kubebuilder:validation:Required
	EmbeddingModelArn *string `json:"embeddingModelArn"`

	// The embeddings model configuration details for the vector model used in Knowledge Base.
	// +optional
	EmbeddingModelConfiguration *EmbeddingModelConfiguration `json:"embeddingModelConfiguration,omitempty"`
}

// EmbeddingModelConfiguration contains the configuration details for the embeddings model.
type EmbeddingModelConfiguration struct {
	// The vector configuration details on the Bedrock embeddings model.
	// +optional
	BedrockEmbeddingModelConfiguration *BedrockEmbeddingModelConfiguration `json:"bedrockEmbeddingModelConfiguration,omitempty"`
}

// BedrockEmbeddingModelConfiguration contains the vector configuration details for the Bedrock embeddings model.
type BedrockEmbeddingModelConfiguration struct {
	// The dimensions details for the vector configuration used on the Bedrock embeddings model.
	// +optional
	Dimensions *int32 `json:"dimensions,omitempty"`

	// The data type for the vectors when using a model to convert text into vector embeddings.
	// +optional
	// +kubebuilder:validation:Enum=FLOAT32;BINARY
	EmbeddingDataType *string `json:"embeddingDataType,omitempty"`
}

// StorageConfiguration contains the storage configuration of the knowledge base.
type StorageConfiguration struct {
	// The vector store service in which the knowledge base is stored.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=OPENSEARCH_SERVERLESS;PINECONE;REDIS_ENTERPRISE_CLOUD;RDS;MONGODB_ATLAS;OPENSEARCH_MANAGED_CLUSTER;NEPTUNE_ANALYTICS;S3_VECTORS
	Type string `json:"type"`

	// The configuration settings for storing knowledge base data using S3 vectors.
	// +optional
	S3VectorsConfiguration *S3VectorsConfiguration `json:"s3VectorsConfiguration,omitempty"`

	// Contains the storage configuration of the knowledge base in Amazon OpenSearch Service.
	// +optional
	OpensearchServerlessConfiguration *OpenSearchServerlessConfiguration `json:"opensearchServerlessConfiguration,omitempty"`

	// Contains the storage configuration of the knowledge base in Pinecone.
	// +optional
	PineconeConfiguration *PineconeConfiguration `json:"pineconeConfiguration,omitempty"`
}

// S3VectorsConfiguration contains the storage configuration of the knowledge base for S3 vectors.
type S3VectorsConfiguration struct {
	// The Amazon Resource Name (ARN) of the vector index used for the knowledge base.
	// +optional
	IndexArn *string `json:"indexArn,omitempty"`

	// The name of the vector index used for the knowledge base.
	// +optional
	IndexName *string `json:"indexName,omitempty"`

	// The Amazon Resource Name (ARN) of the S3 bucket where vector embeddings are stored.
	// +optional
	VectorBucketArn *string `json:"vectorBucketArn,omitempty"`
}

// OpenSearchServerlessConfiguration contains the storage configuration of the knowledge base in Amazon OpenSearch Service.
type OpenSearchServerlessConfiguration struct {
	// The Amazon Resource Name (ARN) of the OpenSearch Service vector store.
	// +kubebuilder:validation:Required
	CollectionArn *string `json:"collectionArn"`

	// Contains the names of the fields to which to map information about the vector store.
	// +kubebuilder:validation:Required
	FieldMapping *OpenSearchServerlessFieldMapping `json:"fieldMapping"`

	// The name of the vector store.
	// +kubebuilder:validation:Required
	VectorIndexName *string `json:"vectorIndexName"`
}

// OpenSearchServerlessFieldMapping contains the names of the fields to which to map information about the vector store.
type OpenSearchServerlessFieldMapping struct {
	// The name of the field in which Amazon Bedrock stores metadata about the vector store.
	// +kubebuilder:validation:Required
	MetadataField *string `json:"metadataField"`

	// The name of the field in which Amazon Bedrock stores the raw text from your data.
	// +kubebuilder:validation:Required
	TextField *string `json:"textField"`

	// The name of the field in which Amazon Bedrock stores the vector embeddings for your data sources.
	// +kubebuilder:validation:Required
	VectorField *string `json:"vectorField"`
}

// PineconeConfiguration contains the storage configuration of the knowledge base in Pinecone.
type PineconeConfiguration struct {
	// The endpoint URL for your index management page.
	// +kubebuilder:validation:Required
	ConnectionString *string `json:"connectionString"`

	// The Amazon Resource Name (ARN) of the secret that you created in Secrets Manager that is linked to your Pinecone API key.
	// +kubebuilder:validation:Required
	CredentialsSecretArn *string `json:"credentialsSecretArn"`

	// Contains the names of the fields to which to map information about the vector store.
	// +kubebuilder:validation:Required
	FieldMapping *PineconeFieldMapping `json:"fieldMapping"`

	// The namespace to be used to write new data to your database.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// PineconeFieldMapping contains the names of the fields to which to map information about the vector store.
type PineconeFieldMapping struct {
	// The name of the field in which Amazon Bedrock stores metadata about the vector store.
	// +kubebuilder:validation:Required
	MetadataField *string `json:"metadataField"`

	// The name of the field in which Amazon Bedrock stores the raw text from your data.
	// +kubebuilder:validation:Required
	TextField *string `json:"textField"`
}

// A KnowledgeBaseSpec defines the desired state of a KnowledgeBase.
type KnowledgeBaseSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       KnowledgeBaseParameters `json:"forProvider"`
}

// A KnowledgeBaseStatus represents the observed state of a KnowledgeBase.
type KnowledgeBaseStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          KnowledgeBaseObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A KnowledgeBase is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,providerawsknowledgebases}
type KnowledgeBase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnowledgeBaseSpec   `json:"spec"`
	Status KnowledgeBaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KnowledgeBaseList contains a list of KnowledgeBase
type KnowledgeBaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnowledgeBase `json:"items"`
}

// KnowledgeBase type metadata.
var (
	KnowledgeBaseKind             = reflect.TypeOf(KnowledgeBase{}).Name()
	KnowledgeBaseGroupKind        = schema.GroupKind{Group: Group, Kind: KnowledgeBaseKind}.String()
	KnowledgeBaseKindAPIVersion   = KnowledgeBaseKind + "." + SchemeGroupVersion.String()
	KnowledgeBaseGroupVersionKind = SchemeGroupVersion.WithKind(KnowledgeBaseKind)
)

func init() {
	SchemeBuilder.Register(&KnowledgeBase{}, &KnowledgeBaseList{})
}
