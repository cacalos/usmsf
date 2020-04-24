#!/bin/sh
docker tag ${DOCKER_USER-flycat1}/usmsf:latest ${DOCKER_USER-flycat1}/usmsf:latest
docker push ${DOCKER_USER-flycat1}/usmsf:latest
