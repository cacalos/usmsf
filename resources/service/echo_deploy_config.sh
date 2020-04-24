#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap usmsf-conf --namespace=shseo
kubectl delete configmap usmsf-if-conf --namespace=shseo
kubectl delete configmap echo-usmsf-udmsim --namespace=shseo
kubectl delete configmap echo-usmsf-amfsim --namespace=shseo

kubectl create configmap usmsf-conf --from-file=$CURR_DIR/echo_config --namespace=shseo
kubectl create configmap usmsf-if-conf --from-file=$CURR_DIR/config_if --namespace=shseo
kubectl create configmap echo-usmsf-udmsim --from-file=$CURR_DIR/echo_config_udmsim --namespace=shseo
kubectl create configmap echo-usmsf-amfsim --from-file=$CURR_DIR/echo_config_amfsim --namespace=shseo

