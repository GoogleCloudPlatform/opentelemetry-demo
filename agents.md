# Agent Guide - OpenTelemetry Demo GCP Fork

This document provides context and guidelines for AI agents (and developers) working on this repository.

## Repository Overview

This repository is a fork of the upstream [open-telemetry/opentelemetry-demo](https://github.com/open-telemetry/opentelemetry-demo). It has been modified to integrate with Google Cloud Observability (Cloud Monitoring, Cloud Logging, and Cloud Trace).

### Key Differences from Upstream
*   Configured to export telemetry to GCP by default.
*   Includes GCP-specific deployment configurations (Helmfile, Cloud Build).
*   Contains additional Kubernetes manifests for GCP services (e.g., Service Directory registration).

## Contribution Guidelines

*   **GCP-Specific Changes:** Contributions that are specific to GCP integrations, deployment, or configuration should be made directly to this repository.
*   **Generic Changes:** Any changes to the core demo application, microservices (unless GCP-specific modification is needed), or generic OpenTelemetry configurations should be contributed to the upstream [open-telemetry/opentelemetry-demo](https://github.com/open-telemetry/opentelemetry-demo) repository.
*   **Public Repository:** This is a public GitHub repository. Do **not** commit any internal Google credentials, project IDs, notification channel details, or other sensitive information.

## Key Workflows

### Syncing with Upstream

To sync this fork with upstream updates, follow these steps:

1.  Add upstream and gcp remotes if you haven't already:
    ```sh
    git remote add upstream git@github.com:open-telemetry/opentelemetry-demo.git
    git remote add gcp git@github.com:GoogleCloudPlatform/opentelemetry-demo.git
    ```
2.  Fetch updates:
    ```sh
    git fetch upstream
    git fetch gcp
    ```
3.  Create a sync branch from the GCP main:
    ```sh
    git checkout gcp/main -b sync-upstream
    ```
4.  Merge the desired upstream tag or branch:
    ```sh
    git merge upstream <tag_name>
    ```
5.  Resolve conflicts.
    *   **Note on `kubernetes/opentelemetry-demo.yaml`**: This file is generated. You can ignore conflicts in this file during the merge. After resolving other conflicts, regenerate it.
6.  Regenerate Kubernetes manifests:
    ```sh
    make generate-kubernetes-manifests
    ```
7.  Commit changes, push, and open a Pull Request.
8.  Verify the deployment in a test GCP project.

### Deployment

*   **Continuous Deployment:** Pushes to the `main` branch trigger a deployment to the internal Telemetry Sandbox GKE cluster via Cloud Build, using `cloudbuild-deploy.yaml`.
*   **Manual Deployment:** Refer to [README_GCP.md](README_GCP.md) for instructions on running the demo on GKE or GCE.

## Testing Changes

Testing is primarily a manual process. The primary method is deploying to GKE.

> [!NOTE]
> Local testing via Docker Compose is currently unsupported in this fork and may not function correctly.

### GCP Integration Testing (GKE)

To verify changes (exporters, metrics, dashboard configurations, or service changes) in a test environment:

1.  **Prepare a Test GCP Project and GKE Cluster:**
    You need a GCP project with a GKE cluster (Autopilot is recommended for testing, but standard works too).

2.  **Prepare the Test Manifest:**
    Instead of deploying to the default `otel-demo` namespace, it is recommended to use a test namespace (e.g., `otel-demo-test`) to avoid conflicts.
    
    You can generate a test manifest from the default one:
    ```sh
    # Generate the default manifest if not already done
    make generate-kubernetes-manifests
    
    # Create a test manifest with a custom namespace
    sed 's/otel-demo/otel-demo-test/g' kubernetes/opentelemetry-demo.yaml > kubernetes/opentelemetry-demo-test.yaml
    ```
    
    > [!IMPORTANT]
    > **GKE Autopilot Memory Limits:** The default memory limits for the OpenTelemetry Collector (`otelcol`) in the generated manifest may be too low for GKE Autopilot, causing OOM kills or data refusal.
    > Edit `kubernetes/opentelemetry-demo-test.yaml` to increase the collector's memory limits:
    > *   Set `limits.memory` to `500Mi` (under `opentelemetry-demo-otelcol` deployment).
    > *   Set `requests.memory` to `500Mi`.
    > *   Increase `GOMEMLIMIT` env var to `400MiB`.

3.  **Configure Workload Identity for the Test Namespace:**
    The demo applications need permissions to write telemetry to GCP. Grant these permissions to the Kubernetes Service Accounts (KSA) in your test namespace.
    
    Set environment variables:
    ```sh
    PROJECT_ID="your-test-project-id"
    NAMESPACE="otel-demo-test" # Use your test namespace
    ```
    
    Allow the `opentelemetry-demo-otelcol` KSA to impersonate the Google Service Account (GSA) used for telemetry (refer to [README_GCP.md](README_GCP.md) for GSA setup):
    ```sh
    # Assuming GSA_NAME is the service account created for telemetry (e.g., opentelemetry-demo)
    GSA_NAME="opentelemetry-demo"
    
    gcloud iam service-accounts add-iam-policy-binding \
      --role roles/iam.workloadIdentityUser \
      --member "serviceAccount:${PROJECT_ID}.svc.id.goog[${NAMESPACE}/opentelemetry-demo-otelcol]" \
      ${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com \
      --project=${PROJECT_ID}
    ```
    Ensure the GSA has the following roles in your project:
    *   `roles/monitoring.metricWriter`
    *   `roles/cloudtrace.agent`
    *   `roles/logging.logWriter`

4.  **Deploy to GKE:**
    Set your `kubectl` context namespace to your test namespace so that resources without explicit namespace are deployed correctly:
    ```sh
    kubectl config set-context --current --namespace=otel-demo-test
    ```
    
    Apply the manifest:
    ```sh
    kubectl apply -f kubernetes/opentelemetry-demo-test.yaml
    ```

5.  **Verify Telemetry:**
    Verify that the workloads are healthy (`kubectl get pods`) and telemetry is arriving:
    *   **Logs:** Check Logs Explorer with query: `resource.type="k8s_container" resource.labels.namespace_name="otel-demo-test"`.
    *   **Metrics:** Check Metrics Explorer for `prometheus.googleapis.com` metrics with label `namespace="otel-demo-test"`.
    *   **Traces:** Check Trace Explorer, filtering by `k8s.namespace.name="otel-demo-test"`.

6.  **Clean up:**
    ```sh
    # Delete the resources
    kubectl delete -f kubernetes/opentelemetry-demo-test.yaml
    
    # Reset context namespace
    kubectl config set-context --current --namespace=default
    
    # (Optional) Remove the Workload Identity binding if no longer needed
    # gcloud iam service-accounts remove-iam-policy-binding ...
    ```

## Important Files

*   [README_GCP.md](README_GCP.md): Setup and run instructions for Google Cloud.
*   `kubernetes/opentelemetry-demo.yaml`: Generated Kubernetes manifests. **Do not edit directly.**
*   `kubernetes/service-directory-registration.yaml`: Configuration for registering the frontend-proxy with Service Directory (used for uptime monitoring).
*   `cloudbuild-deploy.yaml`: Cloud Build configuration for CD.
*   `gcp/`: Contains Terraform/Helm configurations for GCP deployment.
