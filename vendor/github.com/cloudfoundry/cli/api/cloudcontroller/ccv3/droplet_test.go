package ccv3_test

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	. "code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
)

var _ = Describe("Droplet", func() {
	var client *Client

	BeforeEach(func() {
		client, _ = NewTestClient()
	})

	Describe("GetApplicationDropletCurrent", func() {
		var (
			droplet    Droplet
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			droplet, warnings, executeErr = client.GetApplicationDropletCurrent("some-guid")
		})

		When("the request succeeds", func() {
			BeforeEach(func() {
				response := `{
					"guid": "some-guid",
					"state": "STAGED",
					"error": null,
					"lifecycle": {
						"type": "buildpack",
						"data": {}
					},
					"buildpacks": [
						{
							"name": "some-buildpack",
							"detect_output": "detected-buildpack"
						}
					],
					"image": "docker/some-image",
					"stack": "some-stack",
					"created_at": "2016-03-28T23:39:34Z",
					"updated_at": "2016-03-28T23:39:47Z"
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/apps/some-guid/droplets/current"),
						RespondWith(http.StatusOK, response, http.Header{"X-Cf-Warnings": {"warning-1"}}),
					),
				)
			})

			It("returns the given droplet and all warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())

				Expect(droplet).To(Equal(Droplet{
					GUID:  "some-guid",
					Stack: "some-stack",
					State: constant.DropletStaged,
					Buildpacks: []DropletBuildpack{
						{
							Name:         "some-buildpack",
							DetectOutput: "detected-buildpack",
						},
					},
					Image:     "docker/some-image",
					CreatedAt: "2016-03-28T23:39:34Z",
				}))
				Expect(warnings).To(ConsistOf("warning-1"))
			})
		})

		When("cloud controller returns an error", func() {
			BeforeEach(func() {
				response := `{
					"errors": [
						{
							"code": 10010,
							"detail": "Droplet not found",
							"title": "CF-ResourceNotFound"
						}
					]
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/apps/some-guid/droplets/current"),
						RespondWith(http.StatusNotFound, response, http.Header{"X-Cf-Warnings": {"warning-1"}}),
					),
				)
			})

			It("returns the error and all given warnings", func() {
				Expect(executeErr).To(MatchError(ccerror.DropletNotFoundError{}))
				Expect(warnings).To(ConsistOf("warning-1"))
			})
		})
	})

	Describe("GetDroplet", func() {
		var (
			droplet    Droplet
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			droplet, warnings, executeErr = client.GetDroplet("some-guid")
		})

		When("the request succeeds", func() {
			BeforeEach(func() {
				response := `{
					"guid": "some-guid",
					"state": "STAGED",
					"error": null,
					"lifecycle": {
						"type": "buildpack",
						"data": {}
					},
					"buildpacks": [
						{
							"name": "some-buildpack",
							"detect_output": "detected-buildpack"
						}
					],
					"image": "docker/some-image",
					"stack": "some-stack",
					"created_at": "2016-03-28T23:39:34Z",
					"updated_at": "2016-03-28T23:39:47Z"
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/droplets/some-guid"),
						RespondWith(http.StatusOK, response, http.Header{"X-Cf-Warnings": {"warning-1"}}),
					),
				)
			})

			It("returns the given droplet and all warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())

				Expect(droplet).To(Equal(Droplet{
					GUID:  "some-guid",
					Stack: "some-stack",
					State: constant.DropletStaged,
					Buildpacks: []DropletBuildpack{
						{
							Name:         "some-buildpack",
							DetectOutput: "detected-buildpack",
						},
					},
					Image:     "docker/some-image",
					CreatedAt: "2016-03-28T23:39:34Z",
				}))
				Expect(warnings).To(ConsistOf("warning-1"))
			})
		})

		When("cloud controller returns an error", func() {
			BeforeEach(func() {
				response := `{
					"errors": [
						{
							"code": 10010,
							"detail": "Droplet not found",
							"title": "CF-ResourceNotFound"
						}
					]
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/droplets/some-guid"),
						RespondWith(http.StatusNotFound, response, http.Header{"X-Cf-Warnings": {"warning-1"}}),
					),
				)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(ccerror.DropletNotFoundError{}))
				Expect(warnings).To(ConsistOf("warning-1"))
			})
		})
	})

	Describe("GetDroplets", func() {
		var (
			droplets   []Droplet
			warnings   Warnings
			executeErr error
		)

		JustBeforeEach(func() {
			droplets, warnings, executeErr = client.GetDroplets(
				Query{Key: AppGUIDFilter, Values: []string{"some-app-guid"}},
				Query{Key: PerPage, Values: []string{"2"}},
			)
		})

		When("the CC returns back droplets", func() {
			BeforeEach(func() {
				response1 := fmt.Sprintf(`{
					"pagination": {
						"next": {
							"href": "%s/v3/droplets?app_guids=some-app-guid&per_page=2&page=2"
						}
					},
					"resources": [
						{
							"guid": "some-guid-1",
							"stack": "some-stack-1",
							"buildpacks": [{
								"name": "some-buildpack-1",
								"detect_output": "detected-buildpack-1"
							}],
							"state": "STAGED",
							"created_at": "2017-08-16T00:18:24Z",
							"links": {
								"package": "https://api.com/v3/packages/some-package-guid"
							}
						},
						{
							"guid": "some-guid-2",
							"stack": "some-stack-2",
							"buildpacks": [{
								"name": "some-buildpack-2",
								"detect_output": "detected-buildpack-2"
							}],
							"state": "COPYING",
							"created_at": "2017-08-16T00:19:05Z"
						}
					]
				}`, server.URL())
				response2 := `{
					"pagination": {
						"next": null
					},
					"resources": [
						{
							"guid": "some-guid-3",
							"stack": "some-stack-3",
							"buildpacks": [{
								"name": "some-buildpack-3",
								"detect_output": "detected-buildpack-3"
							}],
							"state": "FAILED",
							"created_at": "2017-08-22T17:55:02Z"
						}
					]
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/droplets", "app_guids=some-app-guid&per_page=2"),
						RespondWith(http.StatusOK, response1, http.Header{"X-Cf-Warnings": {"warning-1"}}),
					),
				)
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/droplets", "app_guids=some-app-guid&per_page=2&page=2"),
						RespondWith(http.StatusOK, response2, http.Header{"X-Cf-Warnings": {"warning-2"}}),
					),
				)
			})

			It("returns the droplets and all warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(droplets).To(HaveLen(3))

				Expect(droplets[0]).To(Equal(Droplet{
					GUID:  "some-guid-1",
					Stack: "some-stack-1",
					State: constant.DropletStaged,
					Buildpacks: []DropletBuildpack{
						{
							Name:         "some-buildpack-1",
							DetectOutput: "detected-buildpack-1",
						},
					},
					CreatedAt: "2017-08-16T00:18:24Z",
				}))
				Expect(droplets[1]).To(Equal(Droplet{
					GUID:  "some-guid-2",
					Stack: "some-stack-2",
					State: constant.DropletCopying,
					Buildpacks: []DropletBuildpack{
						{
							Name:         "some-buildpack-2",
							DetectOutput: "detected-buildpack-2",
						},
					},
					CreatedAt: "2017-08-16T00:19:05Z",
				}))
				Expect(droplets[2]).To(Equal(Droplet{
					GUID:  "some-guid-3",
					Stack: "some-stack-3",
					State: constant.DropletFailed,
					Buildpacks: []DropletBuildpack{
						{
							Name:         "some-buildpack-3",
							DetectOutput: "detected-buildpack-3",
						},
					},
					CreatedAt: "2017-08-22T17:55:02Z",
				}))
				Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
			})
		})

		When("cloud controller returns an error", func() {
			BeforeEach(func() {
				response := `{
					"errors": [
						{
							"code": 10010,
							"detail": "App not found",
							"title": "CF-ResourceNotFound"
						}
					]
				}`
				server.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/v3/droplets", "app_guids=some-app-guid&per_page=2"),
						RespondWith(http.StatusNotFound, response, http.Header{"X-Cf-Warnings": {"warning-1"}}),
					),
				)
			})

			It("returns the error", func() {
				Expect(executeErr).To(MatchError(ccerror.ApplicationNotFoundError{}))
				Expect(warnings).To(ConsistOf("warning-1"))
			})
		})
	})
})
