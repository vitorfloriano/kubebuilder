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

package plugins_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

func TestPlugins(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}

var _ = Describe("ScaffolderOption", func() {
	Describe("WithPreScaffoldHook", func() {
		It("should set the PreScaffold hook on ScaffolderHooks", func() {
			called := false
			fn := func(_ machinery.Filesystem) error {
				called = true
				return nil
			}

			hooks := &plugins.ScaffolderHooks{}
			plugins.WithPreScaffoldHook(fn)(hooks)

			Expect(hooks.PreScaffold).NotTo(BeNil())
			Expect(hooks.PreScaffold(machinery.Filesystem{})).To(Succeed())
			Expect(called).To(BeTrue())
		})

		It("should propagate errors from the pre-scaffold hook", func() {
			expectedErr := errors.New("pre-scaffold error")
			fn := func(_ machinery.Filesystem) error { return expectedErr }

			hooks := &plugins.ScaffolderHooks{}
			plugins.WithPreScaffoldHook(fn)(hooks)

			Expect(hooks.PreScaffold(machinery.Filesystem{})).To(MatchError(expectedErr))
		})
	})

	Describe("WithPostScaffoldHook", func() {
		It("should set the PostScaffold hook on ScaffolderHooks", func() {
			called := false
			fn := func() error {
				called = true
				return nil
			}

			hooks := &plugins.ScaffolderHooks{}
			plugins.WithPostScaffoldHook(fn)(hooks)

			Expect(hooks.PostScaffold).NotTo(BeNil())
			Expect(hooks.PostScaffold()).To(Succeed())
			Expect(called).To(BeTrue())
		})

		It("should propagate errors from the post-scaffold hook", func() {
			expectedErr := errors.New("post-scaffold error")
			fn := func() error { return expectedErr }

			hooks := &plugins.ScaffolderHooks{}
			plugins.WithPostScaffoldHook(fn)(hooks)

			Expect(hooks.PostScaffold()).To(MatchError(expectedErr))
		})
	})

	Describe("ScaffolderHooks", func() {
		It("should allow nil PreScaffold without panicking", func() {
			hooks := &plugins.ScaffolderHooks{}
			Expect(hooks.PreScaffold).To(BeNil())
		})

		It("should allow nil PostScaffold without panicking", func() {
			hooks := &plugins.ScaffolderHooks{}
			Expect(hooks.PostScaffold).To(BeNil())
		})

		It("should apply multiple options independently", func() {
			preCallCount := 0
			postCallCount := 0

			hooks := &plugins.ScaffolderHooks{}
			plugins.WithPreScaffoldHook(func(_ machinery.Filesystem) error {
				preCallCount++
				return nil
			})(hooks)
			plugins.WithPostScaffoldHook(func() error {
				postCallCount++
				return nil
			})(hooks)

			Expect(hooks.PreScaffold(machinery.Filesystem{})).To(Succeed())
			Expect(hooks.PostScaffold()).To(Succeed())
			Expect(preCallCount).To(Equal(1))
			Expect(postCallCount).To(Equal(1))
		})
	})
})
