#!/usr/bin/env bash
# Builds the Linux AppImage inside the Steam Linux Runtime 3.0 (sniper) SDK, so
# the binary only needs the old glibc and libraries SteamOS provides. Run it
# from the repository root inside the SDK container, with the frontend already
# built (frontend/dist).
set -euo pipefail

go_version=$(awk '/^go /{print $2}' go.mod)
curl -fsSL "https://go.dev/dl/go${go_version}.linux-amd64.tar.gz" | tar -C /usr/local -xz
export PATH="/usr/local/go/bin:/root/go/bin:$PATH" GOTOOLCHAIN=local GOFLAGS=-buildvcs=false

apt-get update -qq
apt-get install -y -qq libgtk-3-dev libwebkit2gtk-4.0-dev

# Wails 3 asks for webkit2gtk-4.1 with GTK 3, but sniper only ships 4.0. The
# two differ only in the libsoup version, which Wails does not use, so give
# pkg-config a 4.1 name for the 4.0 library.
mkdir -p /tmp/pkgconfig
sed 's/^Name:.*/Name: webkit2gtk-4.1/' \
    "$(pkg-config --variable=pcfiledir webkit2gtk-4.0)/webkit2gtk-4.0.pc" > /tmp/pkgconfig/webkit2gtk-4.1.pc
export PKG_CONFIG_PATH=/tmp/pkgconfig

CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0

app=deckdrop
mkdir -p bin build/linux/appimage
CGO_ENABLED=1 go build -tags gtk3,production -trimpath -ldflags="-w -s" -o "bin/$app" .

wails3 generate .desktop -name "$app" -exec "$app" -icon "$app" \
    -outputfile "build/linux/$app.desktop" -categories "Network;"

cd build/linux/appimage
cp "../../../bin/$app" "$app"
cp ../../appicon.png "$app.png"
# linuxdeploy is itself an AppImage, and containers have no FUSE.
APPIMAGE_EXTRACT_AND_RUN=1 wails3 generate appimage -binary "$app" -icon "$app.png" \
    -desktopfile "../$app.desktop" -outputdir ../../../bin -builddir "$PWD/build"
