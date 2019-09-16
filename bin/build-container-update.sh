#!/bin/sh

# set -e -o pipefail

date
echo Started - Update build using container folders

echo listing /data/idp/src
ls -la /data/idp/src

date
echo copying /data/idp/src to /tmp/app
cp -Rf /data/idp/src/. /tmp/app

date
echo chown -fR 1001 /tmp/app and listing
chown -fR 1001 /tmp/app
cd /tmp/app
ls -la

date
echo running update maven build in /tmp/app
mvn -B package -DskipLibertyPackage -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true

date
echo copying artifacts to /data/idp/buildartifacts/apps 

chown -fR 1001 /tmp/app/target
cp -rf /tmp/app/target/mpnew-1.0-SNAPSHOT.war /data/idp/buildartifacts/apps 

date
echo listing /data/idp/buildartifacts/apps after build and chown 1001 buildoutput
ls -la /data/idp/buildartifacts/apps

date
echo chown the buildartifacts apps dir
chown -fR 1001 /data/idp/buildartifacts/apps

date
echo Finished - Update build using container folders
