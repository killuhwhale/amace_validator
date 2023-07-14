import os
from requests import HTTPError
from rest_framework.views import APIView
from rest_framework.response import Response
from PIL import Image
import smtplib

# Import the email modules we'll need
from email.message import EmailMessage


from google.cloud import storage
# Instantiates a client
storage_client = storage.Client()

from django.core.mail import send_mail
from django.conf import settings
from imageserver.yolov8 import YoloV8
import os

print(f"{os.getcwd()=}")

V8_WEIGHTS=f"{os.getcwd()}/imageserver/imageserver/weights/best_1080_v8m_v3.pt"
detector_v8 = YoloV8(weights=V8_WEIGHTS)


class EmailViewSet(APIView):
    def post(self, req, format=None):
        '''https://www.abstractapi.com/guides/django-send-email'''
        # print(dir(req))
        print(req.data)

        subject = 'Automation bug report'
        message = req.data['msg']
        to = ['andayac@gmail.com', 'buganizer-system+168499@google.com']

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
        bucket_name = "appval-387223.appspot.com"
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