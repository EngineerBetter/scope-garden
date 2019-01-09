package conchhorse_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAtc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "conchhorse Suite")
}
