@echo off
echo Starting Accnotify server with Bark APNs credentials...
echo.

REM Create data directory
if not exist "data" mkdir data

REM Set environment variables
set APNS_KEY_ID=LH4T9V5U4R
set APNS_TEAM_ID=5U8LBRXG3A
set APNS_PRODUCTION=true
set APNS_PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg4vtC3g5L5HgKGJ2+\nT1eA0tOivREvEAY2g+juRXJkYL2gCgYIKoZIzj0DAQehRANCAASmOs3JkSyoGEWZ\nsUGxFs/4pw1rIlSV2IC19M8u3G5kq36upOwyFWj9Gi3Ejc9d3sC7+SHRqXrEAJow\n8/7tRpV+\n-----END PRIVATE KEY-----

echo APNs configured with Bark official credentials
echo Starting server on http://0.0.0.0:8080
echo.

accnotify.exe

pause
