package kafka

import (
	"fmt"
	"testing"

	"github.com/mesosphere/kubeaddons-enterprise/test/utils"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	sparkTest     *testing.T
)

var _ = Describe("SparkTest", func() {
	Describe("[Spark Group Test]", func() {
		Context("addon installation", func() {
			It("Addon should be ready", func() {
				err := utils.GroupTest(sparkTest, "spark")
				Expect(err).To(BeNil())
			})
		})
	})
})

func TestSparkGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	sparkTest = t
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "spark-addon"))
	RunSpecsWithDefaultAndCustomReporters(t, "Spark Addon Test", []Reporter{junitReporter})
}
