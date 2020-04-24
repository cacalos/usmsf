#!/bin/bash

kubectl delete -f svc/svc-pod.yaml
kubectl delete -f dia/usmsf-adif.yaml
kubectl delete -f nrfclient/nrf-client.yaml
kubectl delete configmap -n smsf usmsf-adif-conf
kubectl delete configmap -n smsf usmsf-conf 
