package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/blang/semver"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/mesosphere/kubeaddons/pkg/api/v1beta1"
	"github.com/mesosphere/kubeaddons/pkg/catalog"
	"github.com/mesosphere/kubeaddons/pkg/repositories"
	"github.com/mesosphere/kubeaddons/pkg/repositories/local"
	"github.com/mesosphere/kubeaddons/pkg/test"
	"github.com/mesosphere/kubeaddons/pkg/test/cluster/kind"
)

const (
	controllerBundle         = "https://mesosphere.github.io/kubeaddons/bundle.yaml"
	defaultKubernetesVersion = "1.17.0"
)

var addonTestingGroups = make(map[string][]AddonTestConfiguration)

type AddonTestConfiguration struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Override the values for helm chart and parameters for kudo operators
	Override string `json:"override,omitempty" yaml:"override,omitempty"`
	// List of requirements to be removed from the addon
	RemoveDependencies []string `json:"removeDependencies,omitempty" yaml:"removeDependencies,omitempty"`
}

var (
	cat        catalog.Catalog
	localRepo  repositories.Repository
	groups     map[string][]v1beta1.AddonInterface
	addonsPath = "../../../addons/"
	groupsPath = "../../groups.yaml"
)

func init() {
	var err error
	localRepo, err = local.NewRepository("local", addonsPath)
	if err != nil {
		panic(err)
	}

	cat, err = catalog.NewCatalog(localRepo)
	if err != nil {
		panic(err)
	}

	groups, err = test.AddonsForGroupsFile(groupsPath, cat)
	if err != nil {
		panic(err)
	}

	for group, addons := range groups {
		for _, addon := range addons {
			overrides := overridesForAddon(addon.GetName())
			removeDeps := removeDepsForAddon(addon.GetName())
			cfg := AddonTestConfiguration{
				Name:               addon.GetName(),
				Override:           overrides,
				RemoveDependencies: removeDeps,
			}
			addonTestingGroups[group] = append(addonTestingGroups[group], cfg)
		}
	}
}

// -----------------------------------------------------------------------------
// Private Functions
// -----------------------------------------------------------------------------

func GroupTest(t *testing.T, groupname string) error {
	t.Logf("testing group %s", groupname)

	version, err := semver.Parse(defaultKubernetesVersion)
	if err != nil {
		return err
	}

	cluster, err := kind.NewCluster(version)
	if err != nil {
		return err
	}
	defer cluster.Cleanup()

	if err := kubectl("apply", "-f", controllerBundle); err != nil {
		return err
	}

	if err = waitForPod(cluster.Client(), fmt.Sprintf("etcd-%s-control-plane", cluster.Name()), "kube-system", 120); err != nil {
		return err
	}
	if err = waitForPod(cluster.Client(), fmt.Sprintf("kube-apiserver-%s-control-plane", cluster.Name()), "kube-system", 120); err != nil {
		return err
	}
	if err = waitForPod(cluster.Client(), fmt.Sprintf("kube-scheduler-%s-control-plane", cluster.Name()), "kube-system", 120); err != nil {
		return err
	}
	if err = waitForPod(cluster.Client(), fmt.Sprintf("kube-controller-manager-%s-control-plane", cluster.Name()), "kube-system", 120); err != nil {
		return err
	}
	if err = waitForDeployment(cluster.Client(), "local-path-provisioner", "local-path-storage", 120); err != nil {
		return err
	}

	addons, err := addons(addonTestingGroups[groupname]...)
	if err != nil {
		return err
	}

	err = createNamespaces(cluster.Client(), addons)
	if err != nil {
		return err
	}
	ph, err := test.NewBasicTestHarness(t, cluster, addons...)
	if err != nil {
		return err
	}
	defer ph.Cleanup()

	wg := &sync.WaitGroup{}
	stop := make(chan struct{})
	go test.LoggingHook(t, cluster, wg, stop)

	ph.Validate()
	ph.Deploy()

	close(stop)
	wg.Wait()

	return nil
}

func createNamespaces(client kubernetes.Interface, addons []v1beta1.AddonInterface) error {
	for _, addon := range addons {
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: addon.GetNamespace(),
			},
		}
		_, err := client.CoreV1().Namespaces().Create(ns)
		if err != nil {
			return fmt.Errorf("could not create addon %s namespace: %w", addon.GetName(), err)
		}
	}
	return nil
}

func waitForPod(client kubernetes.Interface, name, namespace string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("Timeout while waiting for pod %s ready replicas count to be %d", namespace, name))
		case <-tick:
			if isPodReady(client, name, namespace) {
				return nil
			}
		}
	}
}

func waitForDeployment(client kubernetes.Interface, name, namespace string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("Timeout while waiting for pod %s ready replicas count to be %d", namespace, name))
		case <-tick:
			if isDeploymentReady(client, name, namespace, int32(1)) {
				return nil
			}
		}
	}
}

