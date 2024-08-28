# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#/bin/bash

# This script set up how to run Tracetest and which test files 
# be executed

set -e

# Availalble services to test
ALL_SERVICES=("adservice" "cartservice" "currencyservice" "checkoutservice" "frontend" "emailservice" "paymentservice" "productcatalogservice" "recommendationservice" "shippingservice")

## Script variables
# Will contain the list of services to test
chosen_services=()
# Array to hold process IDs
pids=()
# Array to hold exit codes
exit_codes=()

## Script functions
check_if_tracetest_is_installed() {
  if ! command -v tracetest &> /dev/null
  then
      echo "tracetest CLI could not be found"
      exit -1
  fi
}

create_env_file() {
  cat << EOF > tracetesting-vars.yaml
type: VariableSet
spec:
  id: tracetesting-vars
  name: tracetesting-vars
  values:
    - key: AD_SERVICE_ADDR
      value: $AD_SERVICE_ADDR
    - key: CART_SERVICE_ADDR
      value: $CART_SERVICE_ADDR
    - key: CHECKOUT_SERVICE_ADDR
      value: $CHECKOUT_SERVICE_ADDR
    - key: CURRENCY_SERVICE_ADDR
      value: $CURRENCY_SERVICE_ADDR
    - key: EMAIL_SERVICE_ADDR
      value: $EMAIL_SERVICE_ADDR
    - key: FRONTEND_ADDR
      value: $FRONTEND_ADDR
    - key: PAYMENT_SERVICE_ADDR
      value: $PAYMENT_SERVICE_ADDR
    - key: PRODUCT_CATALOG_SERVICE_ADDR
      value: $PRODUCT_CATALOG_SERVICE_ADDR
    - key: RECOMMENDATION_SERVICE_ADDR
      value: $RECOMMENDATION_SERVICE_ADDR
    - key: SHIPPING_SERVICE_ADDR
      value: $SHIPPING_SERVICE_ADDR
    - key: KAFKA_SERVICE_ADDR
      value: $KAFKA_SERVICE_ADDR
EOF
}

run_tracetest() {
  service_name=$1
  testsuite_file=./$service_name/all.yaml

  tracetest --config ./cli-config.yml run testsuite --file $testsuite_file --vars ./tracetesting-vars.yaml &
  pids+=($!)
}

## Script execution
while [[ $# -gt 0 ]]; do
  chosen_services+=("$1")
  shift
done

if [ ${#chosen_services[@]} -eq 0 ]; then
  for service in "${ALL_SERVICES[@]}"; do
    chosen_services+=("$service")
  done
fi

check_if_tracetest_is_installed
create_env_file

echo "Starting tests..."
echo "Running trace-based tests for: ${chosen_services[*]} ..."
echo ""

for service in "${chosen_services[@]}"; do
  run_tracetest $service
done

# Wait for processes to finish and capture their exit codes
for pid in "${pids[@]}"; do
    wait $pid
    exit_codes+=($?)
done

# Find the maximum exit code
max_exit_code=0
for code in "${exit_codes[@]}"; do
    if [[ $code -gt $max_exit_code ]]; then
        max_exit_code=$code
    fi
done

echo ""
echo "Tests done! Exit code: $max_exit_code"

exit $max_exit_code
