#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap tmp-config --namespace=shseo
kubectl create configmap tmp-config --from-file=$CURR_DIR/tmp_config --namespace=shseo

docker pull camel.uangel.com:5000/usmsf-tmp_config:latest
#docker tag camel.uangel.com:5000/usmsf:latest kube-registry.kube-system.svc.cluster.local:5000/usmsf-tmp_config:latest
#docker push kube-registry.kube-system.svc.cluster.local:5000/usmsf-tmp_config:latest
