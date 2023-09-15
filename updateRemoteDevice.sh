# This file should be placed in wssTriggerEnv/wssTrigger so that the update client can call it

echo "cd /home/appval002/amace_validator; git pull; bash linkTests.sh;"
cd /home/appval002/amace_validator
git pull
sleep 100
bash linkTests.sh


