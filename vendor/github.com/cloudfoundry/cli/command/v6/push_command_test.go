package v6_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/pushaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v2v3action"
	"code.cloudfoundry.org/cli/actor/v3action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccversion"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/translatableerror"
	. "code.cloudfoundry.org/cli/command/v6"
	"code.cloudfoundry.org/cli/command/v6/shared/sharedfakes"
	"code.cloudfoundry.org/cli/command/v6/v6fakes"
	"code.cloudfoundry.org/cli/types"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/manifest"
	"code.cloudfoundry.org/cli/util/ui"
	"github.com/cloudfoundry/bosh-cli/director/template"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("push Command", func() {
	var (
		cmd                         PushCommand
		testUI                      *ui.UI
		fakeConfig                  *commandfakes.FakeConfig
		fakeSharedActor             *commandfakes.FakeSharedActor
		fakeActor                   *v6fakes.FakeV2PushActor
		fakeRestartActor            *v6fakes.FakeRestartActor
		fakeApplicationSummaryActor *sharedfakes.FakeApplicationSummaryActor
		fakeProgressBar             *v6fakes.FakeProgressBar
		input                       *Buffer
		binaryName                  string

		appName    string
		executeErr error
		pwd        string
	)

	BeforeEach(func() {
		input = NewBuffer()
		testUI = ui.NewTestUI(input, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeV2PushActor)
		fakeRestartActor = new(v6fakes.FakeRestartActor)
		fakeApplicationSummaryActor = new(sharedfakes.FakeApplicationSummaryActor)
		fakeProgressBar = new(v6fakes.FakeProgressBar)

		cmd = PushCommand{
			UI:                      testUI,
			Config:                  fakeConfig,
			SharedActor:             fakeSharedActor,
			Actor:                   fakeActor,
			RestartActor:            fakeRestartActor,
			ApplicationSummaryActor: fakeApplicationSummaryActor,
			ProgressBar:             fakeProgressBar,
		}

		appName = "some-app"
		cmd.OptionalArgs.AppName = appName
		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)

		var err error
		pwd, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Execute", func() {
		JustBeforeEach(func() {
			executeErr = cmd.Execute(nil)
		})

		When("the mutiple buildpacks are provided, and the API version is below the mutiple buildpacks minimum", func() {
			BeforeEach(func() {
				fakeActor.CloudControllerV3APIVersionReturns("3.1.0")
				cmd.Buildpacks = []string{"some-buildpack", "some-other-buildpack"}
			})

			It("returns a MinimumAPIVersionNotMetError", func() {
				Expect(executeErr).To(MatchError(translatableerror.MinimumCFAPIVersionNotMetError{
					Command:        "Multiple option '-b'",
					CurrentVersion: "3.1.0",
					MinimumVersion: ccversion.MinVersionManifestBuildpacksV3,
				}))
			})
		})

		When("checking target fails", func() {
			BeforeEach(func() {
				fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{BinaryName: binaryName})
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))

				Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
				checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
				Expect(checkTargetedOrg).To(BeTrue())
				Expect(checkTargetedSpace).To(BeTrue())
			})
		})

		When("the user is logged in, and org and space are targeted", func() {
			BeforeEach(func() {
				fakeConfig.HasTargetedOrganizationReturns(true)
				fakeConfig.TargetedOrganizationReturns(configv3.Organization{GUID: "some-org-guid", Name: "some-org"})
				fakeConfig.HasTargetedSpaceReturns(true)
				fakeConfig.TargetedSpaceReturns(configv3.Space{GUID: "some-space-guid", Name: "some-space"})
				fakeConfig.CurrentUserReturns(configv3.User{Name: "some-user"}, nil)
			})

			When("the push settings are valid", func() {
				var appManifests []manifest.Application

				BeforeEach(func() {
					appManifests = []manifest.Application{
						{
							Name: appName,
							Path: pwd,
						},
					}
					fakeActor.MergeAndValidateSettingsAndManifestsReturns(appManifests, nil)
				})

				When("buildpacks (plural) is provided in the manifest and the API version is below the minimum", func() {
					BeforeEach(func() {
						appManifests = []manifest.Application{
							{
								Name:       appName,
								Path:       pwd,
								Buildpacks: []string{"ruby-buildpack", "java-buildpack"},
							},
						}

						fakeActor.MergeAndValidateSettingsAndManifestsReturns(appManifests, nil)
						fakeActor.CloudControllerV3APIVersionReturns("3.13.0")
					})

					It("returns a MinimumAPIVersionNotMetError", func() {
						Expect(executeErr).To(MatchError(translatableerror.MinimumCFAPIVersionNotMetError{
							Command:        "'buildpacks' in manifest",
							CurrentVersion: "3.13.0",
							MinimumVersion: ccversion.MinVersionManifestBuildpacksV3,
						}))
					})

				})

				When("the settings can be converted to a valid config", func() {
					var appConfigs []pushaction.ApplicationConfig

					BeforeEach(func() {
						appConfigs = []pushaction.ApplicationConfig{
							{
								CurrentApplication: pushaction.Application{Application: v2action.Application{Name: appName, State: constant.ApplicationStarted}},
								DesiredApplication: pushaction.Application{Application: v2action.Application{Name: appName}},
								CurrentRoutes: []v2action.Route{
									{Host: "route1", Domain: v2action.Domain{Name: "example.com"}},
									{Host: "route2", Domain: v2action.Domain{Name: "example.com"}},
								},
								DesiredRoutes: []v2action.Route{
									{Host: "route3", Domain: v2action.Domain{Name: "example.com"}},
									{Host: "route4", Domain: v2action.Domain{Name: "example.com"}},
								},
								Path: pwd,
							},
						}
						fakeActor.ConvertToApplicationConfigsReturns(appConfigs, pushaction.Warnings{"some-config-warnings"}, nil)
					})

					When("the apply is successful", func() {
						var updatedConfig pushaction.ApplicationConfig

						BeforeEach(func() {
							fakeActor.ApplyStub = func(_ pushaction.ApplicationConfig, _ pushaction.ProgressBar) (<-chan pushaction.ApplicationConfig, <-chan pushaction.Event, <-chan pushaction.Warnings, <-chan error) {
								configStream := make(chan pushaction.ApplicationConfig, 1)
								eventStream := make(chan pushaction.Event)
								warningsStream := make(chan pushaction.Warnings)
								errorStream := make(chan error)

								updatedConfig = pushaction.ApplicationConfig{
									CurrentApplication: pushaction.Application{Application: v2action.Application{Name: appName, GUID: "some-app-guid"}},
									DesiredApplication: pushaction.Application{Application: v2action.Application{Name: appName, GUID: "some-app-guid"}},
									Path:               pwd,
								}

								go func() {
									defer GinkgoRecover()

									Eventually(eventStream).Should(BeSent(pushaction.SettingUpApplication))
									Eventually(eventStream).Should(BeSent(pushaction.CreatedApplication))
									Eventually(eventStream).Should(BeSent(pushaction.UpdatedApplication))
									Eventually(eventStream).Should(BeSent(pushaction.CreatingAndMappingRoutes))
									Eventually(eventStream).Should(BeSent(pushaction.CreatedRoutes))
									Eventually(eventStream).Should(BeSent(pushaction.BoundRoutes))
									Eventually(eventStream).Should(BeSent(pushaction.UnmappingRoutes))
									Eventually(eventStream).Should(BeSent(pushaction.ConfiguringServices))
									Eventually(eventStream).Should(BeSent(pushaction.BoundServices))
									Eventually(eventStream).Should(BeSent(pushaction.ResourceMatching))
									Eventually(eventStream).Should(BeSent(pushaction.UploadingApplication))
									Eventually(eventStream).Should(BeSent(pushaction.CreatingArchive))
									Eventually(eventStream).Should(BeSent(pushaction.UploadingApplicationWithArchive))
									Eventually(fakeProgressBar.ReadyCallCount).Should(Equal(1))
									Eventually(eventStream).Should(BeSent(pushaction.RetryUpload))
									Eventually(eventStream).Should(BeSent(pushaction.UploadWithArchiveComplete))
									Eventually(fakeProgressBar.CompleteCallCount).Should(Equal(1))
									Eventually(configStream).Should(BeSent(updatedConfig))
									Eventually(eventStream).Should(BeSent(pushaction.Complete))
									Eventually(warningsStream).Should(BeSent(pushaction.Warnings{"apply-1", "apply-2"}))
									close(configStream)
									close(eventStream)
									close(warningsStream)
									close(errorStream)
								}()

								return configStream, eventStream, warningsStream, errorStream
							}

							fakeRestartActor.RestartApplicationStub = func(app v2action.Application, client v2action.NOAAClient) (<-chan *v2action.LogMessage, <-chan error, <-chan v2action.ApplicationStateChange, <-chan string, <-chan error) {
								messages := make(chan *v2action.LogMessage)
								logErrs := make(chan error)
								appState := make(chan v2action.ApplicationStateChange)
								warnings := make(chan string)
								errs := make(chan error)

								go func() {
									messages <- v2action.NewLogMessage("log message 1", 1, time.Unix(0, 0), "STG", "1")
									messages <- v2action.NewLogMessage("log message 2", 1, time.Unix(0, 0), "STG", "1")
									appState <- v2action.ApplicationStateStopping
									appState <- v2action.ApplicationStateStaging
									appState <- v2action.ApplicationStateStarting
									close(messages)
									close(logErrs)
									close(appState)
									close(warnings)
									close(errs)
								}()

								return messages, logErrs, appState, warnings, errs
							}

							applicationSummary := v2action.ApplicationSummary{
								Application: v2action.Application{
									DetectedBuildpack:    types.FilteredString{IsSet: true, Value: "some-buildpack"},
									DetectedStartCommand: types.FilteredString{IsSet: true, Value: "some start command"},
									GUID:                 "some-app-guid",
									Instances:            types.NullInt{Value: 3, IsSet: true},
									Memory:               types.NullByteSizeInMb{IsSet: true, Value: 128},
									Name:                 appName,
									PackageUpdatedAt:     time.Unix(0, 0),
									State:                "STARTED",
								},
								Stack: v2action.Stack{
									Name: "potatos",
								},
								Routes: []v2action.Route{
									{
										Host: "banana",
										Domain: v2action.Domain{
											Name: "fruit.com",
										},
										Path: "/hi",
									},
									{
										Domain: v2action.Domain{
											Name: "foobar.com",
										},
										Port: types.NullInt{IsSet: true, Value: 13},
									},
								},
							}
							warnings := []string{"app-summary-warning"}

							applicationSummary.RunningInstances = []v2action.ApplicationInstanceWithStats{{State: "RUNNING"}}

							fakeRestartActor.GetApplicationSummaryByNameAndSpaceReturns(applicationSummary, warnings, nil)
						})

						When("no manifest is provided", func() {
							It("passes through the command line flags", func() {
								Expect(executeErr).ToNot(HaveOccurred())

								Expect(fakeActor.MergeAndValidateSettingsAndManifestsCallCount()).To(Equal(1))
								cmdSettings, _ := fakeActor.MergeAndValidateSettingsAndManifestsArgsForCall(0)
								Expect(cmdSettings).To(Equal(pushaction.CommandLineSettings{
									Name:             appName,
									CurrentDirectory: pwd,
								}))
							})
						})

						When("a manifest is provided", func() {
							var (
								tmpDir       string
								providedPath string

								originalDir string
							)

							BeforeEach(func() {
								var err error
								tmpDir, err = ioutil.TempDir("", "push-command-test")
								Expect(err).ToNot(HaveOccurred())

								// OS X uses weird symlinks that causes problems for some tests
								tmpDir, err = filepath.EvalSymlinks(tmpDir)
								Expect(err).ToNot(HaveOccurred())

								originalDir, err = os.Getwd()
								Expect(err).ToNot(HaveOccurred())

								cmd.OptionalArgs.AppName = ""
							})

							AfterEach(func() {
								Expect(os.Chdir(originalDir)).ToNot(HaveOccurred())
								Expect(os.RemoveAll(tmpDir)).ToNot(HaveOccurred())
							})

							Context("via a manifest.yml in the current directory", func() {
								var expectedApps []manifest.Application

								BeforeEach(func() {
									err := os.Chdir(tmpDir)
									Expect(err).ToNot(HaveOccurred())

									providedPath = filepath.Join(tmpDir, "manifest.yml")
									err = ioutil.WriteFile(providedPath, []byte("some manifest file"), 0666)
									Expect(err).ToNot(HaveOccurred())

									expectedApps = []manifest.Application{{Name: "some-app"}, {Name: "some-other-app"}}
									fakeActor.ReadManifestReturns(expectedApps, nil, nil)
								})

								When("reading the manifest file is successful", func() {
									It("merges app manifest and flags", func() {
										Expect(executeErr).ToNot(HaveOccurred())

										Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
										Expect(fakeActor.ReadManifestArgsForCall(0)).To(Equal(providedPath))

										Expect(fakeActor.MergeAndValidateSettingsAndManifestsCallCount()).To(Equal(1))
										cmdSettings, manifestApps := fakeActor.MergeAndValidateSettingsAndManifestsArgsForCall(0)
										Expect(cmdSettings).To(Equal(pushaction.CommandLineSettings{
											CurrentDirectory: tmpDir,
										}))
										Expect(manifestApps).To(Equal(expectedApps))
									})

									It("outputs corresponding flavor text", func() {
										Expect(executeErr).ToNot(HaveOccurred())

										Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
										Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(providedPath)))
									})
								})

								When("reading manifest file errors", func() {
									var expectedErr error

									BeforeEach(func() {
										expectedErr = errors.New("I am an error!!!")

										fakeActor.ReadManifestReturns(nil, nil, expectedErr)
									})

									It("returns the error", func() {
										Expect(executeErr).To(MatchError(expectedErr))
									})
								})

								When("--no-manifest is specified", func() {
									BeforeEach(func() {
										cmd.NoManifest = true
									})

									It("ignores the manifest file", func() {
										Expect(executeErr).ToNot(HaveOccurred())

										Expect(fakeActor.MergeAndValidateSettingsAndManifestsCallCount()).To(Equal(1))
										cmdSettings, manifestApps := fakeActor.MergeAndValidateSettingsAndManifestsArgsForCall(0)
										Expect(cmdSettings).To(Equal(pushaction.CommandLineSettings{
											CurrentDirectory: tmpDir,
										}))
										Expect(manifestApps).To(BeNil())
									})
								})
							})

							Context("via a manifest.yaml in the current directory", func() {
								BeforeEach(func() {
									err := os.Chdir(tmpDir)
									Expect(err).ToNot(HaveOccurred())

									providedPath = filepath.Join(tmpDir, "manifest.yaml")
									err = ioutil.WriteFile(providedPath, []byte("some manifest file"), 0666)
									Expect(err).ToNot(HaveOccurred())
								})

								It("should read the manifest.yml", func() {
									Expect(executeErr).ToNot(HaveOccurred())

									Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
									Expect(fakeActor.ReadManifestArgsForCall(0)).To(Equal(providedPath))
								})
							})

							Context("via the -f flag", func() {
								Context("given a path with filename 'manifest.yml'", func() {
									BeforeEach(func() {
										providedPath = filepath.Join(tmpDir, "manifest.yml")
									})

									When("the manifest.yml file does not exist", func() {
										BeforeEach(func() {
											cmd.PathToManifest = flag.PathWithExistenceCheck(providedPath)
										})

										It("returns an error", func() {
											Expect(os.IsNotExist(executeErr)).To(BeTrue())

											Expect(testUI.Out).ToNot(Say("Pushing from manifest"))
											Expect(testUI.Out).ToNot(Say("Using manifest file"))

											Expect(fakeActor.ReadManifestCallCount()).To(Equal(0))
										})
									})

									When("the manifest.yml file exists", func() {
										BeforeEach(func() {
											err := ioutil.WriteFile(providedPath, []byte(`key: "value"`), 0666)
											Expect(err).ToNot(HaveOccurred())

											cmd.PathToManifest = flag.PathWithExistenceCheck(providedPath)
										})

										It("should read the manifest.yml file and outputs corresponding flavor text", func() {
											Expect(executeErr).ToNot(HaveOccurred())

											Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
											Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(providedPath)))

											Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
											Expect(fakeActor.ReadManifestArgsForCall(0)).To(Equal(providedPath))
										})

										Context("variable interpolation", func() {
											Context("vars file only", func() {
												When("a vars file is also provided", func() {
													var providedVarsFilePath string

													BeforeEach(func() {
														providedVarsFilePath = filepath.Join(tmpDir, "vars-file.yml")
														cmd.VarsFilePaths = []flag.PathWithExistenceCheck{flag.PathWithExistenceCheck(providedVarsFilePath)}
													})

													It("should read the vars-file.yml file and replace the variables in the manifest.yml file", func() {
														Expect(executeErr).ToNot(HaveOccurred())

														Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
														Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(providedPath)))

														Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
														manifest, varsFiles, vars := fakeActor.ReadManifestArgsForCall(0)
														Expect(manifest).To(Equal(providedPath))
														Expect(varsFiles).To(Equal([]string{providedVarsFilePath}))
														Expect(vars).To(BeEmpty())
													})
												})

												When("multiple vars files are provided", func() {
													var (
														firstProvidedVarsFilePath  string
														secondProvidedVarsFilePath string
													)

													BeforeEach(func() {
														firstProvidedVarsFilePath = filepath.Join(tmpDir, "vars-file-1.yml")
														firstVarsFile := flag.PathWithExistenceCheck(firstProvidedVarsFilePath)

														secondProvidedVarsFilePath = filepath.Join(tmpDir, "vars-file-2.yml")
														secondVarsFile := flag.PathWithExistenceCheck(secondProvidedVarsFilePath)
														cmd.VarsFilePaths = []flag.PathWithExistenceCheck{firstVarsFile, secondVarsFile}
													})

													It("should read the vars-file.yml file and replace the variables in the manifest.yml file", func() {
														Expect(executeErr).ToNot(HaveOccurred())

														Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
														Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(providedPath)))

														Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
														manifest, varsFiles, vars := fakeActor.ReadManifestArgsForCall(0)
														Expect(manifest).To(Equal(providedPath))
														Expect(varsFiles).To(Equal([]string{firstProvidedVarsFilePath, secondProvidedVarsFilePath}))
														Expect(vars).To(BeEmpty())
													})
												})
											})

											Context("vars flag only", func() {
												var vars []template.VarKV

												BeforeEach(func() {
													vars = []template.VarKV{
														{Name: "some-var", Value: "some-value"},
														{Name: "another-var", Value: 1},
													}

													cmd.Vars = vars
												})

												It("should read the vars and pass only the vars array to ReadManifest", func() {
													Expect(executeErr).ToNot(HaveOccurred())

													Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
													Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(providedPath)))

													Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
													manifest, varsFiles, vars := fakeActor.ReadManifestArgsForCall(0)
													Expect(manifest).To(Equal(providedPath))
													Expect(varsFiles).To(BeEmpty())
													Expect(vars).To(ConsistOf([]template.VarKV{
														{Name: "some-var", Value: "some-value"},
														{Name: "another-var", Value: 1},
													}))
												})
											})
										})
									})
								})

								Context("given a path that is a directory", func() {

									var (
										ymlFile  string
										yamlFile string
									)

									BeforeEach(func() {
										providedPath = tmpDir
										cmd.PathToManifest = flag.PathWithExistenceCheck(providedPath)
									})

									When("the directory does not contain a 'manifest.y{a}ml' file", func() {
										It("returns an error", func() {
											Expect(executeErr).To(MatchError(translatableerror.ManifestFileNotFoundInDirectoryError{PathToManifest: providedPath}))
											Expect(testUI.Out).ToNot(Say("Pushing from manifest"))
											Expect(testUI.Out).ToNot(Say("Using manifest file"))

											Expect(fakeActor.ReadManifestCallCount()).To(Equal(0))
										})
									})

									When("the directory contains a 'manifest.yml' file", func() {
										BeforeEach(func() {
											ymlFile = filepath.Join(providedPath, "manifest.yml")
											err := ioutil.WriteFile(ymlFile, []byte(`key: "value"`), 0666)
											Expect(err).ToNot(HaveOccurred())
										})

										It("should read the manifest.yml file and outputs corresponding flavor text", func() {
											Expect(executeErr).ToNot(HaveOccurred())

											Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
											Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(ymlFile)))

											Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
											Expect(fakeActor.ReadManifestArgsForCall(0)).To(Equal(ymlFile))
										})
									})

									When("the directory contains a 'manifest.yaml' file", func() {
										BeforeEach(func() {
											yamlFile = filepath.Join(providedPath, "manifest.yaml")
											err := ioutil.WriteFile(yamlFile, []byte(`key: "value"`), 0666)
											Expect(err).ToNot(HaveOccurred())
										})

										It("should read the manifest.yaml file and outputs corresponding flavor text", func() {
											Expect(executeErr).ToNot(HaveOccurred())

											Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
											Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(yamlFile)))

											Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
											Expect(fakeActor.ReadManifestArgsForCall(0)).To(Equal(yamlFile))
										})
									})

									When("the directory contains both a 'manifest.yml' and 'manifest.yaml' file", func() {
										BeforeEach(func() {
											ymlFile = filepath.Join(providedPath, "manifest.yml")
											err := ioutil.WriteFile(ymlFile, []byte(`key: "value"`), 0666)
											Expect(err).ToNot(HaveOccurred())

											yamlFile = filepath.Join(providedPath, "manifest.yaml")
											err = ioutil.WriteFile(yamlFile, []byte(`key: "value"`), 0666)
											Expect(err).ToNot(HaveOccurred())
										})

										It("should read the manifest.yml file and outputs corresponding flavor text", func() {
											Expect(executeErr).ToNot(HaveOccurred())

											Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
											Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(ymlFile)))

											Expect(fakeActor.ReadManifestCallCount()).To(Equal(1))
											Expect(fakeActor.ReadManifestArgsForCall(0)).To(Equal(ymlFile))
										})
									})
								})
							})
						})

						When("an app name and manifest are provided", func() {
							var (
								tmpDir         string
								pathToManifest string

								originalDir string
							)

							BeforeEach(func() {
								var err error
								tmpDir, err = ioutil.TempDir("", "push-command-test")
								Expect(err).ToNot(HaveOccurred())

								// OS X uses weird symlinks that causes problems for some tests
								tmpDir, err = filepath.EvalSymlinks(tmpDir)
								Expect(err).ToNot(HaveOccurred())

								pathToManifest = filepath.Join(tmpDir, "manifest.yml")
								err = ioutil.WriteFile(pathToManifest, []byte("some manfiest file"), 0666)
								Expect(err).ToNot(HaveOccurred())

								originalDir, err = os.Getwd()
								Expect(err).ToNot(HaveOccurred())

								err = os.Chdir(tmpDir)
								Expect(err).ToNot(HaveOccurred())
							})

							AfterEach(func() {
								Expect(os.Chdir(originalDir)).ToNot(HaveOccurred())
								Expect(os.RemoveAll(tmpDir)).ToNot(HaveOccurred())
							})

							It("outputs corresponding flavor text", func() {
								Expect(executeErr).ToNot(HaveOccurred())

								Expect(testUI.Out).To(Say(`Pushing from manifest to org some-org / space some-space as some-user\.\.\.`))
								Expect(testUI.Out).To(Say("Using manifest file %s", regexp.QuoteMeta(pathToManifest)))
							})
						})

						It("converts the manifests to app configs and outputs config warnings", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(testUI.Err).To(Say("some-config-warnings"))

							Expect(fakeActor.ConvertToApplicationConfigsCallCount()).To(Equal(1))
							orgGUID, spaceGUID, noStart, manifests := fakeActor.ConvertToApplicationConfigsArgsForCall(0)
							Expect(orgGUID).To(Equal("some-org-guid"))
							Expect(spaceGUID).To(Equal("some-space-guid"))
							Expect(noStart).To(BeFalse())
							Expect(manifests).To(Equal(appManifests))
						})

						It("outputs flavor text prior to generating app configuration", func() {
							Expect(executeErr).ToNot(HaveOccurred())
							Expect(testUI.Out).To(Say("Pushing app %s to org some-org / space some-space as some-user", appName))
							Expect(testUI.Out).To(Say(`Getting app info\.\.\.`))
						})

						It("applies each of the application configurations", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(fakeActor.ApplyCallCount()).To(Equal(1))
							config, progressBar := fakeActor.ApplyArgsForCall(0)
							Expect(config).To(Equal(appConfigs[0]))
							Expect(progressBar).To(Equal(fakeProgressBar))
						})

						It("display diff of changes", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(testUI.Out).To(Say(`\s+name:\s+%s`, appName))
							Expect(testUI.Out).To(Say(`\s+path:\s+%s`, regexp.QuoteMeta(appConfigs[0].Path)))
							Expect(testUI.Out).To(Say(`\s+routes:`))
							for _, route := range appConfigs[0].CurrentRoutes {
								Expect(testUI.Out).To(Say(route.String()))
							}
							for _, route := range appConfigs[0].DesiredRoutes {
								Expect(testUI.Out).To(Say(route.String()))
							}
						})

						When("the app starts", func() {
							It("displays app events and warnings", func() {
								Expect(executeErr).ToNot(HaveOccurred())

								Expect(testUI.Out).To(Say(`Creating app with these attributes\.\.\.`))
								Expect(testUI.Out).To(Say(`Mapping routes\.\.\.`))
								Expect(testUI.Out).To(Say(`Unmapping routes\.\.\.`))
								Expect(testUI.Out).To(Say(`Binding services\.\.\.`))
								Expect(testUI.Out).To(Say(`Comparing local files to remote cache\.\.\.`))
								Expect(testUI.Out).To(Say("All files found in remote cache; nothing to upload."))
								Expect(testUI.Out).To(Say(`Waiting for API to complete processing files\.\.\.`))
								Expect(testUI.Out).To(Say(`Packaging files to upload\.\.\.`))
								Expect(testUI.Out).To(Say(`Uploading files\.\.\.`))
								Expect(testUI.Out).To(Say(`Retrying upload due to an error\.\.\.`))
								Expect(testUI.Out).To(Say(`Waiting for API to complete processing files\.\.\.`))
								Expect(testUI.Out).To(Say(`Stopping app\.\.\.`))

								Expect(testUI.Err).To(Say("some-config-warnings"))
								Expect(testUI.Err).To(Say("apply-1"))
								Expect(testUI.Err).To(Say("apply-2"))
							})

							It("displays app staging logs", func() {
								Expect(executeErr).ToNot(HaveOccurred())

								Expect(testUI.Out).To(Say("log message 1"))
								Expect(testUI.Out).To(Say("log message 2"))

								Expect(fakeRestartActor.RestartApplicationCallCount()).To(Equal(1))
								appConfig, _ := fakeRestartActor.RestartApplicationArgsForCall(0)
								Expect(appConfig).To(Equal(updatedConfig.CurrentApplication.Application))
							})

							When("the API is below MinVersionV3", func() {
								BeforeEach(func() {
									fakeApplicationSummaryActor.CloudControllerV3APIVersionReturns(ccversion.MinV3ClientVersion)
								})

								It("displays the app summary with isolation segments as well as warnings", func() {
									Expect(executeErr).ToNot(HaveOccurred())
									Expect(testUI.Out).To(Say(`name:\s+%s`, appName))
									Expect(testUI.Out).To(Say(`requested state:\s+started`))
									Expect(testUI.Out).To(Say(`instances:\s+1\/3`))
									Expect(testUI.Out).To(Say(`usage:\s+128M x 3 instances`))
									Expect(testUI.Out).To(Say(`routes:\s+banana.fruit.com/hi, foobar.com:13`))
									Expect(testUI.Out).To(Say(`last uploaded:\s+\w{3} [0-3]\d \w{3} [0-2]\d:[0-5]\d:[0-5]\d \w+ \d{4}`))
									Expect(testUI.Out).To(Say(`stack:\s+potatos`))
									Expect(testUI.Out).To(Say(`buildpack:\s+some-buildpack`))
									Expect(testUI.Out).To(Say(`start command:\s+some start command`))

									Expect(testUI.Err).To(Say("app-summary-warning"))
								})
							})

							When("the api is at least MinVersionV3", func() {
								BeforeEach(func() {
									fakeApplicationSummaryActor.CloudControllerV3APIVersionReturns(ccversion.MinVersionApplicationFlowV3)
									fakeApplicationSummaryActor.GetApplicationSummaryByNameAndSpaceReturns(
										v2v3action.ApplicationSummary{
											ApplicationSummary: v3action.ApplicationSummary{
												Application: v3action.Application{
													Name: appName,
												},
												ProcessSummaries: v3action.ProcessSummaries{
													{
														Process: v3action.Process{
															Type:       "aba",
															Command:    "some-command-1",
															MemoryInMB: types.NullUint64{Value: 32, IsSet: true},
															DiskInMB:   types.NullUint64{Value: 1024, IsSet: true},
														},
													},
													{
														Process: v3action.Process{
															Type:       "console",
															Command:    "some-command-2",
															MemoryInMB: types.NullUint64{Value: 16, IsSet: true},
															DiskInMB:   types.NullUint64{Value: 512, IsSet: true},
														},
													},
												},
											},
										},
										v2v3action.Warnings{"combo-summary-warning"},
										nil)
								})

								It("displays process information", func() {
									Expect(executeErr).ToNot(HaveOccurred())

									Expect(testUI.Out).To(Say(`name:\s+%s`, appName))
									Expect(testUI.Out).To(Say(`type:\s+aba`))
									Expect(testUI.Out).To(Say(`instances:\s+0/0`))
									Expect(testUI.Out).To(Say(`memory usage:\s+32M`))
									Expect(testUI.Out).To(Say(`start command:\s+some-command-1`))
									Expect(testUI.Out).To(Say(`type:\s+console`))
									Expect(testUI.Out).To(Say(`instances:\s+0/0`))
									Expect(testUI.Out).To(Say(`memory usage:\s+16M`))
									Expect(testUI.Out).To(Say(`start command:\s+some-command-2`))

									Expect(testUI.Err).To(Say("combo-summary-warning"))

									Expect(fakeApplicationSummaryActor.GetApplicationSummaryByNameAndSpaceCallCount()).To(Equal(1))
									passedAppName, spaceGUID, withObfuscatedValues := fakeApplicationSummaryActor.GetApplicationSummaryByNameAndSpaceArgsForCall(0)
									Expect(passedAppName).To(Equal(appName))
									Expect(spaceGUID).To(Equal("some-space-guid"))
									Expect(withObfuscatedValues).To(BeTrue())
								})
							})

							When("the start command is explicitly set", func() {
								BeforeEach(func() {
									applicationSummary := v2action.ApplicationSummary{
										Application: v2action.Application{
											Command:              types.FilteredString{IsSet: true, Value: "a-different-start-command"},
											DetectedBuildpack:    types.FilteredString{IsSet: true, Value: "some-buildpack"},
											DetectedStartCommand: types.FilteredString{IsSet: true, Value: "some start command"},
											GUID:                 "some-app-guid",
											Instances:            types.NullInt{Value: 3, IsSet: true},
											Memory:               types.NullByteSizeInMb{IsSet: true, Value: 128},
											Name:                 appName,
											PackageUpdatedAt:     time.Unix(0, 0),
											State:                "STARTED",
										},
										Stack: v2action.Stack{
											Name: "potatos",
										},
										Routes: []v2action.Route{
											{
												Host: "banana",
												Domain: v2action.Domain{
													Name: "fruit.com",
												},
												Path: "/hi",
											},
											{
												Domain: v2action.Domain{
													Name: "foobar.com",
												},
												Port: types.NullInt{IsSet: true, Value: 13},
											},
										},
									}
									warnings := []string{"app-summary-warning"}

									applicationSummary.RunningInstances = []v2action.ApplicationInstanceWithStats{{State: "RUNNING"}}

									fakeRestartActor.GetApplicationSummaryByNameAndSpaceReturns(applicationSummary, warnings, nil)
								})

								It("displays the correct start command", func() {
									Expect(executeErr).ToNot(HaveOccurred())
									Expect(testUI.Out).To(Say(`name:\s+%s`, appName))
									Expect(testUI.Out).To(Say(`start command:\s+a-different-start-command`))
								})
							})
						})

						When("no-start is set", func() {
							BeforeEach(func() {
								cmd.NoStart = true

								applicationSummary := v2action.ApplicationSummary{
									Application: v2action.Application{
										Command:              types.FilteredString{IsSet: true, Value: "a-different-start-command"},
										DetectedBuildpack:    types.FilteredString{IsSet: true, Value: "some-buildpack"},
										DetectedStartCommand: types.FilteredString{IsSet: true, Value: "some start command"},
										GUID:                 "some-app-guid",
										Instances:            types.NullInt{Value: 3, IsSet: true},
										Memory:               types.NullByteSizeInMb{IsSet: true, Value: 128},
										Name:                 appName,
										PackageUpdatedAt:     time.Unix(0, 0),
										State:                "STOPPED",
									},
									Stack: v2action.Stack{
										Name: "potatos",
									},
									Routes: []v2action.Route{
										{
											Host: "banana",
											Domain: v2action.Domain{
												Name: "fruit.com",
											},
											Path: "/hi",
										},
										{
											Domain: v2action.Domain{
												Name: "foobar.com",
											},
											Port: types.NullInt{IsSet: true, Value: 13},
										},
									},
								}
								warnings := []string{"app-summary-warning"}

								fakeRestartActor.GetApplicationSummaryByNameAndSpaceReturns(applicationSummary, warnings, nil)
							})

							When("the app is not running", func() {
								It("does not start the app", func() {
									Expect(executeErr).ToNot(HaveOccurred())
									Expect(testUI.Out).To(Say(`Waiting for API to complete processing files\.\.\.`))
									Expect(testUI.Out).To(Say(`name:\s+%s`, appName))
									Expect(testUI.Out).To(Say(`requested state:\s+stopped`))

									Expect(fakeRestartActor.RestartApplicationCallCount()).To(Equal(0))
								})
							})
						})
					})

					When("the apply errors", func() {
						var expectedErr error

						BeforeEach(func() {
							expectedErr = errors.New("no wayz dude")
							fakeActor.ApplyStub = func(_ pushaction.ApplicationConfig, _ pushaction.ProgressBar) (<-chan pushaction.ApplicationConfig, <-chan pushaction.Event, <-chan pushaction.Warnings, <-chan error) {
								configStream := make(chan pushaction.ApplicationConfig)
								eventStream := make(chan pushaction.Event)
								warningsStream := make(chan pushaction.Warnings)
								errorStream := make(chan error)

								go func() {
									defer GinkgoRecover()

									Eventually(warningsStream).Should(BeSent(pushaction.Warnings{"apply-1", "apply-2"}))
									Eventually(errorStream).Should(BeSent(expectedErr))
									close(configStream)
									close(eventStream)
									close(warningsStream)
									close(errorStream)
								}()

								return configStream, eventStream, warningsStream, errorStream
							}
						})

						It("outputs the warnings and returns the error", func() {
							Expect(executeErr).To(MatchError(expectedErr))

							Expect(testUI.Err).To(Say("some-config-warnings"))
							Expect(testUI.Err).To(Say("apply-1"))
							Expect(testUI.Err).To(Say("apply-2"))
						})
					})
				})

				When("there is an error converting the app setting into a config", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("no wayz dude")
						fakeActor.ConvertToApplicationConfigsReturns(nil, pushaction.Warnings{"some-config-warnings"}, expectedErr)
					})

					It("outputs the warnings and returns the error", func() {
						Expect(executeErr).To(MatchError(expectedErr))

						Expect(testUI.Err).To(Say("some-config-warnings"))
					})
				})
			})

			When("the push settings are invalid", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("no wayz dude")
					fakeActor.MergeAndValidateSettingsAndManifestsReturns(nil, expectedErr)
				})

				It("returns the error", func() {
					Expect(executeErr).To(MatchError(expectedErr))
				})
			})
		})
	})

	Describe("GetCommandLineSettings", func() {
		Context("valid flag combinations", func() {
			var (
				settings               pushaction.CommandLineSettings
				commandLineSettingsErr error
			)

			JustBeforeEach(func() {
				settings, commandLineSettingsErr = cmd.GetCommandLineSettings()
				Expect(commandLineSettingsErr).ToNot(HaveOccurred())
			})

			When("general app settings are given", func() {
				BeforeEach(func() {
					cmd.Buildpacks = []string{"some-buildpack"}
					cmd.Command = flag.Command{FilteredString: types.FilteredString{IsSet: true, Value: "echo foo bar baz"}}
					cmd.DiskQuota = flag.Megabytes{NullUint64: types.NullUint64{Value: 1024, IsSet: true}}
					cmd.HealthCheckTimeout = 14
					cmd.HealthCheckType = flag.HealthCheckType{Type: "http"}
					cmd.Instances = flag.Instances{NullInt: types.NullInt{Value: 12, IsSet: true}}
					cmd.Memory = flag.Megabytes{NullUint64: types.NullUint64{Value: 100, IsSet: true}}
					cmd.StackName = "some-stack"
				})

				It("sets them on the command line settings", func() {
					Expect(commandLineSettingsErr).ToNot(HaveOccurred())
					Expect(settings.Buildpacks).To(ConsistOf("some-buildpack"))
					Expect(settings.Command).To(Equal(types.FilteredString{IsSet: true, Value: "echo foo bar baz"}))
					Expect(settings.DiskQuota).To(Equal(uint64(1024)))
					Expect(settings.HealthCheckTimeout).To(Equal(14))
					Expect(settings.HealthCheckType).To(Equal("http"))
					Expect(settings.Instances).To(Equal(types.NullInt{Value: 12, IsSet: true}))
					Expect(settings.Memory).To(Equal(uint64(100)))
					Expect(settings.StackName).To(Equal("some-stack"))
				})
			})

			Context("route related flags", func() {
				When("given customed route settings", func() {
					BeforeEach(func() {
						cmd.Domain = "some-domain"
					})

					It("sets NoHostname on the command line settings", func() {
						Expect(settings.DefaultRouteDomain).To(Equal("some-domain"))
					})
				})

				When("--hostname is given", func() {
					BeforeEach(func() {
						cmd.Hostname = "some-hostname"
					})

					It("sets DefaultRouteHostname on the command line settings", func() {
						Expect(settings.DefaultRouteHostname).To(Equal("some-hostname"))
					})
				})

				When("--no-hostname is given", func() {
					BeforeEach(func() {
						cmd.NoHostname = true
					})

					It("sets NoHostname on the command line settings", func() {
						Expect(settings.NoHostname).To(BeTrue())
					})
				})

				When("--random-route is given", func() {
					BeforeEach(func() {
						cmd.RandomRoute = true
					})

					It("sets --random-route on the command line settings", func() {
						Expect(commandLineSettingsErr).ToNot(HaveOccurred())
						Expect(settings.RandomRoute).To(BeTrue())
					})
				})

				When("--route-path is given", func() {
					BeforeEach(func() {
						cmd.RoutePath = flag.RoutePath{Path: "/some-path"}
					})

					It("sets --route-path on the command line settings", func() {
						Expect(commandLineSettingsErr).ToNot(HaveOccurred())
						Expect(settings.RoutePath).To(Equal("/some-path"))
					})
				})

				When("--no-route is given", func() {
					BeforeEach(func() {
						cmd.NoRoute = true
					})

					It("sets NoRoute on the command line settings", func() {
						Expect(settings.NoRoute).To(BeTrue())
					})
				})
			})

			Context("app bits", func() {
				When("-p flag is given", func() {
					BeforeEach(func() {
						cmd.AppPath = "some-directory-path"
					})

					It("sets ProvidedAppPath", func() {
						Expect(settings.ProvidedAppPath).To(Equal("some-directory-path"))
					})
				})

				When("the -o flag is given", func() {
					BeforeEach(func() {
						cmd.DockerImage.Path = "some-docker-image-path"
					})

					It("creates command line setting from command line arguments", func() {
						Expect(settings.DockerImage).To(Equal("some-docker-image-path"))
					})

					Context("--docker-username flags is given", func() {
						BeforeEach(func() {
							cmd.DockerUsername = "some-docker-username"
						})

						Context("the docker password environment variable is set", func() {
							BeforeEach(func() {
								fakeConfig.DockerPasswordReturns("some-docker-password")
							})

							It("creates command line setting from command line arguments and config", func() {
								Expect(testUI.Out).To(Say("Using docker repository password from environment variable CF_DOCKER_PASSWORD."))

								Expect(settings.Name).To(Equal(appName))
								Expect(settings.DockerImage).To(Equal("some-docker-image-path"))
								Expect(settings.DockerUsername).To(Equal("some-docker-username"))
								Expect(settings.DockerPassword).To(Equal("some-docker-password"))
							})
						})

						Context("the docker password environment variable is *not* set", func() {
							BeforeEach(func() {
								input.Write([]byte("some-docker-password\n"))
							})

							It("prompts the user for a password", func() {
								Expect(testUI.Out).To(Say("Environment variable CF_DOCKER_PASSWORD not set."))
								Expect(testUI.Out).To(Say("Docker password"))

								Expect(settings.Name).To(Equal(appName))
								Expect(settings.DockerImage).To(Equal("some-docker-image-path"))
								Expect(settings.DockerUsername).To(Equal("some-docker-username"))
								Expect(settings.DockerPassword).To(Equal("some-docker-password"))
							})
						})
					})
				})
			})
		})

		DescribeTable("validation errors when flags are passed",
			func(setup func(), expectedErr error) {
				setup()
				_, commandLineSettingsErr := cmd.GetCommandLineSettings()
				Expect(commandLineSettingsErr).To(MatchError(expectedErr))
			},

			Entry("--droplet and --docker-username",
				func() {
					cmd.DropletPath = "some-droplet-path"
					cmd.DockerUsername = "some-docker-username"
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--droplet", "--docker-username", "-p"}}),

			Entry("--droplet and --docker-image",
				func() {
					cmd.DropletPath = "some-droplet-path"
					cmd.DockerImage.Path = "some-docker-image"
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--droplet", "--docker-image", "-o"}}),

			Entry("--droplet and -p",
				func() {
					cmd.DropletPath = "some-droplet-path"
					cmd.AppPath = "some-directory-path"
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--droplet", "-p"}}),

			Entry("-o and -p",
				func() {
					cmd.DockerImage.Path = "some-docker-image"
					cmd.AppPath = "some-directory-path"
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--docker-image, -o", "-p"}}),

			Entry("-b and --docker-image",
				func() {
					cmd.DockerImage.Path = "some-docker-image"
					cmd.Buildpacks = []string{"some-buildpack"}
				},
				translatableerror.ArgumentCombinationError{Args: []string{"-b", "--docker-image, -o"}}),

			Entry("--docker-username (without DOCKER_PASSWORD env set)",
				func() {
					cmd.DockerUsername = "some-docker-username"
				},
				translatableerror.RequiredFlagsError{Arg1: "--docker-image, -o", Arg2: "--docker-username"}),

			Entry("-d and --no-route",
				func() {
					cmd.Domain = "some-domain"
					cmd.NoRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"-d", "--no-route"}}),

			Entry("--hostname and --no-hostname",
				func() {
					cmd.Hostname = "po-tate-toe"
					cmd.NoHostname = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--hostname", "-n", "--no-hostname"}}),

			Entry("--hostname and --no-route",
				func() {
					cmd.Hostname = "po-tate-toe"
					cmd.NoRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--hostname", "-n", "--no-route"}}),

			Entry("--no-hostname and --no-route",
				func() {
					cmd.NoHostname = true
					cmd.NoRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--no-hostname", "--no-route"}}),

			Entry("-f and --no-manifest",
				func() {
					cmd.PathToManifest = "/some/path.yml"
					cmd.NoManifest = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"-f", "--no-manifest"}}),

			Entry("--random-route and --hostname",
				func() {
					cmd.Hostname = "po-tate-toe"
					cmd.RandomRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--hostname", "-n", "--random-route"}}),

			Entry("--random-route and --no-hostname",
				func() {
					cmd.RandomRoute = true
					cmd.NoHostname = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--no-hostname", "--random-route"}}),

			Entry("--random-route and --no-route",
				func() {
					cmd.RandomRoute = true
					cmd.NoRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--no-route", "--random-route"}}),

			Entry("--random-route and --route-path",
				func() {
					cmd.RoutePath = flag.RoutePath{Path: "/bananas"}
					cmd.RandomRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--random-route", "--route-path"}}),

			Entry("--route-path and --no-route",
				func() {
					cmd.RoutePath = flag.RoutePath{Path: "/bananas"}
					cmd.NoRoute = true
				},
				translatableerror.ArgumentCombinationError{Args: []string{"--route-path", "--no-route"}}),
		)
	})
})
