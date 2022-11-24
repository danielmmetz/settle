#!/usr/bin/env bash

# This file is largely taken from https://github.com/junegunn/fzf/blob/5f385d88e0a786f20c4231b82f250945a6583a17/install
# As such:
#
# The MIT License (MIT)
#
# Copyright (c) 2013-2021 Junegunn Choi
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

set -euo pipefail

version=0.0.15  # TODO integrate with releases.

settle_base=$(pwd)

check_binary() {
  echo -n "present version: "
  local output
  output=$("$settle_base"/bin/settle version 2>&1)
  if [ $? -ne 0 ]; then
    echo "error: $output"
    binary_error="invalid binary"
  else
    output=${output/ */}
    if [ "$version" != "$output" ]; then
      echo "$output != $version"
      binary_error="invalid version"
    else
      echo "$output"
      binary_error=""
      return 0
    fi
  fi
  rm -f "$settle_base"/bin/settle
  return 1
}

try_curl() {
  command -v curl > /dev/null &&
  curl -fsSL $1 | tar -xzf -
}

try_wget() {
  command -v wget > /dev/null &&
  wget --quiet -O - $1 | tar -xzf -
}

download() {
  if [ -x "$settle_base"/bin/settle ]; then
    echo "settle already exists"
    check_binary && return
  fi
  mkdir -p "$settle_base"/bin && cd "$settle_base"/bin
  if [ $? -ne 0 ]; then
    binary_error="failed to create bin directory"
    return
  fi

  echo "downloading settle"
  local url
  url=https://github.com/danielmmetz/settle/releases/download/v$version/${1}
  set -o pipefail
  if ! (try_curl $url || try_wget $url); then
    set +o pipefail
    binary_error="failed to download with curl and wget"
    return
  fi
  set +o pipefail

  if [ ! -f settle ]; then
    binary_error="failed to download ${1}"
    return
  fi

  chmod +x settle && check_binary
}

archi=$(uname -sm)
binary_available=1
binary_error=""
case "$archi" in
  Darwin\ arm64)   download "settle_$(echo $version)_darwin_arm64.tar.gz" ;;
  Darwin\ x86_64)  download "settle_$(echo $version)_darwin_amd64.tar.gz" ;;
  Linux\ armv8*)   download "settle_$(echo $version)_linux_arm64.tar.gz"  ;;
  Linux\ aarch64*) download "settle_$(echo $version)_linux_arm64.tar.gz"  ;;
  Linux\ x86_64*)  download "settle_$(echo $version)_linux_amd64.tar.gz"  ;;
  *)               binary_available=0 binary_error=1 ;;
esac

cd "$settle_base"
if [ -n "$binary_error" ]; then
  if [ $binary_available -eq 0 ]; then
    echo "no prebuilt binary for $archi"
  else
    echo "  - $binary_error !!!"
  fi
  if command -v go > /dev/null; then
    echo -n "building binary (go install github.com/danielmmetz/settle@latest) ... "
    if go install github.com/danielmmetz/settle@latest; then
      echo "success!"
    else
      echo "install failed: failed to build binary"
      exit 1
    fi
  else
    echo "install failed: go executable not found"
    exit 1
  fi
fi
