

# Auth w/ GCP
gcloud auth application-default login


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

    source bin/activate
    ./imageserver/manage.py runserver 192.168.1.229:8000

    ./manage.py runserver hostIP:8000



# Start AMACE Validator

./startCROS.sh
# Places user in scripts/
./startAMACE.sh root@192.168.1.122
# optional url to override Prod url as second arg.
./startAMACE.sh root@192.168.1.122 http://192.168.1.229:3000/api/amaceResult



# Run Raw TAST command
## Push data to server(default "https://appval-387223.wl.r.appspot.com/api/amaceResult")
tast -verbose run  -var=ui.gaiaPoolDefault=testacct@gmail.com:password root@192.168.1.238 arc.AMACE
./startAMACE.sh root@192.168.1.125 http://192.168.1.229:3000/api/amaceResult

## Push data to localhost (or other url)
tast -verbose run -var=arc.amace.globalPOSTURL="http://192.168.1.229:3000/api/amaceResult" -var=ui.gaiaPoolDefault=testacct@gmail.com:password root@192.168.1.238 arc.AMACE



# Global App List
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



# Capablilities

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
6. Get App Type
7. Get AMACE Status
8. Report app status on a per run basis.
9. Run on multiple devices.


# Notes:
Make sure input data file does not have trailing line.


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