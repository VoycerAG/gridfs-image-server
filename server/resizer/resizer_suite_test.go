package resizer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestResizer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resizer Suite")
}
