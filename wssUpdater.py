import asyncio
import json
import sys
import jwt
import os
import ipaddress
import subprocess
import threading
import time
import websockets
from amace_helpers import line_start, req_env_var, encode_jwt, CONFIG, ping, pj, get_server_wss_url

"""
Location:
    f"/home/{USER}/chromiumos/src/scripts/wssTriggerEnv/wssTrigger"

Useage:
   SUDO_PASSWORD=HOST_USER_PASSWORD python3 wssUpdater.py
"""

SUDO_PASSWORD = os.environ.get("SUDO_PASSWORD")
req_env_var(SUDO_PASSWORD, "Host device password. E.g: appval002's password", 'SUDO_PASSWORD')

process_event = threading.Event()
current_websocket = None  # Global variable to hold the current WebSocket

def cmd():
    return ["bash", "updateRemoteDevice.sh"]

def run_process(cmd, wssToken):
    global process_event
    global current_websocket
    # global exit_signal

    process_event.set()
    # Use Popen to start the process without blocking

    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    while process.poll() is None:  # While the process is still running
        output = ""
        try:
            # output = process.stdout.readline().decode("utf-8").strip("\n")
            output = process.stdout.readline()
            print(line_start, "Progress: ", output)
        except Exception as err:
            print("Error decoding message and sending progress: ", err)
            output = process.stdout.readline()

        if current_websocket:
            asyncio.run(current_websocket.send(ping(f"progress:{line_start}{output}", {}, wssToken)))

        time.sleep(.1)  # Sleep for a short duration before checking again

    process_event.clear()
    # exit_signal.clear()

    # Send a message over the websocket after the process completes
    if current_websocket:
        print(line_start, "Process completed!")
        asyncio.run(current_websocket.send(ping("Process completed!", {}, wssToken)))


# Called to "stop" the wssClient.service when user presses "Stop Run"
def restart_wssClient_service(pswd):
    cmd = ['sudo', '-S', 'systemctl', 'restart', 'wssClient.service']

    proc = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    stdout, stderr = proc.communicate(input=pswd + '\n')

    print("restart_wssClient_service: ")
    print("stdout: ", proc.stdout)
    print("stderr: ", proc.stderr)
    print("stdout: ", stdout)
    print("stderr: ", stderr)


async def listen_to_ws():
    global cmd

    global SUDO_PASSWORD
    global current_websocket
    global process_event

    secret = CONFIG["AMACE_JWT_SECRET"]
    DEVICE_NAME = CONFIG["HOST_DEVICE_NAME"]
    wssToken = encode_jwt({"email": "wssUpdater@ggg.com"}, secret)

    uri = get_server_wss_url()

    print(line_start, f"Device: {DEVICE_NAME} is using URI: ", uri)
    while True:
        try:
            # The connection will persist as long as the server keeps it open
            async with websockets.connect(uri) as websocket:
                current_websocket = websocket
                while True:
                    mping = pj(await websocket.recv())
                    message = mping['msg']
                    data = mping['data']

                    if not message.startswith("progress:"):
                        print(line_start, f"Received message: {message} ")

                    if message == f"update_{DEVICE_NAME}":
                        # Check if the process is not already running
                        if not process_event.is_set():
                            start_cmd = cmd()
                            print(line_start, "using start command: ", start_cmd)
                            thread = threading.Thread(
                                target=run_process,
                                args=(start_cmd, wssToken, )
                            )
                            thread.start()
                            print(line_start, "Update started!")
                            await websocket.send(ping(f"updating:{DEVICE_NAME}", {}, wssToken))
                        else:
                            print(line_start, "Update in progress!")
                            await websocket.send(ping(f"updating:{DEVICE_NAME}:updateinprogress", {}, wssToken))
                    elif message.startswith(f"stoprun_{DEVICE_NAME}"):
                        print(line_start, "Run stopping call restart wssClient.service....")
                        restart_wssClient_service(SUDO_PASSWORD)
                        await websocket.send(ping(f"runstopped:updater:{DEVICE_NAME}", {}, wssToken))

        except websockets.ConnectionClosed:
            print(line_start, "Connection with the server was closed. Retrying in 5 seconds...")
        except Exception as e:
            print(line_start, f"An error occurred: {e}. Retrying in 5 seconds...")

        await asyncio.sleep(5)  # Wait for 5 seconds before trying to rec

if __name__ == "__main__":
    # Run the program using an asyncio event loop
    print("wssUpdater.py starting...")
    loop = asyncio.get_event_loop()
    loop.run_until_complete(listen_to_ws())
    loop.run_forever()




