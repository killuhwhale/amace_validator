# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Starts Tast test and monitors progress. Restarts if TAST fails until app list is exhausted.


'''
tast -verbose run -var=amace.runts=123 -var=amace.runid=123 -var=ui.gaiaPoolDefault=testacct@gmail.com:PASSWORD root@192.168.1.238 arc.AMACE

'''
import argparse
import subprocess
import uuid
from collections import defaultdict
from dataclasses import dataclass
from time import time
import requests

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

    def __init__(self, device: str, BASE_URL: str):
        self.__device = device
        self.__current_package = ""
        self.__BASE_URL = BASE_URL
        self.__run_finished = False
        self.__log_error = False
        self.__package_retries = defaultdict(int)
        self.__packages = defaultdict(int)
        self.__package_arr = []
        self.__api_key = None
        self.__run_ts = int(time()*1000)
        self.__run_id = uuid.uuid4()
        self.__request_body = None
        self.__get_apps()
        self.__get_api_key()

    def __get_apps(self):
        """Get apps from file."""
        with open("../platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/data/AMACE_app_list.tsv", 'r', encoding="utf-8") as f:
            for idx, l in enumerate(f.readlines()):
                pkg = l.split("\t")[1].replace("\n", "")
                self.__package_arr.append(pkg)
                self.__packages[pkg] = idx

    def __get_api_key(self):
        """Get api key from file."""
        with open("../platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/data/AMACE_secret.txt", 'r', encoding="utf-8") as f:
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
                # TODO() Split the line to parse AppResult.
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
        cmd = ("tast", "-verbose", "run", f"-var=arc.amace.startat={self.__current_package}", f"-var=arc.amace.runts={self.__run_ts}", f"-var=arc.amace.runid={self.__run_id}", "-var=ui.gaiaPoolDefault=testacct@gmail.com:PASSWORD", self.__device, "arc.AMACE")
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


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="App validation.")
    parser.add_argument("-d", "--device",
                        help="Device to run on DUT.",
                        default="", type=str)
    parser.add_argument("-u", "--url",
                        help="Base url to post data.",
                        default="https://appval-387223.wl.r.appspot.com/api/amaceResult", type=str)

    ags = parser.parse_args()
    url = ags.url
    print(f"BASEURL {url=}")
    amace = AMACE(device=ags.device, BASE_URL=url)
    amace.start()
