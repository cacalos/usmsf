#!/bin/bash
UAIMGS_HOME=`dirname $0`/../..
CURDIR=`dirname $0`

kubectl delete -f usmsf-adif.yaml
kubectl delete configmap usmsf-adif-conf -n smsf
kubectl create configmap usmsf-adif-conf --from-file=$CURDIR/data -n smsf
docker pull camel.uangel.com:5000/dia_svc:latest
kubectl apply -f ./usmsf-adif.yaml
kubectl get pods -n smsf
