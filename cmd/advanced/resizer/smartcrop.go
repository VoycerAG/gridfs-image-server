package resizer

import (
	"errors"
	"fmt"
	"image"

	"github.com/VoycerAG/gridfs-image-server/server/paint"
	"github.com/muesli/smartcrop"
	"github.com/nfnt/resize"
)

const (
	//TypeSmartcrop will use magic to find the center of attention
	TypeSmartcrop paint.ResizeType = "smartcrop"
)

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

type smartcropResizer struct {
	haarcascade string
}

//NewSmartcrop returns a new resizer for the `TypeSmartcrop`
//it needs opencv internally so this resizer
//WILL NOT ALLOW CROSS COMPILE
func NewSmartcrop(haarcascade string) paint.Resizer {
	return &smartcropResizer{haarcascade: haarcascade}
}

func (s smartcropResizer) Resize(input image.Image, dstWidth, dstHeight int) (image.Image, error) {
	if dstWidth < 0 || dstHeight < 0 {
		return nil, fmt.Errorf("Please specify both width and height for your target image")
	}

	cropSettings := smartcrop.CropSettings{
		FaceDetection:                    true,
		FaceDetectionHaarCascadeFilepath: s.haarcascade,
		InterpolationType:                resize.Bicubic,
		DebugMode:                        false,
	}

	//it only analyzes the image
	crop, err := smartcrop.NewAnalyzerWithCropSettings(cropSettings).FindBestCrop(input, dstWidth, dstHeight)
	if err != nil {
		return nil, err
	}

	if sub, ok := input.(subImager); ok {
		return sub.SubImage(image.Rect(crop.X, crop.Y, crop.Width+crop.X, crop.Height+crop.Y)), nil
	}

	return nil, errors.New("Could not crop image")
}
