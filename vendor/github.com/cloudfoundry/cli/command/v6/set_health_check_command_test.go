package v6_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"
	"code.cloudfoundry.org/cli/command/commandfakes"
	. "code.cloudfoundry.org/cli/command/v6"
	"code.cloudfoundry.org/cli/command/v6/v6fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("set-health-check Command", func() {
	var (
		cmd             SetHealthCheckCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v6fakes.FakeSetHealthCheckActor
		binaryName      string
		executeErr      error
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeSetHealthCheckActor)

		cmd = SetHealthCheckCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
		fakeConfig.TargetedOrganizationReturns(configv3.Organization{
			Name: "some-org",
		})
		fakeConfig.TargetedSpaceReturns(configv3.Space{
			GUID: "some-space-guid",
			Name: "some-space",
		})

		fakeConfig.CurrentUserReturns(configv3.User{Name: "some-user"}, nil)

	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	When("checking the target fails", func() {
		BeforeEach(func() {
			fakeSharedActor.CheckTargetReturns(
				actionerror.NotLoggedInError{BinaryName: binaryName})
		})

		It("returns an error", func() {
			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			targetedOrganizationRequired, targetedSpaceRequired := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(targetedOrganizationRequired).To(Equal(true))
			Expect(targetedSpaceRequired).To(Equal(true))

			Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))
		})
	})

	When("setting the application health check type returns an error", func() {
		var expectedErr error

		BeforeEach(func() {
			cmd.RequiredArgs.AppName = "some-app"
			cmd.RequiredArgs.HealthCheck.Type = "some-health-check-type"

			expectedErr = errors.New("set health check error")
			fakeActor.SetApplicationHealthCheckTypeByNameAndSpaceReturns(
				v2action.Application{}, v2action.Warnings{"warning-1"}, expectedErr)
		})

		It("displays warnings and returns the error", func() {
			Expect(testUI.Err).To(Say("warning-1"))
			Expect(executeErr).To(MatchError(expectedErr))
		})
	})

	When("setting health check is successful", func() {
		BeforeEach(func() {
			cmd.RequiredArgs.AppName = "some-app"
			cmd.RequiredArgs.HealthCheck.Type = "some-health-check-type"
			cmd.HTTPEndpoint = "/"

			fakeActor.SetApplicationHealthCheckTypeByNameAndSpaceReturns(
				v2action.Application{}, v2action.Warnings{"warning-1"}, nil)
		})

		It("informs the user and displays warnings", func() {
			Expect(testUI.Out).To(Say("Updating health check type for app some-app in org some-org / space some-space as some-user..."))
			Expect(testUI.Err).To(Say("warning-1"))
			Expect(testUI.Out).To(Say("OK"))
			Expect(executeErr).ToNot(HaveOccurred())

			Expect(fakeActor.SetApplicationHealthCheckTypeByNameAndSpaceCallCount()).To(Equal(1))
			name, spaceGUID, healthCheckType, healthCheckHTTPEndpoint := fakeActor.SetApplicationHealthCheckTypeByNameAndSpaceArgsForCall(0)
			Expect(name).To(Equal("some-app"))
			Expect(spaceGUID).To(Equal("some-space-guid"))
			Expect(healthCheckType).To(Equal(constant.ApplicationHealthCheckType("some-health-check-type")))
			Expect(healthCheckHTTPEndpoint).To(Equal("/"))
		})
	})

	When("the app is started", func() {
		BeforeEach(func() {
			cmd.RequiredArgs.AppName = "some-app"
			cmd.RequiredArgs.HealthCheck.Type = "some-health-check-type"

			fakeActor.SetApplicationHealthCheckTypeByNameAndSpaceReturns(
				v2action.Application{State: constant.ApplicationStarted}, v2action.Warnings{"warning-1"}, nil)
		})

		It("displays a tip to restart the app", func() {
			Expect(testUI.Out).To(Say("TIP: An app restart is required for the change to take effect."))
		})
	})

})
