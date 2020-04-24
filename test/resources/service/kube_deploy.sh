#!/bin/bash

./deploy_config.sh
kubectl apply -f map/map-pod.yaml
kubectl apply -f dia/dia-pod.yaml
kubectl apply -f svc/svc-pod.yaml
