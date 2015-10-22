package server

import (
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// DEPRECATED this will be removed in a future release
//
// in order for this function to work, you need to have
// imagemagick installed on your unix based os
//
// imageMagickFallback is used to convert a image with image magick
func imageMagickFallback(originalImage ReadSeekCloser) (image.Image, error) {
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
