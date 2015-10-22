package paint

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/VoycerAG/gridfs-image-server/server/manipulation"
)

//Controller lets you completely control
//one image
type Controller interface {
	Encode(target io.Writer) error
	Resize(resizeType manipulation.ResizeType, width, height int) error
	Image() image.Image
	Format() string
}

//NewController returns a new instance of a basic controller
func NewController(data io.Reader) (Controller, error) {
	rawData, format, err := image.Decode(data)

	if err != nil {
		return nil, err
	}

	return &basicController{data: rawData, imageFormat: format}, nil
}

type basicController struct {
	data        image.Image
	imageFormat string
}

func (b basicController) Format() string {
	return b.imageFormat
}

func (b basicController) Image() image.Image {
	return b.data
}

func (b *basicController) Resize(resizeType manipulation.ResizeType, width, height int) error {
	resizer := manipulation.NewResizerByType(resizeType)
	data, err := resizer.Resize(b.data, width, height)
	if err != nil {
		return err
	}
	b.data = data
	return nil
}

func (b basicController) Encode(target io.Writer) error {
	switch b.imageFormat {
	case "jpeg":
		jpeg.Encode(target, b.data, &jpeg.Options{jpeg.DefaultQuality})
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		encoder.Encode(target, b.data)
	case "gif":
		gif.Encode(target, b.data, &gif.Options{256, nil, nil})
	default:
		return fmt.Errorf("invalid imageFormat given")
	}

	return nil
}
