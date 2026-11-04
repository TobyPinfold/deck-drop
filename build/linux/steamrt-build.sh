#!/usr/bin/env bash
# Builds the Linux AppImage inside the Steam Linux Runtime 4.0 SDK, so
# the binary only needs the glibc and libraries SteamOS provides. Run it
# from the repository root inside the SDK container, with the frontend already
# built (frontend/dist).
set -euo pipefail

go_version=$(awk '/^go /{print $2}' go.mod)
curl -fsSL "https://go.dev/dl/go${go_version}.linux-amd64.tar.gz" | tar -C /usr/local -xz
export PATH="/usr/local/go/bin:/root/go/bin:$PATH" GOTOOLCHAIN=local GOFLAGS=-buildvcs=false

apt-get update -qq
apt-get install -y -qq libgtk-4-dev libwebkitgtk-6.0-dev

CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0

app=deckdrop
mkdir -p bin build/linux/appimage
CGO_ENABLED=1 go build -tags production -trimpath -ldflags="-w -s" -o "bin/$app" .

wails3 generate .desktop -name "$app" -exec "$app" -icon "$app" \
    -outputfile "build/linux/$app.desktop" -categories "Network;"

cd build/linux/appimage
cp "../../../bin/$app" "$app"
cp ../../appicon.png "$app.png"
# linuxdeploy is itself an AppImage, and containers have no FUSE.
APPIMAGE_EXTRACT_AND_RUN=1 wails3 generate appimage -binary "$app" -icon "$app.png" \
    -desktopfile "../$app.desktop" -outputdir ../../../bin -builddir "$PWD/build"
