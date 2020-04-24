#!/bin/bash
kubectl delete configmap usmsf-udmsim --namespace=smsf
kubectl delete configmap usmsf-amfsim --namespace=smsf

kubectl delete -f amfsim/amf-pod.yaml
kubectl delete -f udmsim/udm-pod.yaml

