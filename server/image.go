package server

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/VoycerAG/gridfs-image-server/server/manipulation"
)

// ResizeImageByEntry resizes an gridfs image stream
func ResizeImageByEntry(originalImage ReadSeekCloser, entry *Entry) (image.Image, string, error) {
	originalImageData, imageFormat, imgErr := image.Decode(originalImage)
	resizer := manipulation.NewResizerByType(entry.Type)

	if imgErr != nil {
		// reset pointer and re read the gridfs image
		originalImage.Seek(0, 0)

		//if resizing with go tools fails, a fallback is implemented to use imagemagick's convert
		unresizedImage, magickError := imageMagickFallback(originalImage)

		if magickError == nil {
			img, err := resizer.Resize(unresizedImage, int(entry.Width), int(entry.Height))
			return img, imageFormat, err
		}

		return nil, imageFormat, imgErr
	}

	img, err := resizer.Resize(originalImageData, int(entry.Width), int(entry.Height))
	return img, imageFormat, err
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
