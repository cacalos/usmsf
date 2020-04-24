FROM redis:latest as builder

ENV LIBDIR /usr/lib/redis/modules
ENV DEPS "python python-setuptools python-pip wget unzip build-essential"
#COPY redis.conf /usr/local/etc/redis/redis.conf
# Set up a build environment
RUN set -ex;\
    deps="$DEPS";\
    apt-get update; \
	apt-get install -y --no-install-recommends $deps;\
    pip install rmtest; 

# Build the source
ADD rejson-master /REJSON
#ADD RedisJSON /REJSON
WORKDIR /REJSON
RUN set -ex;\
    make clean; \
    deps="$DEPS";\
    make all -j 4; \
    make test;

# Package the runner
FROM redis:alpine
ENV LIBDIR /usr/lib/redis/modules
WORKDIR /data
RUN set -ex;\
    mkdir -p "$LIBDIR";
COPY --from=builder /REJSON/src/rejson.so  "$LIBDIR"
COPY --from=builder /REJSON/redis.conf /usr/local/etc/redis/redis.conf

#CMD ["redis-server", "--loadmodule", "/usr/lib/redis/modules/rejson.so", "/usr/local/etc/redis/redis.conf"]
CMD ["redis-server", "/usr/local/etc/redis/redis.conf"]
