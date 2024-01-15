"""
Executed via systemd service.

Location:
    f"/home/{USER}/chromiumos/src/scripts/wssTriggerEnv/wssTrigger"

Useage:
   python3 wssClient.py
"""

import asyncio
import logging
import os
import subprocess
import threading
import time
import websockets
from amace_helpers import line_start, encode_jwt, CONFIG, ping, pj, USER, get_server_wss_url, CHROMEOS_SCRIPTS

LOG_DIR = f"{CHROMEOS_SCRIPTS}/.config/amaceValidator/logs"  # Replace with your log directory path
if not os.path.exists(LOG_DIR):
    os.makedirs(LOG_DIR)
log_file = os.path.join(LOG_DIR, 'application.log')
logging.basicConfig(filename=log_file, level=logging.DEBUG,
                    format='%(asctime)s:%(levelname)s:%(message)s')

current_websocket = None  # Global variable to hold the current WebSocket
process_event = threading.Event() # Track if the process is already running

def get_d_src_type(playstore: bool):
    return "playstore" if playstore else "pythonstore"

def make_device_args(ips):
    return ["-d", ips]

def cmd(devices, dsrcpath, dsrctype):
    return [
        f"{CONFIG['WSS_TRIGGER_PATH']}/bin/python3",
        f"/home/{USER}/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/amace.py",
        "-w", CONFIG['SKIP_WINDOW_CHECK'], # "Skip amace window check. t|f"
        "-b", CONFIG['SKIP_BROKEN'], # "Skip broken check. t|f"
        "-l", CONFIG['SKIP_LOGIN'], # "Skip login. t|f"
        "--dsrcpath", f"AppLists/{dsrcpath}",
        "--dsrctype", dsrctype,
    ] + make_device_args(devices)

async def async_log_msg(wssToken, msg):
    global current_websocket
    socket_msg = ping(msg, {}, wssToken)
    try:
        print(line_start, msg)
        logging.debug(msg)
        if current_websocket:
            await current_websocket.send(socket_msg)
    except Exception as err:
        logging.debug(f"Error: {str(err)}")
        logging.debug(f"Exception type: {type(err).__name__}")


def log_msg(wssToken, msg):
    global current_websocket
    socket_msg = ping(msg, {}, wssToken)
    try:
        print(line_start, msg)
        logging.debug(msg)
        if current_websocket:
            asyncio.run(current_websocket.send(socket_msg))
    except Exception as err:
        logging.debug(f"Error: {str(err)}")
        logging.debug(f"Exception type: {type(err).__name__}")


def run_process(cmd, wssToken):
    global process_event
    global current_websocket

    process_event.set()
    process = None
    try:
        # Use Popen to start the process without blocking
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    except Exception as err:
        logging.debug(f"Error starting process! {err}")

    output = ""
    while process.poll() is None:  # While the process is still running
        try:
            output = process.stdout.readline()
        except Exception as err:
            print("Error decoding message and sending progress: ", err)
            logging.debug("Error decoding message and sending progress: ", err)
            output = process.stdout.readlines() + process.stderr.readlines()

        log_msg(wssToken, f"progress:{line_start}{output.decode('utf-8')}")
        time.sleep(.3)  # Sleep for a short duration before checking again

    logging.debug("Process done!")
    process_event.clear()

    # Send a message over the websocket after the process completes
    log_msg(wssToken, f"{line_start} Process completed!")


async def listen_to_ws():
    global current_websocket
    global process_event

    host_device_name = CONFIG["HOST_DEVICE_NAME"]
    jwt_secret = CONFIG["AMACE_JWT_SECRET"]
    wssToken = encode_jwt({"email": "wssClient@ggg.com"}, jwt_secret)
    uri = get_server_wss_url()
    log_msg(wssToken, f"{line_start}{host_device_name=} is using URI: {uri} w/ {jwt_secret=}")

    while True:
        try:
            async with websockets.connect(uri) as websocket:
                current_websocket = websocket
                while True:
                    mping = pj(await websocket.recv())
                    message = mping['msg']
                    data = mping['data']
                    await async_log_msg(wssToken, f"{line_start} Received message: {message}")

                    if message == f"startrun_{host_device_name}":
                        if not process_event.is_set():
                            start_cmd = cmd(
                                        data['devices'],
                                        data['listname'],
                                        get_d_src_type(data['playstore']))

                            await async_log_msg(wssToken, f"{line_start} using start command: {start_cmd}")
                            thread = threading.Thread(
                                target=run_process,
                                args=(start_cmd, wssToken, )
                            )
                            thread.start()

                            await async_log_msg(wssToken, f"runstarted:{host_device_name}")
                        else:
                            await async_log_msg(wssToken, f"runstarted:{host_device_name}:runinprogress")
                    elif message == f"querystatus_{host_device_name}":
                        status_msg =  "running" if process_event.is_set() else "stopped"
                        await async_log_msg(wssToken, f"status:{host_device_name}:{status_msg}")
                    elif message == "getdevicename":
                        await async_log_msg(wssToken, f"getdevicename:{host_device_name}")

        except websockets.ConnectionClosed:
            log_msg(wssToken, f"{line_start} Connection with the server was closed. Retrying in 5 seconds...")
        except Exception as e:
            log_msg(wssToken, f"{line_start} An error occurred: {e}. Retrying in 5 seconds...")

        # Wait for 5 seconds before trying to reconnect.
        await asyncio.sleep(5)

if __name__ == "__main__":
    # Run the program using an asyncio event loop
    loop = asyncio.get_event_loop()
    loop.run_until_complete(listen_to_ws())
    loop.run_forever()
