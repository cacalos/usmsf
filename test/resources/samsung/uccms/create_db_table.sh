#!/bin/bash

mysql -h$1 -uroot -proot.123 uccms < mysql/init_db.sql

