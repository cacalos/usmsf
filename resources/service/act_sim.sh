#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap act-config --namespace=usmsf
kubectl create configmap act-config --from-file=$CURR_DIR/act_config --namespace=usmsf

docker pull camel.uangel.com:5000/usmsf-usmsfperf:latest
#docker tag camel.uangel.com:5000/usmsf:latest kube-registry.kube-system.svc.cluster.local:5000/usmsf-tmp_config:latest
#docker push kube-registry.kube-system.svc.cluster.local:5000/usmsf-tmp_config:latest
