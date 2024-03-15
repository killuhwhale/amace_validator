




# Capablilities/ Features

1. Install App
2. Launch App
3. Check Logcat Errors on Launch
    - winDeath
    - forceRmRecord
    - fDebug
    - fatalException
    - procDied
    - anr
4. Check App for Black Screen
5. Get App Version
6. Get App Type - App, Game, PWA
7. Get AMACE Status
8. Attempt Login w/ Google, Facebook and Email.
9. Multi device execution.

# TODO()
1. Create a config files for Host Machine
    - Should contain all the environment vars needed for:
        1. Image Server
        2. TAST Tests
        3. WSS Clients
2. Refactor Code to remove all notions of AMACE_secret and nextAuthSecret.
    - AMACE_secret => AMACEAPIKEY
    - nextAuthSecret => jwtSecret



# Dev notes:
1. - AMACE_secret => AMACEAPIKEY
   - nextAuthSecret => jwtSecret
2. Config file:
    - file_path = f"home/{USER}/.config/amaceValidator/config.json"


# Dev and local host: Auth w/ GCP
gcloud auth list
gcloud auth application-default login
gcloud auth application-default set-quota-project appvalfirebasetest
gcloud config set project appvalfirebasetest


# Google APIs used from Host (local device in lab):
- Google Drive
- Google Cloud Storage


# Google Drive
- Service account
    - Add the service accounts email (GOOGLE_APPLICATION_CREDENTIALS_IMAGE_SERVER_STORAGESERVICEACCOUNTKEY.client_email) to the drive folder so it has access.

# Google Storage
## Note: Different account used from Google Drive due to python client library requiring a service account.
This will use the ADC application default credentails which is created when logging in via `gcloud auth application-default login`
With the email used to login, (find out w/ `gcloud auth list`) add that email as a principal w/ Storage Legacy Object Owner
Cloud Storage > Buckets > Permissions > Grant Access
 - email: Storage Legacy Object Owner

# Enter Chroot
cd ${CHROMEOS_SRC}/src/platform/
code ./tast-tests ./tast

# Update Chroot
 (inside) ./update_chroot

# Delete Chroot
If you need to delete your chroot, use cros_sdk --delete to delete it properly. Using rm -rf could end up deleting your source tree due to the active bind mounts.


# Image Server
    Django server running on Host to accept:
    - Send images from DUTs and push to Server for Apphistory
    - Send email
    - Object detection


# Dev Start
## wssClient (systemd service)
    cd wssTriggerEnv/
    source bin/activate
    cd wssTrigger/
    python3 wssClient.py
            - [Deprecated - CLI args now in config ] DNAME=APPVAL002 TASTACCOUNT=emai@gmail.com:password

## Image Server (systemd service)
    source bin/activate
    ./imageserver/manage.py runserver 192.168.1.229:8000
    ./manage.py runserver hostIP:8000

## AppVal Dashboard
    Open repo AppValDashboard


# Run Raw Python command
python3 /home/killuh/chromiumos/src/platform/tast-tests/src/go.chromium.org/tast-tests/cros/local/bundles/cros/arc/amace.py   -l  t --dsrcpath AppLists/TestList --dsrctype playstore -d  192.168.1.125
tast -verbose run  -var=ui.gaiaPoolDefault=tastarcplusplusappcompat14@gmail.com:2VEKS7H4242 root@192.168.1.192 arc.LoginTest

# Run Raw TAST command
## Push data to server(default "https://appval-387223.wl.r.appspot.com/api/amaceResult")
tast -verbose run  -var=ui.gaiaPoolDefault=testacct@gmail.com:password root@192.168.1.238 arc.AMACE

## Push data to localhost (or other url)
tast -verbose run -var=arc.amace.globalPOSTURL="http://192.168.1.229:3000/api/amaceResult" -var=ui.gaiaPoolDefault=testacct@gmail.com:password root@192.168.1.238 arc.AMACE

