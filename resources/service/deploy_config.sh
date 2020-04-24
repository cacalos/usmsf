#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap usmsf-conf --namespace=usmsf
kubectl delete configmap usmsf-if-conf --namespace=usmsf
kubectl delete configmap usmsf-map-conf0 --namespace=usmsf
kubectl delete configmap usmsf-map-conf1 --namespace=usmsf
kubectl delete configmap usmsf-udmsim --namespace=usmsf
kubectl delete configmap usmsf-amfsim --namespace=usmsf

kubectl delete configmap mysql-conf --namespace=usmsf
kubectl delete configmap usmsf-conf-shseo --namespace=usmsf

kubectl create configmap usmsf-conf --from-file=$CURR_DIR/config --namespace=usmsf
kubectl create configmap usmsf-if-conf --from-file=$CURR_DIR/config_if --namespace=usmsf
kubectl create configmap usmsf-map-conf0 --from-file=$CURR_DIR/srg_conf_0 --namespace=usmsf
kubectl create configmap usmsf-map-conf1 --from-file=$CURR_DIR/srg_conf_1 --namespace=usmsf
kubectl create configmap usmsf-udmsim --from-file=$CURR_DIR/config_udmsim --namespace=usmsf
kubectl create configmap usmsf-amfsim --from-file=$CURR_DIR/config_amfsim --namespace=usmsf

kubectl create configmap mysql-conf --from-file=$CURR_DIR/config_mysql --namespace=usmsf
kubectl create configmap usmsf-conf-shseo --from-file=$CURR_DIR/config_svc_shseo --namespace=usmsf
