#!/bin/bash

# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

apply="kubectl apply -k"

# $helmfile_vars is an associative array that will be passed to helmfile --state-values-set-string
declare -A helmfile_vars
helmfile_vars["namespace"]="otel-demo"

# Parse options
while getopts "N:p:kn" opt; do
  case $opt in
    N)
      helmfile_vars["namespace"]="${OPTARG}"
      ;;
    n)
      # dry-run
      apply="kubectl apply --dry-run=client -o yaml -k"
      ;;
    k)
      # Stop after kustomize
      apply="kubectl kustomize"
      ;;
    p)
      PROJECT_ID="${OPTARG}"
      ;;
    :)
      echo "Error: ${OPTARG} requires an argument"
      exit 1
      ;;
    ?)
      exit 1
      ;;
  esac
done

shift $((OPTIND-1))

# Append the helmfile_vars associative array to the HELMFILE_FLAGS indexed array of flags for helmfile

HELMFILE_FLAGS=("--environment" "cicd")
set_values() {
  local -a out
  for key in "${!helmfile_vars[@]}"; do
    out+=( "${key}=${helmfile_vars[${key}]}" )
  done
  local IFS=,
  if [ ${#out[*]} -gt 0 ]; then
    HELMFILE_FLAGS+=("--state-values-set-string" "${out[*]}")
  fi
}
set_values

src=$(readlink -f "$(dirname "$0")/..")

cd "$src"

# Re-generate the k8s YAML with the new values

make generate-kubernetes-manifests HELMFILE_FLAGS="${HELMFILE_FLAGS[*]}" >&2

kustomize_dir=$(mktemp -d)
cleanup() {
  [ -n "$kustomize_dir" ] && rm -rf "$kustomize_dir"
}
trap cleanup EXIT

# Use Kustomize to transform cluster-wide resources into namespaced resources.
# TODO: Add support for running inside a namespace to the upstream helm charts, so we can get rid of Kustomize.

cd "$kustomize_dir"

cp "$src/kubernetes/opentelemetry-demo.yaml" ./

cat > role.yaml <<EOF
- op: replace
  path: /kind
  value: Role
- op: add
  path: /metadata/namespace
  value: ${helmfile_vars["namespace"]}
EOF
cat > role_binding.yaml <<EOF
- op: replace
  path: /kind
  value: RoleBinding
- op: replace
  path: /roleRef/kind
  value: Role
- op: add
  path: /metadata/namespace
  value: ${helmfile_vars["namespace"]}
EOF
cat > prometheus_role.yaml <<EOF
- op: remove
  path: /rules/3
EOF

cat > kustomization.yaml <<EOF
resources:
- opentelemetry-demo.yaml
namespace: ${helmfile_vars["namespace"]}
patches:
- path: role.yaml
  target:
    kind: ClusterRole
- path: role_binding.yaml
  target:
    kind: ClusterRoleBinding
- path: prometheus_role.yaml
  target:
    kind: Role
    name: opentelemetry-demo-prometheus-server
EOF

$apply .

# If we could use workload identity pools, we'd want to add:
#PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format='get(projectNumber)')
#gcloud projects add-iam-policy-binding projects/${PROJECT_ID} \
#    --role=roles/container.clusterViewer \
#    --member=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/${helmfile_vars["namespace"]}/sa/KSA_NAME \
#    --condition=None
