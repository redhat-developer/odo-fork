#!/bin/sh

# set -e -o pipefail

date
echo Started - Full build using runtime container folders

date
echo cd /home/default/emptydir/ and listing
cd /home/default/emptydir
ls -la

date
echo running full maven build in /home/default/emptydir/src
mvn -B clean package -Dmaven.repo.local=/home/default/emptydir/.m2/repository -DskipTests=true


date
echo listing /data/idp/output after mvn
ls -la /home/default/emptydir/target

date
echo copying artifacts to /config/
rm -rf /home/default/emptydir/buildartifacts
mkdir -p /home/default/emptydir/buildartifacts/
cp -r  /home/default/emptydir/target/liberty/wlp/usr/servers/defaultServer/* /config/
cp -r  /home/default/emptydir/target/liberty/wlp/usr/shared/resources/ /config/
cp  /home/default/emptydir/src/main/liberty/config/jvmbx.options /config/jvm.options 
ls -la /config/

date
echo start the Liberty runtime
/opt/ibm/wlp/bin/server start

date
echo Finished - Full build using container folders
