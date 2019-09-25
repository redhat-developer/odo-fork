#!/bin/sh

# set -e -o pipefail

date
echo Started - Full build using runtime container folders

date
echo cd /home/default/emptydir and listing
cd /home/default/emptydir
ls -la

date
echo running full maven build in /home/default/emptydir/src
mvn -B package -DskipLibertyPackage -Dmaven.repo.local=/home/default/emptydir/.m2/repository -DskipTests=true


date
echo listing /data/idp/output after mvn
ls -la /home/default/emptydir/target

date
echo copying artifacts to /config/apps
cp -rf /home/default/emptydir/target/mpnew-1.0-SNAPSHOT.war /config/apps

date
echo listing /config/apps
ls -la /config/apps

date
echo Finished - Full build using container folders
