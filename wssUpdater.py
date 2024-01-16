"""
Location:
    f"/home/{USER}/chromiumos/src/scripts/wssTriggerEnv/wssTrigger"

Useage:
   SUDO_PASSWORD=HOST_USER_PASSWORD python3 wssUpdater.py
"""
import asyncio
import os
import subprocess
import threading
import time
import websockets
from amace_helpers import line_start, req_env_var, encode_jwt, CONFIG, ping, pj, get_server_wss_url, CHROMEOS_SCRIPTS
import logging

LOG_DIR = f"{CHROMEOS_SCRIPTS}/.config/amaceValidator/logs"  # Replace with your log directory path
if not os.path.exists(LOG_DIR):
    os.makedirs(LOG_DIR)
log_file = os.path.join(LOG_DIR, 'wssUpdater.log')
logging.basicConfig(filename=log_file, level=logging.DEBUG,
                    format='%(asctime)s:%(levelname)s:%(message)s')

current_websocket = None  # Global variable to hold the current WebSocket
process_event = threading.Event()

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


def cmd():
    return ["bash", "updateRemoteDevice.sh"]


def run_process(cmd, wssToken):
    global process_event
    global current_websocket

    process_event.set()
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    while process.poll() is None:  # While the process is still running
        output = ""
        try:
            output = process.stdout.readline()
        except Exception as err:
            print("Error decoding message and sending progress: ", err)
            output = process.stdout.readline()

        log_msg(wssToken, f"progress:{line_start}{output}")
        time.sleep(.5)

    process_event.clear()
    log_msg(wssToken, f"{line_start} Update completed!")


# Called to "stop" the wssClient.service when user presses "Stop Run"
def restart_wssClient_service(wssToken):
    SUDO_PASSWORD = CONFIG["SUDO_PASSWORD"]
    cmd = ['sudo', '-S', 'systemctl', 'restart', 'wssClient.service']

    proc = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    async_log_msg(wssToken, f"{line_start} Restart wssClient.service: {proc.stdout=}, {proc.stderr=}")

    stdout, stderr = proc.communicate(input=f"{SUDO_PASSWORD}\n")
    async_log_msg(wssToken, f"{line_start} Restart wssClient.service: {stdout=}, {stderr=}")


async def listen_to_ws():
    global current_websocket
    global process_event

    host_device_name = CONFIG["HOST_DEVICE_NAME"]
    secret = CONFIG["AMACE_JWT_SECRET"]
    wssToken = encode_jwt({"email": "wssUpdater@ggg.com"}, secret)
    uri = get_server_wss_url()
    async_log_msg(wssToken, f"{line_start} Device: {host_device_name} is using URI: {uri}")

    while True:
        try:
            async with websockets.connect(uri) as websocket:
                current_websocket = websocket
                while True:
                    mping = pj(await websocket.recv())
                    message = mping['msg']
                    data = mping['data']

                    if not message.startswith("progress:"):
                        async_log_msg(wssToken, f"{line_start} Received message: {message}")

                    if message == f"update_{host_device_name}":
                        if not process_event.is_set():
                            start_cmd = cmd()
                            async_log_msg(wssToken, f"{line_start} wssUpdater using {start_cmd=}")
                            thread = threading.Thread(
                                target=run_process,
                                args=(start_cmd, wssToken, )
                            )
                            thread.start()
                            async_log_msg(wssToken, f"{line_start} Update started!")
                            async_log_msg(wssToken, f"updating:{host_device_name}")
                        else:
                            async_log_msg(wssToken, f"{line_start} Update in progress!")

                    elif message.startswith(f"stoprun_{host_device_name}"):
                        restart_wssClient_service(wssToken)
                        async_log_msg(wssToken, f"runstopped:updater:{host_device_name}")

        except websockets.ConnectionClosed:
            async_log_msg(wssToken, f"{line_start} Connection with the server was closed. Retrying in 5 seconds...")
        except Exception as e:
            async_log_msg(wssToken, f"{line_start} An error occurred: {e}. Retrying in 5 seconds...")

        await asyncio.sleep(5)  # Wait for 5 seconds before trying to rec

if __name__ == "__main__":
    loop = asyncio.get_event_loop()
    loop.run_until_complete(listen_to_ws())
    loop.run_forever()




