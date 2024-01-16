# Runs scripts needed to setup automation

CONFIG="./config.json"
SUDO_PASSWORD=$(jq -r '.SUDO_PASSWORD' "$CONFIG")
IMAGE_SERVER_DIR=$(jq -r '.IMAGE_SERVER_DIR' "$CONFIG")
WSS_TRIGGER_PATH=$(jq -r '.WSS_TRIGGER_PATH' "$CONFIG")
# Copy files from Repo to TAST and WSS clients' directories


check_jq_installed() {
    if ! command -v jq &> /dev/null; then
        echo "jq is not installed. Please apt install jq to run this script."
        exit 1
    fi
}


check_empty() {
    if [ -z "$1" ]; then
        echo "$2 is empty. Please add $2 to the config.json file. Exiting the script."
        exit 1
    fi
}

check_jq_installed
# Check if the variables are empty
check_empty "$SUDO_PASSWORD" "SUDO_PASSWORD"
check_empty "$IMAGE_SERVER_DIR" "IMAGE_SERVER_DIR"
check_empty "$WSS_TRIGGER_PATH" "WSS_TRIGGER_PATH"


bash linkTests.sh


asSudo() {
    echo "$SUDO_PASSWORD" | sudo -S $1
}


# Create and start service
asSudo "bash service.sh"
asSudo "bash service_wssClient.sh"
asSudo "bash service_wssUpdater.sh"
asSudo "systemctl daemon-reload"
asSudo "systemctl enable imageserver.service"
asSudo "systemctl enable wssClient.service"
asSudo "systemctl enable wssUpdater.service"
asSudo "systemctl start imageserver.service"
asSudo "systemctl start wssClient.service"
asSudo "systemctl start wssUpdater.service"

echo "Install requirements in python venv"
echo " -> $IMAGE_SERVER_DIR"
echo " -> $WSS_TRIGGER_PATH"