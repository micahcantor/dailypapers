# Dailypapers

Change your Chrome wallpaper to a top landscape photo from Reddit every day.

![Logo](./cmd/daily/banner-logo.png)

## How it works

Dailypapers uses a Go backend to pull the top landscape photo of the day from Reddit's r/EarthPorn. The program then resizes the photo to 1920x1080 using [caire](https://github.com/esimov/caire), a content aware image resizing library written in Go. The resized image is uploaded to Imgur, and its metadata to a MongoDB database which is sent to the client whenever a user opens a new tab.

This repository contains the code for both the backend (in /cmd) and the extension (in /client). The daily routine which uploads the new image is hosted on AWS Lambda, while the http server is hosted on Heroku.

## Local Development

`git clone https://github.com/micahcantor/dailypapers.git`, then:

**To run the daily function**:
- Move the contents in `daily()` in daily/main.go to `main()`
- Either remove the `imgurUpload()` function or register your own app for the Imgur API.
- Connect your own mongodb database in `getCollection()`
- Run with `go run main.go`

**To run the extension**
- Go to chrome://extensions, and turn on 'Developer Mode'
- Click 'Load unpacked' and select the folder dailypapers/client

## Installation
Dailypapers is [available on the Chrome Web Store](https://chrome.google.com/webstore/detail/search-bar-for-classroom/dmlfplbdckbemkkhkojekbagnpldghnc).
![webstore](https://raw.githubusercontent.com/micahcantor/ClassroomSearchbar/master/ChromeWebStoreBadge.png "Webstore")
