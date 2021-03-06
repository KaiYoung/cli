package api

import (
	"bytes"
	"cf"
	"cf/configuration"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type ServiceRepository interface {
	GetServiceOfferings() (offerings []cf.ServiceOffering, err error)
	CreateServiceInstance(name string, plan cf.ServicePlan) (err error)
	CreateUserProvidedServiceInstance(name string, params map[string]string) (err error)
	FindInstanceByName(name string) (instance cf.ServiceInstance, err error)
	BindService(instance cf.ServiceInstance, app cf.Application) (errorCode int, err error)
	UnbindService(instance cf.ServiceInstance, app cf.Application) (err error)
	DeleteService(instance cf.ServiceInstance) (err error)
}

type CloudControllerServiceRepository struct {
	config    *configuration.Configuration
	apiClient ApiClient
}

func NewCloudControllerServiceRepository(config *configuration.Configuration, apiClient ApiClient) (repo CloudControllerServiceRepository) {
	repo.config = config
	repo.apiClient = apiClient
	return
}

func (repo CloudControllerServiceRepository) GetServiceOfferings() (offerings []cf.ServiceOffering, err error) {
	path := fmt.Sprintf("%s/v2/services?inline-relations-depth=1", repo.config.Target)
	request, err := NewRequest("GET", path, repo.config.AccessToken, nil)
	if err != nil {
		return
	}

	response := new(ServiceOfferingsApiResponse)

	_, err = repo.apiClient.PerformRequestAndParseResponse(request, response)

	if err != nil {
		return
	}

	for _, r := range response.Resources {
		plans := []cf.ServicePlan{}
		for _, p := range r.Entity.ServicePlans {
			plans = append(plans, cf.ServicePlan{Name: p.Entity.Name, Guid: p.Metadata.Guid})
		}
		offerings = append(offerings, cf.ServiceOffering{
			Label:       r.Entity.Label,
			Version:     r.Entity.Version,
			Provider:    r.Entity.Provider,
			Description: r.Entity.Description,
			Guid:        r.Metadata.Guid,
			Plans:       plans,
		})
	}

	return
}

func (repo CloudControllerServiceRepository) CreateServiceInstance(name string, plan cf.ServicePlan) (err error) {
	path := fmt.Sprintf("%s/v2/service_instances", repo.config.Target)

	data := fmt.Sprintf(
		`{"name":"%s","service_plan_guid":"%s","space_guid":"%s"}`,
		name, plan.Guid, repo.config.Space.Guid,
	)
	request, err := NewRequest("POST", path, repo.config.AccessToken, strings.NewReader(data))
	if err != nil {
		return
	}

	_, err = repo.apiClient.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) CreateUserProvidedServiceInstance(name string, params map[string]string) (err error) {
	path := fmt.Sprintf("%s/v2/user_provided_service_instances", repo.config.Target)

	type RequestBody struct {
		Name        string            `json:"name"`
		Credentials map[string]string `json:"credentials"`
		SpaceGuid   string            `json:"space_guid"`
	}

	reqBody := RequestBody{name, params, repo.config.Space.Guid}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	request, err := NewRequest("POST", path, repo.config.AccessToken, bytes.NewReader(jsonBytes))
	if err != nil {
		return
	}

	_, err = repo.apiClient.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) FindInstanceByName(name string) (instance cf.ServiceInstance, err error) {
	path := fmt.Sprintf("%s/v2/spaces/%s/service_instances?return_user_provided_service_instances=true&q=name%s&inline-relations-depth=1", repo.config.Target, repo.config.Space.Guid, "%3A"+name)
	request, err := NewRequest("GET", path, repo.config.AccessToken, nil)
	if err != nil {
		return
	}

	response := new(ServiceInstancesApiResponse)
	_, err = repo.apiClient.PerformRequestAndParseResponse(request, response)
	if err != nil {
		return
	}

	if len(response.Resources) == 0 {
		err = errors.New(fmt.Sprintf("Service %s not found", name))
		return
	}

	resource := response.Resources[0]
	instance.Guid = resource.Metadata.Guid
	instance.Name = resource.Entity.Name
	instance.ServiceBindings = []cf.ServiceBinding{}

	for _, bindingResource := range resource.Entity.ServiceBindings {
		newBinding := cf.ServiceBinding{
			Url:     bindingResource.Metadata.Url,
			Guid:    bindingResource.Metadata.Guid,
			AppGuid: bindingResource.Entity.AppGuid,
		}
		instance.ServiceBindings = append(instance.ServiceBindings, newBinding)
	}

	return
}

func (repo CloudControllerServiceRepository) BindService(instance cf.ServiceInstance, app cf.Application) (errorCode int, err error) {
	path := fmt.Sprintf("%s/v2/service_bindings", repo.config.Target)
	body := fmt.Sprintf(
		`{"app_guid":"%s","service_instance_guid":"%s"}`,
		app.Guid, instance.Guid,
	)
	request, err := NewRequest("POST", path, repo.config.AccessToken, strings.NewReader(body))
	if err != nil {
		return
	}

	errorCode, err = repo.apiClient.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) UnbindService(instance cf.ServiceInstance, app cf.Application) (err error) {
	var path string

	for _, binding := range instance.ServiceBindings {
		if binding.AppGuid == app.Guid {
			path = repo.config.Target + binding.Url
			break
		}
	}

	if path == "" {
		err = errors.New("Error finding service binding")
		return
	}

	request, err := NewRequest("DELETE", path, repo.config.AccessToken, nil)
	if err != nil {
		return
	}

	_, err = repo.apiClient.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) DeleteService(instance cf.ServiceInstance) (err error) {
	if len(instance.ServiceBindings) > 0 {
		return errors.New("Cannot delete service instance, apps are still bound to it")
	}

	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.Target, instance.Guid)
	request, err := NewRequest("DELETE", path, repo.config.AccessToken, nil)
	if err != nil {
		return
	}

	_, err = repo.apiClient.PerformRequest(request)
	return
}
