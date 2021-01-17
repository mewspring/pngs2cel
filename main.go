// The pngs2cel tool converts PNG images to a single CEL image.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/mewkiz/pkg/imgutil"
	"github.com/pkg/errors"
)

func usage() {
	const use = `
Usage: pngs2cel [OPTIONS]... FILE.png...`
	fmt.Fprintln(os.Stderr, use[1:])
	flag.PrintDefaults()
}

func main() {
	// Parse command line arguments.
	var (
		// Store output in CL2 format.
		cl2 bool
		// CEL image output path.
		output string
		// Path to levels/towndata/town.pal.
		palPath string
	)
	flag.BoolVar(&cl2, "cl2", false, "store output in Cl2 format")
	flag.StringVar(&output, "o", "output.cel", "CEL image output path")
	flag.StringVar(&palPath, "pal_path", "town.pal", "path to levels/towndata/town.pal")
	flag.Usage = usage
	flag.Parse()
	pngPaths := flag.Args()
	if len(pngPaths) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	// Convert PNG images to a single CEL image.
	if err := pngs2cel(pngPaths, output, palPath, cl2); err != nil {
		log.Fatalf("%+v", err)
	}
}

// pngs2cel converts the given PNG images to a single CEL image, writing the
// output CEL file to the specified output path and parsing the town.pal colour
// palette from the specified PAL path.
func pngs2cel(pngPaths []string, output, palPath string, cl2 bool) error {
	// Parse town.pal.
	pal, err := parsePal(palPath)
	if err != nil {
		return errors.WithStack(err)
	}
	// Parse PNG images.
	var imgs []image.Image
	for _, pngPath := range pngPaths {
		img, err := imgutil.ReadFile(pngPath)
		if err != nil {
			return errors.WithStack(err)
		}
		imgs = append(imgs, img)
	}
	// Convert PNG images to CEL image.
	var celImg celImage
	switch {
	case cl2:
		celImg = createCl2(imgs, pal)
	default:
		celImg = createCel(imgs, pal)
	}
	// Write CEL image to file.
	if cl2 && strings.HasSuffix(output, ".cel") {
		output = strings.TrimSuffix(output, ".cel") + ".cl2"
	}
	if err := dumpCel(celImg, output); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// celImage is a CEL image containing a set of image frames.
type celImage struct {
	// Number of frames.
	nframes uint32
	// Offset to each frame.
	frameOffsets []uint32 // [nframes+1]uint32
	// Header and pixel data contents of each frame.
	//
	//    start: frameOffsets[frameNum]
	//    end:   frameOffsets[frameNum+1]
	frames [][]byte // [nframes]Frame
}

// createCel creates a CEL image based on the given image frames and colour
// palette.
func createCel(imgs []image.Image, pal color.Palette) celImage {
	nframes := len(imgs)
	var frames [][]byte
	for _, img := range imgs {
		frame := getCelFrame(img, pal)
		frames = append(frames, frame)
	}
	frameOffsets := make([]uint32, nframes+1)
	frameOffsets[0] = 4 + 4*uint32(len(frameOffsets))
	for i, frame := range frames {
		frameOffsets[i+1] = uint32(len(frame)) + frameOffsets[i]
	}
	celImg := celImage{
		nframes:      uint32(nframes),
		frameOffsets: frameOffsets,
		frames:       frames,
	}
	return celImg
}

// createCl2 creates a CL2 image based on the given image frames and colour
// palette.
func createCl2(imgs []image.Image, pal color.Palette) celImage {
	nframes := len(imgs)
	var frames [][]byte
	for _, img := range imgs {
		frame := getCl2Frame(img, pal)
		header := []byte{0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		frame = append(header, frame...)
		frames = append(frames, frame)
	}
	frameOffsets := make([]uint32, nframes+1)
	frameOffsets[0] = 4 + 4*uint32(len(frameOffsets))
	for i, frame := range frames {
		frameOffsets[i+1] = uint32(len(frame)) + frameOffsets[i]
	}
	celImg := celImage{
		nframes:      uint32(nframes),
		frameOffsets: frameOffsets,
		frames:       frames,
	}
	return celImg
}

// getCelFrame converts the given image to the corresponding CEL frame contents,
// using the specified palette for colours.
func getCelFrame(img image.Image, pal color.Palette) []byte {
	bounds := img.Bounds()
	var frame []byte
	ntrans := 0       // transparent pixels.
	var pixels []byte // regular pixels.
	// Set regular pixels.
	setRegular := func() {
		cmd := byte(len(pixels))
		frame = append(frame, cmd)
		frame = append(frame, pixels...)
		pixels = pixels[:0] // reset pixel buffer.
	}
	// Set transparent pixels.
	setTrans := func() {
		t := byte(-ntrans)
		frame = append(frame, t)
		ntrans = 0
	}
	for y := bounds.Max.Y - 1; y >= 0; y-- {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if isTransparent(c) {
				if len(pixels) > 0 {
					setRegular()
				}
				ntrans++
			} else {
				if ntrans > 0 {
					setTrans()
				}
				idx := byte(pal.Index(c))
				pixels = append(pixels, idx)
			}
			lastPixelOnRow := x == bounds.Max.X-1
			if len(pixels) >= 0x7F || (len(pixels) > 0 && lastPixelOnRow) {
				setRegular()
				continue
			}
			if ntrans >= 0x80 || (ntrans > 0 && lastPixelOnRow) {
				setTrans()
				continue
			}
		}
	}
	return frame
}

// getCl2Frame converts the given image to the corresponding CL2 frame contents,
// using the specified palette for colours.
func getCl2Frame(img image.Image, pal color.Palette) []byte {
	bounds := img.Bounds()
	var frame []byte
	ntrans := 0       // transparent pixels.
	var pixels []byte // regular pixels.
	// Set regular pixels.
	setRegular := func() {
		cmd := byte(-len(pixels))
		frame = append(frame, cmd)
		frame = append(frame, pixels...)
		pixels = pixels[:0] // reset pixel buffer.
	}
	// Set transparent pixels.
	setTrans := func() {
		t := byte(ntrans)
		frame = append(frame, t)
		ntrans = 0
	}
	for y := bounds.Max.Y - 1; y >= 0; y-- {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if isTransparent(c) {
				if len(pixels) > 0 {
					setRegular()
				}
				ntrans++
			} else {
				if ntrans > 0 {
					setTrans()
				}
				idx := byte(pal.Index(c))
				pixels = append(pixels, idx)
			}
			lastPixel := x == bounds.Max.X-1 && y == bounds.Max.Y-1
			// -1 through -65
			if len(pixels) >= 65 || (len(pixels) > 0 && lastPixel) {
				setRegular()
				continue
			}
			if ntrans >= 0x7F || (ntrans > 0 && lastPixel) {
				setTrans()
				continue
			}
		}
	}
	return frame
}

