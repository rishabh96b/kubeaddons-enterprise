package general_test

import (
	"fmt"
	"github.com/mesosphere/kubeaddons-enterprise/test/utils"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	generalTest *testing.T
)

var _ = Describe("ValidateUnhandled", func() {
	Describe("[Validate Unhandled Addon Test]", func() {
		Context("check for unhandled", func() {
			It("No addons without tests should be present", func() {
				unhandled, err := utils.FindUnhandled()
				Expect(err).To(BeNil())

				if len(unhandled) != 0 {
					names := make([]string, len(unhandled))
					for _, addon := range unhandled {
						names = append(names, addon.GetName())
					}
					Fail(fmt.Sprintf("the following addons are not handled as part of a testing group: %+v", names), 0)
				}
			})
		})
	})
})

func TestValidateUnhandledAddons(t *testing.T) {
	RegisterFailHandler(Fail)
	generalTest = t
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "unhandled-addon"))
	RunSpecsWithDefaultAndCustomReporters(t, "Validate Unhandled Addon Test", []Reporter{junitReporter})
}
