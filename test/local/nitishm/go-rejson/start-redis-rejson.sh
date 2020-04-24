#!/bin/sh
set -ex
./redis-5.0.5/src/redis-server --loadmodule redis-5.0.5/rejson/src/rejson.so &
