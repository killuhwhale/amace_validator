#!/bin/bash
# Executing this file will write this content to a service file. This will start the Django server when the machine turns on which allows the automation program to communicate with the host.

CONFIG="./config.json"
SUDO_PASSWORD=$(jq -r '.SUDO_PASSWORD' "$CONFIG")
USERNAME=$(jq -r '.USER' "$CONFIG")
CROS_SDK_PATH=$(jq -r '.CROS_SDK_PATH' "$CONFIG")

cat << EOF > /etc/systemd/system/wssClient.service
[Unit]
Description=wssClient Service
After=network.target

[Service]
User=$USERNAME
Group=www-data
WorkingDirectory=/home/$USERNAME/chromiumos/src/scripts/wssTriggerEnv/wssTrigger
ExecStart=/usr/bin/env bash -c 'echo "$SUDO_PASSWORD" | sudo -S -E  $CROS_SDK_PATH -- /home/${USERNAME}/chromiumos/src/scripts/wssTriggerEnv/bin/python3 /home/${USERNAME}/chromiumos/src/scripts/wssTriggerEnv/wssTrigger/wssClient.py'
Restart=always

[Install]
WantedBy=multi-user.target
EOF