#!/bin/bash

kubectl delete -f map/map-pod.yaml
kubectl delete -f dia/dia-pod.yaml
kubectl delete -f svc/svc-pod.yaml
kubectl delete configmap usmsf-conf
