package isolated

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccversion"
	"code.cloudfoundry.org/cli/integration/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/ghttp"
)

var _ = Describe("delete-isolation-segment command", func() {
	var isolationSegmentName string

	BeforeEach(func() {
		helpers.SkipIfVersionLessThan(ccversion.MinVersionIsolationSegmentV3)

		isolationSegmentName = helpers.NewIsolationSegmentName()
	})

	Describe("help", func() {
		When("--help flag is set", func() {
			It("Displays command usage to output", func() {
				session := helpers.CF("delete-isolation-segment", "--help")
				Eventually(session).Should(Say("NAME:"))
				Eventually(session).Should(Say("delete-isolation-segment - Delete an isolation segment"))
				Eventually(session).Should(Say("USAGE:"))
				Eventually(session).Should(Say("cf delete-isolation-segment SEGMENT_NAME"))
				Eventually(session).Should(Say("SEE ALSO:"))
				Eventually(session).Should(Say("disable-org-isolation, isolation-segments"))
				Eventually(session).Should(Exit(0))
			})
		})
	})

	When("the environment is not setup correctly", func() {
		It("fails with the appropriate errors", func() {
			helpers.CheckEnvironmentTargetedCorrectly(false, false, ReadOnlyOrg, "delete-isolation-segment", "isolation-segment-name")
		})

		When("the v3 api version is lower than the minimum version", func() {
			var server *Server

			BeforeEach(func() {
				server = helpers.StartAndTargetServerWithAPIVersions(helpers.DefaultV2Version, ccversion.MinV3ClientVersion)
			})

			AfterEach(func() {
				server.Close()
			})

			It("fails with error message that the minimum version is not met", func() {
				session := helpers.CF("delete-isolation-segment", isolationSegmentName)
				Eventually(session).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say(`This command requires CF API version 3\.11\.0 or higher\.`))
				Eventually(session).Should(Exit(1))
			})
		})
	})

	When("the environment is set up correctly", func() {
		BeforeEach(func() {
			helpers.LoginCF()
		})

		When("the isolation segment exists", func() {
			BeforeEach(func() {
				Eventually(helpers.CF("create-isolation-segment", isolationSegmentName)).Should(Exit(0))
			})

			When("passed the force flag", func() {
				It("deletes the isolation segment", func() {
					session := helpers.CF("delete-isolation-segment", "-f", isolationSegmentName)
					userName, _ := helpers.GetCredentials()
					Eventually(session).Should(Say("Deleting isolation segment %s as %s...", isolationSegmentName, userName))
					Eventually(session).Should(Say("OK"))
					Eventually(session).Should(Exit(0))
				})
			})

			When("the force flag is not provided", func() {
				var buffer *Buffer

				BeforeEach(func() {
					buffer = NewBuffer()
				})

				When("'yes' is inputted", func() {
					BeforeEach(func() {
						buffer.Write([]byte("y\n"))
					})

					It("deletes the isolation segment", func() {
						session := helpers.CFWithStdin(buffer, "delete-isolation-segment", isolationSegmentName)
						Eventually(session).Should(Say(`Really delete the isolation segment %s\?`, isolationSegmentName))

						userName, _ := helpers.GetCredentials()
						Eventually(session).Should(Say("Deleting isolation segment %s as %s...", isolationSegmentName, userName))
						Eventually(session).Should(Say("OK"))
						Eventually(session).Should(Exit(0))
					})
				})

				When("'no' is inputted", func() {
					BeforeEach(func() {
						buffer.Write([]byte("n\n"))
					})

					It("cancels the deletion", func() {
						session := helpers.CFWithStdin(buffer, "delete-isolation-segment", isolationSegmentName)
						Eventually(session).Should(Say(`Really delete the isolation segment %s\?`, isolationSegmentName))
						Eventually(session).Should(Say("Delete cancelled"))
						Eventually(session).Should(Exit(0))
					})
				})

				When("using the default value", func() {
					BeforeEach(func() {
						buffer.Write([]byte("\n"))
					})

					It("cancels the deletion", func() {
						session := helpers.CFWithStdin(buffer, "delete-isolation-segment", isolationSegmentName)
						Eventually(session).Should(Say(`Really delete the isolation segment %s\?`, isolationSegmentName))
						Eventually(session).Should(Say("Delete cancelled"))
						Eventually(session).Should(Exit(0))
					})
				})
			})
		})

		When("the isolation segment does not exist", func() {
			It("returns an ok and warning", func() {
				session := helpers.CF("delete-isolation-segment", "-f", isolationSegmentName)
				Eventually(session.Err).Should(Say("Isolation segment %s does not exist.", isolationSegmentName))
				Eventually(session).Should(Say("OK"))
				Eventually(session).Should(Exit(0))
			})
		})
	})
})