func isPodReady(client kubernetes.Interface, name, namespace string) bool {
	pod, err := client.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	if pod.Status.Phase == v1.PodRunning {
		return true
	}
	return false
}

func isDeploymentReady(client kubernetes.Interface, name, namespace string, expectedCount int32) bool {
	deployment, err := client.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		return false
	}
	if deployment.Status.ReadyReplicas == expectedCount {
		return true
	}
	return false
}

func addons(addonConfigs ...AddonTestConfiguration) ([]v1beta1.AddonInterface, error) {
	var testAddons []v1beta1.AddonInterface

	repo, err := local.NewRepository("base", addonsPath)
	if err != nil {
		return testAddons, err
	}
	for _, addonConfig := range addonConfigs {
		addon, err := repo.GetAddon(addonConfig.Name)
		if err != nil {
			return testAddons, err
		}
		overrides(addon[0], addonConfig)
		if addon[0].GetNamespace() == "" {
			addon[0].SetNamespace(addon[0].GetName())
		}
		// TODO - we need to re-org where these filters are done (see: https://jira.mesosphere.com/browse/DCOS-63260)
		testAddons = append(testAddons, addon[0])
	}

	if len(testAddons) != len(addonConfigs) {
		return testAddons, fmt.Errorf("got %d addons, expected %d", len(testAddons), len(addonConfigs))
	}

	return testAddons, nil
}

func FindUnhandled() ([]v1beta1.AddonInterface, error) {
	var unhandled []v1beta1.AddonInterface
	repo, err := local.NewRepository("base", addonsPath)
	if err != nil {
		return unhandled, err
	}
	addons, err := repo.ListAddons()
	if err != nil {
		return unhandled, err
	}

	for _, revisions := range addons {
		addon := revisions[0]
		found := false
		for _, v := range addonTestingGroups {
			for _, addonConfig := range v {
				if addonConfig.Name == addon.GetName() {
					found = true
				}
			}
		}
		if !found {
			unhandled = append(unhandled, addon)
		}
	}

	return unhandled, nil
}

// -----------------------------------------------------------------------------
// Private - CI Values Overrides
// -----------------------------------------------------------------------------

func overrides(addon v1beta1.AddonInterface, config AddonTestConfiguration) {
	if config.Override != "" {
		// override helm chart values
		if addon.GetAddonSpec().ChartReference != nil {
			addon.GetAddonSpec().ChartReference.Values = &config.Override
		}
		//override kudo operator default values
		if addon.GetAddonSpec().KudoReference != nil {
			addon.GetAddonSpec().KudoReference.Parameters = &config.Override
		}
	}

	for _, toRemove := range config.RemoveDependencies {
		removeDependencyFromAddon(addon, toRemove)
	}

}

func removeDependencyFromAddon(addon v1beta1.AddonInterface, toRemove string) {
	for index, labelSelector := range addon.GetAddonSpec().Requires {
		for label, value := range labelSelector.MatchLabels {
			if value == toRemove {
				delete(labelSelector.MatchLabels, label)
				if len(labelSelector.MatchLabels) == 0 {
					addon.GetAddonSpec().Requires = removeLabelsIndex(addon.GetAddonSpec().Requires, index)
				}
				return
			}

		}
	}
}

func removeLabelsIndex(s []metav1.LabelSelector, index int) []metav1.LabelSelector {
	return append(s[:index], s[index+1:]...)
}

// TODO currently overrides are hardcoded but will be promoted into test.Groups in the future
// see D2IQ-64898
func overridesForAddon(name string) string {
	switch name {
	case "kafka":
		return `ZOOKEEPER_URI: zookeeper-cs.zookeeper.svc.cluster.local
BROKER_MEM: 32Mi
BROKER_CPUS: 20m
BROKER_COUNT: 1
ADD_SERVICE_MONITOR: false
`
	case "cassandra":
		return `NODE_COUNT: 1
NODE_DISK_SIZE_GIB: 1
NODE_MEM_MIB: 128
PROMETHEUS_EXPORTER_ENABLED: "false"
`
	case "spark":
		return `enableMetrics: "false"`
	case "zookeeper":
		return `MEMORY: "32Mi"
CPUS: 50m
NODE_COUNT: 1
`
	}

	return ""
}

// TODO currently depremovals are hardcoded but will be promoted into test.Groups in the future
// see D2IQ-64898
func removeDepsForAddon(name string) []string {
	switch name {
	// remove promethues dependency for CI
	// https://jira.d2iq.com/browse/D2IQ-63819
	case "spark":
		return []string{"prometheus"}
	case "kafka":
		return []string{"prometheus"}
	}
	return []string{}
}

func kubectl(args ...string) error {
	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

