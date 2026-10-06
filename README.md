# TiLapseCam

A lightweight, beautiful, and pure-Go webcam timelapse creator. Designed as a web app embedded within a single Go executable.

## Features
- **Pure Go**: No C++ compiler, OpenCV, or external dependencies like FFmpeg needed.
- **Single Executable**: HTML, CSS, and JS files are embedded directly into the binary using `go:embed`.
- **Beautiful UI**: Glassmorphism aesthetic and smooth interactions.
- **Auto Video Rendering**: Native MJPEG video generation turns captured frames into an `.avi` video directly within the app.

## How to Run
1. Download or build the `TiLapseCam.exe`.
2. Run the executable.
3. Your default web browser will open automatically.
4. Allow webcam permissions, set your interval, and start your timelapse!

## Build from Source
```bash
go mod tidy
go build -o TiLapseCam.exe
```
