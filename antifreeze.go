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
	if args[0] != "check-manifest" {
		os.Exit(0)
	}

	fmt.Println("Running check-manifest...")

	appName, manifestPath, err := ParseArgs(args)
	fatalIf(err)

	manifestEnv, manifestServices, err := ParseManifest(manifestPath, appName)
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

func (c *AntifreezePlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "antifreeze",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 3,
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
	Name     string                 `yaml:"name"`
	Env      map[string]interface{} `yaml:"env"`
	Services []string               `yaml:"services"`
}

func ParseManifest(manifestPath, appName string) (manifestEnv []string, manifestServices []string, err error) {
	document, err := loadYAML(manifestPath)

	if err != nil {
		return manifestEnv, manifestServices, err
	}

	app, err := findApp(appName, document.Applications)

	if err != nil {
		return manifestEnv, manifestServices, err
	}

	for k := range app.Env {
		manifestEnv = append(manifestEnv, k)
	}

	return manifestEnv, app.Services, nil
}

func MissingFromManifest(manifestList, appList []string) (missing []string) {
	for _, appValue := range appList {
		if !stringInSlice(appValue, manifestList) {
			missing = append(missing, appValue)
		}
	}
	return missing
}

func loadYAML(manifestPath string) (manifest YManifest, err error) {
	b, err := ioutil.ReadFile(manifestPath)

	if err != nil {
		return YManifest{}, fmt.Errorf("Unable to read manifest file: %s", manifestPath)
	}

	var document YManifest
	err = yaml.Unmarshal(b, &document)

	if err != nil {
		return YManifest{}, fmt.Errorf("Unable to parse manifest YAML")
	}

	return document, nil
}

func findApp(appName string, apps []YApplication) (app YApplication, err error) {
	if len(apps) == 0 {
		return YApplication{}, fmt.Errorf("No application found in manifest")
	}

	appIndex := notFoundIndex

	for i := range apps {
		if apps[i].Name == appName {
			appIndex = i
			break
		}
	}

	if appIndex == notFoundIndex {
		return YApplication{}, fmt.Errorf("Application '%s' not found in manifest", appName)
	}

	return apps[appIndex], nil
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
		printListAsBullets(missingEnv)
	}

	if len(missingServices) > 0 {
		fmt.Printf("\nApp '%s' has unexpected services (missing from manifest %s):\n", appName, manifestPath)
		printListAsBullets(missingServices)
	}
}

func printListAsBullets(list []string) {
	for _, v := range list {
		fmt.Printf("- %s\n", v)
	}
}

const notFoundIndex = -1
