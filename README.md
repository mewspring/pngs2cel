# pngs2cel

## Installation

```
git clone https://github.com/mewspring/pngs2cel
cd pngs2cel
go build
```

## Usage

### Single PNG image to CEL

```bash
# Convert single PNG image into a corresponding CEL image.
./pngs2cel -o panel8.cel -pal_path /path/to/town.pal panel8.png
```

### Multiple PNG images to CEL

```bash
# Convert multiple PNG images into a corresponding CEL image.
./pngs2cel -o health_orb.cel -pal_path /path/to/town.pal health_0001.png health_0002.png health_0003.png
```

![Custom health and mana orb graphics](inc/cel.png "Custom health and mana orb graphics")

### Multiple PNG images to CL2

```bash
# Convert multiple PNG images into a corresponding CL2 image.
./pngs2cel -cl2 -o portal2.cl2 -pal_path /path/to/town.pal portal_*.png
```

![Custom town portal graphics](inc/cl2.png "Custom town portal graphics")

### Multiple PNG images for multiple direction to CL2 archive

```bash
# Convert multiple PNG images into a corresponding CL2 image.
./pngs2cel -cl2_archive -o cow.cl2 -pal_path /path/to/town.pal dir1/ dir2/ dir3/ dir4/ dir5/ dir6/ dir7/ dir8/
```
