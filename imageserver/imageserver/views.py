import os
import subprocess
from django.http import FileResponse
from requests import HTTPError
from google.oauth2.service_account import Credentials
from googleapiclient.discovery import build
from rest_framework.views import APIView
from rest_framework.response import Response
from rest_framework import viewsets
from PIL import Image
import smtplib
from googleapiclient.http import MediaIoBaseDownload
from rest_framework.decorators import action
# Import the email modules we'll need
from email.message import EmailMessage

from imageserver.settings import env

from django.core.mail import send_mail
from django.conf import settings
from imageserver.yolov8 import YoloV8
import os
from imageserver.settings import BASE_DIR

from google.cloud import storage
# Instantiates a client
storage_client = storage.Client()

# Set up Google Drive service
print("Env ", os.environ.get("GOOGLE_APPLICATION_CREDENTIALS"))
creds = Credentials.from_service_account_file(os.environ.get("GOOGLE_APPLICATION_CREDENTIALS"),
                                              scopes=['https://www.googleapis.com/auth/drive.readonly'])
service = build('drive', 'v3', credentials=creds)

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
    cmd = ('adb', 'devices', '-l')
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
    
def installMultipleADB(tid, file_path):
    try:
        print(f"Attempting to install {file_path}")
        cmd = ('unzip', '-o', file_path)
        outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                capture_output=True).stdout.strip()
        print(outstr)

        apkFiles = []
        for file in os.listdir("."):
            if file.endswith(".apk"):
                print(file)
                apkFiles.append(file)

        print("Attempting install-multiple")
        cmd = (['adb', 'install-multiple'] + apkFiles)
        print(cmd)
        outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                capture_output=True).stdout.strip()
        print(outstr)
        
        print("Attempting to remove apk files")
        cmd = (['rm'] + apkFiles)
        outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                capture_output=True).stdout.strip()
        print(outstr)
    except Exception as err:
        print("Error installing: ", file_path, err.output())
        return False


def download_file_from_drive(file_id, output_path):
    print("Downloading file from drive")
    request = service.files().get_media(fileId=file_id)
    with open(output_path, 'wb') as f:
        downloader = MediaIoBaseDownload(f, request)
        done = False
        while done is False:
            status, done = downloader.next_chunk()


class ConnectADBViewSet(APIView):

    def post(self, req, pk=None):
        dutIP = req.data['dutIP']
        print(f"Dut {dutIP} asking to conect to ADB.... ")
        # adb connect dutIP
        try:
            cmd = ('adb', 'connect', dutIP)
            outstr = subprocess.run(cmd, check=True, encoding='utf-8',
                                    capture_output=True).stdout.strip()
            failed_msg = "failed to connect to"
            if outstr.startswith(failed_msg):
                print(failed_msg)
                return Response({"data": None, "error": failed_msg})

            print(outstr)
            return Response({"data": outstr, "error": None})
        except Exception as err:
            print("Error connecting to ADB", err)
            return Response({"data": None, "error": f"Failed to connect to ADB {err}"})
        

class APKList(APIView):

    def get(self, req, format=None):
        print(req)
        drive_url = req.data['driveURL']
        print("Drive URL ", drive_url)

        try:
            file_names = []
            response = service.files().list(q=f"'{drive_url}' in parents").execute()
            #print("Google drive response: ", response)
            files = response.get('files', [])
            #print("Google drive response files: ", files)
            file_id = None
            for file in files:
                file_names.append(file['name'] + "\t" + file['name'].split('-')[0])
            print(file_names)
            print(Response({"data": file_names, "error": None}))
            return Response({"data": file_names, "error": None})
        except Exception as err:
            return Response({"data": None, "error": f"Failed to get list of packages from apks to check: {err}"})


class PythonStoreViewSet(APIView):

    def post(self, req, format=None):
        print("In Python Store View Set post")
        file_name = req.data['aName']
        pkg_name = req.data['pkgName']
        drive_url = req.data['driveURL']
        dutIP = req.data['dutIP']
        # TODO() find transport id from ip dutIP
        tid = find_transport_id(dutIP)
        print(f"DUT requested {file_name} from {drive_url}")
        print("Drive URL ", drive_url)
        try:
            # Assuming files are stored in a folder named 'files' in the server's directory
            print("Looking for file in folder ", APKFolder, "with pkg_name", file_name)
            file_path = os.path.join(APKFolder, file_name)
            print(file_path)

            # Check if file exists on server
            if not os.path.exists(file_path):
                print("In if")
                # If not, fetch from Google Drive and store on server
                # Here, you'd need a way to determine the correct file ID based on package_name
                # For now, I'm assuming file_id is passed but you may want to create a mapping
                # or a database lookup to get the file ID based on the package_name
                # folder_id = "1Lq_IdWlN9KOJT-h8dPiJsLFaRnHusg6e"
                folder_id = drive_url
                response = service.files().list(q=f"'{folder_id}' in parents").execute()
                #print("Google drive response: ", response)
                files = response.get('files', [])
                #print("Google drive response files: ", files)
                file_id = None
                for file in files:
                    print(file)
                    # print("File in folder: ", file, file['name'])
                    if str(file['name']).startswith(pkg_name):
                        file_id = file['id']
                        print("file_id " , file_id)
                        print("file_name ", file['name'])
                        #file_path = os.path.join(file_path, file['name'].split(".")[:-1])
                        print("file_path ", file_path)
                        break

                if file_id:
                    download_file_from_drive(file_id, file_path)
                else:
                    return Response({"data": None, "error": "File not found in Google Drive"})

            if ".apk" in file_path:
                if installADB(tid, file_path):
                    return Response({"data": "Installed.", "error": None})
            else:
                if installMultipleADB(tid, file_path):
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
        # print(dir(req))
        print(req.data)
        print(req.FILES)

        img = req.FILES['image']
        path = req.data['imgPath']



        # The ID of your GCS bucket
        bucket_name = env("BUCKET_NAME")
        # The path to your file to upload
        # source_file_name = "local/path/to/file"
        # The ID of your GCS object
        # destination_blob_name = "storage-object-name"
        destination_blob_name = path
        # storage_client = storage.Client()
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