#! /bin/bash

kubectl delete configmap usmsf-uccms-conf -n smsf

kubectl delete -f uccms/uccms_deploy.yaml

