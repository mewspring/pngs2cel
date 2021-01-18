#!/bin/bash

set -e

./download_graphics.sh
./split_tiles.sh
./copy_graphic.sh
./gen_cl2.sh
