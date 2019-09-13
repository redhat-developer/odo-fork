#!/bin/sh

# set -e -o pipefail

date
echo Started - Full build using container folders

echo listing /data/idp/src
ls -la /data/idp/src

date
echo copying /data/idp/src to /tmp/app
cp -rf /data/idp/src /tmp/app

date
echo chown -fR 1001 /tmp/app and listing
chown -fR 1001 /tmp/app
cd /tmp/app
ls -la

date
echo running full maven build in /tmp/app
mvn -B clean package -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true

date
echo copying target to output dir
rm -rf /data/idp/output
mkdir -p /data/idp/output
cp -rf /tmp/app/target /data/idp/output
chown -fR 1001 /data/idp/output

date
echo listing /data/idp/output after mvn and chown 1001 buildoutput
ls -la /data/idp/output/target

date
echo rm -rf /data/idp/buildartifacts and copying artifacts
rm -rf /data/idp/buildartifacts
mkdir -p /data/idp/buildartifacts/
cp -r /data/idp/output/target/liberty/wlp/usr/servers/defaultServer/* /data/idp/buildartifacts/
cp -r /data/idp/output/target/liberty/wlp/usr/shared/resources/ /data/idp/buildartifacts/
cp /data/idp/src/src/main/liberty/config/jvmbx.options /data/idp/buildartifacts/jvm.options

date
echo chown the buildartifacts dir
chown -fR 1001 /data/idp/buildartifacts

date
echo Finished - Full build using container folders
