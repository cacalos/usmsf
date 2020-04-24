#!/bin/bash

CURR_DIR=`dirname $0`

kubectl create configmap usmsf-smsc --from-file=$CURR_DIR/config_sim --namespace=smsf
kubectl apply -f smsc-pod.yaml
