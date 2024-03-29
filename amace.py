# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Starts Tast test and monitors progress. Restarts if TAST fails until app list is exhausted.


'''
tast -verbose run -var=arc.amace.posturl=http://xyz.com -var=arc.amace.hostip=http://192.168.1.123  -var=arc.amace.device=root@192.168.1.456 -var=amace.runts=123 -var=amace.runid=123  -var=ui.gaiaPoolDefault=email@gmail.com:password root@192.168.1.238 arc.AMACE
'''
import argparse
import ipaddress
import json
import os
import subprocess
import sys
import uuid
from collections import defaultdict
from dataclasses import dataclass
from multiprocessing import Process
from time import time
from typing import Dict, List
from amace_helpers import CONFIG, CHROMEOS_SRC, RESET, Red, Green, Yellow, Blue, Blue, Cyan
import requests

chroot_data_path = f"{CHROMEOS_SRC}/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/data"

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


# enum AppType {
#   APP = "App",
#   Game = "Game",
#   PWA = "PWA",
# }

# type HistoryStep = {
#   msg: string;
#   url: string;
# };

def get_server_domain(url):
    ip_addr = url
    if ":" in ip_addr:
        ip_addr, port = ip_addr.split(":")

    if ip_addr.lower().startswith("localhost"):
        return f"http://{url}"

    ip = ipaddress.ip_address(ip_addr)
    if ip.is_loopback:
        return f"http://{url}"

    return "https://{url}"

@dataclass
class RequestBody:
    """Request data for app error. Reflects amace.go and backend."""
    appName: str
    pkgName: str
    runID: str
    runTS: str
    appTS: int
    status: int
    brokenStatus: int
    buildInfo: str
    deviceInfo: str
    appType: str
    appVersion: str
    history: str
    logs: str
    loginResults: int



def get_local_ip():
    '''Gets host deivce local ip address.'''
    result = subprocess.run(['ifconfig'], capture_output=True, text=True)
    output = result.stdout
    s = "192.168.1."
    try:
        idx = output.index(s)
        idx += len(s)
        return f"192.168.1.{output[idx:idx+3]}"
    except Exception:
        pass

    s = "10.0.0."
    try:
        idx = output.index(s)
        idx += len(s)
        return f"10.0.0.{output[idx:idx+3]}"
    except Exception:
        pass

    s = "192.168.0."
    try:
        idx = output.index(s)
        idx += len(s)
        return f"192.168.0.{output[idx:idx+3]}"
    except Exception:
        sys.exit("Failed to get local ip!")


def fetch_apps(server_url, amace_api_key, dsrcpath, dsrctype):
    '''Fetch apps from backend. NextJS -> FirebaseDB'''
    headers = {"Authorization": amace_api_key}
    try:
        url = f"{server_url}/api/applist?dsrctype={dsrctype}&dsrcpath={dsrcpath}"
        print("Getting app list: ", url)
        res = requests.get(url, headers=headers)
        print("Res fetch apps: ", res)
        if res.status_code == 404:
            sys.exit(f"Failed to get list of apps to check due to 404 bad URL: {url}")

        result = json.loads(res.text)
        print("Results fetch apps: ", result)
        s = result['data']['data']

        driveURL = ""
        try:
            driveURL = result['data']['driveURL']
            print("Run requested files from driveURL: ", driveURL)
        except Exception as err:
            print("No drive url returned!", err)

        # results = s.replace("\\t", "\t").split("\\n")
        results = s.replace("\\t", "\t").replace("\\n", "\n")
        print(f"{driveURL=} -- {results=}")
        return results, driveURL
    except Exception as err:
        sys.exit(f"Failed to get list of apps to check: {err}")

def load_apps(server_url, amace_api_key, dsrcpath, dsrctype):
    apps,  driveURL = fetch_apps(server_url, amace_api_key, dsrcpath,  dsrctype)
    return apps, driveURL

