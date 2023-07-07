#!/bin/bash
# tast -verbose run  -var=ui.gaiaPoolDefault=testaccount@gmail.com:PASSWORD $device_address arc.AMACE
# Place in ~/chromiumos/src/scripts

# ./startAMACE.sh -d root@192.168.1.125 -d root@192.168.1.141 -u  http://192.168.1.229:3000/api/amaceResult

function usage {
    echo ""
    echo "Starts automation."
    echo ""
    echo "usage:  -d root@192.168.1.123 -d root@192.168.123 -u http://192.168.1.229:3000/api/amaceResult"
    echo ""
    echo "  -d  string             Device to test on."
    echo "                          (example: root@192.168.1.123 root@192.168.123)"
    echo "  -u  string             Url of server to post results to."
    echo "                          (example: http://192.168.1.229:3000/api/amaceResult)"
    echo ""
}

# Parse command-line options
while getopts ":d:u:" opt; do
  case ${opt} in
    d)
      # Device addresses
      device_addresses+=("$OPTARG")
      ;;
    u)
      # URL
      url=$OPTARG
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

devices=""
for device_address in "${device_addresses[@]}"; do
  devices="${devices} ${device_address}"
done


if [[ -z ${url} ]]; then
  echo "Using default URL"
  python3 ../platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace.py -d "${devices}"
else
  echo "Using URL: ${url}"
  python3 ../platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace.py -d "${devices}" -u "${url}"
fi
