#!/bin/bash

sp=`dirname $0`
echo "script path = $sp"

ULIB_CAHCE_DIR=`(cd $sp && go mod download -json camel.uangel.com/ua5g/ulib.git) | awk 'BEGIN { FS="\""; RS="," }; { if ($2 == "Dir") {print $4} }'`
echo "ULIB_CAHCE_DIR : $ULIB_CAHCE_DIR"

if [ "$ULIB_CAHCE_DIR" = "" ]; then
	echo "Can't find cached ulib.git directory"
	exit 1
fi

bash $ULIB_CAHCE_DIR/scripts/build_examples/personal_computer/docker_build.sh $sp


