package ccv3_test

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"

	. "code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
)

var _ = Describe("Task", func() {
	var client *Client

	BeforeEach(func() {
		client, _ = NewTestClient()
	})

	Describe("GetDeployments", func() {
		var (
			deployments []Deployment
			warnings    Warnings
			executeErr  error
		)

		JustBeforeEach(func() {
			deployments, warnings, executeErr = client.GetDeployments(Query{Key: AppGUIDFilter, Values: []string{"some-app-guid"}}, Query{Key: OrderBy, Values: []string{"-created_at"}}, Query{Key: PerPage, Values: []string{"1"}})
		})

		var response string
		var response2 string
		BeforeEach(func() {
			response = fmt.Sprintf(`{
	"pagination": {
		"next": {
			"href": "%s/v3/deployments?app_guids=some-app-guid&order_by=-created_at&page=2&per_page=1"
		}
	},
	"resources": [
		{
      		"guid": "newest-deployment-guid",
      		"created_at": "2018-05-25T22:42:10Z"
   	}
	]
}`, server.URL())
			response2 = `{
  	"pagination": {
		"next": null
	},
	"resources": [
		{
      		"guid": "oldest-deployment-guid",
      		"created_at": "2018-04-25T22:42:10Z"
   	}
	]
}`
		})

		Context("when the deployment exists", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/deployments", "app_guids=some-app-guid&order_by=-created_at&per_page=1"),
						RespondWith(http.StatusAccepted, response),
					),
				)

				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/deployments", "app_guids=some-app-guid&order_by=-created_at&page=2&per_page=1"),
						RespondWith(http.StatusAccepted, response2, http.Header{"X-Cf-Warnings": {"warning"}}),
					),
				)

			})

			It("returns the deployment guid of the most recent deployment", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(warnings).To(ConsistOf("warning"))
				Expect(deployments).To(ConsistOf(
					Deployment{GUID: "newest-deployment-guid", CreatedAt: "2018-05-25T22:42:10Z"},
					Deployment{GUID: "oldest-deployment-guid", CreatedAt: "2018-04-25T22:42:10Z"},
				))
			})
		})
	})

	Describe("CancelDeployment", func() {
		var (
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			warnings, executeErr = client.CancelDeployment("some-deployment-guid")
		})

		Context("when the deployment exists", func() {
			Context("when cancelling the deployment succeeds", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						CombineHandlers(
							VerifyRequest(http.MethodPost, "/v3/deployments/some-deployment-guid/actions/cancel"),
							RespondWith(http.StatusAccepted, "", http.Header{"X-Cf-Warnings": {"warning"}}),
						),
					)
				})

				It("cancels the deployment with no errors and returns all warnings", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning"))
				})
			})
		})
	})

	Describe("CreateApplicationDeployment", func() {
		var (
			deploymentGUID string
			warnings       Warnings
			executeErr     error
			dropletGUID    string
		)

		JustBeforeEach(func() {
			deploymentGUID, warnings, executeErr = client.CreateApplicationDeployment("some-app-guid", dropletGUID)
		})

		Context("when the application exists", func() {
			var response string
			BeforeEach(func() {
				dropletGUID = "some-droplet-guid"
				response = `{
  "guid": "some-deployment-guid",
  "created_at": "2018-04-25T22:42:10Z",
  "relationships": {
    "app": {
      "data": {
        "guid": "some-app-guid"
      }
    }
  }
}`
			})

			Context("when creating the deployment succeeds", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						CombineHandlers(
							VerifyRequest(http.MethodPost, "/v3/deployments"),
							VerifyJSON(`{"droplet":{ "guid":"some-droplet-guid" }, "relationships":{"app":{"data":{"guid":"some-app-guid"}}}}`),
							RespondWith(http.StatusAccepted, response, http.Header{"X-Cf-Warnings": {"warning"}}),
						),
					)
				})

				It("creates the deployment with no errors and returns all warnings", func() {
					Expect(deploymentGUID).To(Equal("some-deployment-guid"))
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning"))
				})

			})

			Context("when no droplet guid is provided", func() {
				BeforeEach(func() {
					dropletGUID = ""
					server.AppendHandlers(
						CombineHandlers(
							VerifyRequest(http.MethodPost, "/v3/deployments"),
							VerifyJSON(`{"relationships":{"app":{"data":{"guid":"some-app-guid"}}}}`),
							RespondWith(http.StatusAccepted, response, http.Header{"X-Cf-Warnings": {"warning"}}),
						),
					)
				})

				It("omits the droplet object in the JSON", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning"))
				})
			})
		})
	})

	Describe("GetDeployment", func() {
		var response string
		Context("When the deployments exists", func() {
			BeforeEach(func() {
				response = `{
				    "guid": "some-deployment-guid",
					"state": "DEPLOYING",
					"droplet": {
 					  "guid": "some-droplet-guid"
					},
 					"previous_droplet": {
 					  "guid": "some-other-droplet-guid"
 					},
 					"created_at": "some-time",
 					"updated_at": "some-later-time",
 					"relationships": {
 					  "app": {
 					    "data": {
 					      "guid": "some-app-guid"
 					    }
 					  }
 					}
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/deployments/some-deployment-guid"),
						RespondWith(http.StatusAccepted, response, http.Header{"X-Cf-Warnings": {"warning"}}),
					),
				)
			})
			It("Successfully returns a deployment object", func() {
				deployment, warnings, err := client.GetDeployment("some-deployment-guid")
				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).To(ConsistOf("warning"))
				Expect(deployment).To(Not(BeNil()))
				Expect(deployment.GUID).To(Equal("some-deployment-guid"))
				Expect(deployment.State).To(Equal(constant.DeploymentDeploying))
			})
		})

		Context("when the deployment doesn't exist", func() {
			BeforeEach(func() {
				response := `{
					"errors": [
						{
							"code": 10010,
							"detail": "Deployment not found",
							"title": "CF-ResourceNotFound"
						}
					]
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/deployments/not-a-deployment"),
						RespondWith(http.StatusNotFound, response, http.Header{"X-Cf-Warnings": {"warning-deployment"}}),
					),
				)
			})

			It("returns the error", func() {
				_, warnings, err := client.GetDeployment("not-a-deployment")
				Expect(err).To(MatchError(ccerror.DeploymentNotFoundError{}))
				Expect(warnings).To(ConsistOf("warning-deployment"))
			})
		})
	})
})
