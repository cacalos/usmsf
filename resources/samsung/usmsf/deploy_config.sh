#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap -n smsf usmsf-conf 
kubectl delete configmap -n smsf usmsf-if-conf
kubectl delete configmap -n smsf usmsf-adif-conf
kubectl delete configmap -n smsf usmsf-nrf-conf

kubectl create configmap -n smsf usmsf-nrf-conf --from-file=$CURR_DIR/config_nrf
kubectl create configmap -n smsf usmsf-conf --from-file=$CURR_DIR/config
kubectl create configmap -n smsf usmsf-if-conf --from-file=$CURR_DIR/config_if
kubectl create configmap -n smsf usmsf-adif-conf --from-file=dia/data 
