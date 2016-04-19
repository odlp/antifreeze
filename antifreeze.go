package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/cli/plugin"

	"gopkg.in/yaml.v2"
)

func main() {
	plugin.Start(&AntifreezePlugin{})
}

type AntifreezePlugin struct{}

func (c *AntifreezePlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "check-manifest" {
		fmt.Println("Running check-manifest...")

		appName, manifestPath, err := ParseArgs(args)
		fatalIf(err)

		manifestEnv, manifestServices, err := ParseManifest(manifestPath)
		fatalIf(err)

		appEnv, appServices, err := GetAppEnvAndServices(cliConnection, appName)
		fatalIf(err)

		missingEnv := MissingFromManifest(manifestEnv, appEnv)
		missingServices := MissingFromManifest(manifestServices, appServices)

		if len(missingEnv) == 0 && len(missingServices) == 0 {
			os.Exit(0)
		}

		printMissingValues(appName, manifestPath, missingEnv, missingServices)
		os.Exit(1)
	}
}

func (c *AntifreezePlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "antifreeze",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 2,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     "check-manifest",
				HelpText: "Check your manifest isn't missing any ENV vars or services currently in an app",
			},
		},
	}
}

func ParseArgs(args []string) (string, string, error) {
	flags := flag.NewFlagSet("check-manifest", flag.ContinueOnError)
	manifestPath := flags.String("f", "", "path to an application manifest")
	err := flags.Parse(args[2:])

	if err != nil {
		return "", "", err
	}

	if *manifestPath == "" {
		return "", "", fmt.Errorf("Missing manifest argument")
	}

	appName := args[1]
	return appName, *manifestPath, nil
}

func GetAppEnvAndServices(cliConnection plugin.CliConnection, appName string) (appEnv []string, appServices []string, err error) {
	app, _ := cliConnection.GetApp(appName)

	for k := range app.EnvironmentVars {
		appEnv = append(appEnv, k)
	}

	for _, s := range app.Services {
		appServices = append(appServices, s.Name)
	}

	return appEnv, appServices, err
}

type YManifest struct {
	Applications []YApplication `yaml:"applications"`
}

type YApplication struct {
	Env      map[string]interface{} `yaml:"env"`
	Services []string               `yaml:"services"`
}

func ParseManifest(manifestPath string) ([]string, []string, error) {
	b, err := ioutil.ReadFile(manifestPath)

	if err != nil {
		return []string{}, []string{}, fmt.Errorf("Unable to read manifest file: %s", manifestPath)
	}

	var document YManifest
	err = yaml.Unmarshal(b, &document)

	if err != nil {
		return []string{}, []string{}, fmt.Errorf("Unable to parse manifest YAML")
	}

	if len(document.Applications) == 0 {
		return []string{}, []string{}, fmt.Errorf("No application found in manifest")
	}

	envKeys := []string{}
	for k := range document.Applications[0].Env {
		envKeys = append(envKeys, k)
	}

	return envKeys, document.Applications[0].Services, nil
}

func MissingFromManifest(manifestList, appList []string) (missing []string) {
	for _, appValue := range appList {
		if !stringInSlice(appValue, manifestList) {
			missing = append(missing, appValue)
		}
	}
	return missing
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if a == b {
			return true
		}
	}
	return false
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		os.Exit(1)
	}
}

func printMissingValues(appName string, manifestPath string, missingEnv []string, missingServices []string) {
	if len(missingEnv) > 0 {
		fmt.Printf("\nApp '%s' has unexpected ENV vars (missing from manifest %s):\n", appName, manifestPath)
		for _, v := range missingEnv {
			fmt.Printf("- %s\n", v)
		}
	}

	if len(missingServices) > 0 {
		fmt.Printf("\nApp '%s' has unexpected services (missing from manifest %s):\n", appName, manifestPath)
		for _, v := range missingServices {
			fmt.Printf("- %s\n", v)
		}
	}
}
