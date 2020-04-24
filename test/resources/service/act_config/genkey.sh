#!/bin/bash

openssl req -new -config rootca.cnf -keyout rootca.key -out rootca.csr
openssl x509 -req -days 365 -in rootca.csr -signkey rootca.key -out rootca.crt
openssl req -in rootca.csr -noout -text -nameopt sep_multiline

openssl x509 -req -days 365 -in ../smsf.uangel.com/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../smsf.uangel.com/server.crt
openssl x509 -req -days 365 -in ../amf.uangel.com/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../amf.uangel.com/server.crt
openssl x509 -req -days 365 -in ../udm.uangel.com/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../udm.uangel.com/server.crt
openssl x509 -req -days 365 -in ../smsc.uangel.com/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../smsc.uangel.com/server.crt
