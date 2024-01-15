#!/bin/bash
# Executing this file will write this content to a service file. This will start the Django server when the machine turns on which allows the automation program to communicate with the host.

CONFIG="./config.json"
SUDO_PASSWORD=$(jq -r '.SUDO_PASSWORD' "$CONFIG")
USERNAME=$(jq -r '.USER' "$CONFIG")
WSS_TRIGGER_PATH=$(jq -r '.WSS_TRIGGER_PATH' "$CONFIG")
cat << EOF > /etc/systemd/system/wssUpdater.service
[Unit]
Description=wssUpdater Service
After=network.target

[Service]
User=$USERNAME
Group=www-data
WorkingDirectory=/home/$USERNAME/chromiumos/src/scripts/wssTriggerEnv/wssTrigger
ExecStart=/usr/bin/env bash -c '$WSS_TRIGGER_PATH/bin/python3 /home/${USERNAME}/chromiumos/src/scripts/wssTriggerEnv/wssTrigger/wssUpdater.py'
Restart=always

[Install]
WantedBy=multi-user.target
EOF