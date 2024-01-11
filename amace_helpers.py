import ipaddress
import json
import os
import sys

import jwt

USER = os.environ.get("USER")
CHROMEOS_SRC = f"/home/{USER}/chromiumos/src"
CHROMEOS_SCRIPTS = f"/home/{USER}/chromiumos/src/scripts"

def load_config_file():
    file_path = f"{CHROMEOS_SCRIPTS}/.config/amaceValidator/config.json"
    try:
        return json.load(open(file_path, "r"))
    except FileNotFoundError:
        print(f"File '{file_path}' not found.")
        return {}
    except json.JSONDecodeError as e:
        print(f"Error decoding JSON in '{file_path}': {e}")
        return {}

CONFIG = load_config_file()

def req_env_var(value, name, env_var):
    if value is None:
        print(f"Env var: {name} not found, must enter env var: {env_var}")
        sys.exit(1)


def encode_jwt(payload, secret, algorithm='HS512'):
    """
    Encode a payload into a JWT token.

    Parameters:
    - payload: The data you want to encode into the JWT.
    - secret: The secret key to sign the JWT.
    - algorithm: The algorithm to use for signing. Default is 'HS512'.

    Returns:
    - Encoded JWT token as a string.
    """
    encoded_jwt = jwt.encode(payload, secret, algorithm=algorithm)
    return encoded_jwt

def get_server_wss_url():
    # uri = "ws://localhost:3001/wss/"
    # uri = "wss://appvaldashboard.com/wss/"

    SERVER_IP = CONFIG["WSS_SERVER_IP"]
    ip_addr = CONFIG["WSS_SERVER_IP"]

    if ":" in ip_addr:
        ip_addr, port = ip_addr.split(":")

    if ip_addr.lower() == "localhost":
        return f"ws://{SERVER_IP}/wss/"

    ip = ipaddress.ip_address(ip_addr)
    if ip.is_loopback:
        return f"ws://{SERVER_IP}/wss/"

    return f"wss://{SERVER_IP}/wss/"

def ping(msg, data, wssToken):
    return str(json.dumps({"msg": msg, "data": {**data, "wssToken": wssToken}}))

def pj(s: str):
    # parse json
    return json.loads(s)


Red     = "\033[31m"
Black   = "\033[30m"
Green   = "\033[32m"
Yellow  = "\033[33m"
Blue    = "\033[34m"
Purple  = "\033[35m"
Cyan    = "\033[36m"
White   = "\033[37m"
RESET   = "\033[0m"

line_start = f"{Blue}>{Red}>{Yellow}>{Green}>{Blue}{RESET} "