#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap usmsf-actsim --namespace=smsf
kubectl create configmap usmsf-actsim --from-file=$CURR_DIR/config_actsim --namespace=smsf

