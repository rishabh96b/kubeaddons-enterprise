package kafka

import (
	"fmt"
	"github.com/mesosphere/kubeaddons-enterprise/test/utils"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	kafkaTest     *testing.T
)

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Group Test]", func() {
		Context("addon installation", func() {
			It("Addon should be ready", func() {
				err := utils.GroupTest(kafkaTest, "kafka")
				Expect(err).To(BeNil())
			})
		})
	})
})

func TestKafkaGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	kafkaTest = t
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-addon"))
	RunSpecsWithDefaultAndCustomReporters(t, "Kafka Addon Test", []Reporter{junitReporter})
}
