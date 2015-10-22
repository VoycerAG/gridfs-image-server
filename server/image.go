package server

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/disintegration/imaging"
)

//Resizer can resize an image
//dstWidth and dstHeight are the desired output values
//but it is not promised that the output image has exactly those bounds
type Resizer interface {
	Resize(input image.Image, dstWidth, dstHeight int) (image.Image, error)
}

// ResizeImageByEntry resizes an gridfs image stream
func ResizeImageByEntry(originalImage ReadSeekCloser, entry *Entry) (image.Image, string, error) {
	originalImageData, imageFormat, imgErr := image.Decode(originalImage)

	if imgErr != nil {
		// reset pointer and re read the gridfs image
		originalImage.Seek(0, 0)

		//if resizing with go tools fails, a fallback is implemented to use convett
		unresizedImage, magickError := imageMagickFallback(originalImage)

		if magickError == nil {
			img, err := ResizeImage(unresizedImage, entry)
			return img, imageFormat, err
		}

		return nil, imageFormat, imgErr
	}

	img, err := ResizeImage(originalImageData, entry)
	return img, imageFormat, err
}

type plainResizer struct {
}

func (p plainResizer) Resize(input image.Image, dstWidth, dstHeight int) (image.Image, error) {
	if dstWidth < 0 && dstHeight < 0 {
		return nil, fmt.Errorf("Either width or height must be greater zero to keep the existing ratio")
	}

	//since we use -1 as optional and imaging uses zero as optional
	//we change -1 to 0 to keep the aspect ratio
	if dstWidth < 0 {
		dstWidth = 0
	}

	if dstHeight < 0 {
		dstHeight = 0
	}

	return imaging.Resize(input, dstWidth, dstHeight, imaging.Lanczos), nil
}

type fitResizer struct {
}

func (f fitResizer) Resize(input image.Image, dstWidth, dstHeight int) (image.Image, error) {
	if dstWidth < 0 || dstHeight < 0 {
		return nil, fmt.Errorf("Please specify both width and height for your target image")
	}

	originalBounds := input.Bounds()
	originalRatio := float64(originalBounds.Dx()) / float64(originalBounds.Dy())

	targetRatio := float64(dstWidth) / float64(dstHeight)

	if targetRatio < originalRatio {
		dstHeight = int(float64(dstWidth) / originalRatio)
	} else {
		dstWidth = int(float64(dstHeight) * originalRatio)
	}

	return imaging.Resize(input, int(dstWidth), int(dstHeight), imaging.Lanczos), nil
}

type cropResizer struct {
}

func (c cropResizer) Resize(input image.Image, dstWidth, dstHeight int) (image.Image, error) {
	if dstWidth < 0 && dstHeight < 0 {
		return nil, fmt.Errorf("Either width or height must be greater zero to keep the existing ratio")
	}

	originalBounds := input.Bounds()
	originalRatio := float64(originalBounds.Dx()) / float64(originalBounds.Dy())

	if dstWidth < 0 {
		dstWidth = int(float64(dstHeight) * originalRatio)
	}

	if dstHeight < 0 {
		dstHeight = int(float64(dstWidth) / originalRatio)
	}

	return imaging.Thumbnail(input, dstWidth, dstHeight, imaging.Lanczos), nil
}

// ResizeImage resizes images or crops them if either size is not defined
func ResizeImage(originalImageData image.Image, entry *Entry) (image.Image, error) {
	resizers := map[string]Resizer{
		TypeResize: plainResizer{},
		TypeFit:    fitResizer{},
		TypeCrop:   cropResizer{},
	}

	resizer, found := resizers[entry.Type]

	if !found {
		// an error here would be a regression
		// so for now we use a fallback behaviour
		// in the future we can refactor it to
		// only support registered resizers
		resizer = resizers[TypeResize]
	}

	dst, err := resizer.Resize(originalImageData, int(entry.Width), int(entry.Height))

	return dst, err
}

// EncodeImage encodes the image with the given format
func EncodeImage(targetImage io.Writer, imageData image.Image, imageFormat string) error {
	switch imageFormat {
	case "jpeg":
		jpeg.Encode(targetImage, imageData, &jpeg.Options{jpeg.DefaultQuality})
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		encoder.Encode(targetImage, imageData)
	case "gif":
		gif.Encode(targetImage, imageData, &gif.Options{256, nil, nil})
	default:
		return fmt.Errorf("invalid imageFormat given")
	}

	return nil
}

// deprecated this will be removed in a future release
// imageMagickFallback is used to convert a image with image magick
func imageMagickFallback(originalImage ReadSeekCloser) (image.Image, error) {
	tempDirectory := os.TempDir()

	file, err := ioutil.TempFile(tempDirectory, "magick_original_")
	target, targetErr := ioutil.TempFile(tempDirectory, "magick_target_")

	if targetErr != nil {
		return nil, targetErr
	}

	defer syscall.Unlink(target.Name())

	if err != nil {
		return nil, err
	}

	defer syscall.Unlink(file.Name())

	io.Copy(file, originalImage)
	file.Close()
	originalImage.Close()

	log.Printf("convert %s %s", file.Name(), target.Name())

	cmd := exec.Command("convert", file.Name(), target.Name())

	// blocking execution, since we need to read the image afterwards
	someErr := cmd.Run()

	if someErr != nil {
		return nil, someErr
	}

	targetPNG, openErr := os.Open(target.Name())

	if openErr != nil {
		return nil, openErr
	}

	return png.Decode(targetPNG)
}
