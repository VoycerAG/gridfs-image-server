package paint_test

import (
	"bufio"
	"bytes"
	"image"
	"io"
	"os"

	// png file formats
	_ "image/png"
	// gif file formats
	_ "image/gif"
	// jpeg file formats
	_ "image/jpeg"

	. "github.com/VoycerAG/gridfs-image-server/server/paint"

	. "github.com/sharpner/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller to resize all types of images", func() {
	loadImage := func(path string) (image.Image, error) {
		fp, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		img, _, err := image.Decode(fp)
		return img, err
	}

	saveImage := func(path string, r io.Reader) error {
		fp, err := os.Create(path)
		if err != nil {
			return err
		}

		defer fp.Close()
		io.Copy(fp, r)
		return nil
	}

	_ = saveImage

	Context("Resize for all file types", func() {
		It("should resize normal image.jpg", func() {
			testFile, err := os.Open("../testdata/image.jpg")
			Expect(err).ToNot(HaveOccurred())
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeResize, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/resize_20_10_image.jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
			Expect(controller.Format()).To(Equal("jpeg"))
		})

		It("should resize normal non-animated.gif", func() {
			testFile, err := os.Open("../testdata/non-animated.gif")
			Expect(err).ToNot(HaveOccurred())
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeResize, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/resize_20_10_non-animated.gif")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
			Expect(controller.Format()).To(Equal("gif"))
		})

		It("should resize normal png", func() {
			testFile, err := os.Open("../testdata/normal.png")
			Expect(err).ToNot(HaveOccurred())
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeResize, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/resize_20_10_normal.png")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
			Expect(controller.Format()).To(Equal("png"))
		})

		It("should resize transparent png", func() {
			testFile, err := os.Open("../testdata/transparent.png")
			Expect(err).ToNot(HaveOccurred())
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeResize, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/resize_20_10_transparent.png")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
			Expect(controller.Format()).To(Equal("png"))
		})

		It("should resize interlaced png", func() {
			testFile, err := os.Open("../testdata/interlaced.png")
			Expect(err).ToNot(HaveOccurred())
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeResize, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/resize_20_10_interlaced.png")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
			Expect(controller.Format()).To(Equal("png"))
		})
	})

	Context("JPG Manipulation", func() {
		var (
			testFile io.Reader
			err      error
		)

		BeforeEach(func() {
			testFile, err = os.Open("../testdata/failure.jpg")
			Expect(err).ToNot(HaveOccurred())
		})
		It("should be resized by type Resize", func() {
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeResize, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			//encode actual image to have the same compression
			//as the result file
			//this check could probably fail with every go version which introduces
			//encoding changes
			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/resize_20_10_failure.jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
		})

		It("should be resized by type Fit", func() {
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeFit, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(13))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/fit_20_10_failure.jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
		})

		It("should be resized by type Crop", func() {
			controller, err := NewController(testFile, map[ResizeType]Resizer{})
			Expect(err).ToNot(HaveOccurred())
			controller.Resize(TypeCrop, 20, 10)
			Expect(controller.Image().Bounds().Dx()).To(Equal(20))
			Expect(controller.Image().Bounds().Dy()).To(Equal(10))

			var buffer bytes.Buffer
			w := bufio.NewWriter(&buffer)
			controller.Encode(w)
			w.Flush()
			actual, _, err := image.Decode(bufio.NewReader(&buffer))

			expected, err := loadImage("./expected/crop_20_10_failure.jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(EqualImage(actual))
		})
	})
})
