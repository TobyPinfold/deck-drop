#!/usr/bin/env bash
# Stamps a release version into the app and every platform's metadata.
# The release workflow runs it with the tag's version before building:
#
#   build/set-version.sh 0.1.0
#   build/set-version.sh 0.1.0-rc.5
#
# The app shows the full version. macOS and Windows only accept numbers in
# their version fields, so those get the numeric part (0.1.0 for 0.1.0-rc.5).
# It edits tracked files, so run it on a clean checkout and don't commit the
# result.
set -euo pipefail

version=${1:?usage: build/set-version.sh <version, for example 0.1.0>}
version=${version#v}
if [[ ! $version =~ ^([0-9]+\.[0-9]+\.[0-9]+)(-[0-9A-Za-z.-]+)?$ ]]; then
    echo "set-version: '$version' is not a version like 0.1.0 or 0.1.0-rc.5" >&2
    exit 1
fi
numeric=${BASH_REMATCH[1]}

cd "$(dirname "$0")/.."

# replace FILE PERL_REGEX: edits FILE in place and fails if nothing matched.
replace() {
    local file=$1 regex=$2
    VERSION=$version NUMERIC=$numeric perl -0pi -e "$regex or die qq{set-version: no match in $file\n}" "$file"
}

printf '%s\n' "$version" > internal/version/version.txt
replace build/config.yml           's/(\n\s*version: )"[^"]*"/$1"$ENV{VERSION}"/'
replace build/darwin/Info.plist    's{(<key>CFBundle(?:ShortVersionString|Version)</key>\s*<string>)[^<]*}{$1$ENV{NUMERIC}}g'
replace build/windows/info.json    's/("(?:file_version|ProductVersion)":\s*)"[^"]*"/$1"$ENV{NUMERIC}"/g'
replace build/windows/nsis/wails_tools.nsh 's/(!define INFO_PRODUCTVERSION )"[^"]*"/$1"$ENV{NUMERIC}"/'
replace build/linux/nfpm/nfpm.yaml 's/(\nversion: )"[^"]*"/$1"$ENV{VERSION}"/'

echo "set-version: $version (numeric $numeric)"
