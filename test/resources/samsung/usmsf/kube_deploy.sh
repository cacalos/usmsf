#!/bin/bash

echo "Set configmap for USMSF"
echo "------------------------------------------------------------"
./deploy_config.sh

echo "Deploy POD for Diameter Interface"
echo "------------------------------------------------------------"
kubectl apply -f dia/usmsf-adif.yaml

echo "Deploy POD for HTTP service"
echo "------------------------------------------------------------"
kubectl apply -f svc/svc-pod.yaml

echo "------------------------------------------------------------"
kubectl apply -f nrfclient/nrf-client.yaml
echo "deploy fiinsh"
