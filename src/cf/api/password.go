package api

import "cf/configuration"


type PasswordRepository interface {
	GetScore(password string) string
	UpdatePassword(old string, new string) error
}

type CloudControllerPasswordRepository struct {
	config    *configuration.Configuration
	apiClient ApiClient
}

func NewCloudControllerPasswordRepository(config *configuration.Configuration, apiClient ApiClient) (repo CloudControllerPasswordRepository) {
	repo.config = config
	repo.apiClient = apiClient
	return
}

func (repo CloudControllerPasswordRepository) GetScore(password string) (score string) {
	return
}

func (repo CloudControllerPasswordRepository) UpdatePassword(old string, new string) (err error) {
	return
}
