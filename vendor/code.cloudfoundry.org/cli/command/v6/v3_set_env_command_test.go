package v6_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v3action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccversion"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/translatableerror"
	. "code.cloudfoundry.org/cli/command/v6"
	"code.cloudfoundry.org/cli/command/v6/v6fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("v3-set-env Command", func() {
	var (
		cmd             V3SetEnvCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v6fakes.FakeV3SetEnvActor
		binaryName      string
		executeErr      error
		appName         string
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeV3SetEnvActor)

		cmd = V3SetEnvCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
		appName = "some-app"

		cmd.RequiredArgs.AppName = appName
		cmd.RequiredArgs.EnvironmentVariableName = "some-key"
		cmd.RequiredArgs.EnvironmentVariableValue = "some-value"
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	When("the API version is below the minimum", func() {
		BeforeEach(func() {
			fakeActor.CloudControllerAPIVersionReturns(ccversion.MinV3ClientVersion)
		})

		It("returns a MinimumAPIVersionNotMetError", func() {
			Expect(executeErr).To(MatchError(translatableerror.MinimumCFAPIVersionNotMetError{
				CurrentVersion: ccversion.MinV3ClientVersion,
				MinimumVersion: ccversion.MinVersionApplicationFlowV3,
			}))
		})

		It("displays the experimental warning", func() {
			Expect(testUI.Err).To(Say("This command is in EXPERIMENTAL stage and may change without notice"))
		})
	})

	When("checking target fails", func() {
		BeforeEach(func() {
			fakeActor.CloudControllerAPIVersionReturns(ccversion.MinVersionApplicationFlowV3)
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

	When("the user is logged in, an org is targeted and a space is targeted", func() {
		BeforeEach(func() {
			fakeActor.CloudControllerAPIVersionReturns(ccversion.MinVersionApplicationFlowV3)
			fakeConfig.TargetedSpaceReturns(configv3.Space{Name: "some-space", GUID: "some-space-guid"})
			fakeConfig.TargetedOrganizationReturns(configv3.Organization{Name: "some-org"})
		})

		When("getting the current user returns an error", func() {
			BeforeEach(func() {
				fakeConfig.CurrentUserReturns(configv3.User{}, errors.New("some-error"))
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError("some-error"))
			})
		})

		When("getting the current user succeeds", func() {
			BeforeEach(func() {
				fakeConfig.CurrentUserReturns(configv3.User{Name: "banana"}, nil)
			})

			When("setting the environment succeeds", func() {
				BeforeEach(func() {
					fakeActor.SetEnvironmentVariableByApplicationNameAndSpaceReturns(v3action.Warnings{"set-warning-1", "set-warning-2"}, nil)
				})

				It("sets the environment variable and value pair", func() {
					Expect(executeErr).ToNot(HaveOccurred())

					Expect(testUI.Out).To(Say(`Setting env variable some-key for app some-app in org some-org / space some-space as banana\.\.\.`))

					Expect(testUI.Err).To(Say("set-warning-1"))
					Expect(testUI.Err).To(Say("set-warning-2"))
					Expect(testUI.Out).To(Say("OK"))
					Expect(testUI.Out).To(Say(`TIP: Use 'cf v3-stage some-app' to ensure your env variable changes take effect\.`))

					Expect(fakeActor.SetEnvironmentVariableByApplicationNameAndSpaceCallCount()).To(Equal(1))
					appName, spaceGUID, envVariablePair := fakeActor.SetEnvironmentVariableByApplicationNameAndSpaceArgsForCall(0)
					Expect(appName).To(Equal("some-app"))
					Expect(spaceGUID).To(Equal("some-space-guid"))
					Expect(envVariablePair.Key).To(Equal("some-key"))
					Expect(envVariablePair.Value).To(Equal("some-value"))
				})
			})

			When("the set environment variable returns an unknown error", func() {
				var expectedErr error
				BeforeEach(func() {
					expectedErr = errors.New("some-error")
					fakeActor.SetEnvironmentVariableByApplicationNameAndSpaceReturns(v3action.Warnings{"get-warning-1", "get-warning-2"}, expectedErr)
				})

				It("returns the error", func() {
					Expect(executeErr).To(Equal(expectedErr))
					Expect(testUI.Out).To(Say(`Setting env variable some-key for app some-app in org some-org / space some-space as banana\.\.\.`))

					Expect(testUI.Err).To(Say("get-warning-1"))
					Expect(testUI.Err).To(Say("get-warning-2"))
					Expect(testUI.Out).ToNot(Say("OK"))
				})
			})
		})
	})
})
