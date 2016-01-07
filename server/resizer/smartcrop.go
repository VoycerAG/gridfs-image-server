package resizer

import (
	"errors"
	"fmt"
	"image"
	"log"

	"github.com/VoycerAG/gridfs-image-server/server/paint"
	"github.com/VoycerAG/smartcrop"
	"github.com/disintegration/imaging"
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
		Prescale:                         true,
		PrescaleValue:                    400,
	}

	if input.Bounds().Dx() < 400 || input.Bounds().Dy() < 300 {
		log.Println("input to small, skipping face detection")
		return imaging.Thumbnail(input, dstWidth, dstHeight, imaging.Lanczos), nil
	}

	//it only analyzes the image
	crop, err := smartcrop.NewAnalyzerWithCropSettings(cropSettings).FindBestCrop(input, 400, 300)
	if err != nil {
		if err == smartcrop.ErrNoFacesFound {
			log.Println("No faces found, using fallback resizer")
			fallback := paint.CropResizer{}
			return fallback.Resize(input, dstWidth, dstHeight)
		}

		return nil, err
	}

	startX := crop.X
	startY := crop.Y

	log.Printf("Cropping Position: %d x %d Crop: (%d|%d) -> (%d|%d)\n", startX, startY, crop.X, crop.Y, crop.X+crop.Width, crop.Y+crop.Height)
	if sub, ok := input.(subImager); ok {
		cropImage := sub.SubImage(image.Rect(startX, startY, crop.Width, crop.Height))

		//cropImage must now be resized to the desired format

		originalBounds := input.Bounds()
		originalRatio := float64(originalBounds.Dx()) / float64(originalBounds.Dy())

		if dstWidth < 0 {
			dstWidth = int(float64(dstHeight) * originalRatio)
		}

		if dstHeight < 0 {
			dstHeight = int(float64(dstWidth) / originalRatio)
		}

		return imaging.Thumbnail(cropImage, dstWidth, dstHeight, imaging.Lanczos), nil
	}

	return nil, errors.New("Could not crop image")
}
