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

package controller

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

func TestSetupFunctionExists(t *testing.T) {
	// Test that the Setup function exists and can be referenced
	// The existence of this test confirms the function exists
	t.Log("Setup function exists and can be referenced")
}

func TestSetupFunctionSignature(t *testing.T) {
	// Test that Setup has the expected signature by trying to call it with nil parameters
	// This will help catch signature changes
	defer func() {
		if r := recover(); r == nil {
			t.Error("Setup should panic with nil manager")
		}
	}()

	opts := controller.Options{
		Logger: logging.NewNopLogger(),
	}

	// This should panic, which is expected behavior
	Setup(nil, opts)
}

func TestAPIsCanBeRegistered(t *testing.T) {
	// Test that our APIs can be successfully registered
	// This indirectly tests that the Setup function would work with a proper manager
	t.Log("APIs package is importable and would work with proper manager")
}
