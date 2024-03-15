import os
import glob
import os
import subprocess
import zipfile
from time import sleep
from google.oauth2.service_account import Credentials
from googleapiclient.discovery import build
from rest_framework.views import APIView
from rest_framework.response import Response
from rest_framework import viewsets
from PIL import Image
from googleapiclient.http import MediaIoBaseDownload
from django.core.mail import send_mail
from django.conf import settings

from imageserver.yolov8 import YoloV8
from imageserver.settings import BASE_DIR, CONFIG
from google.cloud import storage

'''
Google APIs
  - Manage files in Drive
  - Upload screenshots to Storage
'''
# GOOGLE_APPLICATION_CREDENTIALS_IMAGE_SERVER_STORAGESERVICEACCOUNTKEY

# Instantiates a client
# Set up Google Drive service
# creds = Credentials.from_service_account_info(CONFIG['GOOGLE_APPLICATION_CREDENTIALS_IMAGE_SERVER_STORAGESERVICEACCOUNTKEY'],
#                                               scopes=['https://www.googleapis.com/auth/drive.readonly'])


# ~/.config/gcloud/application_default_credentials.json
# creds = Credentials.from_service_account_file(filename=CONFIG['ADC'], scopes=['https://www.googleapis.com/auth/drive.readonly'])
service = build('drive', 'v3') # ADC via gcloud auth application-default login

# Set up Google Cloud Storage service
storage_client = storage.Client() # ADC via gcloud auth application-default login

# Get all environment variables
def print_env():
    print(f"Current env vars: ")
    # Print all environment variables
    for key, value in os.environ.items():
        print(f'{key}: {value}')


APKFolder = f'{BASE_DIR}/files'
os.makedirs(APKFolder, exist_ok=True)


V8_WEIGHTS=f"{BASE_DIR}/weights/best_1080_v8m_v3.pt"
print(f"Loaded weights from: {V8_WEIGHTS}")
detector_v8 = YoloV8(weights=V8_WEIGHTS)


def find_transport_id(ip_address)-> str:
    ''' Gets the transport_id from ADB devices command.

        ['192.168.1.113:5555', 'device', 'product:strongbad', 'model:strongbad',
            'device:strongbad_cheets', 'transport_id:1']

        Params:
            ip_address: A string representing the name of the device
                according to ADB devices, typically the ip address.

        Returns:
            A string representing the transport id for the device matching the
                @ip_adress

    '''
    cmd = (CONFIG['IMAGE_SERVER_ADB_PATH'], 'devices', '-l')
    outstr = subprocess.run(cmd, check=True, encoding='utf-8', capture_output=True).stdout.strip()
    # Split the output into a list of lines
    lines = outstr.split("\n")
    for line in lines:
        # Split the line into words
        words = line.split()
        print("finding tid words: ", words)
        if f"{ip_address}:5555" in words:
            # The transport ID is the last word in the line
            return words[-1].split(":")[-1]
    # If the IP address was not found, return None
    return '-1'

def installADB(tid, file_path):
    try:
        print(f"Attempting to install {file_path}")
        cmd = ('adb', '-t', tid, "install", file_path)
        outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                capture_output=True).stdout.strip()

        print(outstr)
        return True
    except Exception as err:
        print("Error installing: ", file_path, err)
        return False

def installMultiADB(tid, file_path):
    proc= None
    try:
        print(f"Attempting to install multi {file_path}")
        apk_files = glob.glob(f"{file_path}.zip_extracted/*.apk")
        cmd = ['adb', '-t', tid, "install-multiple", *apk_files]
        print(f"Attempting to install multi {cmd}")
        outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                capture_output=True).stdout.strip()
        print(outstr)
        return True
    except TypeError as err:
        print("\n\n\n Error installing multiple: ", file_path, err, "\n\n\n")
    except subprocess.CalledProcessError as err:
        print("\n\n\n Error installing multiple: ", file_path, err.stderr, "\n\n\n")
    except Exception as err:
        print("\n\n\n Error installing multiple: ", file_path, err, "\n\n\n")
        return False

def extract_zip(file_path):
    "Given a filepath ending in the package name of a APK, extract the file_path.zip into a folder named file_path.zip_extracted"
    # Define the extraction directory
    extract_dir = f"{file_path}.zip_extracted"

    # Create the directory if it doesn't exist
    if not os.path.exists(extract_dir):
        os.makedirs(extract_dir)

    # Extract the zip file
    with zipfile.ZipFile(f"{file_path}.zip", 'r') as zip_ref:
        zip_ref.extractall(extract_dir)

    print(f"Extracted {file_path}.zip to {extract_dir}")
    remove_zip(file_path)

def download_file_from_drive(file_id, output_path, ext):
    request = service.files().get_media(fileId=file_id)
    with open(f"{output_path}.{ext}", 'wb') as f:
        downloader = MediaIoBaseDownload(f, request)
        done = False
        while done is False:
            status, done = downloader.next_chunk()


class ConnectADBViewSet(APIView):

    def post(self, req, pk=None):
        dutIP = req.data['dutIP']
        kill_server = req.data['killServer']
        print(f"Dut {dutIP} asking to conect to ADB.... ")
        # adb connect dutIP
        try:
            if kill_server == "kill":
                cmd = ('adb', 'kill-server')
                outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                        capture_output=True).stdout.strip()
                print(f"Killed server: {outstr}")
                sleep(1)

                cmd = ('adb', 'devices')
                outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                        capture_output=True).stdout.strip()
                print(f"ADB devices: {outstr}")
                sleep(1)

            cmd = ('adb', 'connect', dutIP)
            outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                    capture_output=True).stdout.strip()
            failed_msg = "failed to connect to"
            if outstr.startswith(failed_msg):
                print(failed_msg)
                return Response({"data": None, "error": failed_msg})

            print(f"Connected to ADB {outstr}")
            return Response({"data": outstr, "error": None})
        except Exception as err:
            print("Error connecting to ADB", err)
            return Response({"data": None, "error": f"Failed to connect to ADB {err}"})


