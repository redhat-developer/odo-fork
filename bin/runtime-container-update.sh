#!/bin/sh

# set -e -o pipefail

date
echo Started - Update build using runtime container folders

date
echo cd /home/default/idp/src and listing
cd /home/default/idp/src
ls -la

date
echo running full maven build in /home/default/idp/src
mvn -B package -DskipLibertyPackage -Dmaven.repo.local=/home/default/idp/cache/.m2/repository -DskipTests=true


date
echo listing /data/idp/output after mvn
ls -la /home/default/idp/src/target

date
echo copying artifacts to /config/apps
cp -rf /home/default/idp/src/target/mpnew-1.0-SNAPSHOT.war /config/apps

date
echo listing /config/apps
ls -la /config/apps

date
echo Finished - Full build using container folders
