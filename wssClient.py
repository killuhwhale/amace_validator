import asyncio
import os
import select
import signal
import subprocess
import threading
import websockets
import psutil

DEVICE_NAME = value = os.environ.get('DNAME')

process = None
current_websocket = None  # Global variable to hold the current WebSocket
process_event = threading.Event()

ip_address = "192.168.1.125"
account = "tastarcplusplusappcompat14@gmail.com:1Z5-LT4Q1337 "
USER = os.environ.get("USER")
cmd = [
        "python3",
        f"/home/{USER}/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/amace.py",
        "-d", ip_address,
        "-a", account,
        "-p", f"/home/{USER}/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_secret.txt",
        "-l", "t",
]
# cmd =["sleep", "30"]

def kill_proc_tree(pid, including_parent=True):
    global process
    parent = psutil.Process(pid)
    children = parent.children(recursive=True)
    for child in children:
        child.terminate()
    gone, still_alive = psutil.wait_procs(children, timeout=5)
    if including_parent:
        parent.terminate()
        parent.wait(5)
    process = None

def run_process(cmd):
    global process_event
    global current_websocket

    process_event.set()
    subprocess.run(cmd)
    process_event.clear()

    # Send a message over the websocket after the process completes
    if current_websocket:
        print("Process completed!")
        asyncio.run(current_websocket.send("Process completed!"))

async def listen_to_ws():
    """TODO()

    So far we have a system where we can query for all devices running the client program.

    Then we can get the status of device (running automatuion or not), start & stop automation.

    Callback when automation is done. Reconnecting socket if server does down.

    Point of failure:

        Maybe we need a server based system with firebase to monitor the progress
            - we can send a message to firebase saying we are in progress'
            - If no progress is made within 10 mins we can check to see if device is online, check status, stop if neccessary, then restart.

            - We then would then a way to start off at a certain package.
                - We would need to pipe this through to Amace.py

        1. Server VM -Beginning of Transcaction
            - host website and Websocket server
                - If this goes down, communication stops but automation continues.

        2. Host/ Lab Device - Receives start signal and begins running automation
            - Device turns off, loses wifi
                - Only way to fail automation without automatic recovery is when the device loses power or wifi.
                    - If device turns off or loses wifi, what do we do?

            # Should be robust against programming errors....
            - WSS Program will reconnect to socket for communcation
            - TAST Python program will monitor, and finishes runs

        3. Dut
            - If device loses power or wifi
                - as long as its connected to power and previously connected to wifi it should persist and handle errors.


    """
    global process
    global cmd
    global DEVICE_NAME
    global current_websocket
    global process_event

    uri = "ws://localhost:3001/wss/"
    uri = "wss://appvaldashboard.com/wss/"
    print(f"Device: {DEVICE_NAME} is using URI: ", uri)
    while True:
        try:
            # The connection will persist as long as the server keeps it open
            async with websockets.connect(uri) as websocket:
                current_websocket = websocket
                while True:
                    message = await websocket.recv()
                    print(f"Received message: {message} ")
                    if message == f"startrun_{DEVICE_NAME}":
                        if process is None or process.poll() is not None:  # Start only if not already running
                            # process = subprocess.Popen(cmd)
                            if not process_event.is_set():  # Check if the process is not already running
                                thread = threading.Thread(target=run_process, args=(cmd,))
                                thread.start()
                            print("Run started!")
                            await websocket.send(f"runstarted:{DEVICE_NAME}")
                        else:
                            print("Run in progress!")
                            await websocket.send(f"runstarted:{DEVICE_NAME}:runinprogress")
                    elif message == f"querystatus_{DEVICE_NAME}":
                        status_msg =  "running" if process and process.poll() is None is None else "stopped"
                        status = f"status:{DEVICE_NAME}:{status_msg}"
                        print("Sending status: ", status)
                        await websocket.send(status)
                    elif message == "getdevicename":
                        print("Sending name: ", DEVICE_NAME)
                        await websocket.send(f"getdevicename:{DEVICE_NAME}")

                    elif message == f"stoprun_{DEVICE_NAME}":
                        print("Run stopped!")
                        if process and process.poll() is None:  # Check if process is running
                            kill_proc_tree(process.pid)
                            print("Run stopped!")
                            await websocket.send(f"runstopped:{DEVICE_NAME}")
                    elif not process is None:
                        print("We can print out the output from process here every 2s...", process)


        except websockets.ConnectionClosed:
            print("Connection with the server was closed. Retrying in 5 seconds...")
        except Exception as e:
            print(f"An error occurred: {e}. Retrying in 5 seconds...")

        await asyncio.sleep(5)  # Wait for 5 seconds before trying to rec

if __name__ == "__main__":
    # Run the program using an asyncio event loop
    loop = asyncio.get_event_loop()
    loop.run_until_complete(listen_to_ws())
    loop.run_forever()
