#!/bin/bash
# Executing this file will write this content to a service file. This will start the Django server when the machine turns on which allows the automation program to communicate with the host.

cat << EOF > /etc/systemd/system/wssUpdater.service
[Unit]
Description=wssUpdater Service
After=network.target

[Service]
User=appval002
Group=www-data
WorkingDirectory=/home/appval002/chromiumos/src/scripts/wssTriggerEnv/wssTrigger
ExecStart=/usr/bin/env bash -c 'python3 /home/appval002/chromiumos/src/scripts/wssTriggerEnv/wssTrigger/wssUpdater.py'
Restart=always

[Install]
WantedBy=multi-user.target
EOF