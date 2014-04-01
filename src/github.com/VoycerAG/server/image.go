package server

import (
	"code.google.com/p/graphics-go/graphics"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"labix.org/v2/mgo"
)

// ResizeImageFromGridfs resizes an gridfs image stream
func ResizeImageFromGridfs(originalImage *mgo.GridFile, entry *Entry) (*image.Image, string, error) {

	originalImageData, imageFormat, imgErr := image.Decode(originalImage)

	if imgErr != nil {
		return nil, imageFormat, imgErr
	}

	return ResizeImage(originalImageData, imageFormat, entry)
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

	if entry.Type == TypeResize {
		// the Resize method automatically adjusts ratio based format when one parameter is zero
		if targetWidth < 0 {
			targetWidth = 0
		}

		if targetHeight < 0 {
			targetHeight = 0
		}

		dst = resize.Resize(uint(targetWidth), uint(targetHeight), originalImageData, resize.Lanczos3)
	} else {
		// the Thumbnail method needs correctly adjusted bounds in order to work
		originalBounds := originalImageData.Bounds()
		originalRatio := float64(originalBounds.Dx()) / float64(originalBounds.Dy())

		if targetWidth < 0 {
			targetWidth = float64(targetHeight) * originalRatio
		}

		if targetHeight < 0 {
			targetHeight = float64(targetWidth) / originalRatio
		}

		imageRGBA := image.NewRGBA(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
		err = graphics.Thumbnail(imageRGBA, originalImageData)
		dst = imageRGBA.SubImage(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	}

	return &dst, imageFormat, err
}

// EncodeImage encodes the image with the given format
func EncodeImage(targetImage *mgo.GridFile, imageData image.Image, imageFormat string) error {
	switch imageFormat {
	case "jpeg":
		jpeg.Encode(targetImage, imageData, &jpeg.Options{JpegMaximumQuality})
	case "png":
		png.Encode(targetImage, imageData)
	case "gif":

	default:
		return fmt.Errorf("invalid imageFormat given")
	}

	return nil
}
