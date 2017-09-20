#!/bin/bash

#bin/rcma -c ./config.json --pretty --privateip Name=*epoch*,Team=infra "ls -la"
#bin/rcma -c ./config.json --privateip Name=*epoch*,Team=infra "ls -la"
#echo "base test"

echo 
echo "############################################################"
echo 
bin/rcma -c ./config.json --quiet --privateip Name=*epoch*,Team=infra "date"
echo 
echo "############################################################"
echo 
bin/rcma -c ./config.json --privateip -x Name=epoch Team=infra "date"

echo 
echo "############################################################"
echo 
bin/rcma -c ./config.json --json --privateip Name=*epoch*,Team=infra "date"
