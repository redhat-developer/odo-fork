#!/bin/sh

# set -e -o pipefail

echo Started - Update build using volume folders
date 
chown -R 1001 /data/idp/src 
echo chown done 

cd /data/idp/src 

echo Run update maven build directly from volume (/data/idp/src)
date 
mvn -B package -DskipLibertyPackage -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true 

date
echo maven build finished 
ls -la target 

echo starting copy of war 
date 
cp target/mpnew-1.0-SNAPSHOT.war /data/idp/buildartifacts/apps 
date 

echo chown the buildartifacts dir 
chown 1001 /data/idp/buildartifacts/apps/mpnew-1.0-SNAPSHOT.war 

echo list /data/idp/buildartifacts/apps 
ls -al /data/idp/buildartifacts/apps 

echo Finished - Update build using volume folders
date
