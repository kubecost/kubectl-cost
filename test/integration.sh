#!/bin/bash

set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

set -o xtrace

binary=$1

# These commands are copied from usage. This script assumes
# a cluster with kubecost is configured as the current
# kube context on the machine this script runs on, hence
# it is an integration test.
#
# WARNING: the nightly cluster does not have a staging install
# so tests that use -N and/or --service-name should be changed.
 

# Show the projected monthly rate for each namespace
# based on the last 5 days of activity.
$binary namespace --window 5d

# Show how much each namespace cost over the past 5 days
# with additional CPU and memory cost and without efficiency.
$binary namespace \
  --historical \
  --window 5d \
  --show-cpu \
  --show-memory \
  --show-efficiency=false

# Show the projected monthly rate for each controller
# based on the last 5 days of activity with PV (persistent
# volume) cost breakdown.
$binary controller --window 5d --show-pv

# Show costs over the past 5 days broken down by the value
# of the "app" label:
$binary label --historical -l app

# Show the projected monthly rate for each deployment
# based on the last month of activity with CPU, memory,
# GPU, PV, and network cost breakdown.
$binary deployment --window month -A

# Show the projected monthly rate for each deployment
# in the "kubecost" namespace based on the last 3 days
# of activity with CPU cost breakdown.
$binary deployment \
  --window 3d \
  --show-cpu \
  -n kubecost

# The same, but with a non-standard Kubecost deployment
# in the namespace "kubecost-staging" with the cost
# analyzer service called "kubecost-staging-cost-analyzer".
# 
# WARNING: modified from usage examles because test cluster
# doesn't have a staging install
$binary deployment \
  --window 3d \
  --show-cpu \
  -n kubecost \
  -N kubecost \
  --service-name kubecost-cost-analyzer


# Show how much each pod in the "kube-system" namespace
# cost yesterday, including CPU-specific cost.
$binary pod \
  --historical \
  --window yesterday \
  --show-cpu \
  -n kube-system

# use proxy
$binary namespace --use-proxy
