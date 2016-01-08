package resizer

import (
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"time"

	"github.com/VoycerAG/gridfs-image-server/server/paint"
	"github.com/disintegration/imaging"
	"github.com/lazywei/go-opencv/opencv"
)

const (
	//TypeSmartcrop will use magic to find the center of attention
	TypeSmartcrop paint.ResizeType = "smartcrop"
)

var (
	//ErrNoFacesFound this error will be produced if no face could be found in the image
	ErrNoFacesFound = errors.New("No faces found")
)

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

type smartcropResizer struct {
	haarcascade     string
	fallbackResizer paint.Resizer
}

var nilFallbackResizer paint.Resizer

func normalizeInput(input image.Image, maxSize int) (image.Image, float64, error) {
	var scale float64
	if input.Bounds().Dx() > maxSize {
		scale = float64(input.Bounds().Dx()) / float64(maxSize)
	} else {
		scale = float64(input.Bounds().Dy()) / float64(maxSize)
	}

	fmt.Printf("Normalizing to %dx%d\n", int(float64(input.Bounds().Dx())/scale), int(float64(input.Bounds().Dy())/scale))
	resized := imaging.Resize(input, int(float64(input.Bounds().Dx())/scale), int(float64(input.Bounds().Dy())/scale), imaging.Lanczos)

	return resized, scale, nil
}

//NewSmartcrop returns a new resizer for the `TypeSmartcrop`
//it needs opencv internally so this resizer
//Warning: will not allow cross compilation
func NewSmartcrop(haarcascade string, fallbackResizer paint.Resizer) paint.Resizer {
	return &smartcropResizer{haarcascade: haarcascade, fallbackResizer: fallbackResizer}
}

func (s smartcropResizer) Resize(input image.Image, dstWidth, dstHeight int) (image.Image, error) {
	if dstWidth < 0 || dstHeight < 0 {
		return nil, fmt.Errorf("Please specify both width and height for your target image")
	}

	if input.Bounds().Dx() < 400 || input.Bounds().Dy() < 300 {
		log.Println("input to small, skipping face detection")
		return imaging.Thumbnail(input, dstWidth, dstHeight, imaging.Lanczos), nil
	}

	start := time.Now()

	scaledInput, scale, err := normalizeInput(input, 1024)
	if err != nil {
		return input, err
	}

	cvImage := opencv.FromImage(scaledInput)
	_, err = os.Stat(s.haarcascade)
	if err != nil {
		return input, err
	}

	cascade := opencv.LoadHaarClassifierCascade(s.haarcascade)
	faces := cascade.DetectObjects(cvImage)

	if len(faces) == 0 {
		return nil, ErrNoFacesFound
	}

	var biggestFace *opencv.Rect

	for _, f := range faces {
		if biggestFace == nil {
			biggestFace = f
			continue
		}

		if biggestFace.Width() < f.Width() {
			biggestFace = f
		}
	}

	if biggestFace == nil {
		return nil, ErrNoFacesFound
	}

	if sub, ok := input.(subImager); ok {
		x := int(float64(biggestFace.X()) * scale)
		y := int(float64(biggestFace.Y()) * scale)
		width := int(float64(biggestFace.Width()) * scale)
		height := int(float64(biggestFace.Height()) * scale)
		dstWidthScaled := int(float64(dstWidth) * scale)
		dstHeightScaled := int(float64(dstHeight) * scale)

		translateX := int(float64(dstWidthScaled-width) / 2)
		translateY := int(float64(dstHeightScaled-height) / 2)

		log.Printf("Translation: (%d|%d)\n", translateX, translateY)

		diffX := x - translateX
		if diffX < 0 {
			diffX = x
		}

		diffY := y - translateY
		if diffY < 0 {
			diffY = y
		}

		toX := x + width + translateX
		toY := y + height + translateY

		log.Printf("Cutout: (%d|%d) to (%d|%d). Face at (%d|%d)\n", diffX, diffY, toX, toY, x, y)

		cropImage := sub.SubImage(image.Rect(diffX, diffY, toX, toY))

		log.Printf("Face detection took %s\n", time.Now().Sub(start))

		return imaging.Thumbnail(cropImage, dstWidth, dstHeight, imaging.Lanczos), nil
	}

	return input, err
}
