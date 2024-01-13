# Runs scripts needed to setup automation

# Copy files from Repo to TAST and WSS clients' directories
bash linkTests.sh

# Create and start service
sudo bash service.sh
sudo systemctl daemon-reload
sudo systemctl enable imageserver.service
sudo systemctl start imageserver.service

# Create and start service
sudo bash service_wssClient.sh
sudo systemctl daemon-reload
sudo systemctl enable wssClient.service
sudo systemctl start wssClient.service

# Create and start service
sudo bash service_wssUpdater.sh
sudo systemctl daemon-reload
sudo systemctl enable wssUpdater.service
sudo systemctl start wssUpdater.service
