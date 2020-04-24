#!/bin/sh

# 실행 파일이  cmd 디렉토리밑에 있고
# 디렉토리 이름과 실행파일 이름이 동일하다고 가정합니다.

sp=`dirname $0`
echo "script path = $sp"

pd=$sp
if [ "$1" != "" ]; then
	pd=$1
fi

DOCKER_USER=${DOCKER_USER-camel.uangel.com:5000}
echo "docker user : $DOCKER_USER"

ULIB_CAHCE_DIR=`(cd $pd && go mod download -json camel.uangel.com/ua5g/ulib.git) | awk 'BEGIN { FS="\""; RS="," }; { if ($2 == "Dir") {print $4} }'`
echo "ulib.git cache dir : $ULIB_CAHCE_DIR"

if [ "$ULIB_CAHCE_DIR" = "" ]; then
	echo "Can't find cached ulib.git directory"
	exit 1
fi

. $ULIB_CAHCE_DIR/scripts/docker_build/docker_functions

apd=`abspath $pd`

SCRIPT_DIR=`script_dir`
echo "absolute script path = $SCRIPT_DIR"

PACKAGE_NAME=`basename $apd`
echo "package name : $PACKAGE_NAME"

PROJECT_DIR=`abspath $apd/../..`
PROJECT_NAME=`basename $PROJECT_DIR`

#PACKAGE_DOCKER_NAME="$PROJECT_NAME-$PACKAGE_NAME"
#if [ "$PROJECT_NAME" = "$PACKAGE_NAME" ]; then
#	PACKAGE_DOCKER_NAME=$PACKAGE_NAME
#fi
PACKAGE_DOCKER_NAME=$PACKAGE_NAME
echo "docker image name will be $DOCKER_USER/$PACKAGE_DOCKER_NAME:latest"

cp $ULIB_CAHCE_DIR/scripts/docker_build/camel_git_cert.pem $apd/../..

(cd $apd && go clean)

docker rmi $DOCKER_USER/$PACKAGE_DOCKER_NAME:latest
docker build --build-arg command=$PACKAGE_NAME -f $SCRIPT_DIR/DockerfileJenkins -t $DOCKER_USER/$PACKAGE_DOCKER_NAME:latest $apd/../..

rm -f $apd/../../camel_git_cert.pem
