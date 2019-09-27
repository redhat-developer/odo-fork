#!/bin/sh

# set -e -o pipefail

date
echo Started - Full build using runtime container folders

date
echo cd /home/default/idp/src/ and listing
cd /home/default/idp/src
ls -la

date
echo running full maven build in /home/default/idp/src
mvn -B clean package -Dmaven.repo.local=/home/default/idp/cache/.m2/repository -DskipTests=true


date
echo listing /home/default/idp/src/target after mvn
ls -la /home/default/idp/src/target

date
echo copying artifacts to /config/
rm -rf /config/*
cp -r  /home/default/idp/src/target/liberty/wlp/usr/servers/defaultServer/* /config/
cp -r  /home/default/idp/src/target/liberty/wlp/usr/shared/resources/ /config/
cp  /home/default/idp/src/src/main/liberty/config/jvmbx.options /config/jvm.options 
ls -la /config/

date
echo start the Liberty runtime
/opt/ibm/wlp/bin/server start

date
echo Finished - Full build using container folders
