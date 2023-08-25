import asyncio
import os
import select
import signal
import subprocess
import threading
import time
import websockets
import psutil

DEVICE_NAME = value = os.environ.get('DNAME')
exit_signal = threading.Event()
process_event = threading.Event()
current_websocket = None  # Global variable to hold the current WebSocket
ip_address = "192.168.1.125"
account = "email@gmail.com:password"
USER = os.environ.get("USER")
cmd = [
        "python3",
        f"/home/{USER}/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/amace.py",
        "-d", ip_address,
        "-a", account,
        "-p", f"/home/{USER}/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_secret.txt",
        "-l", "t",
        "--dsrcpath", "AppLists/bstar",
        "--dsrctype", "pythonstore",
]
# cmd =["sleep", "30"]

def kill():
    exit_signal.set()

def kill_proc_tree(pid, including_parent=True):
    print("kill proc tree")
    parent = psutil.Process(pid)
    children = parent.children(recursive=True)
    for child in children:
        print("Terminating child: ", child)
        child.terminate()
    gone, still_alive = psutil.wait_procs(children, timeout=5)
    if including_parent:
        parent.terminate()
        parent.wait(5)

def run_process(cmd):
    global process_event
    global current_websocket
    global exit_signal

    process_event.set()
    # Use Popen to start the process without blocking
    process = subprocess.Popen(cmd)

    while process.poll() is None:  # While the process is still running
        if exit_signal.is_set():  # Check if exit signal is set
            print("TERMINATING PROCESS")
            kill_proc_tree(process.pid)
            break
        time.sleep(1)  # Sleep for a short duration before checking again

    process_event.clear()
    exit_signal.clear()

    # Send a message over the websocket after the process completes
    if current_websocket:
        print("Process completed!")
        asyncio.run(current_websocket.send("Process completed!"))


async def listen_to_ws():
    """TODO()

    Statuses:
        STARTED

    Create and endpoint that we can send a runID to
        - We will first start by Checking in the brand new Run via RUN ID and a status STARTED

        When done we send SUCCUSS

        If something fails


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
    global cmd
    global DEVICE_NAME
    global current_websocket
    global process_event

    uri = "wss://appvaldashboard.com/wss/"
    uri = "ws://localhost:3001/wss/"
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
                        if not process_event.is_set():  # Check if the process is not already running
                            thread = threading.Thread(target=run_process, args=(cmd,))
                            thread.start()
                            print("Run started!")
                            await websocket.send(f"runstarted:{DEVICE_NAME}")
                        else:
                            print("Run in progress!")
                            await websocket.send(f"runstarted:{DEVICE_NAME}:runinprogress")
                    elif message == f"querystatus_{DEVICE_NAME}":
                        status_msg =  "running" if process_event.is_set() else "stopped"
                        status = f"status:{DEVICE_NAME}:{status_msg}"
                        print("Sending status: ", status)
                        await websocket.send(status)
                    elif message == "getdevicename":
                        print("Sending name: ", DEVICE_NAME)
                        await websocket.send(f"getdevicename:{DEVICE_NAME}")

                    elif message == f"stoprun_{DEVICE_NAME}":
                        print("Run stopping....")
                        if process_event.is_set():  # Check if process is running
                            kill()
                            print("Run stopped!")
                            await websocket.send(f"runstopped:{DEVICE_NAME}")
                    # elif not thread is None:
                    #     print("We can print out the output from process here every 2s...", thread)


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




