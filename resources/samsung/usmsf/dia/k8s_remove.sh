#!/bin/bash
UAIMGS_HOME=`dirname $0`/../../..

kubectl delete -f ./usmsf-adif.yaml
kubectl delete configmap usmsf-adif-conf -n smsf

