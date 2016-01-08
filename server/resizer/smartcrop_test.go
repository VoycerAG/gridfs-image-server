package resizer_test

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"

	. "github.com/VoycerAG/gridfs-image-server/server/resizer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// png file formats
	"image/png"
	// gif file formats
	"image/gif"
	// jpeg file formats
	"image/jpeg"
)

const (
	inputFolder  = "./testimages/"
	outputFolder = "./testimages_output/"
	haarCascade  = "../../scripts/haarcascade_frontalface_alt.xml"
)

var _ = Describe("Smartcrop testsuite", func() {
	loadImage := func(path string) (image.Image, string, error) {
		fp, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}

		return image.Decode(fp)
	}

	saveImage := func(path string, format string, i image.Image) error {
		fp, err := os.Create(path)
		if err != nil {
			return err
		}

		defer fp.Close()
		switch format {
		case "jpeg":
			jpeg.Encode(fp, i, &jpeg.Options{jpeg.DefaultQuality})
		case "png":
			encoder := png.Encoder{CompressionLevel: png.BestCompression}
			encoder.Encode(fp, i)
		case "gif":
			gif.Encode(fp, i, &gif.Options{256, nil, nil})
		default:
			return fmt.Errorf("invalid imageFormat given")
		}

		return nil
	}

	Measure("it will generate multiple face detected images", func(b Benchmarker) {
		b.Time("runtime", func() {
			resizer := NewSmartcrop(haarCascade, nil)
			facesFound := 0
			noFacesFound := 0
			errors := 0
			dstWidth := 300
			dstHeight := 294

			err := filepath.Walk(inputFolder, func(path string, f os.FileInfo, err error) error {
				if f.IsDir() || filepath.Base(path) == ".gitkeep" {
					return nil
				}

				image, format, err := loadImage(path)
				if err != nil {
					errors++
					return nil
				}

				output, err := resizer.Resize(image, dstWidth, dstHeight)
				if err != nil {
					if err == ErrNoFacesFound {
						noFacesFound++
						return nil
					}

					errors++
					return nil
				}

				outputPath := strings.Replace(path, "testimages", "testimages_output", 1)
				log.Printf("Writing %s\n", outputPath)
				saveImage(outputPath, format, output)
				facesFound++
				return nil
			})

			log.Printf("Faces %d, No Faces %d, errors %d\n", facesFound, noFacesFound, errors)
			Expect(err).ToNot(HaveOccurred())
			Expect(errors).To(Equal(0))
		})
	}, 1)
})
