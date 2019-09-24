#!/bin/sh

# set -e -o pipefail

date
echo Started - Full build using runtime container folders

# temporary until the syncing is in place
echo listing /home/default/pvc/src
ls -la /home/default/pvc/src

# temporary until the syncing is in place
date
echo copying /home/default/pvc/src/. to /home/default/emptydir/src
cp -rf /home/default/pvc/src/. /home/default/emptydir/src/

date
echo cd /home/default/emptydir/src and listing
cd /home/default/emptydir/src
ls -la

date
echo running full maven build in /home/default/emptydir/src
mvn -B clean package -Dmaven.repo.local=/home/default/emptydir/cache/.m2/repository -DskipTests=true


date
echo listing /data/idp/output after mvn
ls -la /home/default/emptydir/src/target

date
echo rm -rf /home/default/emptydir/buildartifacts and copying artifacts
rm -rf /home/default/emptydir/buildartifacts
mkdir -p /home/default/emptydir/buildartifacts/
cp -r  /home/default/emptydir/src/target/liberty/wlp/usr/servers/defaultServer/*  /home/default/emptydir/buildartifacts/
cp -r  /home/default/emptydir/src/target/liberty/wlp/usr/shared/resources/  /home/default/emptydir/buildartifacts/
cp  /home/default/emptydir/src/src/main/liberty/config/jvmbx.options  /home/default/emptydir/buildartifacts/jvm.options

date
echo remove /config dir contents and update with build artifacts
rm -rf /config/*
cp -rf /home/default/emptydir/buildartifacts/. /config
ls -la /config/

date
echo start the Liberty runtime
/opt/ibm/wlp/bin/server start

date
echo Finished - Full build using container folders
