#!/bin/bash

kubectl port-forward --address 0.0.0.0 svc/usmsf-adif-active 30868:3868
