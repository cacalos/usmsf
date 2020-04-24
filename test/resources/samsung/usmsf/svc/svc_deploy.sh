#!/bin/bash

kubectl delete -f svc-pod.yaml 

docker pull camel.uangel.com:5000/usmsf:latest
../deploy_config.sh
kubectl apply -f svc-pod.yaml
kubectl get pods -n usmsf
