#!/bin/bash
# Executing this file will write this content to a service file. This will start the Django server when the machine turns on which allows the automation program to communicate with the host.

cat << EOF > /etc/systemd/system/wssClient.service
[Unit]
Description=wssClient Service
After=network.target

[Service]
User=appval002
Group=www-data
WorkingDirectory=/home/appval002/chromiumos/src/scripts/wssTriggerEnv/wssTrigger
ExecStart=/usr/bin/env bash -c 'echo "testminnie123" | sudo -S -E /home/appval002/dtools/depot_tools/cros_sdk -- /home/appval002/chromiumos/src/scripts/wssTriggerEnv/bin/python3 /home/appval002/chromiumos/src/scripts/wssTriggerEnv/wssTrigger/wssClient.py'
Restart=always

[Install]
WantedBy=multi-user.target
EOF