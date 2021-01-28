#!/bin/bash

set -e

if [ $# -ne 1 ]; then
	echo "Usage: all.sh /path/to/town.pal"
	exit 1
fi

PAL_PATH=$1

./download_graphics.sh
./split_tiles.sh
./copy_graphic.sh
./gen_cl2.sh ${PAL_PATH}
