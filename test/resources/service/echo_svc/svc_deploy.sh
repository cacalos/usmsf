#!/bin/bash

kubectl delete -f echo-svc-pod.yaml 

docker pull camel.uangel.com:5000/usmsf_echo-usmsf:latest
kubectl apply -f echo-svc-pod.yaml
kubectl get pods -n shseo
