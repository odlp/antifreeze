package main_test

import (
	"testing"

	"github.com/cloudfoundry/cli/plugin/models"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"
	. "github.com/odlp/antifreeze"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAntifreeze(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Antifreeze Suite")
}

var _ = Describe("Flag Parsing", func() {
	It("parses args", func() {
		appName, manifestPath, err := ParseArgs(
			[]string{
				"validate-manifest-ok",
				"app-name",
				"-f", "manifest-path",
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(appName).To(Equal("app-name"))
		Expect(manifestPath).To(Equal("manifest-path"))
	})

	It("requires a manifest", func() {
		_, _, err := ParseArgs(
			[]string{
				"validate-manifest-ok",
				"app-name",
			},
		)
		Expect(err).To(MatchError("Missing manifest argument"))
	})
})

var _ = Describe("Parsing Manifest", func() {
	It("parses the ENV keys", func() {
		envKeys, _, err := ParseManifest("./examples/manifest.yml")
		Expect(err).ToNot(HaveOccurred())
		Expect(envKeys).To(HaveLen(2))
		Expect(envKeys).To(ContainElement("ENV_VAR_1"))
		Expect(envKeys).To(ContainElement("ENV_VAR_2"))
	})

	It("parses the service names", func() {
		_, serviceNames, err := ParseManifest("./examples/manifest.yml")
		Expect(err).ToNot(HaveOccurred())
		Expect(serviceNames).To(HaveLen(2))
		Expect(serviceNames).To(ContainElement("service-1"))
		Expect(serviceNames).To(ContainElement("service-2"))
	})

	Context("invalid manifest path", func() {
		It("returns an error", func() {
			_, _, err := ParseManifest("./pure-fiction")
			Expect(err).To(MatchError("Unable to read manifest file: ./pure-fiction"))
		})
	})

	Context("invalid manifest", func() {
		It("returns an error", func() {
			_, _, err := ParseManifest("./examples/invalid-manifest.json")
			Expect(err).To(MatchError("No application found in manifest"))
		})
	})
})

var _ = Describe("Get App Env And Services", func() {
	var cliConnection *pluginfakes.FakeCliConnection
	var fakeApp plugin_models.GetAppModel

	BeforeEach(func() {
		cliConnection = &pluginfakes.FakeCliConnection{}
		fakeAppEnv := map[string]interface{}{"ENV_VAR_1": "foo", "ENV_VAR_2": "bar"}
		service1 := plugin_models.GetApp_ServiceSummary{
			Name: "service-1",
		}
		service2 := plugin_models.GetApp_ServiceSummary{
			Name: "service-2",
		}
		fakeApp = plugin_models.GetAppModel{
			EnvironmentVars: fakeAppEnv,
			Services:        []plugin_models.GetApp_ServiceSummary{service1, service2},
		}

		cliConnection.GetAppStub = func(arg1 string) (plugin_models.GetAppModel, error) {
			Expect(arg1).To(Equal("app-name"))
			return fakeApp, nil
		}
	})

	It("returns the ENV keys from the application", func() {
		appEnv, _, err := GetAppEnvAndServices(cliConnection, "app-name")

		Expect(err).ToNot(HaveOccurred())
		Expect(appEnv).To(HaveLen(2))
		Expect(appEnv).To(ContainElement("ENV_VAR_1"))
		Expect(appEnv).To(ContainElement("ENV_VAR_2"))
	})

	It("returns the service keys from the application", func() {
		_, appServices, err := GetAppEnvAndServices(cliConnection, "app-name")

		Expect(err).ToNot(HaveOccurred())
		Expect(appServices).To(HaveLen(2))
		Expect(appServices).To(ContainElement("service-1"))
		Expect(appServices).To(ContainElement("service-2"))
	})
})

var _ = Describe("Missing From Manifest", func() {
	Context("no differences", func() {
		It("returns an empty slice", func() {
			manifestEnv := []string{"ENV_VAR_1"}
			appEnv := []string{"ENV_VAR_1"}

			difference := MissingFromManifest(manifestEnv, appEnv)
			Expect(difference).To(HaveLen(0))
		})
	})

	Context("manifest has additional values", func() {
		It("returns an empty slice", func() {
			manifestEnv := []string{"ENV_VAR_1", "ENV_VAR_2"}
			appEnv := []string{"ENV_VAR_1"}

			difference := MissingFromManifest(manifestEnv, appEnv)
			Expect(difference).To(HaveLen(0))
		})
	})

	Context("app has additional values", func() {
		It("returns the offending values", func() {
			manifestEnv := []string{"ENV_VAR_1"}
			appEnv := []string{"ENV_VAR_1", "ENV_SNOW", "ENV_FLAKE"}

			difference := MissingFromManifest(manifestEnv, appEnv)
			Expect(difference).To(HaveLen(2))
			Expect(difference).To(ContainElement("ENV_SNOW"))
			Expect(difference).To(ContainElement("ENV_FLAKE"))
		})
	})
})

var _ = Describe("GetMetadata", func() {
	It("returns valid metadata", func() {
		plugin := AntifreezePlugin{}
		metadata := plugin.GetMetadata()

		Expect(metadata.Name).To(Equal("antifreeze"))
		Expect(metadata.Version).ToNot(BeNil())
		Expect(metadata.MinCliVersion).ToNot(BeNil())
		Expect(metadata.Commands).To(HaveLen(1))
	})
})
