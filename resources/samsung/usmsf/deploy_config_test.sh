#!/bin/bash

CURR_DIR=`dirname $0`
kubectl delete configmap -n test usmsf-conf-test 
kubectl create configmap -n test usmsf-conf-test --from-file=$CURR_DIR/config_test
