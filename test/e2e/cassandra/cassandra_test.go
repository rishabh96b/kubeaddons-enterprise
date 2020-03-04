package cassandra

import (
	"fmt"
	"testing"

	"github.com/mesosphere/kubeaddons-enterprise/test/utils"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	cassandraTest     *testing.T
)

var _ = Describe("CassandraTest", func() {
	Describe("[Cassandra Group Test]", func() {
		Context("addon installation", func() {
			It("Addon should be ready", func() {
				err := utils.GroupTest(cassandraTest, "cassandra")
				Expect(err).To(BeNil())
			})
		})
	})
})

func TestCassandraGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	cassandraTest = t
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "cassandra-addon"))
	RunSpecsWithDefaultAndCustomReporters(t, "Cassandra Addon Test", []Reporter{junitReporter})
}
