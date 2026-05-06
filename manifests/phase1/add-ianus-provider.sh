#!/bin/bash

set -x

KUBECONFIG=/home/sava/repos/ecotwin/helm-charts/.secret/kcp/admin.kubeconfig

kubectl create-workspace ianus-provider --type=root:provider --ignore-existing --server="https://localhost:8443/clusters/root:providers"
kubectl apply -k ./providers/ianus-provider --server="https://localhost:8443/clusters/root:providers:ianus-provider"
kubectl apply -k ./orgs --server="https://localhost:8443/clusters/root:orgs"