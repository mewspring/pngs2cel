#!/bin/bash

mkdir -p cl2/

if [ $# -ne 1 ]; then
	echo "Usage: gen_cl2.sh /path/to/town.pal"
	exit 1
fi

PAL_PATH=$1

# Add pngs2cel repository root to PATH for the duration of this script.
PATH=../:$PATH


pngs2cel -cl2_archive -o cl2/wyvern_breathe.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_breathe/wyvern_breathe_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_die.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_die/wyvern_die_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_fly.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_fly/wyvern_fly_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_hit.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_hit/wyvern_hit_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_hover.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_hover/wyvern_hover_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_ram.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_ram/wyvern_ram_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_sting.cl2 -pal_path ${PAL_PATH} wyvern/wyvern_sting/wyvern_sting_{1,2,3,4,5,6,7,8}
