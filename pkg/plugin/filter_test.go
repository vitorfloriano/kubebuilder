/*
Copyright 2022 The Kubernetes Authors.

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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

var _ = Describe("FilterPlugins", func() {
	var (
		p1         mockPlugin
		p2         mockPlugin
		p3         mockPlugin
		p4         mockPlugin
		p5         mockPlugin
		allPlugins []Plugin
	)

	BeforeEach(func() {
		p1 = mockPlugin{
			name:                     "go.kubebuilder.io",
			version:                  Version{Number: 2},
			supportedProjectVersions: []config.Version{{Number: 2}, {Number: 3}},
		}
		p2 = mockPlugin{
			name:                     "go.kubebuilder.io",
			version:                  Version{Number: 3},
			supportedProjectVersions: []config.Version{{Number: 3}},
		}
		p3 = mockPlugin{
			name:                     "example.kubebuilder.io",
			version:                  Version{Number: 1},
			supportedProjectVersions: []config.Version{{Number: 2}},
		}
		p4 = mockPlugin{
			name:                     "test.kubebuilder.io",
			version:                  Version{Number: 1},
			supportedProjectVersions: []config.Version{{Number: 3}},
		}
		p5 = mockPlugin{
			name:                     "go.test.domain",
			version:                  Version{Number: 2},
			supportedProjectVersions: []config.Version{{Number: 2}},
		}

		allPlugins = []Plugin{p1, p2, p3, p4, p5}
	})

	DescribeTable("should filter by key",
		func(key string, expectedPlugins func() []Plugin) {
			filtered, err := FilterPluginsByKey(allPlugins, key)
			Expect(err).NotTo(HaveOccurred())
			Expect(filtered).To(Equal(expectedPlugins()))
		},
		Entry("go plugins", "go", func() []Plugin { return []Plugin{p1, p2, p5} }),
		Entry("go plugins (kubebuilder domain)", "go.kubebuilder", func() []Plugin { return []Plugin{p1, p2} }),
		Entry("go v2 plugins", "go/v2", func() []Plugin { return []Plugin{p1, p5} }),
		Entry("go v2 plugins (kubebuilder domain)", "go.kubebuilder/v2", func() []Plugin { return []Plugin{p1} }),
	)

	It("should fail for invalid versions", func() {
		_, err := FilterPluginsByKey(allPlugins, "go/a")
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("should filter by project version",
		func(projectVersion config.Version, expectedPlugins func() []Plugin) {
			Expect(FilterPluginsByProjectVersion(allPlugins, projectVersion)).To(Equal(expectedPlugins()))
		},
		Entry("project v2 plugins", config.Version{Number: 2}, func() []Plugin { return []Plugin{p1, p3, p5} }),
		Entry("project v3 plugins", config.Version{Number: 3}, func() []Plugin { return []Plugin{p1, p2, p4} }),
	)
})
