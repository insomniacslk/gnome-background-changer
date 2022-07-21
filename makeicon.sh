#!/bin/bash
set -exu

icon=$1

icon_name=Icon
package_name=main

# convert your square image to 64x64, e.g. convert -resize 64x yourimage.png yourimage.ico
# call this script on yourimage.ico
go run ithub.com/cratonica/2goarray "${icon_name}" "${package_name}" < "${1}" > icon.go
