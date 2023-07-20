# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Starts Tast test and monitors progress. Restarts if TAST fails until app list is exhausted.


'''
tast -verbose run -var=arc.amace.posturl=http://xyz.com -var=arc.amace.hostip=http://192.168.1.123  -var=arc.amace.device=root@192.168.1.456 -var=amace.runts=123 -var=amace.runid=123  -var=ui.gaiaPoolDefault=email@gmail.com:password root@192.168.1.238 arc.AMACE
./startAMACE.sh -d root@192.168.1.125 -d root@192.168.1.141 -a email@gmail.com:password
'''
import argparse
import json

import requests
import subprocess
import uuid


from collections import defaultdict
from dataclasses import dataclass
from multiprocessing import Process
from time import sleep, time
from typing import List



Red = "\033[31m"
Black = "\033[30m"
Green = "\033[32m"
Yellow = "\033[33m"
Blue = "\033[34m"
Purple = "\033[35m"
Cyan = "\033[36m"
White = "\033[37m"
RESET = "\033[0m"

def p_red(*args, end='\n'):
    print(Red, *args, RESET, end=end)

def p_green(*args, end='\n'):
    print(Green, *args, RESET, end=end)

def p_yellow(*args, end='\n'):
    print(Yellow, *args, RESET, end=end)

def p_blue(*args, end='\n'):
    print(Blue, *args, RESET, end=end)

def p_purple(*args, end='\n'):
    print(Blue, *args, RESET, end=end)

def p_cyan(*args, end='\n'):
    print(Cyan, *args, RESET, end=end)

@dataclass
class RequestBody:
    """Request data for app error. Reflects amace.go and backend."""
    runID: str
    runTS: str
    pkgName: str
    appName: str
    status: int
    isGame: bool
    appTS: int
    buildInfo: str
    deviceInfo: str

class AMACE:
    """Runs TAST test to completion.

    If the test fails early for any reason, the test will be re run.
    """

    def __init__(self, device: str, BASE_URL: str, host_ip: str, run_id: str, run_ts: int, test_account: str):
        self.__test_account = test_account
        self.__device = device
        self.__current_package = ""
        self.__BASE_URL = BASE_URL
        self.__host_ip = host_ip
        self.__run_finished = False
        self.__log_error = False
        self.__package_retries = defaultdict(int)
        self.__packages = defaultdict(int)
        self.__package_arr = []
        self.__api_key = None
        self.__run_id = run_id
        self.__run_ts = run_ts
        self.__request_body = None
        self.__get_apps()
        self.__get_api_key()

    def __get_apps(self):
        """Get apps from file."""
        with open("../platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_app_list.tsv", 'r', encoding="utf-8") as f:
            for idx, l in enumerate(f.readlines()):
                pkg = l.split("\t")[1].replace("\n", "")
                self.__package_arr.append(pkg)
                self.__packages[pkg] = idx

    def __get_api_key(self):
        """Get api key from file."""
        with open("../platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_secret.txt", 'r', encoding="utf-8") as f:
            self.__api_key = f.readline()

    def __get_next_app(self, pkg: str) -> str:
        """Gets next app given a package name.

            Used when a TAST test fails too many times on the same app.

            Args:
                pkg: Package name of the last app.
            Returns:
                The app's package name that is next in the list.
        """
        if self.__packages[pkg] + 1 >= len(self.__packages.keys()):
            self.__run_finished = True
            return ""
        return self.__package_arr[self.__packages[pkg] + 1]

    def __split_app_result(self, msg: str):
        """Splits and stores App/run info."""
        info = msg.split("|~|")
        p_cyan(f"App info picked up: ", info)
        self.__current_package = info[3]
        self.__request_body = RequestBody(
            runID = info[1],
            runTS = info[2],
            pkgName = info[3],
            appName = info[4],
            status = info[5],
            isGame = info[6],
            appTS = info[7],
            buildInfo = info[8],
            deviceInfo = info[9],
        )

    def __run_command(self, command):
        """Runs command and processes output.

            Used to start and monitor TAST test.
        """
        with subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT) as process:
            # Read output in real-time and log it
            for line in iter(process.stdout.readline, b''):
                msg = line.decode().strip()
                if "--appstart@" in msg:
                    self.__split_app_result(msg)

                if "--~~rundone" in msg:
                    self.__run_finished = True

                # Error output from TAST when test fails to complete.
                if "Error: Test did not finish" in msg:
                    self.__log_error = True
                print(msg)

            # Wait for the process to complete and get the return code
            process.wait()
            return process.returncode

    def __run_tast(self):
        """Command for the TAST test with required params."""
        cmd = ("tast", "-verbose", "run",  f"-var=arc.amace.device={self.__device}", f"-var=arc.amace.hostip={self.__host_ip}", f"-var=arc.amace.posturl={self.__BASE_URL}" , f"-var=arc.amace.startat={self.__current_package}", f"-var=arc.amace.runts={self.__run_ts}", f"-var=arc.amace.runid={self.__run_id}", f"-var=ui.gaiaPoolDefault={self.__test_account}", f"-var=arc.amace.account={self.__test_account}" , self.__device, "arc.AMACE")

        return self.__run_command(cmd)

    def __post_err(self):
        """Sends post request to backed to store result in Firebase when an error happens."""
        print(f"Posting error from python ")
        headers = {'Authorization': self.__api_key}
        res = requests.post(self.__BASE_URL, data=self.__request_body.__dict__, headers=headers)
        print(f"{res=}")

    def start(self):
        """Starts the TAST test and ensures it completes."""
        N = len(self.__packages)
        print("Num apps to test: ", N)
        print("Running tests now!")

        while not self.__run_finished:
            p_green(f"Starting a TAST run with {self.__current_package=}")
            self.__run_tast()
            if not self.__run_finished:
                self.__package_retries[self.__current_package] += 1
                if self.__package_retries[self.__current_package] > 1:
                    if self.__log_error:
                        self.__post_err()
                    self.__current_package = self.__get_next_app(self.__current_package)
            p_red(f"Tast run over with: {self.__current_package=}")


