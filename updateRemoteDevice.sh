# This file should be placed in wssTriggerEnv/wssTrigger so that the update client can call it
lastdir=$(pwd)
echo "Current at ${lastdir}"
echo "cd /home/appval002/amace_validator; git pull; bash linkTests.sh;"
cd /home/appval002/amace_validator
git pull
sleep 10000
bash linkTests.sh
sleep 1000
cd ${lastdir}

