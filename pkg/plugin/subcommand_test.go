/*
Copyright 2026 The Kubernetes Authors.

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

package plugin

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("PreScaffoldFunc", func() {
	It("should implement HasPreScaffold", func() {
		var _ HasPreScaffold = PreScaffoldFunc(nil)
	})

	It("should call the underlying function", func() {
		called := false
		fn := PreScaffoldFunc(func(_ machinery.Filesystem) error {
			called = true
			return nil
		})
		Expect(fn.PreScaffold(machinery.Filesystem{})).To(Succeed())
		Expect(called).To(BeTrue())
	})

	It("should propagate errors from the underlying function", func() {
		expectedErr := errors.New("pre-scaffold error")
		fn := PreScaffoldFunc(func(_ machinery.Filesystem) error {
			return expectedErr
		})
		Expect(fn.PreScaffold(machinery.Filesystem{})).To(MatchError(expectedErr))
	})
})

var _ = Describe("PostScaffoldFunc", func() {
	It("should implement HasPostScaffold", func() {
		var _ HasPostScaffold = PostScaffoldFunc(nil)
	})

	It("should call the underlying function", func() {
		called := false
		fn := PostScaffoldFunc(func() error {
			called = true
			return nil
		})
		Expect(fn.PostScaffold()).To(Succeed())
		Expect(called).To(BeTrue())
	})

	It("should propagate errors from the underlying function", func() {
		expectedErr := errors.New("post-scaffold error")
		fn := PostScaffoldFunc(func() error {
			return expectedErr
		})
		Expect(fn.PostScaffold()).To(MatchError(expectedErr))
	})
})
