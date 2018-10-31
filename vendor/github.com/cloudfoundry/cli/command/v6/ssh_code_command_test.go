package v6_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/command/commandfakes"
	. "code.cloudfoundry.org/cli/command/v6"
	"code.cloudfoundry.org/cli/command/v6/v6fakes"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("ssh-code Command", func() {
	var (
		cmd             SSHCodeCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v6fakes.FakeSSHCodeActor
		binaryName      string
		executeErr      error
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeSSHCodeActor)

		cmd = SSHCodeCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
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
			Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))

			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			targetedOrganizationRequired, targetedSpaceRequired := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(targetedOrganizationRequired).To(Equal(false))
			Expect(targetedSpaceRequired).To(Equal(false))
		})
	})

	When("the user is logged in", func() {
		var code string

		BeforeEach(func() {
			code = "s3curep4ss"
			fakeActor.GetSSHPasscodeReturns(code, nil)
		})

		It("displays the ssh code", func() {
			Expect(executeErr).NotTo(HaveOccurred())
			Expect(testUI.Out).To(Say(code))
			Expect(fakeActor.GetSSHPasscodeCallCount()).To(Equal(1))
		})

		When("an error is encountered getting the ssh code", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("get ssh code error")
				fakeActor.GetSSHPasscodeReturns("", expectedErr)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(expectedErr))

				Expect(fakeActor.GetSSHPasscodeCallCount()).To(Equal(1))
			})
		})
	})
})
