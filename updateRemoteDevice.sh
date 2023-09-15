# This file should be placed in wssTriggerEnv/wssTrigger so that the update client can call it

cd /home/appval002/amace_validator
git pull # update source files from repo
bash /home/appval002/amace_validator/linkTests.sh # Place src files in required location on host device.

echo "cd /home/appval002/amace_validator; git pull; bash linkTests.sh;"

