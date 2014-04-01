package server

import (
	"code.google.com/p/graphics-go/graphics"
	"fmt"
	"github.com/nfnt/resize"
	"image"
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

	if targetWidth < 0 {
		targetWidth = 0
	}

	if targetHeight < 0 {
		targetHeight = 0
	}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	err := graphics.Thumbnail(imageRGBA, originalImageData)

	var dst image.Image

	if entry.Type == TypeResize {
		dst = resize.Resize(uint(targetWidth), uint(targetHeight), originalImageData, resize.Lanczos3)
	} else {
		dst = imageRGBA.SubImage(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	}

	return &dst, imageFormat, err
}
