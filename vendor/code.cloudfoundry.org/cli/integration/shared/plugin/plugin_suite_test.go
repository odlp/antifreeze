package plugin

import (
	"testing"
	"time"

	"code.cloudfoundry.org/cli/integration/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const (
	CFEventuallyTimeout   = 180 * time.Second
	CFConsistentlyTimeout = 500 * time.Millisecond
)

var (
	// Suite Level
	testPluginPath         string
	overrideTestPluginPath string
	panicTestPluginPath    string
	apiURL                 string
	skipSSLValidation      string

	// Per Test Level
	homeDir string
)

func TestGlobal(t *testing.T) {
	RegisterFailHandler(Fail)
	reporters := []Reporter{}

	honeyCombReporter := helpers.GetHoneyCombReporter("Plugin Suite")

	if honeyCombReporter != nil {
		reporters = append(reporters, honeyCombReporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Plugin Suite", reporters)
}

var _ = SynchronizedBeforeSuite(func() []byte {
	return nil
}, func(path []byte) {
	// Ginkgo Globals
	SetDefaultEventuallyTimeout(CFEventuallyTimeout)
	SetDefaultConsistentlyDuration(CFConsistentlyTimeout)

	// Setup common environment variables
	helpers.TurnOffColors()

	var err error
	testPluginPath, err = Build("code.cloudfoundry.org/cli/integration/assets/test_plugin")
	Expect(err).ToNot(HaveOccurred())

	overrideTestPluginPath, err = Build("code.cloudfoundry.org/cli/integration/assets/test_plugin_with_command_overrides")
	Expect(err).ToNot(HaveOccurred())

	panicTestPluginPath, err = Build("code.cloudfoundry.org/cli/integration/assets/test_plugin_with_panic")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	CleanupBuildArtifacts()
})

var _ = BeforeEach(func() {
	homeDir = helpers.SetHomeDir()
	apiURL, skipSSLValidation = helpers.SetAPI()
	helpers.LoginCF()
	Eventually(helpers.CF("remove-plugin-repo", "CF-Community")).Should(Exit(0))
})

var _ = AfterEach(func() {
	GinkgoWriter.Write([]byte("==============================Global After Each=============================="))
	helpers.DestroyHomeDir(homeDir)
})

func installTestPlugin() {
	session := helpers.CF("install-plugin", "-f", testPluginPath)
	Eventually(session).Should(Exit(0))
}

func uninstallTestPlugin() {
	session := helpers.CF("uninstall-plugin", "CF-CLI-Integration-Test-Plugin")
	Eventually(session).Should(Exit(0))
}

func createTargetedOrgAndSpace() (string, string) {
	org := helpers.NewOrgName()
	space := helpers.NewSpaceName()
	helpers.CreateOrgAndSpace(org, space)
	helpers.TargetOrgAndSpace(org, space)
	return org, space
}

func confirmTestPluginOutput(command string, output ...string) {
	session := helpers.CF(command)
	for _, val := range output {
		Eventually(session).Should(Say(val))
	}
	Eventually(session).Should(Exit(0))
}

func confirmTestPluginOutputWithArg(command string, arg string, output ...string) {
	session := helpers.CF(command, arg)
	for _, val := range output {
		Eventually(session).Should(Say(val))
	}
	Eventually(session).Should(Exit(0))
}
