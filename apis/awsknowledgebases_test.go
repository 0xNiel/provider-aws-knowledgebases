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

package apis

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	awskbv1alpha1 "github.com/0xNiel/provider-aws-knowledgebases/apis/awskb/v1alpha1"
	providerawsknowledgebasesv1alpha1 "github.com/0xNiel/provider-aws-knowledgebases/apis/v1alpha1"
)

func TestAddToScheme(t *testing.T) {
	scheme := runtime.NewScheme()

	tests := []struct {
		name    string
		scheme  *runtime.Scheme
		wantErr bool
	}{
		{
			name:    "successful scheme registration",
			scheme:  scheme,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddToScheme(tt.scheme)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddToScheme() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemeBuilderRegistration(t *testing.T) {
	scheme := runtime.NewScheme()

	// Test that all expected scheme builders are registered
	err := AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Failed to add to scheme: %v", err)
	}

	// Verify that the main provider API types are registered
	providerConfigGVK := providerawsknowledgebasesv1alpha1.ProviderConfigGroupVersionKind
	if !scheme.Recognizes(providerConfigGVK) {
		t.Errorf("Scheme does not recognize ProviderConfig GVK: %v", providerConfigGVK)
	}

	providerConfigUsageGVK := providerawsknowledgebasesv1alpha1.ProviderConfigUsageGroupVersionKind
	if !scheme.Recognizes(providerConfigUsageGVK) {
		t.Errorf("Scheme does not recognize ProviderConfigUsage GVK: %v", providerConfigUsageGVK)
	}

	storeConfigGVK := providerawsknowledgebasesv1alpha1.StoreConfigGroupVersionKind
	if !scheme.Recognizes(storeConfigGVK) {
		t.Errorf("Scheme does not recognize StoreConfig GVK: %v", storeConfigGVK)
	}

	// Verify that the KnowledgeBase API types are registered
	knowledgeBaseGVK := awskbv1alpha1.KnowledgeBaseGroupVersionKind
	if !scheme.Recognizes(knowledgeBaseGVK) {
		t.Errorf("Scheme does not recognize KnowledgeBase GVK: %v", knowledgeBaseGVK)
	}
}

func TestAddToSchemesInitialization(t *testing.T) {
	// Test that AddToSchemes is properly initialized
	if len(AddToSchemes) == 0 {
		t.Error("AddToSchemes should not be empty after package initialization")
	}

	// We expect at least 2 scheme builders (provider v1alpha1 and awskb v1alpha1)
	if len(AddToSchemes) < 2 {
		t.Errorf("Expected at least 2 scheme builders, got %d", len(AddToSchemes))
	}
}

func TestMultipleSchemeRegistrations(t *testing.T) {
	scheme := runtime.NewScheme()

	// Register the scheme multiple times to ensure it's idempotent
	for i := 0; i < 3; i++ {
		err := AddToScheme(scheme)
		if err != nil {
			t.Fatalf("Failed to add to scheme on iteration %d: %v", i, err)
		}
	}

	// Verify that types are still recognized after multiple registrations
	providerConfigGVK := providerawsknowledgebasesv1alpha1.ProviderConfigGroupVersionKind
	if !scheme.Recognizes(providerConfigGVK) {
		t.Errorf("Scheme does not recognize ProviderConfig GVK after multiple registrations: %v", providerConfigGVK)
	}

	knowledgeBaseGVK := awskbv1alpha1.KnowledgeBaseGroupVersionKind
	if !scheme.Recognizes(knowledgeBaseGVK) {
		t.Errorf("Scheme does not recognize KnowledgeBase GVK after multiple registrations: %v", knowledgeBaseGVK)
	}
}
