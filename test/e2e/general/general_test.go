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

var _ = Describe("GeneralTest", func() {
	Describe("[General Group Test]", func() {
		Context("addon installation", func() {
			It("Addon should be ready", func() {
				err := utils.GroupTest(generalTest, "general")
				Expect(err).To(BeNil())
			})
		})
	})
})

func TestGeneralGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	generalTest = t
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "general-addon"))
	RunSpecsWithDefaultAndCustomReporters(t, "General Addon Test", []Reporter{junitReporter})
}
