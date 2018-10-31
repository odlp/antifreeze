package v2action_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	. "code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v2action/v2actionfakes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2/constant"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service Instance Actions", func() {
	var (
		actor                     *Actor
		fakeCloudControllerClient *v2actionfakes.FakeCloudControllerClient
	)

	BeforeEach(func() {
		fakeCloudControllerClient = new(v2actionfakes.FakeCloudControllerClient)
		actor = NewActor(fakeCloudControllerClient, nil, nil)
	})

	Describe("ServiceInstance", func() {
		DescribeTable("IsManaged",
			func(iType constant.ServiceInstanceType, expected bool) {
				Expect(ServiceInstance{Type: iType}.IsManaged()).To(Equal(expected))
			},

			Entry("return true for managed service", constant.ServiceInstanceTypeManagedService, true),
			Entry("return false for any other type of service", constant.ServiceInstanceTypeUserProvidedService, false),
		)

		DescribeTable("IsUserProvided",
			func(iType constant.ServiceInstanceType, expected bool) {
				Expect(ServiceInstance{Type: iType}.IsUserProvided()).To(Equal(expected))
			},

			Entry("return true for UserProvidedService service", constant.ServiceInstanceTypeUserProvidedService, true),
			Entry("return false for any other type of service", constant.ServiceInstanceTypeManagedService, false),
		)
	})

	Describe("GetServiceInstance", func() {
		var (
			serviceInstanceGUID string

			serviceInstance ServiceInstance
			warnings        Warnings
			executeErr      error
		)

		BeforeEach(func() {
			serviceInstanceGUID = "service-instance-guid"
		})

		JustBeforeEach(func() {
			serviceInstance, warnings, executeErr = actor.GetServiceInstance(serviceInstanceGUID)
		})

		When("the service instance exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetServiceInstanceReturns(ccv2.ServiceInstance{Name: "some-service-instance", GUID: "service-instance-guid"}, ccv2.Warnings{"service-instance-warnings"}, nil)
			})

			It("returns the service instance and warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(serviceInstance).To(Equal(ServiceInstance{
					GUID: "service-instance-guid",
					Name: "some-service-instance",
				}))
				Expect(warnings).To(Equal(Warnings{"service-instance-warnings"}))

				Expect(fakeCloudControllerClient.GetServiceInstanceCallCount()).To(Equal(1))
				Expect(fakeCloudControllerClient.GetServiceInstanceArgsForCall(0)).To(Equal(serviceInstanceGUID))
			})
		})

		When("the service instance does not exist", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetServiceInstanceReturns(ccv2.ServiceInstance{}, ccv2.Warnings{"service-instance-warnings-1"}, ccerror.ResourceNotFoundError{})
			})

			It("returns errors and warnings", func() {
				Expect(executeErr).To(MatchError(actionerror.ServiceInstanceNotFoundError{GUID: serviceInstanceGUID}))
				Expect(warnings).To(ConsistOf("service-instance-warnings-1"))
			})
		})

		When("retrieving the application's bound services returns an error", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("this is indeed an error, kudos!")
				fakeCloudControllerClient.GetServiceInstanceReturns(ccv2.ServiceInstance{}, ccv2.Warnings{"service-instance-warnings-1"}, expectedErr)
			})

			It("returns errors and warnings", func() {
				Expect(executeErr).To(MatchError(expectedErr))
				Expect(warnings).To(ConsistOf("service-instance-warnings-1"))
			})
		})
	})

	Describe("GetServiceInstanceByNameAndSpace", func() {
		When("the service instance exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetSpaceServiceInstancesReturns(
					[]ccv2.ServiceInstance{
						{
							GUID: "some-service-instance-guid",
							Name: "some-service-instance",
						},
					},
					ccv2.Warnings{"foo"},
					nil,
				)
			})

			It("returns the service instance and warnings", func() {
				serviceInstance, warnings, err := actor.GetServiceInstanceByNameAndSpace("some-service-instance", "some-space-guid")
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstance).To(Equal(ServiceInstance{
					GUID: "some-service-instance-guid",
					Name: "some-service-instance",
				}))
				Expect(warnings).To(Equal(Warnings{"foo"}))

				Expect(fakeCloudControllerClient.GetSpaceServiceInstancesCallCount()).To(Equal(1))

				spaceGUID, includeUserProvidedServices, queries := fakeCloudControllerClient.GetSpaceServiceInstancesArgsForCall(0)
				Expect(spaceGUID).To(Equal("some-space-guid"))
				Expect(includeUserProvidedServices).To(BeTrue())
				Expect(queries).To(ConsistOf([]ccv2.Filter{
					ccv2.Filter{
						Type:     constant.NameFilter,
						Operator: constant.EqualOperator,
						Values:   []string{"some-service-instance"},
					},
				}))
			})
		})

		When("the service instance does not exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetSpaceServiceInstancesReturns([]ccv2.ServiceInstance{}, nil, nil)
			})

			It("returns a ServiceInstanceNotFoundError", func() {
				_, _, err := actor.GetServiceInstanceByNameAndSpace("some-service-instance", "some-space-guid")
				Expect(err).To(MatchError(actionerror.ServiceInstanceNotFoundError{Name: "some-service-instance"}))
			})
		})

		When("the cloud controller client returns an error", func() {
			var expectedError error

			BeforeEach(func() {
				expectedError = errors.New("I am a CloudControllerClient Error")
				fakeCloudControllerClient.GetSpaceServiceInstancesReturns([]ccv2.ServiceInstance{}, nil, expectedError)
			})

			It("returns the error", func() {
				_, _, err := actor.GetServiceInstanceByNameAndSpace("some-service-instance", "some-space-guid")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("GetServiceInstancesByApplication", func() {
		var (
			appGUID string

			serviceInstances []ServiceInstance
			warnings         Warnings
			executeErr       error
		)

		BeforeEach(func() {
			appGUID = "some-app-guid"
		})

		JustBeforeEach(func() {
			serviceInstances, warnings, executeErr = actor.GetServiceInstancesByApplication(appGUID)
		})

		When("the application has services bound", func() {
			var serviceBindings []ccv2.ServiceBinding

			BeforeEach(func() {
				serviceBindings = []ccv2.ServiceBinding{
					{ServiceInstanceGUID: "service-instance-guid-1"},
					{ServiceInstanceGUID: "service-instance-guid-2"},
					{ServiceInstanceGUID: "service-instance-guid-3"},
				}

				fakeCloudControllerClient.GetServiceBindingsReturns(serviceBindings, ccv2.Warnings{"service-bindings-warnings-1", "service-bindings-warnings-2"}, nil)
			})

			When("retrieving the service instances is successful", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetServiceInstanceReturnsOnCall(0, ccv2.ServiceInstance{Name: "some-service-instance-1"}, ccv2.Warnings{"service-instance-warnings-1"}, nil)
					fakeCloudControllerClient.GetServiceInstanceReturnsOnCall(1, ccv2.ServiceInstance{Name: "some-service-instance-2"}, ccv2.Warnings{"service-instance-warnings-2"}, nil)
					fakeCloudControllerClient.GetServiceInstanceReturnsOnCall(2, ccv2.ServiceInstance{Name: "some-service-instance-3"}, ccv2.Warnings{"service-instance-warnings-3"}, nil)
				})

				It("returns the service instances and warnings", func() {
					Expect(executeErr).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("service-bindings-warnings-1", "service-bindings-warnings-2", "service-instance-warnings-1", "service-instance-warnings-2", "service-instance-warnings-3"))
					Expect(serviceInstances).To(ConsistOf(
						ServiceInstance{Name: "some-service-instance-1"},
						ServiceInstance{Name: "some-service-instance-2"},
						ServiceInstance{Name: "some-service-instance-3"},
					))

					Expect(fakeCloudControllerClient.GetServiceInstanceCallCount()).To(Equal(3))
					Expect(fakeCloudControllerClient.GetServiceInstanceArgsForCall(0)).To(Equal("service-instance-guid-1"))
					Expect(fakeCloudControllerClient.GetServiceInstanceArgsForCall(1)).To(Equal("service-instance-guid-2"))
					Expect(fakeCloudControllerClient.GetServiceInstanceArgsForCall(2)).To(Equal("service-instance-guid-3"))
				})
			})

			When("retrieving the service instances returns an error", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("this is indeed an error, kudos!")
					fakeCloudControllerClient.GetServiceInstanceReturns(ccv2.ServiceInstance{}, ccv2.Warnings{"service-instance-warnings-1", "service-instance-warnings-2"}, expectedErr)
				})

				It("returns errors and warnings", func() {
					Expect(executeErr).To(MatchError(expectedErr))
					Expect(warnings).To(ConsistOf("service-bindings-warnings-1", "service-bindings-warnings-2", "service-instance-warnings-1", "service-instance-warnings-2"))
				})
			})
		})

		When("the application has no services bound", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetServiceBindingsReturns(nil, ccv2.Warnings{"service-bindings-warnings-1", "service-bindings-warnings-2"}, nil)
			})

			It("returns an empty list and warnings", func() {
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(warnings).To(ConsistOf("service-bindings-warnings-1", "service-bindings-warnings-2"))
				Expect(serviceInstances).To(BeEmpty())
			})
		})

		When("retrieving the application's bound services returns an error", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("this is indeed an error, kudos!")
				fakeCloudControllerClient.GetServiceBindingsReturns(nil, ccv2.Warnings{"service-bindings-warnings-1", "service-bindings-warnings-2"}, expectedErr)
			})

			It("returns errors and warnings", func() {
				Expect(executeErr).To(MatchError(expectedErr))
				Expect(warnings).To(ConsistOf("service-bindings-warnings-1", "service-bindings-warnings-2"))
			})
		})
	})

	Describe("GetServiceInstancesBySpace", func() {
		When("there are service instances", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetSpaceServiceInstancesReturns(
					[]ccv2.ServiceInstance{
						{
							GUID: "some-service-instance-guid-1",
							Name: "some-service-instance-1",
						},
						{
							GUID: "some-service-instance-guid-2",
							Name: "some-service-instance-2",
						},
					},
					ccv2.Warnings{"warning-1", "warning-2"},
					nil,
				)
			})

			It("returns the service instances and warnings", func() {
				serviceInstances, warnings, err := actor.GetServiceInstancesBySpace("some-space-guid")
				Expect(err).ToNot(HaveOccurred())
				Expect(serviceInstances).To(ConsistOf(
					ServiceInstance{
						GUID: "some-service-instance-guid-1",
						Name: "some-service-instance-1",
					},
					ServiceInstance{
						GUID: "some-service-instance-guid-2",
						Name: "some-service-instance-2",
					},
				))
				Expect(warnings).To(ConsistOf("warning-1", "warning-2"))

				Expect(fakeCloudControllerClient.GetSpaceServiceInstancesCallCount()).To(Equal(1))

				spaceGUID, includeUserProvidedServices, queries := fakeCloudControllerClient.GetSpaceServiceInstancesArgsForCall(0)
				Expect(spaceGUID).To(Equal("some-space-guid"))
				Expect(includeUserProvidedServices).To(BeTrue())
				Expect(queries).To(BeNil())
			})
		})

		When("the cloud controller client returns an error", func() {
			var expectedError error

			BeforeEach(func() {
				expectedError = errors.New("I am a CloudControllerClient Error")
				fakeCloudControllerClient.GetSpaceServiceInstancesReturns(
					[]ccv2.ServiceInstance{},
					ccv2.Warnings{"warning-1", "warning-2"},
					expectedError)
			})

			It("returns the error and warnings", func() {
				_, warnings, err := actor.GetServiceInstancesBySpace("some-space-guid")
				Expect(err).To(MatchError(expectedError))
				Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
			})
		})
	})
})
