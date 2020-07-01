# pngs2cel

## Installation

```
go get github.com/mewspring/pngs2cel
```

## Usage

```bash
# Convert single PNG image into corresponding CEL image.
$GOPATH/bin/pngs2cel -o panel8.cel -pal_path /path/to/town.pal panel8.png
```

```bash
# Convert multiple PNG images into corresponding CEL image.
$GOPATH/bin/pngs2cel -o health_orb.cel -pal_path /path/to/town.pal health_0001.png health_0002.png health_0003.png
```
