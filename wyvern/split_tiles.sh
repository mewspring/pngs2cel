#!/bin/bash

set -e

mkdir -p wyvern_tiles/

convert \
	wyvern_noshadow.png \
	-crop 256x256 \
	-set filename:tile "%[fx:page.y / 256 + 1]_%[fx:page.x/256 + 1]" \
	+repage +adjoin \
	"wyvern_tiles/wyvern_tile_%[filename:tile].png"

find wyvern_tiles -type f | xargs -I '{}' convert '{}' -crop 128x128+64+64 '{}'
