/*
Copyright 2024 The Kubernetes Authors.

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

package scaffolds

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// render executes a named template string with the given data.
// Returns the rendered content as a string.
func render(name, tmplStr string, data any) (string, error) {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title, //nolint:staticcheck // acceptable for code generation
	}
	tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", name, err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
	}

	return buf.String(), nil
}

// renderMap renders a map of path→template strings with the given data.
// The keys of tmplMap are the destination file paths; the values are template strings.
// Returns a map of path→rendered content.
func renderMap(tmplMap map[string]string, data any) (map[string]string, error) {
	out := make(map[string]string, len(tmplMap))
	for path, tmplStr := range tmplMap {
		content, err := render(path, tmplStr, data)
		if err != nil {
			return nil, err
		}
		out[path] = content
	}
	return out, nil
}