// dumpCel writes the given CEL image in binary format to the specified output
// path.
func dumpCel(celImg celImage, output string) error {
	f, err := os.Create(output)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := binary.Write(f, binary.LittleEndian, celImg.nframes); err != nil {
		return errors.WithStack(err)
	}
	if err := binary.Write(f, binary.LittleEndian, celImg.frameOffsets); err != nil {
		return errors.WithStack(err)
	}
	for _, frame := range celImg.frames {
		if _, err := f.Write(frame); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// ### [ Helper functions ] ####################################################

// isTransparent reports whether the given colour is transparent.
func isTransparent(c color.Color) bool {
	_, _, _, a := c.RGBA()
	return a == 0
}

// parsePal parses the given PAL file and returns the corresponding palette.
//
// Below follows a pseudo-code description of the PAL file format.
//
//    // A PAL file contains a sequence of colour definitions, representing a
//    // palette.
//    type PAL [256]Color
//
//    // A Color represents a colour specified by red, green and blue intensity
//    // levels.
//    type Color struct {
//       red, green, blue byte
//    }
func parsePal(palPath string) (color.Palette, error) {
	buf, err := ioutil.ReadFile(palPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	const (
		// Number of colours within a palette.
		ncolors = 256
		// The size of each colour in bytes.
		colorSize = 3
	)
	if len(buf) != ncolors*colorSize {
		return nil, errors.Errorf("invalid PAL file size for %q; expected %d, got %d", palPath, ncolors*colorSize, len(buf))
	}
	pal := make(color.Palette, ncolors)
	for i := range pal {
		pal[i] = color.RGBA{
			R: buf[i*colorSize],
			G: buf[i*colorSize+1],
			B: buf[i*colorSize+2],
			A: 0xFF,
		}
	}
	return pal, nil
}
