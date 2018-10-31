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

var _ = Describe("disable-org-isolation Command", func() {
	var (
		cmd              DisableOrgIsolationCommand
		testUI           *ui.UI
		fakeConfig       *commandfakes.FakeConfig
		fakeSharedActor  *commandfakes.FakeSharedActor
		fakeActor        *v6fakes.FakeDisableOrgIsolationActor
		binaryName       string
		executeErr       error
		isolationSegment string
		org              string
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeDisableOrgIsolationActor)

		cmd = DisableOrgIsolationCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
		org = "org1"
		isolationSegment = "segment1"
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
				MinimumVersion: ccversion.MinVersionIsolationSegmentV3,
			}))
		})
	})

	When("checking target fails", func() {
		BeforeEach(func() {
			fakeActor.CloudControllerAPIVersionReturns(ccversion.MinVersionIsolationSegmentV3)
			fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{BinaryName: binaryName})
		})

		It("returns an error", func() {
			Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))

			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(checkTargetedOrg).To(BeFalse())
			Expect(checkTargetedSpace).To(BeFalse())
		})
	})

	When("user is logged in", func() {
		BeforeEach(func() {
			fakeActor.CloudControllerAPIVersionReturns(ccversion.MinVersionIsolationSegmentV3)
			fakeConfig.CurrentUserReturns(configv3.User{Name: "admin"}, nil)
			cmd.RequiredArgs.OrganizationName = org
			cmd.RequiredArgs.IsolationSegmentName = isolationSegment
		})

		It("Outputs a mesaage", func() {
			Expect(testUI.Out).To(Say("Removing entitlement to isolation segment segment1 from org org1 as admin..."))
		})

		When("revoking is successful", func() {
			BeforeEach(func() {
				fakeActor.DeleteIsolationSegmentOrganizationByNameReturns(v3action.Warnings{"warning 1", "warning 2"}, nil)
			})

			It("Isolation segnment is revoked from org", func() {
				Expect(executeErr).ToNot(HaveOccurred())

				Expect(testUI.Out).To(Say("OK"))
				Expect(testUI.Err).To(Say("warning 1"))
				Expect(testUI.Err).To(Say("warning 2"))

				Expect(fakeActor.DeleteIsolationSegmentOrganizationByNameCallCount()).To(Equal(1))
				actualIsolationSegmentName, actualOrgName := fakeActor.DeleteIsolationSegmentOrganizationByNameArgsForCall(0)
				Expect(actualIsolationSegmentName).To(Equal(isolationSegment))
				Expect(actualOrgName).To(Equal(org))
			})
		})

		Context("generic error while revoking segment isolation", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("ZOMG")
				fakeActor.DeleteIsolationSegmentOrganizationByNameReturns(v3action.Warnings{"warning 1", "warning 2"}, expectedErr)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(expectedErr))
				Expect(testUI.Err).To(Say("warning 1"))
				Expect(testUI.Err).To(Say("warning 2"))
			})
		})
	})
})
