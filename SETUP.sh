# Runs scripts needed to setup automation

CONFIG="./config.json"
SUDO_PASSWORD=$(jq -r '.SUDO_PASSWORD' "$CONFIG")
# Copy files from Repo to TAST and WSS clients' directories
bash linkTests.sh

# Create and start service
echo "$SUDO_PASSWORD" | sudo -S bash service.sh
echo "$SUDO_PASSWORD" | sudo -S systemctl daemon-reload
echo "$SUDO_PASSWORD" | sudo -S systemctl enable imageserver.service
echo "$SUDO_PASSWORD" | sudo -S systemctl start imageserver.service

# Create and start service
echo "$SUDO_PASSWORD" | sudo -S sudo bash service_wssClient.sh
echo "$SUDO_PASSWORD" | sudo -S sudo systemctl daemon-reload
echo "$SUDO_PASSWORD" | sudo -S sudo systemctl enable wssClient.service
echo "$SUDO_PASSWORD" | sudo -S sudo systemctl start wssClient.service

# Create and start service
echo "$SUDO_PASSWORD" | sudo -S sudo bash service_wssUpdater.sh
echo "$SUDO_PASSWORD" | sudo -S sudo systemctl daemon-reload
echo "$SUDO_PASSWORD" | sudo -S sudo systemctl enable wssUpdater.service
echo "$SUDO_PASSWORD" | sudo -S sudo systemctl start wssUpdater.service