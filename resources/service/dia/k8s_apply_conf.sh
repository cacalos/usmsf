#!/bin/bash
UAIMGS_HOME=`dirname $0`/../..
CURDIR=`dirname $0`

kubectl create configmap usmsf-adif-conf --from-file=$CURDIR/data -o yaml --dry-run -n usmsf | kubectl replace -f -
