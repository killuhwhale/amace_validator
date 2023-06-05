#!/bin/bash
# tast -verbose run  -var=ui.gaiaPoolDefault=testaccount@gmail.com:PASSWORD $device_address arc.AMACE
# Place in ~/chromiumos/src/scripts

device_address=$1
url=$2

if [[ -z ${device_address} ]]; then
  echo "Device address is missing. Please provide the device address in the format 'user@123.123.1.123'."
  exit 1
fi

if [[ -z ${url} ]]; then
  echo "Using default URL"
  python3 ../platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace.py -d "${device_address}"
else
  echo "Using URL: ${url}"
  python3 ../platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace.py -d "${device_address}" -u "${url}"
fi
