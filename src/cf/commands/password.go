package commands

import (
	term "cf/terminal"
	"cf/requirements"
	"github.com/codegangsta/cli"
	"cf/api"
)

type Password struct {
	ui            term.UI
	pwdRepo api.PasswordRepository
}

func NewPassword(ui term.UI, pwdRepo api.PasswordRepository) (cmd Password) {
	cmd.ui = ui
	cmd.pwdRepo = pwdRepo
	return
}

func (cmd Password) GetRequirements(reqFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	reqs = []requirements.Requirement{
		reqFactory.NewLoginRequirement(),
	}
	return
}

func (cmd Password) Run(c *cli.Context) {
	oldPassword := cmd.ui.Ask("Current Password%s", term.Cyan(">"))
	newPassword := cmd.ui.Ask("New Password%s", term.Cyan(">"))
	cmd.ui.Ask("Verify Password%s", term.Cyan(">"))

	score := cmd.pwdRepo.GetScore(newPassword)
	cmd.ui.Say("Your password strength is: %s", score)

	cmd.ui.Say("Changing password...")
	err := cmd.pwdRepo.UpdatePassword(oldPassword, newPassword)

	if err != nil {
		cmd.ui.Failed("Error changing password", err)
		return
	}

	cmd.ui.Ok()
}
