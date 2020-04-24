#!/bin/sh
docker tag ${DOCKER_USER-UNKNOWN}/ubsf-nrfsim:latest ${DOCKER_USER-UNKNOWN}/ubsf-nrfsim:latest
docker push ${DOCKER_USER-UNKNOWN}/ubsf-nrfsim:latest
