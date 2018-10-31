package v6_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/flag"
	. "code.cloudfoundry.org/cli/command/v6"
	"code.cloudfoundry.org/cli/command/v6/v6fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("bind-security-group Command", func() {
	var (
		cmd             BindSecurityGroupCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v6fakes.FakeBindSecurityGroupActor
		binaryName      string
		executeErr      error
	)

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeBindSecurityGroupActor)

		cmd = BindSecurityGroupCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)

		// Stubs for the happy path.
		cmd.RequiredArgs.SecurityGroupName = "some-security-group"
		cmd.RequiredArgs.OrganizationName = "some-org"

		fakeConfig.CurrentUserReturns(
			configv3.User{Name: "some-user"},
			nil)
		fakeActor.GetSecurityGroupByNameReturns(
			v2action.SecurityGroup{Name: "some-security-group", GUID: "some-security-group-guid"},
			v2action.Warnings{"get security group warning"},
			nil)
		fakeActor.GetOrganizationByNameReturns(
			v2action.Organization{Name: "some-org", GUID: "some-org-guid"},
			v2action.Warnings{"get org warning"},
			nil)
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	When("lifecycle is 'running'", func() {
		BeforeEach(func() {
			cmd.Lifecycle = flag.SecurityGroupLifecycle(constant.SecurityGroupLifecycleRunning)
		})

		When("checking target fails", func() {
			BeforeEach(func() {
				fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{BinaryName: binaryName})
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: "faceman"}))

				Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
				checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
				Expect(checkTargetedOrg).To(BeFalse())
				Expect(checkTargetedSpace).To(BeFalse())
			})
		})

		When("getting the current user returns an error", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("getting current user error")
				fakeConfig.CurrentUserReturns(
					configv3.User{},
					expectedErr)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(expectedErr))
			})
		})

		When("the provided security group does not exist", func() {
			BeforeEach(func() {
				fakeActor.GetSecurityGroupByNameReturns(
					v2action.SecurityGroup{},
					v2action.Warnings{"get security group warning"},
					actionerror.SecurityGroupNotFoundError{Name: "some-security-group"})
			})

			It("returns a SecurityGroupNotFoundError and displays all warnings", func() {
				Expect(executeErr).To(MatchError(actionerror.SecurityGroupNotFoundError{Name: "some-security-group"}))
				Expect(testUI.Err).To(Say("get security group warning"))
			})
		})

		When("an error is encountered getting the provided security group", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("get security group error")
				fakeActor.GetSecurityGroupByNameReturns(
					v2action.SecurityGroup{},
					v2action.Warnings{"get security group warning"},
					expectedErr)
			})

			It("returns the error and displays all warnings", func() {
				Expect(executeErr).To(MatchError(expectedErr))
				Expect(testUI.Err).To(Say("get security group warning"))
			})
		})

		When("the provided org does not exist", func() {
			BeforeEach(func() {
				fakeActor.GetOrganizationByNameReturns(
					v2action.Organization{},
					v2action.Warnings{"get organization warning"},
					actionerror.OrganizationNotFoundError{Name: "some-org"})
			})

			It("returns an OrganizationNotFoundError and displays all warnings", func() {
				Expect(executeErr).To(MatchError(actionerror.OrganizationNotFoundError{Name: "some-org"}))
				Expect(testUI.Err).To(Say("get security group warning"))
				Expect(testUI.Err).To(Say("get organization warning"))
			})
		})

		When("an error is encountered getting the provided org", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("get org error")
				fakeActor.GetOrganizationByNameReturns(
					v2action.Organization{},
					v2action.Warnings{"get org warning"},
					expectedErr)
			})

			It("returns the error and displays all warnings", func() {
				Expect(executeErr).To(MatchError(expectedErr))
				Expect(testUI.Err).To(Say("get security group warning"))
				Expect(testUI.Err).To(Say("get org warning"))
			})
		})

		When("a space is provided", func() {
			BeforeEach(func() {
				cmd.RequiredArgs.SpaceName = "some-space"
			})

			When("the space does not exist", func() {
				BeforeEach(func() {
					fakeActor.GetSpaceByOrganizationAndNameReturns(
						v2action.Space{},
						v2action.Warnings{"get space warning"},
						actionerror.SpaceNotFoundError{Name: "some-space"})
				})

				It("returns a SpaceNotFoundError", func() {
					Expect(executeErr).To(MatchError(actionerror.SpaceNotFoundError{Name: "some-space"}))
					Expect(testUI.Err).To(Say("get security group warning"))
					Expect(testUI.Err).To(Say("get org warning"))
					Expect(testUI.Err).To(Say("get space warning"))
				})
			})

			When("the space exists", func() {
				BeforeEach(func() {
					fakeActor.GetSpaceByOrganizationAndNameReturns(
						v2action.Space{
							GUID: "some-space-guid",
							Name: "some-space",
						},
						v2action.Warnings{"get space by org warning"},
						nil)
				})

				When("no errors are encountered binding the security group to the space", func() {
					BeforeEach(func() {
						fakeActor.BindSecurityGroupToSpaceReturns(
							v2action.Warnings{"bind security group to space warning"},
							nil)
					})

					It("binds the security group to the space and displays all warnings", func() {
						Expect(executeErr).NotTo(HaveOccurred())

						Expect(testUI.Out).To(Say(`Assigning security group some-security-group to space some-space in org some-org as some-user\.\.\.`))
						Expect(testUI.Out).To(Say("OK"))
						Expect(testUI.Out).To(Say(`TIP: Changes require an app restart \(for running\) or restage \(for staging\) to apply to existing applications\.`))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get space by org warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning"))

						Expect(fakeActor.CloudControllerAPIVersionCallCount()).To(Equal(0))

						Expect(fakeActor.GetSecurityGroupByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetSecurityGroupByNameArgsForCall(0)).To(Equal("some-security-group"))

						Expect(fakeActor.GetOrganizationByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetOrganizationByNameArgsForCall(0)).To(Equal("some-org"))

						Expect(fakeActor.GetSpaceByOrganizationAndNameCallCount()).To(Equal(1))
						orgGUID, spaceName := fakeActor.GetSpaceByOrganizationAndNameArgsForCall(0)
						Expect(orgGUID).To(Equal("some-org-guid"))
						Expect(spaceName).To(Equal("some-space"))

						Expect(fakeActor.BindSecurityGroupToSpaceCallCount()).To(Equal(1))
						securityGroupGUID, spaceGUID, lifecycle := fakeActor.BindSecurityGroupToSpaceArgsForCall(0)
						Expect(securityGroupGUID).To(Equal("some-security-group-guid"))
						Expect(spaceGUID).To(Equal("some-space-guid"))
						Expect(lifecycle).To(Equal(constant.SecurityGroupLifecycleRunning))
					})
				})

				When("an error is encountered binding the security group to the space", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("bind error")
						fakeActor.BindSecurityGroupToSpaceReturns(
							v2action.Warnings{"bind security group to space warning"},
							expectedErr)
					})

					It("returns the error and displays all warnings", func() {
						Expect(executeErr).To(MatchError(expectedErr))

						Expect(testUI.Out).NotTo(Say("OK"))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get space by org warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning"))
					})
				})
			})

			When("an error is encountered getting the space", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("get org error")
					fakeActor.GetSpaceByOrganizationAndNameReturns(
						v2action.Space{},
						v2action.Warnings{"get space by org warning"},
						expectedErr)
				})

				It("returns the error and displays all warnings", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(testUI.Err).To(Say("get security group warning"))
					Expect(testUI.Err).To(Say("get org warning"))
					Expect(testUI.Err).To(Say("get space by org warning"))
				})
			})
		})

		When("a space is not provided", func() {
			When("there are no spaces in the org", func() {
				BeforeEach(func() {
					fakeActor.GetOrganizationSpacesReturns(
						[]v2action.Space{},
						v2action.Warnings{"get org spaces warning"},
						nil)
				})

				It("does not perform any bindings and displays all warnings", func() {
					Expect(executeErr).NotTo(HaveOccurred())

					Expect(testUI.Out).NotTo(Say("Assigning security group"))
					Expect(testUI.Out).NotTo(Say("OK"))

					Expect(testUI.Err).To(Say("get security group warning"))
					Expect(testUI.Err).To(Say("get org warning"))
					Expect(testUI.Err).To(Say("get org spaces warning"))
				})
			})

			When("there are spaces in the org", func() {
				BeforeEach(func() {
					fakeActor.GetOrganizationSpacesReturns(
						[]v2action.Space{
							{
								GUID: "some-space-guid-1",
								Name: "some-space-1",
							},
							{
								GUID: "some-space-guid-2",
								Name: "some-space-2",
							},
						},
						v2action.Warnings{"get org spaces warning"},
						nil)
				})

				When("no errors are encountered binding the security group to the spaces", func() {
					BeforeEach(func() {
						fakeActor.BindSecurityGroupToSpaceReturnsOnCall(
							0,
							v2action.Warnings{"bind security group to space warning 1"},
							nil)
						fakeActor.BindSecurityGroupToSpaceReturnsOnCall(
							1,
							v2action.Warnings{"bind security group to space warning 2"},
							nil)
					})

					It("binds the security group to each space and displays all warnings", func() {
						Expect(executeErr).NotTo(HaveOccurred())

						Expect(testUI.Out).To(Say(`Assigning security group some-security-group to space some-space-1 in org some-org as some-user\.\.\.`))
						Expect(testUI.Out).To(Say("OK"))
						Expect(testUI.Out).To(Say(`Assigning security group some-security-group to space some-space-2 in org some-org as some-user\.\.\.`))
						Expect(testUI.Out).To(Say("OK"))
						Expect(testUI.Out).To(Say(`TIP: Changes require an app restart \(for running\) or restage \(for staging\) to apply to existing applications\.`))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get org spaces warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning 1"))
						Expect(testUI.Err).To(Say("bind security group to space warning 2"))

						Expect(fakeActor.GetSecurityGroupByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetSecurityGroupByNameArgsForCall(0)).To(Equal("some-security-group"))

						Expect(fakeActor.GetOrganizationByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetOrganizationByNameArgsForCall(0)).To(Equal("some-org"))

						Expect(fakeActor.BindSecurityGroupToSpaceCallCount()).To(Equal(2))
						securityGroupGUID, spaceGUID, lifecycle := fakeActor.BindSecurityGroupToSpaceArgsForCall(0)
						Expect(securityGroupGUID).To(Equal("some-security-group-guid"))
						Expect(spaceGUID).To(Equal("some-space-guid-1"))
						Expect(lifecycle).To(Equal(constant.SecurityGroupLifecycleRunning))
						securityGroupGUID, spaceGUID, lifecycle = fakeActor.BindSecurityGroupToSpaceArgsForCall(1)
						Expect(securityGroupGUID).To(Equal("some-security-group-guid"))
						Expect(spaceGUID).To(Equal("some-space-guid-2"))
						Expect(lifecycle).To(Equal(constant.SecurityGroupLifecycleRunning))
					})
				})

				When("an error is encountered binding the security group to a space", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("bind security group to space error")
						fakeActor.BindSecurityGroupToSpaceReturns(
							v2action.Warnings{"bind security group to space warning"},
							expectedErr)
					})

					It("returns the error and displays all warnings", func() {
						Expect(executeErr).To(MatchError(expectedErr))

						Expect(testUI.Out).NotTo(Say("OK"))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get org spaces warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning"))
					})
				})
			})

			When("an error is encountered getting spaces in the org", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("get org spaces error")
					fakeActor.GetOrganizationSpacesReturns(
						nil,
						v2action.Warnings{"get org spaces warning"},
						expectedErr)
				})

				It("returns the error and displays all warnings", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(testUI.Err).To(Say("get security group warning"))
					Expect(testUI.Err).To(Say("get org warning"))
					Expect(testUI.Err).To(Say("get org spaces warning"))
				})
			})
		})
	})

	When("lifecycle is 'staging'", func() {
		BeforeEach(func() {
			cmd.Lifecycle = "staging"
		})

		When("a space is provided", func() {
			BeforeEach(func() {
				cmd.RequiredArgs.SpaceName = "some-space"
			})

			When("the space exists", func() {
				BeforeEach(func() {
					fakeActor.GetSpaceByOrganizationAndNameReturns(
						v2action.Space{
							GUID: "some-space-guid",
							Name: "some-space",
						},
						v2action.Warnings{"get space by org warning"},
						nil)
				})

				When("no errors are encountered binding the security group to the space", func() {
					BeforeEach(func() {
						fakeActor.BindSecurityGroupToSpaceReturns(
							v2action.Warnings{"bind security group to space warning"},
							nil)
					})

					It("binds the security group to the space and displays all warnings", func() {
						Expect(executeErr).NotTo(HaveOccurred())

						Expect(testUI.Out).To(Say(`Assigning security group some-security-group to space some-space in org some-org as some-user\.\.\.`))
						Expect(testUI.Out).To(Say("OK"))
						Expect(testUI.Out).To(Say(`TIP: Changes require an app restart \(for running\) or restage \(for staging\) to apply to existing applications\.`))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get space by org warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning"))

						Expect(fakeActor.CloudControllerAPIVersionCallCount()).To(Equal(0))
						Expect(fakeActor.GetSecurityGroupByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetSecurityGroupByNameArgsForCall(0)).To(Equal("some-security-group"))

						Expect(fakeActor.GetOrganizationByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetOrganizationByNameArgsForCall(0)).To(Equal("some-org"))

						Expect(fakeActor.GetSpaceByOrganizationAndNameCallCount()).To(Equal(1))
						orgGUID, spaceName := fakeActor.GetSpaceByOrganizationAndNameArgsForCall(0)
						Expect(orgGUID).To(Equal("some-org-guid"))
						Expect(spaceName).To(Equal("some-space"))

						Expect(fakeActor.BindSecurityGroupToSpaceCallCount()).To(Equal(1))
						securityGroupGUID, spaceGUID, lifecycle := fakeActor.BindSecurityGroupToSpaceArgsForCall(0)
						Expect(securityGroupGUID).To(Equal("some-security-group-guid"))
						Expect(spaceGUID).To(Equal("some-space-guid"))
						Expect(lifecycle).To(Equal(constant.SecurityGroupLifecycleStaging))
					})
				})
			})
		})

		When("a space is not provided", func() {
			When("there are no spaces in the org", func() {
				BeforeEach(func() {
					fakeActor.GetOrganizationSpacesReturns(
						[]v2action.Space{},
						v2action.Warnings{"get org spaces warning"},
						nil)
				})

				It("does not perform any bindings and displays all warnings", func() {
					Expect(executeErr).NotTo(HaveOccurred())

					Expect(testUI.Out).NotTo(Say("Assigning security group"))
					Expect(testUI.Out).NotTo(Say("OK"))

					Expect(testUI.Err).To(Say("get security group warning"))
					Expect(testUI.Err).To(Say("get org warning"))
					Expect(testUI.Err).To(Say("get org spaces warning"))
				})
			})

			When("there are spaces in the org", func() {
				BeforeEach(func() {
					fakeActor.GetOrganizationSpacesReturns(
						[]v2action.Space{
							{
								GUID: "some-space-guid-1",
								Name: "some-space-1",
							},
							{
								GUID: "some-space-guid-2",
								Name: "some-space-2",
							},
						},
						v2action.Warnings{"get org spaces warning"},
						nil)
				})

				When("no errors are encountered binding the security group to the spaces", func() {
					BeforeEach(func() {
						fakeActor.BindSecurityGroupToSpaceReturnsOnCall(
							0,
							v2action.Warnings{"bind security group to space warning 1"},
							nil)
						fakeActor.BindSecurityGroupToSpaceReturnsOnCall(
							1,
							v2action.Warnings{"bind security group to space warning 2"},
							nil)
					})

					It("binds the security group to each space and displays all warnings", func() {
						Expect(executeErr).NotTo(HaveOccurred())

						Expect(testUI.Out).To(Say(`Assigning security group some-security-group to space some-space-1 in org some-org as some-user\.\.\.`))
						Expect(testUI.Out).To(Say("OK"))
						Expect(testUI.Out).To(Say(`Assigning security group some-security-group to space some-space-2 in org some-org as some-user\.\.\.`))
						Expect(testUI.Out).To(Say("OK"))
						Expect(testUI.Out).To(Say(`TIP: Changes require an app restart \(for running\) or restage \(for staging\) to apply to existing applications\.`))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get org spaces warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning 1"))
						Expect(testUI.Err).To(Say("bind security group to space warning 2"))

						Expect(fakeActor.GetSecurityGroupByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetSecurityGroupByNameArgsForCall(0)).To(Equal("some-security-group"))

						Expect(fakeActor.GetOrganizationByNameCallCount()).To(Equal(1))
						Expect(fakeActor.GetOrganizationByNameArgsForCall(0)).To(Equal("some-org"))

						Expect(fakeActor.BindSecurityGroupToSpaceCallCount()).To(Equal(2))
						securityGroupGUID, spaceGUID, lifecycle := fakeActor.BindSecurityGroupToSpaceArgsForCall(0)
						Expect(securityGroupGUID).To(Equal("some-security-group-guid"))
						Expect(spaceGUID).To(Equal("some-space-guid-1"))
						Expect(lifecycle).To(Equal(constant.SecurityGroupLifecycleStaging))
						securityGroupGUID, spaceGUID, lifecycle = fakeActor.BindSecurityGroupToSpaceArgsForCall(1)
						Expect(securityGroupGUID).To(Equal("some-security-group-guid"))
						Expect(spaceGUID).To(Equal("some-space-guid-2"))
						Expect(lifecycle).To(Equal(constant.SecurityGroupLifecycleStaging))
					})
				})

				When("an error is encountered binding the security group to a space", func() {
					var expectedErr error

					BeforeEach(func() {
						expectedErr = errors.New("bind security group to space error")
						fakeActor.BindSecurityGroupToSpaceReturns(
							v2action.Warnings{"bind security group to space warning"},
							expectedErr)
					})

					It("returns the error and displays all warnings", func() {
						Expect(executeErr).To(MatchError(expectedErr))

						Expect(testUI.Out).NotTo(Say("OK"))

						Expect(testUI.Err).To(Say("get security group warning"))
						Expect(testUI.Err).To(Say("get org warning"))
						Expect(testUI.Err).To(Say("get org spaces warning"))
						Expect(testUI.Err).To(Say("bind security group to space warning"))
					})
				})
			})

			When("an error is encountered getting spaces in the org", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("get org spaces error")
					fakeActor.GetOrganizationSpacesReturns(
						nil,
						v2action.Warnings{"get org spaces warning"},
						expectedErr)
				})

				It("returns the error and displays all warnings", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(testUI.Err).To(Say("get security group warning"))
					Expect(testUI.Err).To(Say("get org warning"))
					Expect(testUI.Err).To(Say("get org spaces warning"))
				})
			})
		})
	})
})
