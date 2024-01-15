# This file should be placed in wssTriggerEnv/wssTrigger so that the update client can call it
CONFIG="./config.json"
SUDO_PASSWORD=$(jq -r '.SUDO_PASSWORD' "$CONFIG")
IMAGE_SERVER_DIR=$(jq -r '.IMAGE_SERVER_DIR' "$CONFIG")

asSudo() {
    echo "$SUDO_PASSWORD" | sudo -S $1
}


lastdir=$(pwd)
echo "Current at ${lastdir}"
echo "cd ${IMAGE_SERVER_DIR}; git pull; bash linkTests.sh;"
cd "${IMAGE_SERVER_DIR}"

# get updates
git pull
sleep 2

# update files
bash linkTests.sh

# reset services
asSudo "systemctl restart imageserver.service"
asSudo "systemctl restart wssClient.service"
asSudo "systemctl restart wssUpdater.service"

sleep 1
cd ${lastdir}
echo "Done updating"

