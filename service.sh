#!/bin/bash
# Executing this file will write this content to a service file. This will start the Django server when the machine turns on which allows the automation program to communicate with the host.
CONFIG="./config.json"
USERNAME=$(jq -r '.USER' "$CONFIG")
IMAGE_SERVER_DIR=$(jq -r '.IMAGE_SERVER_DIR' "$CONFIG")
IMAGE_SERVER_VENV_BIN=$(jq -r '.IMAGE_SERVER_VENV_BIN' "$CONFIG")


cat << EOF > /etc/systemd/system/imageserver.service
[Unit]
Description=imageserver Service
After=network.target

[Service]
User=$USERNAME
Group=www-data
WorkingDirectory=${IMAGE_SERVER_DIR}
ExecStart=/bin/bash -c '${IMAGE_SERVER_VENV_BIN}/python ${IMAGE_SERVER_DIR}/imageserver/manage.py runserver $(hostname -I | awk '{print $1}'):8000'
Restart=always

[Install]
WantedBy=multi-user.target
EOF