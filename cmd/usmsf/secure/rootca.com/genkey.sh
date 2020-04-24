#!/bin/bash

openssl req -new -config rootca.cnf -keyout rootca.key -out rootca.csr
openssl x509 -req -days 365 -in rootca.csr -signkey rootca.key -out rootca.crt
openssl req -in rootca.csr -noout -text -nameopt sep_multiline

openssl x509 -req -days 365 -in ../sepp.uangel.com/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../sepp.uangel.com/server.crt
openssl x509 -req -days 365 -in ../nrf.uangel.com/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../nrf.uangel.com/server.crt
openssl x509 -req -days 365 -in ../sepp.skytel.mn/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../sepp.skytel.mn/server.crt
openssl x509 -req -days 365 -in ../sepp.unitel.mn/server.csr -CA rootca.crt -CAcreateserial -CAkey rootca.key -out ../sepp.unitel.mn/server.crt
