package v2action_test

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cli/actor/actionerror"
	. "code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v2action/v2actionfakes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/types"
)

var _ = Describe("Buildpack", func() {
	var (
		actor                     *Actor
		fakeCloudControllerClient *v2actionfakes.FakeCloudControllerClient
	)

	BeforeEach(func() {
		fakeCloudControllerClient = new(v2actionfakes.FakeCloudControllerClient)
		actor = NewActor(fakeCloudControllerClient, nil, nil)
	})

	Describe("Buildpack", func() {
		Describe("NoStack", func() {
			var buildpack Buildpack

			When("the stack is empty", func() {
				BeforeEach(func() {
					buildpack.Stack = ""
				})

				It("returns true", func() {
					Expect(buildpack.NoStack()).To(BeTrue())
				})
			})

			When("the stack is set", func() {
				BeforeEach(func() {
					buildpack.Stack = "something i guess"
				})

				It("returns false", func() {
					Expect(buildpack.NoStack()).To(BeFalse())
				})
			})
		})
	})

	Describe("CreateBuildpack", func() {
		var (
			buildpack  Buildpack
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			buildpack, warnings, executeErr = actor.CreateBuildpack("some-bp-name", 42, true)
		})

		When("creating the buildpack is successful", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.CreateBuildpackReturns(ccv2.Buildpack{GUID: "some-guid"}, ccv2.Warnings{"some-create-warning"}, nil)
			})

			It("returns the buildpack and all warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeCloudControllerClient.CreateBuildpackCallCount()).To(Equal(1))
				Expect(fakeCloudControllerClient.CreateBuildpackArgsForCall(0)).To(Equal(ccv2.Buildpack{
					Name:     "some-bp-name",
					Position: types.NullInt{IsSet: true, Value: 42},
					Enabled:  types.NullBool{IsSet: true, Value: true},
				}))

				Expect(buildpack).To(Equal(Buildpack{GUID: "some-guid"}))
				Expect(warnings).To(ConsistOf("some-create-warning"))
			})
		})

		When("the buildpack already exists with nil stack", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.CreateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-create-warning"}, ccerror.BuildpackAlreadyExistsWithoutStackError{Message: ""})
			})

			It("returns a BuildpackAlreadyExistsWithoutStackError error and all warnings", func() {
				Expect(warnings).To(ConsistOf("some-create-warning"))
				Expect(executeErr).To(MatchError(actionerror.BuildpackAlreadyExistsWithoutStackError{BuildpackName: "some-bp-name"}))
			})
		})

		When("the buildpack name is taken", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.CreateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-create-warning"}, ccerror.BuildpackNameTakenError{Message: ""})
			})

			It("returns a BuildpackAlreadyExistsWithoutStackError error and all warnings", func() {
				Expect(warnings).To(ConsistOf("some-create-warning"))
				Expect(executeErr).To(MatchError(actionerror.BuildpackNameTakenError{Name: "some-bp-name"}))
			})
		})

		When("a cc create error occurs", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.CreateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-create-warning"}, errors.New("kaboom"))
			})

			It("returns an error and all warnings", func() {
				Expect(warnings).To(ConsistOf("some-create-warning"))
				Expect(executeErr).To(MatchError("kaboom"))
			})
		})
	})

	Describe("GetBuildpackByName", func() {
		var (
			buildpack  Buildpack
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			buildpack, warnings, executeErr = actor.GetBuildpackByName("some-bp-name")
		})

		When("one buildpack with the same name exists", func() {
			When("the buildpack also has no stack", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
						{
							Name:  "some-bp-name",
							GUID:  "some-bp-guid",
							Stack: "",
						},
					}, ccv2.Warnings{"some-warning"}, nil)
				})

				It("returns the buildpack", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))

					Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
					Expect(buildpack).To(Equal(Buildpack{
						Name: "some-bp-name",
						GUID: "some-bp-guid",
					}))
				})
			})

			When("the buildpack has a stack", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
						{
							Name:  "some-bp-name",
							GUID:  "some-bp-guid",
							Stack: "some-stack-name",
						},
					}, ccv2.Warnings{"some-warning"}, nil)
				})

				It("returns the buildpack", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))

					Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
					Expect(buildpack).To(Equal(Buildpack{
						Name:  "some-bp-name",
						GUID:  "some-bp-guid",
						Stack: "some-stack-name",
					}))
				})
			})
		})

		When("the client returns an empty set of buildpacks", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, nil)
			})

			It("returns a buildpack not found error", func() {
				Expect(executeErr).To(MatchError(actionerror.BuildpackNotFoundError{BuildpackName: "some-bp-name"}))
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
				Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
			})
		})

		When("the client returns more than one buildpack", func() {
			When("one of the buildpacks has no stack", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
						{
							Name:  "some-bp-name",
							GUID:  "bp-guid-1",
							Stack: "some-stack-name",
						},
						{
							Name:  "some-bp-name",
							GUID:  "bp-guid-2",
							Stack: "",
						},
					}, ccv2.Warnings{"some-warning"}, nil)
				})

				It("returns the correct buildpack", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))

					Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
					Expect(buildpack).To(Equal(Buildpack{
						Name:  "some-bp-name",
						GUID:  "bp-guid-2",
						Stack: "",
					}))
				})

			})
			Context("none of the buildpacks have no stack", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
						{
							Name:  "some-bp-name",
							GUID:  "bp-guid-1",
							Stack: "some-stack-1",
						},
						{
							Name:  "some-bp-name",
							GUID:  "bp-guid-2",
							Stack: "some-stack-2",
						},
					}, ccv2.Warnings{"some-warning"}, nil)
				})
				It("returns a multiple buildpacks found error", func() {
					Expect(executeErr).To(MatchError(actionerror.MultipleBuildpacksFoundError{BuildpackName: "some-bp-name"}))
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
					Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
				})
			})
		})

		When("the client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, ccerror.APINotFoundError{})
			})

			It("returns a buildpack not found error", func() {
				Expect(executeErr).To(MatchError(ccerror.APINotFoundError{}))
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
				Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
			})
		})
	})

	Describe("GetBuildpackByNameAndStack", func() {
		var (
			buildpack  Buildpack
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			buildpack, warnings, executeErr = actor.GetBuildpackByNameAndStack("some-bp-name", "some-stack-name")
		})

		When("the client returns a buildpack with the same name and stack", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
					{
						Name:  "some-bp-name",
						GUID:  "some-bp-guid",
						Stack: "some-stack-name",
					},
				}, ccv2.Warnings{"some-warning"}, nil)
			})

			It("returns the buildpack", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))

				Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
				Expect(buildpack).To(Equal(Buildpack{
					Name:  "some-bp-name",
					GUID:  "some-bp-guid",
					Stack: "some-stack-name",
				}))
			})
		})

		When("the client returns an empty set of buildpacks", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, nil)
			})

			It("returns a buildpack not found error", func() {
				Expect(executeErr).To(MatchError(actionerror.BuildpackNotFoundError{BuildpackName: "some-bp-name", StackName: "some-stack-name"}))
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
				Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
			})
		})

		When("the client returns more than one buildpack", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
					{
						Name:  "some-bp-name",
						GUID:  "bp-guid-1",
						Stack: "some-stack-name",
					},
					{
						Name:  "some-bp-name",
						GUID:  "bp-guid-2",
						Stack: "some-stack-name",
					},
				}, ccv2.Warnings{"some-warning"}, nil)
			})

			It("returns a multiple buildpacks found error", func() {
				Expect(executeErr).To(MatchError(actionerror.MultipleBuildpacksFoundError{BuildpackName: "some-bp-name"}))
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
				Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
			})
		})

		When("the client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, ccerror.APINotFoundError{})
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(ccerror.APINotFoundError{}))
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
				Expect(fakeCloudControllerClient.GetBuildpacksCallCount()).To(Equal(1))
			})
		})
	})

	Describe("PrepareBuildpackBits", func() {
		var (
			inPath         string
			outPath        string
			tmpDirPath     string
			fakeDownloader *v2actionfakes.FakeDownloader

			executeErr error
		)

		BeforeEach(func() {
			fakeDownloader = new(v2actionfakes.FakeDownloader)
		})

		JustBeforeEach(func() {
			outPath, executeErr = actor.PrepareBuildpackBits(inPath, tmpDirPath, fakeDownloader)
		})

		When("the buildpack path is a url", func() {
			BeforeEach(func() {
				inPath = "http://buildpacks.com/a.zip"
				fakeDownloader = new(v2actionfakes.FakeDownloader)

				var err error
				tmpDirPath, err = ioutil.TempDir("", "buildpackdir-")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(tmpDirPath)).ToNot(HaveOccurred())
			})

			When("downloading the file succeeds", func() {
				BeforeEach(func() {
					fakeDownloader.DownloadReturns("/tmp/buildpackdir-100/a.zip", nil)
				})

				It("downloads the buildpack to a local file", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(fakeDownloader.DownloadCallCount()).To(Equal(1))

					inputPath, inputTmpDirPath := fakeDownloader.DownloadArgsForCall(0)
					Expect(inputPath).To(Equal("http://buildpacks.com/a.zip"))
					Expect(inputTmpDirPath).To(Equal(tmpDirPath))
				})
			})

			When("downloading the file fails", func() {
				BeforeEach(func() {
					fakeDownloader.DownloadReturns("", errors.New("some-download-error"))
				})

				It("returns the error", func() {
					Expect(executeErr).To(MatchError("some-download-error"))
				})
			})
		})

		When("the buildpack path points to a directory", func() {
			var tempFile *os.File
			BeforeEach(func() {
				var err error
				inPath, err = ioutil.TempDir("", "buildpackdir-")
				Expect(err).ToNot(HaveOccurred())

				tempFile, err = ioutil.TempFile(inPath, "foo")
				Expect(err).ToNot(HaveOccurred())

				tmpDirPath, err = ioutil.TempDir("", "buildpackdir-")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				tempFile.Close()
				Expect(os.RemoveAll(inPath)).ToNot(HaveOccurred())
				Expect(os.RemoveAll(tmpDirPath)).ToNot(HaveOccurred())
			})

			It("returns a path to the zipped directory", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeDownloader.DownloadCallCount()).To(Equal(0))

				Expect(filepath.Base(outPath)).To(Equal(filepath.Base(inPath) + ".zip"))
			})
		})

		When("the buildpack path points to an empty directory", func() {
			BeforeEach(func() {
				var err error
				inPath, err = ioutil.TempDir("", "some-empty-dir")
				Expect(err).ToNot(HaveOccurred())

				tmpDirPath, err = ioutil.TempDir("", "buildpackdir-")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(actionerror.EmptyBuildpackDirectoryError{Path: inPath}))
			})
		})

		When("the buildpack path points to a zip file", func() {
			BeforeEach(func() {
				inPath = "/foo/buildpacks/a.zip"
			})

			It("returns the local filepath", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeDownloader.DownloadCallCount()).To(Equal(0))
				Expect(outPath).To(Equal("/foo/buildpacks/a.zip"))
			})
		})
	})

	Describe("RenameBuildpack", func() {
		var (
			oldName    string
			newName    string
			stackName  string
			warnings   Warnings
			executeErr error
		)

		BeforeEach(func() {
			oldName = "some-old-name"
			newName = "some-new-name"
		})

		JustBeforeEach(func() {
			warnings, executeErr = actor.RenameBuildpack(oldName, newName, stackName)
		})

		When("the lookup succeeds", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
					{},
				},
					ccv2.Warnings{"warning-1", "warning-2"},
					nil)
			})

			When("the update succeeds", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{},
						ccv2.Warnings{"warning-3", "warning-4"},
						nil)
				})
				It("returns warnings", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "warning-3", "warning-4"))
				})
			})

			When("the update errors", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{},
						ccv2.Warnings{"warning-3", "warning-4"},
						errors.New("some-error"))
				})

				It("returns the error and warnings", func() {
					Expect(executeErr).To(MatchError("some-error"))
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "warning-3", "warning-4"))
				})
			})
		})

		When("the lookup errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns(nil,
					ccv2.Warnings{"warning-1", "warning-2"},
					errors.New("some-lookup-error"))
			})
			It("returns the error and warnings", func() {
				Expect(executeErr).To(MatchError("some-lookup-error"))
				Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
			})
		})
	})

	Describe("UpdateBuildpack", func() {
		var (
			buildpack        Buildpack
			updatedBuildpack Buildpack
			warnings         Warnings
			executeErr       error
		)

		JustBeforeEach(func() {
			buildpack = Buildpack{
				Name:  "some-bp-name",
				GUID:  "some-bp-guid",
				Stack: "some-stack",
			}
			updatedBuildpack, warnings, executeErr = actor.UpdateBuildpack(buildpack)
		})

		When("there are no errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{
					Name:  "some-bp-name",
					GUID:  "some-bp-guid",
					Stack: "some-stack",
				}, ccv2.Warnings{"some-warning"}, nil)
			})

			It("returns the updated buildpack", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
				Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))

				Expect(updatedBuildpack).To(Equal(buildpack))
			})
		})

		When("the client errors", func() {
			When("the buildpack is not found", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, ccerror.ResourceNotFoundError{})
				})

				It("returns a buildpack not found error", func() {
					Expect(executeErr).To(MatchError(actionerror.BuildpackNotFoundError{BuildpackName: "some-bp-name"}))
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
				})
			})

			When("the buildpack already exists without a stack association", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, ccerror.BuildpackAlreadyExistsWithoutStackError{})
				})

				It("returns a buildpack already exists without stack error", func() {
					Expect(executeErr).To(MatchError(actionerror.BuildpackAlreadyExistsWithoutStackError{BuildpackName: "some-bp-name"}))
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
				})
			})

			When("the buildpack already exists with a stack association", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, ccerror.BuildpackAlreadyExistsForStackError{Message: "some-message"})
				})

				It("returns a buildpack already exists for stack error", func() {
					Expect(executeErr).To(MatchError(actionerror.BuildpackAlreadyExistsForStackError{Message: "some-message"}))
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
				})
			})

			When("the client returns a generic error", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{}, ccv2.Warnings{"some-warning"}, errors.New("some-error"))
				})

				It("returns the error", func() {
					Expect(executeErr).To(MatchError("some-error"))
					Expect(warnings).To(ConsistOf(Warnings{"some-warning"}))
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
				})
			})
		})
	})

	Describe("UpdateBuildpackByNameAndStack", func() {
		var (
			expectedError        error
			warnings             Warnings
			executeErr           error
			newPosition          types.NullInt
			newLocked            types.NullBool
			newEnabled           types.NullBool
			fakeProgressBar      *v2actionfakes.FakeSimpleProgressBar
			updatedBuildpackGuid string
			stackName            string
		)

		JustBeforeEach(func() {
			fakeProgressBar = new(v2actionfakes.FakeSimpleProgressBar)
			updatedBuildpackGuid, warnings, executeErr = actor.UpdateBuildpackByNameAndStack("some-bp-name", stackName, newPosition, newLocked, newEnabled)
		})

		When("stack is an empty string", func() {
			BeforeEach(func() {
				stackName = ""
			})

			It("gets the buildpack by name only", func() {
				args := fakeCloudControllerClient.GetBuildpacksArgsForCall(0)
				Expect(len(args)).To(Equal(1))
				Expect(args[0].Values[0]).To(Equal("some-bp-name"))
			})
		})

		When("a non-empty stack name is passed", func() {
			BeforeEach(func() {
				stackName = "some-stack"
			})

			It("gets the buildpack by name and stack", func() {
				args := fakeCloudControllerClient.GetBuildpacksArgsForCall(0)
				Expect(len(args)).To(Equal(2))
				Expect(args[0].Values[0]).To(Equal("some-bp-name"))
				Expect(args[1].Values[0]).To(Equal(stackName))
			})
		})

		When("getting the buildpack fails", func() {
			BeforeEach(func() {
				expectedError = errors.New("some-error")
				fakeCloudControllerClient.GetBuildpacksReturns(nil, nil, expectedError)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(expectedError))
			})
		})

		When("getting the buildpack succeeds", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetBuildpacksReturns([]ccv2.Buildpack{
					ccv2.Buildpack{}}, ccv2.Warnings{"get warning"}, nil)
			})

			It("does not return an error", func() {
				Expect(executeErr).ToNot(HaveOccurred())
			})

			It("returns any warnings", func() {
				Expect(warnings).To(ConsistOf("get warning"))
			})

			When("no changes to the buildpack record are specified", func() {
				BeforeEach(func() {
					newPosition = types.NullInt{}
					newLocked = types.NullBool{}
					newEnabled = types.NullBool{}
				})

				It("doesn't call the CC API", func() {
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(0))
				})
			})

			When("a new position is specified", func() {
				BeforeEach(func() {
					newPosition = types.NullInt{IsSet: true, Value: 3}
					newLocked = types.NullBool{}
					newEnabled = types.NullBool{}
				})

				It("makes an API call to update the position", func() {
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
					passedBuildpack := fakeCloudControllerClient.UpdateBuildpackArgsForCall(0)
					Expect(passedBuildpack.Position).To(Equal(newPosition))
				})
			})

			When("a new locked state is specified", func() {
				BeforeEach(func() {
					newPosition = types.NullInt{}
					newLocked = types.NullBool{IsSet: true, Value: true}
					newEnabled = types.NullBool{}
				})

				It("makes an API call to update the locked state", func() {
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
					passedBuildpack := fakeCloudControllerClient.UpdateBuildpackArgsForCall(0)
					Expect(passedBuildpack.Locked).To(Equal(newLocked))
				})
			})

			When("a new enabled state is specified", func() {
				BeforeEach(func() {
					newPosition = types.NullInt{}
					newLocked = types.NullBool{}
					newEnabled = types.NullBool{IsSet: true, Value: true}
				})

				It("makes an API call to update the enabled state", func() {
					Expect(fakeCloudControllerClient.UpdateBuildpackCallCount()).To(Equal(1))
					passedBuildpack := fakeCloudControllerClient.UpdateBuildpackArgsForCall(0)
					Expect(passedBuildpack.Enabled).To(Equal(newEnabled))
				})
			})

			When("some arguments are specified and buildpack record update is needed", func() {
				BeforeEach(func() {
					newPosition = types.NullInt{IsSet: true, Value: 3}
					newLocked = types.NullBool{IsSet: true, Value: true}
					newEnabled = types.NullBool{IsSet: true, Value: true}
				})

				When("updating the buildpack record returns an error", func() {
					BeforeEach(func() {
						fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{}, nil, errors.New("failed"))
					})

					It("returns the error", func() {
						Expect(executeErr).To(MatchError("failed"))
					})
				})

				When("updating the buildpack record succeeds", func() {
					BeforeEach(func() {
						fakeCloudControllerClient.UpdateBuildpackReturns(ccv2.Buildpack{GUID: "some guid"}, ccv2.Warnings{"update warning"}, nil)
					})

					It("does not return an error", func() {
						Expect(executeErr).ToNot(HaveOccurred())
					})

					It("returns any warnings", func() {
						Expect(warnings).To(ConsistOf("get warning", "update warning"))
					})
				})
			})
		})
	})

	Describe("UploadBuildpack", func() {
		var (
			bpFile     io.Reader
			bpFilePath string
			fakePb     *v2actionfakes.FakeSimpleProgressBar

			warnings   Warnings
			executeErr error
		)

		BeforeEach(func() {
			bpFile = strings.NewReader("")
		})

		JustBeforeEach(func() {
			fakePb = new(v2actionfakes.FakeSimpleProgressBar)
			fakePb.InitializeReturns(bpFile, 0, nil)
			bpFilePath = "tmp/buildpack.zip"
			warnings, executeErr = actor.UploadBuildpack("some-bp-guid", bpFilePath, fakePb)
		})

		It("tracks the progress of the upload", func() {
			Expect(executeErr).ToNot(HaveOccurred())
			Expect(fakePb.InitializeCallCount()).To(Equal(1))
			Expect(fakePb.InitializeArgsForCall(0)).To(Equal(bpFilePath))
			Expect(fakePb.TerminateCallCount()).To(Equal(1))
		})

		When("the upload errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.UploadBuildpackReturns(ccv2.Warnings{"some-upload-warning"}, errors.New("some-upload-error"))
			})

			It("returns warnings and errors", func() {
				Expect(warnings).To(ConsistOf("some-upload-warning"))
				Expect(executeErr).To(MatchError("some-upload-error"))
			})
		})

		When("the cc returns an error because the buildpack and stack combo already exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.UploadBuildpackReturns(ccv2.Warnings{"some-upload-warning"}, ccerror.BuildpackAlreadyExistsForStackError{Message: "ya blew it"})
			})

			It("returns warnings and a BuildpackAlreadyExistsForStackError", func() {
				Expect(warnings).To(ConsistOf("some-upload-warning"))
				Expect(executeErr).To(MatchError(actionerror.BuildpackAlreadyExistsForStackError{Message: "ya blew it"}))
			})
		})

		When("the upload is successful", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.UploadBuildpackReturns(ccv2.Warnings{"some-create-warning"}, nil)
			})

			It("uploads the buildpack and returns any warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(fakeCloudControllerClient.UploadBuildpackCallCount()).To(Equal(1))
				guid, path, pbReader, size := fakeCloudControllerClient.UploadBuildpackArgsForCall(0)
				Expect(guid).To(Equal("some-bp-guid"))
				Expect(size).To(Equal(int64(0)))
				Expect(path).To(Equal(bpFilePath))
				Expect(pbReader).To(Equal(bpFile))
				Expect(warnings).To(ConsistOf("some-create-warning"))
			})
		})
	})

	Describe("Zipit", func() {
		//tested in buildpack_linux_test.go and buildpack_windows_test.go
		var (
			source string
			target string

			executeErr error
		)

		JustBeforeEach(func() {
			executeErr = Zipit(source, target, "testzip-")
		})

		When("the source directory does not exist", func() {
			BeforeEach(func() {
				source = ""
				target = ""
			})

			It("returns an error", func() {
				Expect(os.IsNotExist(executeErr)).To(BeTrue())
			})
		})
	})
})
