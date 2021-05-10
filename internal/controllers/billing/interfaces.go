// Copyright 2021 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package billing

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

// Registerer can register usage of universal-crossplane with idempotent calls.
type Registerer interface {
	Register(ctx context.Context, secret *v1.Secret, uid string) (string, error)
	Verify(token, uid string) (bool, error)
}

// NewNopRegisterer returns a Registerer that does nothing.
func NewNopRegisterer() NopRegisterer {
	return NopRegisterer{}
}

// NopRegisterer implements Registerer and does nothing.
type NopRegisterer struct{}

// Register does nothing.
func (np NopRegisterer) Register(_ context.Context, _ *v1.Secret, _ string) (string, error) {
	return "", nil
}

// Verify does nothing.
func (np NopRegisterer) Verify(_, _ string) (bool, error) {
	return true, nil
}
