# Running the demo on Google Cloud

The demo can send logs, traces, and metrics to Google Cloud. The easiest way to
do this is with the [`gcp/helmfile.yaml`](gcp/helmfile.yaml) (when
deploying via Helmfile on GKE) or
[`src/otelcollector/otelcol-config-extras.yml`](src/otelcollector/otelcol-config-extras.yml)
when running with `docker-compose` on GCE.

## Running on GKE

The recommended way to run the demo on GKE is with the official Helm chart. If
running on a GKE Autopilot cluster (or any cluster with Workload Identity), you
must follow the prerequisite steps to set up a Workload Identity-enabled service
account below. Otherwise, you can skip to the next section.

### Workload Identity prequisites

Follow the [Workload Identity
docs](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#authenticating_to)
to set up an IAM service account in your GCP project with permission to use
Workload Identity and write logs, traces, and metrics:

```console
export GCLOUD_PROJECT=<your project id>
```

```console
gcloud iam service-accounts create opentelemetry-demo \
    --project=${GCLOUD_PROJECT}
gcloud projects add-iam-policy-binding ${GCLOUD_PROJECT} \
    --member "serviceAccount:opentelemetry-demo@${GCLOUD_PROJECT}.iam.gserviceaccount.com" \
    --role "roles/logging.logWriter"
gcloud projects add-iam-policy-binding ${GCLOUD_PROJECT} \
    --member "serviceAccount:opentelemetry-demo@${GCLOUD_PROJECT}.iam.gserviceaccount.com" \
    --role "roles/monitoring.metricWriter"
gcloud projects add-iam-policy-binding ${GCLOUD_PROJECT} \
    --member "serviceAccount:opentelemetry-demo@${GCLOUD_PROJECT}.iam.gserviceaccount.com" \
    --role "roles/cloudtrace.agent"
gcloud iam service-accounts add-iam-policy-binding opentelemetry-demo@${GCLOUD_PROJECT}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${GCLOUD_PROJECT}.svc.id.goog[otel-demo/opentelemetry-demo-otelcol]"
```

### Deploying the Helmfile

Make sure you have the following installed:

* [Helmfile](https://helmfile.readthedocs.io/en/stable/#installation)
* [Helm](https://helm.sh/docs/intro/install/)
* [helm-diff plugin](https://github.com/databus23/helm-diff)

Without Workload Identity, run

```console
helmfile --interactive apply -f gcp/helmfile.yaml
```

With Workload Identity, run

```console
helmfile --interactive apply -f gcp/helmfile.yaml --state-values-set-string workload_identity_project_id=${GCLOUD_PROJECT}
```

### (Alternative) Using `kubectl apply`

Installing with the Helm chart is recommended, but you can also use `kubectl
apply` to install the manifests directly.

First, make sure you have followed the Workload Identity setup steps above.

Update
[`kubernetes/opentelemetry-demo.yaml`](kubernetes/opentelemetry-demo.yaml) to
annotate the Kubernetes service account with your project:

```console
sed -i "s/%GCLOUD_PROJECT%/${GCLOUD_PROJECT}/g" ./kubernetes/opentelemetry-demo.yaml
```

Install the manifests:

```console
kubectl apply -n otel-demo -f ./kubernetes/opentelemetry-demo.yaml
```

## Running on GCE

Follow the [OpenTelemetry docs to run with Docker](https://opentelemetry.io/docs/demo/docker-deployment/):

```console
make start
```

## Seeing telemetry

With the demo running, you should see telemetry automatically created by the
demo's load generator. You can see metrics under "Prometheus Target" in Cloud
Monitoring:

![metrics](gcp_metrics.png)

Traces in the Trace explorer:

![traces](gcp_traces.png)

And logs in the Logs explorer organized by service:

![logs](gcp_logs.png)
