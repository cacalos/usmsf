#!/bin/bash

openssl req -new -config server.cnf -newkey rsa:2048 -keyout server.key -out server.csr
#openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
#openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj "/C=KO/ST=KyeongKi/L=SeongNam/O=UANGEL/OU=CoreTech/CN=sepp.uangel.com"
openssl req -in server.csr -noout -text -nameopt sep_multiline
