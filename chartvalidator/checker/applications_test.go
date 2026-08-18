package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeApplicationFile writes content to <root>/<env>/applications/<name>.
func writeApplicationFile(t *testing.T, root, env, name, content string) {
	t.Helper()
	dir := filepath.Join(root, env, "applications")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

const multiSourceApp = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: core-traefik-main
  namespace: argocd
spec:
  project: cluster-gateway
  sources:
    - repoURL: git@github.com:interledger/clusternation-deploy.git
      targetRevision: main
      ref: values
    - repoURL: https://traefik.github.io/charts
      chart: traefik
      targetRevision: 41.2.0
      helm:
        valueFiles:
          - $values/base/values/blank.yaml
          - $values/clusters/ilf-1/values/core-traefik.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: traefik
`

func TestChartsFromApplications_MultiSource(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "core-traefik.yaml", multiSourceApp)

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	require.Len(t, charts, 1, "the values-only ref source must not yield a chart")

	c := charts[0]
	assert.Equal(t, "ilf-1", c.Env)
	assert.Equal(t, "traefik", c.ChartName)
	assert.Equal(t, "https://traefik.github.io/charts", c.RepoURL)
	assert.Equal(t, "41.2.0", c.ChartVersion)
	assert.Equal(t, []string{
		srcPrefix + "base/values/blank.yaml",
		srcPrefix + "clusters/ilf-1/values/core-traefik.yaml",
	}, c.ValueFiles)
}

func TestChartsFromApplications_SingleSource(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "app.yaml", `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: single
spec:
  source:
    repoURL: https://prometheus-community.github.io/helm-charts
    chart: kube-prometheus-stack
    targetRevision: 88.3.0
`)

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	require.Len(t, charts, 1)
	assert.Equal(t, "kube-prometheus-stack", charts[0].ChartName)
	assert.Equal(t, "88.3.0", charts[0].ChartVersion)
	assert.Empty(t, charts[0].ValueFiles, "no helm block means no values files")
}

// An App-of-Apps points at a git path in another repository; there is no chart
// to render, and that must not be an error.
func TestChartsFromApplications_SkipsNonChartSources(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "app-of-apps.yaml", `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: business-apps
spec:
  source:
    repoURL: git@github.com:interledger/some-other-repo.git
    targetRevision: main
    path: argocd/applications
`)

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	assert.Empty(t, charts)
}

func TestChartsFromApplications_MultiDocumentAndOtherKinds(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "combined.yaml", `apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: cluster-gateway
spec:
  sourceRepos:
    - '*'
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: chart-app
spec:
  source:
    repoURL: https://example.com/charts
    chart: example
    targetRevision: 1.2.3
`)

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	require.Len(t, charts, 1, "the AppProject document must be ignored")
	assert.Equal(t, "example", charts[0].ChartName)
}

func TestChartsFromApplications_MultipleChartSourcesInOneApp(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "two-charts.yaml", `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: two-charts
spec:
  sources:
    - repoURL: https://example.com/charts
      chart: first
      targetRevision: 1.0.0
    - repoURL: https://example.com/charts
      chart: second
      targetRevision: 2.0.0
`)

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	require.Len(t, charts, 2)
	assert.Equal(t, "first", charts[0].ChartName)
	assert.Equal(t, "second", charts[1].ChartName)
}

// A valueFiles entry without a "$ref/" prefix is resolved by ArgoCD inside the
// chart, so it cannot be located in the repository and is skipped. A custom ref
// name is honoured.
func TestChartsFromApplications_ValueFilePrefixHandling(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "prefixes.yaml", `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: prefixes
spec:
  sources:
    - repoURL: https://example.com/charts
      chart: example
      targetRevision: 1.0.0
      helm:
        valueFiles:
          - $custom-ref/clusters/ilf-1/values/example.yaml
          - values-production.yaml
`)

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	require.Len(t, charts, 1)
	assert.Equal(t, []string{srcPrefix + "clusters/ilf-1/values/example.yaml"}, charts[0].ValueFiles)
}

func TestChartsFromApplications_RecursesIntoSubfolders(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ilf-1", "applications", "nested")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nested
spec:
  source:
    repoURL: https://example.com/charts
    chart: nested-chart
    targetRevision: 9.9.9
`), 0o644))

	charts, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.NoError(t, err)
	require.Len(t, charts, 1)
	assert.Equal(t, "nested-chart", charts[0].ChartName)
}

func TestChartsFromApplications_NoApplicationsDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sandbox", "appsets"), 0o755))

	charts, err := chartsFromApplications("sandbox", filepath.Join(root, "sandbox"))
	require.NoError(t, err)
	assert.Empty(t, charts)
}

func TestChartsFromApplications_InvalidYAML(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, root, "ilf-1", "broken.yaml", "kind: Application\n\tbad indentation:\n")

	_, err := chartsFromApplications("ilf-1", filepath.Join(root, "ilf-1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

// findCharts must pick up both layouts, so a tree mixing ApplicationSets and
// Applications is fully covered.
func TestFindCharts_BothLayouts(t *testing.T) {
	root := t.TempDir()

	appsets := filepath.Join(root, "mixed", "appsets")
	require.NoError(t, os.MkdirAll(appsets, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appsets, "cluster-appset.yaml"), []byte(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: cluster-appset
spec:
  generators:
    - list:
        elements:
          - name: from-appset
            chartName: appset-chart
            repoURL: https://example.com/charts
            chartVersion: 1.0.0
            baseValuesFile: base/values/blank.yaml
            valuesOverride: env/mixed/values/appset-chart.yaml
`), 0o644))

	writeApplicationFile(t, root, "mixed", "app.yaml", `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: from-application
spec:
  source:
    repoURL: https://example.com/charts
    chart: application-chart
    targetRevision: 2.0.0
`)

	charts, err := findCharts(root, "mixed")
	require.NoError(t, err)
	require.Len(t, charts, 2)

	names := []string{charts[0].ChartName, charts[1].ChartName}
	assert.ElementsMatch(t, []string{"appset-chart", "application-chart"}, names)
}

func TestResolvedValueFiles(t *testing.T) {
	t.Run("falls back to the appset base/override pair", func(t *testing.T) {
		refs := ChartRenderParams{
			BaseValuesFile: "../base/values/blank.yaml",
			ValuesOverride: "../env/sandbox/values/traefik.yaml",
		}.resolvedValueFiles()

		require.Len(t, refs, 2)
		assert.Equal(t, "../base/values/blank.yaml", refs[0].path)
		assert.Equal(t, "base values file", refs[0].label)
		assert.Equal(t, "../env/sandbox/values/traefik.yaml", refs[1].path)
		assert.Equal(t, "values override file", refs[1].label)
	})

	t.Run("ValueFiles replaces the pair", func(t *testing.T) {
		refs := ChartRenderParams{
			BaseValuesFile: "ignored.yaml",
			ValuesOverride: "ignored-too.yaml",
			ValueFiles:     []string{"a.yaml", "b.yaml", "c.yaml"},
		}.resolvedValueFiles()

		require.Len(t, refs, 3)
		assert.Equal(t, []string{"a.yaml", "b.yaml", "c.yaml"},
			[]string{refs[0].path, refs[1].path, refs[2].path})
	})
}
