#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN mkdir /app
COPY nrfclient /app/nrfclient
ENV GIN_MODE release
ENTRYPOINT /app/nrfclient
#EXPOSE 8100
