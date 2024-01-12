# Runs scripts need to setup automation


# Copy files from Repo to TAST and WSS clients' directories
bash linkTests.sh

# Create service and start server
sudo bash service.sh
sudo systemctl start imageserver.service
sudo systemctl enable imageserver.service


# Create service and start server
sudo bash service_wssClient.sh
sudo systemctl start wssClient.service
sudo systemctl enable wssClient.service


# Create service and start server
sudo bash service_wssUpdater.sh
sudo systemctl start wssUpdater.service
sudo systemctl enable wssUpdater.service
