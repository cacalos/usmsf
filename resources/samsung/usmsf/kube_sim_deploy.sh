#!/bin/bash

echo "Set configmap for Simulator(AMF,UDM)"
echo "------------------------------------------------------------"
./deploy_sim_config.sh

echo "Deployement AMF_SIM"
echo "------------------------------------------------------------"
kubectl apply -f amfsim/amf-pod.yaml

echo "Deployement UDM_SIM"
echo "------------------------------------------------------------"
#kubectl apply -f udmsim/udm-pod.yaml

