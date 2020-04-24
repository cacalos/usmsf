#!/bin/bash

kubectl delete -f service_account.yaml
kubectl delete -f role.yaml
kubectl delete -f role_binding.yaml
kubectl delete -f operator.yaml
