#!/bin/sh

# set -e -o pipefail

echo Started - Full build using volume folders
date
echo listing /data/idp/src
ls -la /data/idp/src

echo chown, listing and running mvn in /data/idp/src
chown -R 1001 /data/idp/src
cd /data/idp/src

echo list contents before maven build
ls -la

echo Run full maven build directly from volume (/data/idp/src)
date
mvn -B clean package -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true
date

echo listing after mvn
ls -la

echo copying /data/idp/target to /data/idp/output
date
cp -rf /data/idp/src/target /data/idp/output
date
chown -fR 1001 /data/idp/output

echo listing /data/idp/output after mvn and chown 1001 buildoutput
ls -la /data/idp/output

echo rm -rf /data/idp/buildartifacts and copying artifacts
date
rm -rf /data/idp/buildartifacts
mkdir -p /data/idp/buildartifacts/
cp -r /data/idp/output/target/liberty/wlp/usr/servers/defaultServer/* /data/idp/buildartifacts/
cp -r /data/idp/output/target/liberty/wlp/usr/shared/resources/ /data/idp/buildartifacts/
cp /data/idp/src/src/main/liberty/config/jvmbx.options /data/idp/buildartifacts/jvm.options
date

echo chown the buildartifacts dir
chown -fR 1001 /data/idp/buildartifacts 

echo Finished - Full build using volume folders
date
