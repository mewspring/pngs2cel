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
	"path/filepath"
	"strings"

	ciede2000 "github.com/mattn/go-ciede2000"
	"github.com/mewkiz/pkg/imgutil"
	"github.com/pkg/errors"
	"github.com/rickypai/natsort"
)

func usage() {
	const use = `
Usage:
   pngs2cel [OPTIONS]... FILE.png...
   pngs2cel -cl2_archive [OPTIONS]... DIR...`
	fmt.Fprintln(os.Stderr, use[1:])
	flag.PrintDefaults()
}

var (
	// Use CIE Delta E 2000 for colour conversion.
	useCIE2000 bool
	// Threshold amount for Euclidean method.
	useThreshold int
	// Transparent colour value.
	colourKey int
	// Only use colours of the lower part of the palette (128 first colours).
	lowerPal bool
	// Only use colours of the upper part of the palette (128 last colours).
	upperPal bool
)

func main() {
	// Parse command line arguments.
	var (
		// Store output in CL2 format.
		cl2Flag bool
		// Store output in CL2 archive format.
		cl2ArchiveFlag bool
		// CEL image output path.
		output string
		// Path to levels/towndata/town.pal.
		palPath string
	)
	flag.BoolVar(&cl2Flag, "cl2", false, "store output in CL2 format")
	flag.BoolVar(&cl2ArchiveFlag, "cl2_archive", false, "store output in CL2 archive format")
	flag.BoolVar(&useCIE2000, "cie2000", false, "use CIE Delta E 2000 instead of Euclidean colour conversion")
	flag.IntVar(&useThreshold, "threshold", 0, "threshold amount for Euclidean colour conversion")
	flag.IntVar(&colourKey, "col_key", -1, "manually specify RGB value of transparent colour (e.g. 0xFF0000 for red)")
	flag.StringVar(&output, "o", "output.cel", "CEL or CL2 image output path")
	flag.StringVar(&palPath, "pal_path", "town.pal", "path to levels/towndata/town.pal")
	flag.BoolVar(&lowerPal, "lower_pal", false, "only use colours of the lower part of the palette (128 first colours)")
	flag.BoolVar(&upperPal, "upper_pal", false, "only use colours of the upper part of the palette (128 last colours)")
	flag.Usage = usage
	flag.Parse()
	paths := flag.Args()
	if len(paths) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	switch {
	case cl2ArchiveFlag:
		dirs := paths
		natsort.Strings(dirs)
		cl2Archive, err := dirs2CL2Archive(dirs, output, palPath)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		if err := dumpCL2Archive(cl2Archive, output); err != nil {
			log.Fatalf("%+v", err)
		}
	default:
		// Convert PNG images to a single CEL image.
		pngPaths := paths
		natsort.Strings(pngPaths)
		celImg, err := pngs2CEL(pngPaths, output, palPath, cl2Flag, false)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		// Write CEL image to file.
		if cl2Flag && strings.HasSuffix(output, ".cel") {
			output = strings.TrimSuffix(output, ".cel") + ".cl2"
		}
		if err := dumpCEL(celImg, output); err != nil {
			log.Fatalf("%+v", err)
		}
	}
}

// dirs2CL2Archive converts the PNG images found in the given directories to a
// single CL2 archive, writing to output CL2 archive file to the specified
// output path and parsing the town.pal colour palette from the specified PAL
// path.
func dirs2CL2Archive(dirs []string, output, palPath string) (*CL2Archive, error) {
	// Convert the PNG files contained within each directory to a corresponding CEL
	// file.
	var celImgs []*CELImage
	for _, dir := range dirs {
		pngPaths, err := findFilesInDir(dir)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		natsort.Strings(pngPaths)
		const cl2Flag = true
		celImg, err := pngs2CEL(pngPaths, output, palPath, cl2Flag, true)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		celImgs = append(celImgs, celImg)
	}
	// Pack CEL files in CL2 archive.
	cl2Archive := &CL2Archive{
		celImgs: celImgs,
	}
	return cl2Archive, nil
}