def get_local_ip():
    '''
    '''
    result = subprocess.run(['ifconfig'], capture_output=True, text=True)
    output = result.stdout
    s = "192.168.1."
    try:
        idx = output.index(s)
        idx += len(s)
        return f"192.168.1.{output[idx:idx+3]}"
    except Exception as err:
        pass

    s = "192.168.0."
    try:
        idx = output.index(s)
        idx += len(s)
        return f"192.168.1.{output[idx:idx+3]}"
    except Exception as err:
        print("Errir: ", err)

    return ""


def load_apps():
    apps = fetch_apps()
    write_apps(apps)

def fetch_apps():
    '''Fetch apps from backend. NextJS -> FirebaseDB'''
    headers = {"Authorization": read_secret()}
    # res = requests.get("http://localhost:3000/api/applist", headers=headers)
    res = requests.get(f"https://appval-387223.wl.r.appspot.com/api/applist", headers=headers)
    result = json.loads(res.text)

    s = result['data']['data']['apps']
    results = s.replace("\\t", "\t").split("\\n")
    print(f"{results=}")
    return results

def write_apps(apps: List[str]):
    '''Overwrite /home/USER/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_app_list.tsv
        platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/amace.py
    '''
    filepath = f"../platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_app_list.tsv"
    with open(filepath, "w", encoding="utf-8") as f:
        for idx, line in enumerate(apps):
            if idx == len(apps) - 1:
                f.write(f"{line}")  # dont not write empty line on last entry
            else:
                f.write(f"{line}\n")

def read_secret():
    """Get api key from file."""
    secret = ""
    with open("../platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data/AMACE_secret.txt", 'r', encoding="utf-8") as f:
        secret = f.readline()
    return secret


def task(device: str, url, host_ip, run_id, run_ts, test_account):
    amace = AMACE(device=device.strip(), BASE_URL=url, host_ip=host_ip, run_id=run_id, run_ts=run_ts, test_account=test_account)
    amace.start()


class MultiprocessTaskRunner:
    ''' Starts running AMACE() on each device/ ip. '''
    def __init__(self, url: str, host_ip: str,  ips: List[str], test_account: str):

        self.__test_account = test_account
        self.__run_ts = int(time()*1000)
        self.__run_id = uuid.uuid4()
        self.__url = url
        self.__host_ip = host_ip
        self.__ips = ips
        self.__processes = []

    def __start_process(self, ip):
        try:
            process = Process(target=task, args=(ip, self.__url, self.__host_ip, self.__run_id, self.__run_ts, self.__test_account))
            process.start()
            self.__processes.append(process)
        except Exception as error:
            print("Error start process: ",  error)

    def run(self):
        # start process
        for ip in self.__ips:
            self.__start_process(ip)

        for p in self.__processes:
            p.join()


if __name__ == "__main__":
    load_apps()
    parser = argparse.ArgumentParser(description="App validation.")
    parser.add_argument("-d", "--device",
                        help="Device to run on DUT.",
                        default="", type=str)

    parser.add_argument("-u", "--url",
                        help="Base url to post data.",
                        default="https://appval-387223.wl.r.appspot.com/api/amaceResult", type=str)

    parser.add_argument("-a", "--account",
                        help="Test account for DUT.",
                        default="", type=str)


    ags = parser.parse_args()
    url = ags.url
    print(f"BASEURL {url=}")

    host_ip = get_local_ip()
    print(f"{host_ip=}")



    test_account = ags.account

    # Read list of ips from CLI
    # Loops and create a new process for each
    ips = [d for d in ags.device.split(" ") if d]
    print("Starting on devices: ", ips)
    # sleep(10)
    tr = MultiprocessTaskRunner(url, host_ip, ips=ips, test_account=test_account)

    tr.run()
