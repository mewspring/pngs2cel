#!/bin/bash

mkdir -p cl2/

pngs2cel -cl2_archive -o cl2/wyvern_breathe.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_breathe/wyvern_breathe_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_die.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_die/wyvern_die_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_fly.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_fly/wyvern_fly_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_hit.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_hit/wyvern_hit_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_hover.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_hover/wyvern_hover_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_ram.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_ram/wyvern_ram_{1,2,3,4,5,6,7,8}
pngs2cel -cl2_archive -o cl2/wyvern_sting.cl2 -pal_path ~/_share_/diabdat/levels/towndata/town.pal wyvern/wyvern_sting/wyvern_sting_{1,2,3,4,5,6,7,8}
