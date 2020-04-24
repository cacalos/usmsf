#!/bin/bash

function help() {
	echo "Usage: $0 <command> <parameters>"
	echo "[command]"
	echo " list       : list usmsf-adif's pods "
	echo " status     : print usmsf-adif pods HA status"
	echo " active     : list usmsf-adif active pod"
	echo " standby    : list usmsf-adif standby pod"
	echo " set-active : set usmsf-adif active pod"
	echo " chg-active : change usmsf-adif active pod"
}

if [ "$#" -lt 1 ]; then
	help $@
	exit 1
fi

active=""
if [ "$1" == "list" ]; then
	kubectl get pods --show-labels | grep "usmsf-adif" | cut -d' ' -f1
	exit 0
elif [ "$1" == "status" ]; then
	if [ "$#" -lt 2 ]; then
		echo "Usage: $0 status <pod-name>"
		exit 1
	fi
	exist=`kubectl get pods --show-labels | grep "$2" | cut -d' ' -f1`
	if [ "$exist" != "$2" ]; then 
		echo "Unknown POD '$2'."
		exit 1
	fi
	bact=`kubectl get pods --show-labels | grep "$2" | grep "qoqo.dev/pod-designation" | cut -d' ' -f1`
	if [ "$bact" == "" ]; then
		echo "$2 is standby."
	else
		echo "$2 is active."
	fi
elif [ "$1" == "active" ]; then
	kubectl get pods --show-labels | grep "usmsf-adif" | grep "qoqo.dev/pod-designation" | cut -d' ' -f1
	exit 0
elif [ "$1" == "standby" ]; then
	kubectl get pods --show-labels | grep "usmsf-adif" | grep -v "qoqo.dev/pod-designation" | cut -d' ' -f1
	exit 0
elif [ "$1" == "set-active" ]; then
	if [ "$#" -lt 2 ]; then
		echo "Usage: $0 set-active <active-pod>"
		exit 1
	fi
	bact=`kubectl get pods --show-labels | grep "usmsf-adif" | grep "qoqo.dev/pod-designation" | cut -d' ' -f1`
	if [ "$bact" == "$2" ]; then
		echo "$2 is aleady active."
		exit 0
	else
		active=$2
	fi
elif [ "$1" == "chg-active" ]; then
	active=`kubectl get pods --show-labels | grep "usmsf-adif" | grep -v "qoqo.dev/pod-designation" | head -n 1 | cut -d' ' -f1`
	echo "$active will be active."
else
	help $@
	exit 1
fi


if [ "$active" == "usmsf-adif-0" ]; then
  kubectl label pods usmsf-adif-0 "qoqo.dev/pod-designation=active" && kubectl label pods usmsf-adif-1 "qoqo.dev/pod-designation"-
elif [ "$active" == "usmsf-adif-1" ]; then
  kubectl label pods usmsf-adif-1 "qoqo.dev/pod-designation=active" && kubectl label pods usmsf-adif-0 "qoqo.dev/pod-designation"-
fi
