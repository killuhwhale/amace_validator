
CONFIG="./config.json"
SUDO_PASSWORD=$(jq -r '.SUDO_PASSWORD' "$CONFIG")


asSudo() {
    echo "$SUDO_PASSWORD" | sudo -S $1
}


asSudo "systemctl stop imageserver.service"
asSudo "systemctl stop wssClient.service"
asSudo "systemctl stop wssUpdater.service"
asSudo "rm /etc/systemd/system/imageserver.service"
asSudo "rm /etc/systemd/system/wssClient.service"
asSudo "rm /etc/systemd/system/wssUpdater.service"

asSudo "systemctl daemon-reload"
