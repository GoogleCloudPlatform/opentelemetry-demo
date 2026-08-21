---
name: update-upstream
description: >-
  Updates the Google Cloud OpenTelemetry Demo fork with the latest upstream
  Helm chart release. Updates version in gcp/helmfile.yaml, regenerates
  Kubernetes manifests, and verifies changes. Use when updating or syncing the
  repo with upstream OpenTelemetry demo releases.
---

# Update Upstream OpenTelemetry Demo

This skill guides you through updating the Google Cloud integration fork to the
latest release of the upstream OpenTelemetry Demo Helm chart.

> [!NOTE]
> For this repository, updating to upstream is done by bumping the Helm chart
> version in `gcp/helmfile.yaml` and regenerating
> `kubernetes/opentelemetry-demo.yaml`, rather than git rebasing the repository
> history.

## Prerequisites

* GitHub CLI (`gh`)
* `helmfile` and `helm`
* `make`

## Step-by-Step Instructions

### Step 1: Find Latest Upstream Helm Release

Check the latest release version of the upstream Helm chart:

```bash
gh release list -R open-telemetry/opentelemetry-helm-charts | grep 'opentelemetry-demo-'
```

### Step 2: Update Helmfile Version

Update the `version` field in `gcp/helmfile.yaml` to use the target upstream
version (e.g., `version: ~0.40.9`):

```yaml
releases:
  - name: opentelemetry-demo
    chart: open-telemetry/opentelemetry-demo
    version: ~<VERSION>
```

### Step 3: Regenerate Kubernetes Manifests

Run the Makefile target to re-render `kubernetes/opentelemetry-demo.yaml`:

```bash
make generate-kubernetes-manifests
```

### Step 4: Handle Breaking Changes or Values Updates

Inspect upstream release notes for any breaking changes or updated component
configurations. If necessary, update:

* `gcp/opentelemetry-demo-values.yaml`
* `kubernetes/service-directory-registration.yaml`

### Step 5: Validate Changes Locally

Run local linter and license checks:

```bash
make check
```

### Step 6: Commit and Push

Commit the updated manifests and configuration:

```bash
git add gcp/helmfile.yaml kubernetes/opentelemetry-demo.yaml gcp/opentelemetry-demo-values.yaml
git commit -m "Update upstream opentelemetry-demo to <VERSION>"
```