// CL2Archive is a CL2 archive containing multiple CEL files.
type CL2Archive struct {
	// CEL files contained within CL2 archive.
	celImgs []*CELImage
}

// pngs2CEL converts the given PNG images to a single CEL image, writing the
// output CEL file to the specified output path and parsing the town.pal colour
// palette from the specified PAL path.
func pngs2CEL(pngPaths []string, output, palPath string, cl2Flag, cl2ArchiveFlag bool) (*CELImage, error) {
	// Parse town.pal.
	pal, err := parsePal(palPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// Parse PNG images.
	var imgs []image.Image
	for _, pngPath := range pngPaths {
		img, err := imgutil.ReadFile(pngPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		imgs = append(imgs, img)
	}
	// Convert PNG images to CEL image.
	switch {
	case cl2Flag, cl2ArchiveFlag:
		return createCL2(imgs, pal, cl2ArchiveFlag), nil
	default:
		return createCEL(imgs, pal), nil
	}
}

// CELImage is a CEL image containing a set of image frames.
type CELImage struct {
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

// createCEL creates a CEL image based on the given image frames and colour
// palette.
func createCEL(imgs []image.Image, pal color.Palette) *CELImage {
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
	celImg := &CELImage{
		nframes:      uint32(nframes),
		frameOffsets: frameOffsets,
		frames:       frames,
	}
	return celImg
}

// createCL2 creates a CL2 image based on the given image frames and colour
// palette.
func createCL2(imgs []image.Image, pal color.Palette, cl2ArchiveFlag bool) *CELImage {
	nframes := len(imgs)
	var frames [][]byte
	for _, img := range imgs {
		var (
			frame  []byte
			header []byte
		)
		if cl2ArchiveFlag {
			frame, header = getCL2EmbeddedFrame(img, pal)
		} else {
			frame, header = getCL2Frame(img, pal)
		}
		frame = append(header, frame...)
		frames = append(frames, frame)
	}
	frameOffsets := make([]uint32, nframes+1)
	frameOffsets[0] = 4 + 4*uint32(len(frameOffsets))
	for i, frame := range frames {
		frameOffsets[i+1] = uint32(len(frame)) + frameOffsets[i]
	}
	celImg := &CELImage{
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
				idx := byte(FindClosest(pal, c))
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

// rleEncode returns an RLE-encoded version of the given pixels.
func rleEncode(pixels []byte) (outPixels, buf []byte) {
	var i int
	start := 0
	for i = 0; i < len(pixels); {
		n := runLength(pixels[i:])
		if n >= 3 {
			// store regular pixels.
			if len(pixels[start:i]) > 0 {
				m := len(pixels[start:i])
				cmd := uint8(int8(-m))
				buf = append(buf, cmd)
				buf = append(buf, pixels[start:i]...)
				start = i
				return pixels[start:], buf
			}
			// store RLE-encoded pixels.
			idx := pixels[i]
			cmd := uint8(int8(-(n + 65)))
			buf = append(buf, cmd)
			buf = append(buf, idx)
			i += n
			start = i
			return pixels[start:], buf
		} else {
			i++
		}
	}
	// store regular pixels.
	if len(pixels[start:i]) > 0 {
		m := len(pixels[start:i])
		cmd := uint8(int8(-m))
		buf = append(buf, cmd)
		buf = append(buf, pixels[start:i]...)
		start = i
	}
	return pixels[start:], buf
}

// runLength returns the number of identical pixels in a row, as used for
// run-length encoding.
func runLength(pixels []byte) int {
	n := 1
	b := pixels[0]
	for j := 1; j < len(pixels); j++ {
		if pixels[j] != b {
			return n
		}
		n++
	}
	return n
}

// getCL2EmbeddedFrame converts the given image to the corresponding CL2 frame
// contents (as embedded within a CL2 archive), using the specified palette for
// colours.
func getCL2EmbeddedFrame(img image.Image, pal color.Palette) (frame, header []byte) {
	bounds := img.Bounds()
	ntrans := 0       // transparent pixels.
	var pixels []byte // regular pixels.
	// Set regular pixels.
	setRegular := func() {
		var buf []byte
		pixels, buf = rleEncode(pixels)
		frame = append(frame, buf...)
		//cmd := byte(-len(pixels))
		//frame = append(frame, cmd)
		//frame = append(frame, pixels...)
		//pixels = pixels[:0] // reset pixel buffer.
	}
	setAllRegular := func() {
		for len(pixels) > 0 {
			setRegular()
		}
	}
	// Set transparent pixels.
	setTrans := func() {
		t := byte(ntrans)
		frame = append(frame, t)
		ntrans = 0
	}
	const headerSize = 10
	header = []byte{
		0x0A, 0x00, // offset to pixel row 0 (0xA bytes)
		0x00, 0x00, // offset to pixel row 32 (placehodler value)
		0x00, 0x00, // offset to pixel row 64 (placehodler value)
		0x00, 0x00, // offset to pixel row 96 (placehodler value)
		0x00, 0x00, // offset to pixel row 128 (placehodler value)
	}
	i := 0
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		if (y+1)%32 == 0 && i*2+1 < len(header) {
			// Flush pixel line for Slab table.
			if len(pixels) > 0 {
				setAllRegular()
			}
			if ntrans > 0 {
				setTrans()
			}
			offset := headerSize + len(frame)
			binary.LittleEndian.PutUint16(header[i*2:], uint16(offset))
			i++
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if isTransparent(c) {
				if len(pixels) > 0 {
					setAllRegular()
				}
				ntrans++
			} else {
				if ntrans > 0 {
					setTrans()
				}
				idx := byte(FindClosest(pal, c))
				pixels = append(pixels, idx)
			}
			// -1 through -65
			lastPixel := x == bounds.Max.X-1 && y == bounds.Min.Y
			if lastPixel && len(pixels) > 0 {
				setAllRegular()
				continue
			}
			if len(pixels) >= 65 { // TODO: double check; should be `len(pixels) >= 64`?
				// Append one control byte sequence of pixels. This is to ensure
				// that no more than 65 non-run length pixels fit into a control
				// byte. The pixel array is not emptied entirely, it is just made
				// one control byte sequence shorter.
				setRegular()
				continue
			}
			if ntrans >= 0x7F || (ntrans > 0 && lastPixel) {
				setTrans()
				continue
			}
		}
	}
	return frame, header
}

// getCL2Frame converts the given image to the corresponding CL2 frame contents,
// using the specified palette for colours.
func getCL2Frame(img image.Image, pal color.Palette) (frame, header []byte) {
	bounds := img.Bounds()
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
	const headerSize = 10
	header = []byte{
		0x0A, 0x00, // offset to pixel row 0 (0xA bytes)
		0x00, 0x00, // offset to pixel row 32 (placehodler value)
		0x00, 0x00, // offset to pixel row 64 (placehodler value)
		0x00, 0x00, // offset to pixel row 96 (placehodler value)
		0x00, 0x00, // offset to pixel row 128 (placehodler value)
	}
	i := 0
	for y := bounds.Max.Y - 1; y >= 0; y-- {
		if (y+1)%32 == 0 {
			offset := headerSize + len(frame)
			binary.LittleEndian.PutUint16(header[i*2:], uint16(offset))
			i++
		}
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
				idx := byte(FindClosest(pal, c))
				pixels = append(pixels, idx)
			}
			lastPixel := x == bounds.Max.X-1 && y == bounds.Min.Y
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
	return frame, header
}

// dumpCEL writes the given CEL image in binary format to the specified output
// path.
func dumpCEL(celImg *CELImage, output string) error {
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

// dumpCL2Archive writes the given CL2 archive in binary format to the specified
// output path.
func dumpCL2Archive(cl2Archive *CL2Archive, output string) error {
	f, err := os.Create(output)
	if err != nil {
		return errors.WithStack(err)
	}
	// CL2Archive format:
	//
	//    type CL2Archive struct {
	//       // CL2 archive header.
	//       Hdr CL2ArchiveHeader
	//       // CEL headers.
	//       CELHdrs [ndirs]CELHeader
	//       // CEL bodies, containing pixel data.
	//       CELBodies [ndirs]CELBody
	//    }
	//
	//    // Number of directions; one per embedded CEL file.
	//    const ndirs = 8
	//
	//    type CL2ArchiveHeader struct {
	//       // offset from start of CL2 archive file to CEL header start.
	//       CELHdrOffsets [ndirs]uint32
	//    }
	//
	//    type CELHeader struct {
	//       // Number of frames in CEL file.
	//       NFrames uint32
	//       // CEL body offsets; from start of this CEL header.
	//       CELBodyOffsets [NFrames+1]uint32
	//    }
	//
	//    type CELBody struct {
	//       // slab line offsets into pixel data.
	//       LineOffets [5]uint16
	//       // RLE-encoded pixel data
	//       Data []byte
	//    }
	const (
		// Number of directions; one per embedded CEL file.
		ndirs = 8
		// 2 bytes.
		uint16Size = 2
		// 4 bytes.
		uint32Size = 4
	)
	// Write CL2 archive header.
	cl2ArchiveHdrSize := ndirs * uint32Size
	nframes := cl2Archive.celImgs[0].nframes
	celHdrSize := 1*uint32Size + (nframes+1)*uint32Size
	var celHdrOffsets [ndirs]uint32
	offset := uint32(cl2ArchiveHdrSize)
	for i := range celHdrOffsets {
		celHdrOffsets[i] = uint32(offset)
		offset += uint32(celHdrSize)
	}
	if err := binary.Write(f, binary.LittleEndian, celHdrOffsets); err != nil {
		return errors.WithStack(err)
	}
	// Write CEL headers.
	var celBodySizes [ndirs]uint32
	for i := range celBodySizes {
		celImg := cl2Archive.celImgs[i]
		size := 5 * uint16Size // size of slab line offsets into data array.
		for _, frame := range celImg.frames {
			size += len(frame)
		}
		celBodySizes[i] = uint32(size)
	}
	// offset currently at start of first cel body.
	for i, celImg := range cl2Archive.celImgs {
		for j := range celImg.frameOffsets {
			celImg.frameOffsets[j] = offset - celHdrOffsets[i]
			if j < int(celImg.nframes) {
				offset += uint32(len(celImg.frames[j]))
			}
		}
	}
	// Write CEL headers.
	for _, celImg := range cl2Archive.celImgs {
		if err := binary.Write(f, binary.LittleEndian, celImg.nframes); err != nil {
			return errors.WithStack(err)
		}
		if err := binary.Write(f, binary.LittleEndian, celImg.frameOffsets); err != nil {
			return errors.WithStack(err)
		}
	}
	// Write CEL bodies.
	for _, celImg := range cl2Archive.celImgs {
		for _, frame := range celImg.frames {
			if _, err := f.Write(frame); err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

// ### [ Helper functions ] ####################################################

// isTransparent reports whether the given colour is transparent.
func isTransparent(c color.Color) bool {
	r, g, b, a := c.RGBA()
	if a < 32768 { // treat < 50% alpha as transparent.
		return true
	}

	if colourKey >= 0 { // user-specified alpha RGB
		rr := (colourKey >> 16) & 0xFF
		gg := (colourKey >> 8) & 0xFF
		bb := colourKey & 0xFF
		if int(r>>8) == rr && int(g>>8) == gg && int(b>>8) == bb {
			return true
		}
	}

	return false
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
	// Null out skipped indices, we'll force index 0 for black later
	switch {
	case lowerPal:
		// Set indices 128-254 to black
		for i := 128; i <= 254; i++ {
			pal[i] = color.RGBA{
				R: 0x00,
				G: 0x00,
				B: 0x00,
				A: 0xFF,
			}
		}
	case upperPal:
		// Set indices 1-127 to black
		for i := 1; i <= 127; i++ {
			pal[i] = color.RGBA{
				R: 0x00,
				G: 0x00,
				B: 0x00,
				A: 0xFF,
			}
		}
	}
	return pal, nil
}

// findFilesInDir returns a list of paths to the files located in the given
// directory.
func findFilesInDir(dir string) ([]string, error) {
	var filePaths []string
	// TODO: use fs.FileInfo directly when Go 1.16 has been released.
	visit := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		if info.IsDir() {
			return nil
		}
		filePaths = append(filePaths, path)
		return nil
	}
	if err := filepath.Walk(dir, visit); err != nil {
		return nil, errors.WithStack(err)
	}
	return filePaths, nil
}

// IndexCIEDE2000 returns the index of the palette colour closest to c using the
// CIE Delta E 2000 Color-Difference algorithm.
func IndexCIEDE2000(pal color.Palette, orig color.Color) int {
	var (
		bestDiff float64
		ret      int
	)
	for i, c2 := range pal {
		diff := ciede2000.Diff(orig, c2)
		if diff == 0 {
			return i
		}
		if i == 0 || diff < bestDiff {
			bestDiff = diff
			ret = i
		}
	}
	return ret
}

// Channel specifies a colour channel.
type Channel int

// Colour channels.
const (
	ChannelGray  Channel = 0
	ChannelRed   Channel = 1
	ChannelGreen Channel = 2
	ChannelBlue  Channel = 3
)

// GreatestColor finds the brightest colour in an R,G,B space.
func GreatestColor(r int, g int, b int) Channel {
	// Note that the human eye is more sensitive to green hence the reason
	// 16-bit colors often use 6 bits for green and 5 for red/blue.
	// This can be simulated in 32-bit by increasing green by 20% and may
	// produce more accurate results with images heavy in yellow/green.
	// g += g / 5
	if r > g && r > b {
		return ChannelRed
	}
	if g > r && g > b {
		return ChannelGreen
	}
	if b > r && b > g {
		return ChannelBlue
	}
	// All channels are equal so the color is grayscale
	return ChannelGray
}

// IndexMult returns the index of the palette colour closest to c in Euclidean
// R,G,B,A space. Strongest colour multiplied by the threshold value.
func IndexMult(p color.Palette, c color.Color, thresh uint32) int {
	cr, cg, cb, _ := c.RGBA()
	// Is this colour visibly red, green, or blue?
	brightest := GreatestColor(int(cr), int(cg), int(cb))
	ret, bestSum := 0, uint32(1<<32-1)
	for i, v := range p {
		vr, vg, vb, _ := v.RGBA()
		rr := sqDiff(cr, vr)
		gg := sqDiff(cg, vg)
		bb := sqDiff(cb, vb)
		switch brightest {
		case ChannelRed:
			rr *= thresh
		case ChannelGreen:
			gg *= thresh
		case ChannelBlue:
			bb *= thresh
		}
		sum := rr + gg + bb
		if sum < bestSum {
			if sum == 0 {
				return i
			}
			ret, bestSum = i, sum
		}
	}
	return ret
}

// sqDiff returns the squared-difference of x and y, shifted by 2 so that
// adding three of those won't overflow a uint32.
//
// x and y are both assumed to be in the range [0, 0xffff].
func sqDiff(x, y uint32) uint32 {
	d := x - y
	return (d * d) >> 2
}

// FindClosest returns the palette index of the closest colour to orig based on
// the chosen colour matching algorithm.
func FindClosest(pal color.Palette, orig color.Color) int {
	var idx int
	if useCIE2000 {
		idx = IndexCIEDE2000(pal, orig)
	} else if useThreshold != 0 {
		idx = IndexMult(pal, orig, uint32(useThreshold))
	} else {
		idx = pal.Index(orig)
	}
	if lowerPal || upperPal {
		// Ensure index 0 is always chosen for black
		r, g, b, _ := pal[idx].RGBA()
		if r == 0 && g == 0 && b == 0 {
			return 0
		}
	}
	return idx
}