def fetch_app_creds(server_domain, amace_api_key):
    '''Fetch apps creds from backend. NextJS -> FirebaseDB'''
    try:
        headers = {"Authorization": amace_api_key}
        # res = requests.get("http://localhost:3000/api/appCreds", headers=headers)
        res = requests.get(f"{server_domain}/api/appCreds", headers=headers)
        creds = json.loads(res.text)['data']['data']
        return creds
    except Exception as err:
        sys.exit(f"Failed to get app creds: {str(err)}")

def task(device: str, post_url, host_ip, amace_api_key, run_id, run_ts, test_account, creds, skip_amace, skip_broken, skip_login, dsrcpath, dsrctype, appsToTest, driveURL):
    amace = AMACE(
        device=device.strip(),
        post_url=post_url,
        host_ip=host_ip,
        amace_api_key=amace_api_key,
        run_id=run_id,
        run_ts=run_ts,
        test_account=test_account,
        creds=creds,
        skip_amace=skip_amace,
        skip_broken=skip_broken,
        skip_login=skip_login,
        dsrcpath=dsrcpath,
        dsrctype=dsrctype,
        appsToTest=appsToTest,
        driveURL=driveURL,
        )
    amace.start()

class AMACE:
    """Runs TAST test to completion.

    If the test fails early for any reason, the test will be re run.
    """

    def __init__(self, device: str, post_url: str, host_ip: str, amace_api_key: str, run_id: str, run_ts: int, test_account: str, creds: Dict[str, Dict[str, str]], skip_amace, skip_broken, skip_login, dsrcpath, dsrctype, appsToTest, driveURL):
        self.__test_account = test_account
        self.__device = device
        self.__current_package = ""
        self.__post_url = post_url
        self.__host_ip = host_ip
        self.__run_finished = False
        self.__log_error = False
        self.__package_retries = defaultdict(int)
        self.__packages = defaultdict(int)
        self.__package_arr = []
        self.__amace_api_key = amace_api_key
        self.__run_id = run_id
        self.__run_ts = run_ts
        self.__creds = json.dumps(creds)
        self.__request_body = None
        self.__skip_amace = skip_amace
        self.__skip_broken = skip_broken
        self.__skip_login = skip_login
        self.__dsrcpath = dsrcpath
        self.__dsrctype = dsrctype
        self.__appsToTest = appsToTest
        self.__driveURL = driveURL
        self.__get_apps()

    def __get_apps(self):
        """Get apps from file."""
        filepath = f"{chroot_data_path}/AMACE_app_list.tsv"
        with open(filepath, 'r', encoding="utf-8") as f:
            for idx, l in enumerate(f.readlines()):
                pkg = l.split("\t")[1].replace("\n", "")
                self.__package_arr.append(pkg)
                self.__packages[pkg] = idx

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
        # The idea is to output the needed info to identify the app and report an error in the case that seomthing with the host device goes wrong.
        self.__request_body = RequestBody(
            appName = info[4],
            pkgName = info[3],
            runID = info[1],
            runTS = info[2],
            appTS = info[7],
            status = info[5],
            buildInfo = info[8],
            deviceInfo = info[9],
            appType=info[6],
            brokenStatus=0,
            appVersion="",
            history="",
            logs="",
            loginResults=0,
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
        cmd = (
            "tast", "-verbose", "run",
                f"-var=arc.amace.posturl={self.__post_url}",
                f"-var=arc.amace.creds={self.__creds}",  # Not currently used. TODO() Figure out method to store and retrieve creds
                f"-var=arc.amace.dsrcpath={self.__dsrcpath}",
                f"-var=arc.amace.dsrctype={self.__dsrctype}",
                f"-var=arc.amace.driveurl={self.__driveURL}",
                f"-var=arc.amace.skipamace={self.__skip_amace}",
                f"-var=arc.amace.skipbrokencheck={self.__skip_broken}",
                f"-var=arc.amace.skiploggin={self.__skip_login}",
                f"-var=arc.amace.device={self.__device}",
                f"-var=arc.amace.hostip={self.__host_ip}",
                f"-var=arc.amace.amaceapikey={self.__amace_api_key}",
                f"-var=arc.amace.appsToTest={self.__appsToTest}",
                f"-var=arc.amace.startat={self.__current_package}",
                f"-var=arc.amace.runts={self.__run_ts}",
                f"-var=arc.amace.runid={self.__run_id}",
                f"-var=ui.gaiaPoolDefault={self.__test_account}",
                f"-var=arc.amace.account={self.__test_account}" , self.__device, "arc.AMACE")

        return self.__run_command(cmd)

    def __post_err(self):
        """Sends post request to backed to store result in Firebase when an error happens."""
        print(f"Posting error from python ")
        headers = {'Authorization': self.__amace_api_key}
        res = requests.post(self.__post_url, data=self.__request_body.__dict__, headers=headers)
        print(f"{res=}")
        self.__log_error = False

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


class MultiprocessTaskRunner:
    ''' Starts running AMACE() on each device/ ip. '''
    def __init__(self, post_url: str, host_ip: str, amace_api_key: str,  ips: List[str], test_account: str, creds: Dict[str, Dict[str, str]],  skip_amace, skip_broken, skip_login, dsrcpath, dsrctype, appsToTest, driveURL):

        self.__test_account = test_account
        self.__run_ts = int(time()*1000)
        self.__run_id = uuid.uuid4()
        self.__post_url = post_url
        self.__host_ip = host_ip
        self.__ips = ips
        self.__creds = creds
        self.__amace_api_key = amace_api_key
        self.__processes = []
        self.__skip_amace = skip_amace
        self.__skip_broken = skip_broken
        self.__skip_login = skip_login
        self.__dsrcpath = dsrcpath
        self.__dsrctype = dsrctype
        self.__driveURL = driveURL
        self.__appsToTest = appsToTest

    def __start_process(self, ip):
        try:
            process = Process(target=task, args=(ip, self.__post_url, self.__host_ip, self.__amace_api_key, self.__run_id, self.__run_ts, self.__test_account, self.__creds, self.__skip_amace, self.__skip_broken, self.__skip_login, self.__dsrcpath, self.__dsrctype, self.__appsToTest, self.__driveURL))
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


    parser = argparse.ArgumentParser(description="App validation.")
    parser.add_argument("-d", "--device",
                        help="Device to run on DUT.",
                        default="", type=str)

    parser.add_argument("-w", "--samace",
                        help="Skip amace window check.",
                        default="f", type=str)
    parser.add_argument("-b", "--sbroken",
                        help="Skip broken check.",
                        default="f", type=str)
    parser.add_argument("-l", "--slogin",
                        help="Skip login.",
                        default="f", type=str)
    parser.add_argument("-s", "--dsrcpath",
                        help="Firebase document path for data/app list to test.",
                        default=f"AppLists/live", type=str)
    parser.add_argument("-t", "--dsrctype",
                        help="Data/ app list type: [playstore | pythonstore]",
                        default=f"playstore", type=str)



    print("Found config: ", CONFIG)
    test_account = CONFIG["TAST_ACCOUNT"]
    server_url = CONFIG["SERVER_IP"]
    amace_api_key = CONFIG["AMACE_API_KEY"]
    server_domain = get_server_domain(server_url)
    host_ip = get_local_ip()

    ags = parser.parse_args()
    skip_amace = ags.samace
    skip_broken = ags.sbroken
    skip_login = ags.slogin
    dsrcpath=ags.dsrcpath
    dsrctype=ags.dsrctype
    ips = [d for d in ags.device.split(" ") if d]

    appsToTest, driveURL = load_apps(server_domain, amace_api_key, dsrcpath,  dsrctype)
    creds = fetch_app_creds(server_domain, amace_api_key)
    post_url = f"http://{host_ip}:3000/api/amaceResult"

    print("Starting on devices: ", ips)
    print(f"\n\nCLIGGGGGG args: {server_url=} {host_ip=} {test_account=} {skip_amace=} {skip_broken=} {skip_login=}\n\n")
    tr = MultiprocessTaskRunner(post_url, host_ip, amace_api_key=amace_api_key, ips=ips, test_account=test_account, creds=creds, skip_amace= skip_amace, skip_broken= skip_broken, skip_login= skip_login, dsrcpath=dsrcpath, dsrctype=dsrctype, appsToTest=appsToTest, driveURL=driveURL)
    tr.run()