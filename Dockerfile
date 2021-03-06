FROM golang:alpine AS builder
#ARG command
ENV COMMAND=usmsf
RUN apk add --no-cache git
RUN apk add build-base
COPY camel_git_cert.pem /
RUN cat /camel_git_cert.pem >> /etc/ssl/certs/ca-certificates.crt
#WORKDIR /src
#COPY . .
#WORKDIR /src/cmd/$COMMAND
#RUN go install -v

#final stage
FROM alpine:latest
#ARG command
ENV COMMAND=usmsf
RUN apk --no-cache add ca-certificates
RUN mkdir /app
#COPY --from=builder /go/bin/$COMMAND /app/$COMMAND
#ENV GIN_MODE release
#ENTRYPOINT /app/$COMMAND