tast -verbose run -var=arc.amace.posturl=http://192.168.1.229:3000/api/amaceResult -var=arc.amace.creds={} -var=arc.amace.dsrcpath=AppLists/TestList -var=arc.amace.dsrctype=playstore -var=arc.amace.driveurl= -var=arc.amace.skipamace=f -var=arc.amace.skipbrokencheck=f -var=arc.amace.skiploggin=t -var=arc.amace.device=192.168.1.125 -var=arc.amace.hostip=192.168.1.229 -var=arc.amace.amaceapikey=7cYt4FP9CDDG6i5tX5R3waR/HIwARyAu8KuZKlPhjZU= -var=arc.amace.appsToTest=$'Instagram\tcom.instagram.android\nNetflix\tcom.netflix.mediaclient\nRoblox\tcom.roblox.client' -var=arc.amace.startat= -var=arc.amace.runts=1706307423479 -var=arc.amace.runid=e9d23f6a-464c-4edf-9d9d-cfa1d056c1c2 -var=ui.gaiaPoolDefault=tastarcplusplusappcompat14@gmail.com:2VEKS7H4242 -var=arc.amace.account=tastarcplusplusappcompat14@gmail.com:2VEKS7H4242 192.168.1.125 arc.AMACE



# Global App List - Rough Draft Impl.
## collection(GlobalAMACEStatus).doc(packageName).collection(kevin).doc(build 0).[status | version | packageName]
We can place the latest status here with version. We dont expect the status to change from Device to Device so we can maintain a global list of latest status....
 As long as all devices are on ARC-R, we expect the status to only change on an update app version.
 We wont update if the app version is not newer....
 We will still check on new device builds...

 So we process an app, Roblox version 1 on Kevin B 1
 Get latest build for kevin: collection(GlobalAMACEStatus).doc(packageName).collection(kevin).doc(@latest) => build 0
 We get Build 0 and we are on build 1, so we continue to check....
 We then check the verison: .doc(@latest).version => 0
     .doc(@latest).version < Version 1 ? continue check: skip package, last checked version is >= current version

 If we are on a newer build and have a newer app, we just continue with the run, install launch, etc.....
 Then we update the list:
     - collection(GlobalAMACEStatus).doc(packageName).collection(kevin).doc(1).version = 1
     - collection(GlobalAMACEStatus).doc(packageName).collection(kevin).doc(1).status = XX

 Then update the Global List top level if verison is newer:
     collection(GlobalAMACEStatus).doc(packageName).version => 0
     collection(GlobalAMACEStatus).doc(packageName).version < Version 1 ? collection(GlobalAMACEStatus).doc(packageName).version = 1 : "Do nothing, app version is not newer. We must check because of different devices can potentially have varying app verisons..."

    collection(GlobalAMACEStatus).doc(packageName).date = new Date() // Update date


 Top level status has a goal, the goal is to provide the latest status regardless of device. We expect an app verison to behave the same on all ARC-R devices....






# Notes:
Make sure input data file does not have trailing line. AMACE-app_list


SSH Works
    ssh root@192.168.1.238
    cros shell 192.168.1.238
    cros shell 192.168.1.238 uname -r


    # ENABLE DUT SSH

    Flags
    Gain Access: at the developer console, enter “root” as the login, use a signed image
    Navigate Features Path: move, type “cd /usr/libexec/debugd/helpers”
    Initiate Flags: execute feature “./dev_features_rootfs_verification” then “sudo reboot”
    Reload Access: at the developer console, enter “root” as the login
    Reload Path: move to debugging features, type “cd /usr/libexec/debugd/helpers”
    Permissions: modify by typing “chmod 777 dev_features_ssh”
    Execute: type “./dev_features_ssh” & then “sudo reboot”
    Navigate .SSH: move, type “cd /home/chronos/user/”
    Generate .SSH: create, type “mkdir .ssh”
    Permissions .SSH: modify, type “sudo chmod 777 .ssh”
    Local: now let’s local loopback ssh, “ssh localhost” and type “yes” to accept fingerprint
    Remote: now let’s remote ssh, “ssh root@IP” and type “yes” to accept fingerprint
    Link (Debugging Features): https://support.google.com/chromebook/thread/105387132/
    Open Tunnel
    Remove File Blocking:
    move to “cd /etc/ssh”
    backup to “sshd_not_meant_to_run >> ~/temp” (if it exists, note, it should not)
    type to “rm sshd_not_meant_to_run”
    Close Tunnel
    Attach File Blocking:
    copy to “cp ~/temp /etc/ssh/sshd_not_meant_to_run”
    Revert
    Powerwash Device and Check Flags:
    go into dev console as root, check flags are off
    , you can tell by the permissions being reverted back to the original state.