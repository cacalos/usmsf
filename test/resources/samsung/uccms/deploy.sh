#! /bin/bash

echo "set configmap for uccms-conf"
kubectl create configmap usmsf-uccms-conf -n smsf --from-file=config

echo "deploy uccms service"
kubectl apply -f uccms/uccms_deploy.yaml

echo "
finish"


