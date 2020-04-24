#!/bin/bash

openssl req -new -config server.cnf -keyout server.key -out server.csr

openssl req -in server.csr -noout -text -nameopt sep_multiline

openssl x509 -req -in server.csr -days 3650 -sha1 -CAcreateserial -CA ../rootca.com/rootca.crt -CAkey ../rootca.com/rootca.key -out server.crt