def remove_zip(file_path):
    """
    Removes the specified file from the filesystem.

    Args:
    - file_path (str): The path to the file to be removed.

    Returns:
    - True if the file was successfully removed, False otherwise.
    """
    try:
        os.remove(f"{file_path}.zip")
        return True
    except Exception as e:
        print(f"Error removing {file_path}: {e}")
        return False

def installByBlankFilePath(tid, file_path):

    if os.path.exists(f"{file_path}.apk"):
        print(" installADB:  ")
        return installADB(tid, f"{file_path}.apk")

    if os.path.exists(f"{file_path}.zip_extracted"):
        print(" installMultiADB:  ")
        return installMultiADB(tid, f"{file_path}")

    if os.path.exists(f"{file_path}.zip"):
        print("(extract_zip & installMultiADB:  ")
        extract_zip(file_path)
        return installMultiADB(tid, file_path)

    print(f"Unable to install {file_path}, file not found on local server")


class PythonStoreViewSet(APIView):

    def post(self, req, format=None):
        print("PythonStoreViewSet: ", req.data)
        pkg_name = req.data['pkgName']
        file_name = req.data['fileName']
        drive_url = req.data['driveURL']
        dutIP = req.data['dutIP']
        tid = find_transport_id(dutIP)
        print(f"\n\n\n\n DUT requested {pkg_name=} {file_name=} from {drive_url} \n\n\n\n")

        try:
            # Assuming files are stored in a folder named 'files' in the server's directory
            file_path = os.path.join(APKFolder, file_name)

            # Check if file exists on server
            if not os.path.exists(f"{file_path}.apk") and not os.path.exists(f"{file_path}.zip_extracted"):
                # If not, fetch from Google Drive and store on server
                # Here, you'd need a way to determine the correct file ID based on package_name
                # For now, I'm assuming file_id is passed but you may want to create a mapping
                # or a database lookup to get the file ID based on the package_name
                # folder_id = "1Lq_IdWlN9KOJT-h8dPiJsLFaRnHusg6e"
                print("downloaing from google drive: ", file_name)
                response = service.files().list(q=f"'{drive_url}' in parents").execute()
                print("Google drive response: ", response)
                files = response.get('files', [])
                print("Google drive response files: ", files)
                file_id = None
                ext = ""
                for file in files:
                    # print("File in folder: ", file, file['name'])
                    if str(file['name']).startswith(file_name):
                        file_id = file['id']
                        ext = file['name'].split(".")[-1]
                        break

                if file_id:
                    download_file_from_drive(file_id, file_path, ext)
                else:
                    err = f"File {file_name} not found in Google Drive"
                    print(err)
                    return Response({"data": None, "error": err})

            if installByBlankFilePath(tid, file_path):
                return Response({"data": "Installed.", "error": None})
            return Response({"data": None, "error": f"Failed to install: {pkg_name}"})
        except Exception as err:
            print("Failed to get APK: ", err)
            return Response({"data": None, "error": f"Failed to get APK: {err}"})


class EmailViewSet(APIView):
    def post(self, req, format=None):
        '''https://www.abstractapi.com/guides/django-send-email'''
        # print(dir(req))
        print(req.data)

        subject = 'Automation bug report'
        message = req.data['msg']
        to = []

        try:
           send_mail( subject=subject, message=message, from_email=settings.EMAIL_HOST_USER, recipient_list=to)
        except Exception as err:
            print("Email err: ", err)
            return Response({"success": False})
        return Response({"success": True})


class YoloViewSet(APIView):
    def post(self, req, format=None):
        print(req.FILES)

        img = req.FILES['image']

        res = detector_v8.detect(Image.open(img))
        print(f"{res=}")
        return Response({"data": res, "error": None})


class ImageViewSet(APIView):
    def post(self, req, format=None):
        print(f"{req.data=}")
        print(f"{req.FILES=}")
        test = f"{req.data=}, {req.FILES=}"
        img, path = "", ""
        try:
            img = req.FILES['image']
            path = req.data['imgPath']
        except Exception as err:
            print("Error: ", err)
            return Response({"success": False, "err": str(err), "test": test})


        # The ID of your GCS bucket
        bucket_name = CONFIG['IMAGE_SERVER_BUCKET_NAME']
        print(f"Uploading to {bucket_name}. {len(path)=} {path=}")
        # The path to your file to upload
        # source_file_name = "local/path/to/file"
        # The ID of your GCS object
        # destination_blob_name = "storage-object-name"
        destination_blob_name = path

        bucket = storage_client.bucket(bucket_name)
        blob = bucket.blob(destination_blob_name)

        # Optional: set a generation-match precondition to avoid potential race conditions
        # and data corruptions. The request to upload is aborted if the object's
        # generation number does not match your precondition. For a destination
        # object that does not yet exist, set the if_generation_match precondition to 0.
        # If the destination object already exists in your bucket, set instead a
        # generation-match precondition using its generation number.
        generation_match_precondition = 0

        # blob.upload_from_filename(source_file_name, if_generation_match=generation_match_precondition)
        blob.upload_from_file(img)

        return Response({"success": True})