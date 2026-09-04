package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// valuesRefPrefix matches the "$<ref-name>/" prefix ArgoCD uses in a
// multi-source Application's helm.valueFiles to point at a sibling source in
// the same Application (conventionally "$values/"). What follows the prefix is
// a path relative to the root of the referenced git repository.
var valuesRefPrefix = regexp.MustCompile(`^\$[A-Za-z0-9_.-]+/`)

// chartsFromApplications scans ArgoCD Application manifests under
// <envPath>/applications and returns one ChartRenderParams per Helm chart
// source found. This is the layout used by clusters, which declare their core
// charts as individual Applications rather than through an ApplicationSet.
//
// Applications that reference no chart are skipped rather than treated as an
// error: App-of-Apps entries point at a git path in another repository, and the
// values-only "ref" source of a multi-source Application has nothing to render.
func chartsFromApplications(envName, envPath string) ([]ChartRenderParams, error) {
	appsPath := filepath.Join(envPath, "applications")
	ok, err := existsDir(appsPath)
	if err != nil || !ok {
		return []ChartRenderParams{}, err
	}

	// Recursive, so charts stay discoverable if applications/ grows subfolders.
	files, err := findYAMLFiles(appsPath)
	if err != nil {
		return nil, err
	}

	var charts []ChartRenderParams
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		docs, err := decodeYAMLDocuments(data, f)
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			charts = append(charts, extractApplicationCharts(doc, envName, f)...)
		}
	}
	return charts, nil
}

// decodeYAMLDocuments decodes every mapping document in a possibly
// multi-document YAML file.
func decodeYAMLDocuments(data []byte, path string) ([]map[string]any, error) {
	var out []map[string]any
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node any
		err := dec.Decode(&node)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML %s: %w", path, err)
		}
		if m, ok := node.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// extractApplicationCharts pulls the Helm chart sources out of a single
// Application document. Both spec.source (single-source) and spec.sources
// (multi-source) are handled.
func extractApplicationCharts(doc map[string]any, envName, path string) []ChartRenderParams {
	if str(doc["kind"]) != "Application" {
		return nil
	}
	spec, _ := doc["spec"].(map[string]any)
	if spec == nil {
		return nil
	}

	var sources []map[string]any
	if list, ok := spec["sources"].([]any); ok {
		for _, s := range list {
			if m, ok := s.(map[string]any); ok {
				sources = append(sources, m)
			}
		}
	}
	if single, ok := spec["source"].(map[string]any); ok {
		sources = append(sources, single)
	}

	appName := "<unnamed>"
	if md, ok := doc["metadata"].(map[string]any); ok {
		if n := str(md["name"]); n != "" {
			appName = n
		}
	}

	var charts []ChartRenderParams
	for _, src := range sources {
		chartName := str(src["chart"])
		repoURL := str(src["repoURL"])

		if chartName == "" {
			// Argo's native OCI sources carry no separate `chart` field.
			// The chart itself is the full repoURL, for example
			// oci://ghcr.io/interledger/charts/merchant. The chart name
			// is that URL's last path segment.
			//
			// This check only recognizes an explicit "oci://" scheme. It
			// does not recognize the scheme-less form ApplicationSets
			// accept (see normalizeRepoURL). A scheme-less git remote,
			// for example git@github.com:org/repo.git, also has no
			// "://". Reading it as OCI would misread it as a chart.
			//
			// A non-OCI source with no chart field is a values-only ref
			// source, or a git App-of-Apps path. It has nothing to
			// render.
			if !isOCIRepo(repoURL) {
				continue
			}
			chartName = lastPathSegment(repoURL)
			if chartName == "" {
				continue
			}
		}

		charts = append(charts, ChartRenderParams{
			Env:          envName,
			ChartName:    chartName,
			RepoURL:      repoURL,
			ChartVersion: str(src["targetRevision"]),
			ValueFiles:   applicationValueFiles(src, appName, path),
		})
	}
	return charts
}

// lastPathSegment returns a URL or path's final "/"-separated, non-empty
// segment. It names a chart when repoURL already points at the chart
// itself. In that case, no separate chart name field exists.
func lastPathSegment(url string) string {
	trimmed := strings.TrimRight(url, "/")
	if trimmed == "" {
		return ""
	}
	idx := strings.LastIndex(trimmed, "/")
	return trimmed[idx+1:]
}

// applicationValueFiles maps an Application source's helm.valueFiles onto paths
// this tool can read, which are relative to the checker's working directory.
func applicationValueFiles(src map[string]any, appName, path string) []string {
	helm, _ := src["helm"].(map[string]any)
	if helm == nil {
		return nil
	}

	if helm["values"] != nil || helm["valuesObject"] != nil {
		fmt.Printf("WARNING: %s (%s): inline helm values are not applied by the chart checker; only valueFiles are rendered\n", appName, path)
	}

	list, _ := helm["valueFiles"].([]any)
	var out []string
	for _, v := range list {
		p := str(v)
		if p == "" {
			continue
		}
		prefix := valuesRefPrefix.FindString(p)
		if prefix == "" {
			// Without a "$ref/" prefix ArgoCD resolves the path inside the chart
			// itself, so there is no corresponding file in this repository.
			fmt.Printf("WARNING: %s (%s): skipping chart-internal values file %q\n", appName, path, p)
			continue
		}
		out = append(out, srcPrefix+strings.TrimPrefix(p, prefix))
	}
	return out
}
