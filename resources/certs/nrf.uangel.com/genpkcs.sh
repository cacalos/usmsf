#!/bin/bash
openssl pkcs12 -export -in server.crt -inkey server.key -out nrf.uangel.com.p12
