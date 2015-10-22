package paint_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPaint(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Paint Suite")
}
