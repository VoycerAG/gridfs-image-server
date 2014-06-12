package server

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// ResizeImageFromGridfs resizes an gridfs image stream
func ResizeImageFromGridfs(originalImage *mgo.GridFile, entry *Entry) (*image.Image, string, error) {
	originalImageData, imageFormat, imgErr := image.Decode(originalImage)

	if imgErr != nil {
		// reset pointer and re read the gridfs image
		originalImage.Seek(0, 0)
		// now it gets really hacky, since go does not support interlaced pngs
		// http://code.google.com/p/go/issues/detail?id=6293
		// we will call imagemagick in order to remove interlacing, and save the image
		// if this fails as well, there must be something wrong
		unresizedImage, magickError := imageMagickFallback(originalImage)

		if magickError == nil {
			return ResizeImage(unresizedImage, imageFormat, entry)
		}

		return nil, imageFormat, imgErr
	}

	return ResizeImage(originalImageData, imageFormat, entry)
}

// imageMagickFallback is used to convert a image with image magick
func imageMagickFallback(originalImage *mgo.GridFile) (image.Image, error) {
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

// ResizeImage resizes images or crops them if either size is not defined
func ResizeImage(originalImageData image.Image, imageFormat string, entry *Entry) (*image.Image, string, error) {
	if entry.Width < 0 && entry.Height < 0 {
		return nil, "", fmt.Errorf("At least one parameter of width or height must be specified")
	}

	targetHeight := float64(entry.Height)
	targetWidth := float64(entry.Width)

	var dst image.Image
	var err error

	// the Thumbnail method needs correctly adjusted bounds in order to work
	originalBounds := originalImageData.Bounds()
	originalRatio := float64(originalBounds.Dx()) / float64(originalBounds.Dy())

	if entry.Type == TypeResize {
		// the Resize method automatically adjusts ratio based format when one parameter is zero
		if targetWidth < 0 {
			targetWidth = 0
		}

		if targetHeight < 0 {
			targetHeight = 0
		}

		dst = resize.Resize(uint(targetWidth), uint(targetHeight), originalImageData, resize.Lanczos3)
	} else if entry.Type == TypeFit {
		if targetWidth < 0 || targetHeight < 0 {
			return nil, "", fmt.Errorf("When using type fit, both height and width must be specified")
		}

		targetRatio := targetWidth / targetHeight

		if targetRatio < originalRatio {
			targetHeight = targetWidth / originalRatio
		} else {
			targetWidth = targetHeight * originalRatio
		}

		dst = resize.Resize(uint(targetWidth), uint(targetHeight), originalImageData, resize.Lanczos3)
	} else {
		// typeCut
		if targetWidth < 0 {
			targetWidth = float64(targetHeight) * originalRatio
		}

		if targetHeight < 0 {
			targetHeight = float64(targetWidth) / originalRatio
		}

		dst = imaging.Thumbnail(originalImageData, int(targetWidth), int(targetHeight), imaging.Lanczos)
	}

	return &dst, imageFormat, err
}

// EncodeImage encodes the image with the given format
func EncodeImage(targetImage io.Writer, imageData image.Image, imageFormat string) error {
	switch imageFormat {
	case "jpeg":
		jpeg.Encode(targetImage, imageData, &jpeg.Options{JpegMaximumQuality})
	case "png":
		png.Encode(targetImage, imageData)
	case "gif":
		gif.Encode(targetImage, imageData, &gif.Options{256, nil, nil})
	default:
		return fmt.Errorf("invalid imageFormat given")
	}

	return nil
}
