#!/bin/bash

echo "Install HTTPIF-GO Module"
stopmc -b httpif

echo "Delete Exist HTTPIF-GO Module"
rm -f $HOME/home/bin/httpif

echo "Go Module Build"
go build httpif.go

echo "Go Module Move - bin"
mv httpif $HOME/home/bin/.

echo "Build SUCC................."

startmc -b httpif
