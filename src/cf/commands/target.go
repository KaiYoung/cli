package commands

import (
	"cf"
	"cf/api"
	"cf/configuration"
	"cf/requirements"
	term "cf/terminal"
	"github.com/codegangsta/cli"
)

type InfoResponse struct {
	ApiVersion            string `json:"api_version"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
}

type Target struct {
	ui         term.UI
	config     *configuration.Configuration
	configRepo configuration.ConfigurationRepository
	orgRepo    api.OrganizationRepository
	spaceRepo  api.SpaceRepository
}

func NewTarget(ui term.UI, configRepo configuration.ConfigurationRepository, orgRepo api.OrganizationRepository, spaceRepo api.SpaceRepository) (t Target) {
	t.ui = ui
	t.configRepo = configRepo
	t.config, _ = configRepo.Get()
	t.orgRepo = orgRepo
	t.spaceRepo = spaceRepo

	return
}

func (cmd Target) GetRequirements(reqFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	return
}

func (t Target) Run(c *cli.Context) {
	argsCount := len(c.Args())
	orgName := c.String("o")
	spaceName := c.String("s")

	if argsCount == 0 && orgName == "" && spaceName == "" {
		t.ui.ShowConfiguration(t.config)

		if !t.config.IsLoggedIn() {
			return
		}
		if !t.config.HasOrganization() {
			t.ui.Say("No org targeted. Use 'cf target -o' to target an org.")
		}
		if !t.config.HasSpace() {
			t.ui.Say("No space targeted. Use 'cf target -s' to target a space.")
		}
		return
	}

	if argsCount > 0 {
		t.setNewTarget(c.Args()[0])
		return
	}

	if orgName != "" {
		t.setOrganization(orgName)
		if t.config.IsLoggedIn() {
			t.ui.Say("No space targeted. Use 'cf target -s' to target a space.")
		}
		return
	}

	if spaceName != "" {
		t.setSpace(spaceName)
		return
	}

	return
}

func (t Target) setNewTarget(target string) {
	t.ui.Say("Setting target to %s...", term.Yellow(target))

	request, err := api.NewRequest("GET", target+"/v2/info", "", nil)

	if err != nil {
		t.ui.Failed("URL invalid.", err)
		return
	}

	scheme := request.URL.Scheme
	if scheme != "http" && scheme != "https" {
		t.ui.Failed("API Endpoints should start with https:// or http://", nil)
		return
	}

	serverResponse := new(InfoResponse)
	_, err = api.PerformRequestAndParseResponse(request, &serverResponse)

	if err != nil {
		t.ui.Failed("", err)
		return
	}

	err = t.saveTarget(target, serverResponse)

	if err != nil {
		t.ui.Failed("Error saving configuration", err)
		return
	}

	t.ui.Ok()

	if scheme == "http" {
		t.ui.Say(term.Magenta("\nWarning: Insecure http API Endpoint detected. Secure https API Endpoints are recommended.\n"))
	}
	t.ui.ShowConfiguration(t.config)
}

func (t *Target) saveTarget(target string, info *InfoResponse) (err error) {
	t.configRepo.ClearSession()
	t.config.Target = target
	t.config.ApiVersion = info.ApiVersion
	t.config.AuthorizationEndpoint = info.AuthorizationEndpoint
	return t.configRepo.Save()
}

func (t Target) setOrganization(orgName string) {
	if !t.config.IsLoggedIn() {
		t.ui.Failed("You must be logged in to set an organization. Use 'cf login'.", nil)
		return
	}

	org, err := t.orgRepo.FindByName(orgName)
	if err != nil {
		t.ui.Failed("Could not set organization.", nil)
		return
	}

	t.config.Organization = org
	t.config.Space = cf.Space{}
	t.saveAndShowConfig()
}

func (t Target) setSpace(spaceName string) {
	if !t.config.IsLoggedIn() {
		t.ui.Failed("You must be logged in to set a space. Use 'cf login'.", nil)
		return
	}

	if !t.config.HasOrganization() {
		t.ui.Failed("Organization must be set before targeting space.", nil)
		return
	}

	space, err := t.spaceRepo.FindByName(spaceName)
	if err != nil {
		t.ui.Failed("You do not have access to that space.", nil)
		return
	}

	t.config.Space = space
	t.saveAndShowConfig()
}

func (t Target) saveAndShowConfig() {
	err := t.configRepo.Save()
	if err != nil {
		t.ui.Failed("Error saving configuration", err)
		return
	}
	t.ui.ShowConfiguration(t.config)
}
