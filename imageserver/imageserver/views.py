import os
from requests import HTTPError
from rest_framework.views import APIView
from rest_framework.response import Response

import smtplib

# Import the email modules we'll need
from email.message import EmailMessage

# cred = cred.with_subject(email)

# creds, _ = google.auth.load_credentials_from_file(f"{BASE_DIR}/imageserver/appvalEmailKey.json")

class EmailViewSet(APIView):
    def post(self, req, format=None):
        # print(dir(req))
        print(req.data)
        try:
           # Import smtplib for the actual sending function


# Open the plain text file whose name is in textfile for reading.
            msg = EmailMessage()
            msg.set_content("Testing")


            # me == the sender's email address
            # you == the recipient's email address
            msg['Subject'] = f'Testing msgs'
            msg['From'] = "andayac@gmail.com"
            msg['To'] = "andayac@gmail.com"

            # Send the message via our own SMTP server.
            s = smtplib.SMTP('localhost')
            s.send_message(msg)
            s.quit()

        except HTTPError as error:
            print(F'An error occurred: {error}')
            draft = None

        return Response({"testing": True})





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