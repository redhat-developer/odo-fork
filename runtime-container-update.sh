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
cp -Rf /home/default/pvc/src/. /home/default/emptydir/src/

date
echo cd /home/default/emptydir/src and listing
cd /home/default/emptydir/src
ls -la

date
echo running full maven build in /home/default/emptydir/src
mvn -B package -DskipLibertyPackage -Dmaven.repo.local=/home/default/emptydir/cache/.m2/repository -DskipTests=true


date
echo listing /data/idp/output after mvn
ls -la /home/default/emptydir/src/target

date
echo copying artifacts to /config/apps
cp -rf /home/default/emptydir/src/target/mpnew-1.0-SNAPSHOT.war /config/apps

date
echo listing /config/apps
ls -la /config/apps

date
echo Finished - Full build using container folders
