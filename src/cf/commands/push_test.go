package commands_test

import (
	"bytes"
	"cf"
	"cf/api"
	. "cf/commands"
	"github.com/stretchr/testify/assert"
	"os"
	"testhelpers"
	"testing"
)

type FakeAppStarter struct {
	StartedApp cf.Application
}

func (starter *FakeAppStarter) ApplicationStart(app cf.Application) {
	starter.StartedApp = app
}

func TestPushingRequirements(t *testing.T) {
	fakeUI := new(testhelpers.FakeUI)
	starter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}
	appRepo := &testhelpers.FakeApplicationRepository{}
	domainRepo := &testhelpers.FakeDomainRepository{}
	routeRepo := &testhelpers.FakeRouteRepository{}
	stackRepo := &testhelpers.FakeStackRepository{}

	cmd := NewPush(fakeUI, starter, zipper, appRepo, domainRepo, routeRepo, stackRepo)
	ctxt := testhelpers.NewContext("push", []string{})

	reqFactory := &testhelpers.FakeReqFactory{LoginSuccess: true, SpaceSuccess: true}
	testhelpers.RunCommand(cmd, ctxt, reqFactory)
	assert.True(t, testhelpers.CommandDidPassRequirements)

	reqFactory = &testhelpers.FakeReqFactory{LoginSuccess: false, SpaceSuccess: true}
	testhelpers.RunCommand(cmd, ctxt, reqFactory)
	assert.False(t, testhelpers.CommandDidPassRequirements)

	testhelpers.CommandDidPassRequirements = true

	reqFactory = &testhelpers.FakeReqFactory{LoginSuccess: true, SpaceSuccess: false}
	testhelpers.RunCommand(cmd, ctxt, reqFactory)
	assert.False(t, testhelpers.CommandDidPassRequirements)
}

func TestPushingAppWhenItDoesNotExist(t *testing.T) {
	domains := []cf.Domain{
		cf.Domain{Name: "foo.cf-app.com", Guid: "foo-domain-guid"},
	}
	domainRepo := &testhelpers.FakeDomainRepository{FindByNameDomain: domains[0]}
	routeRepo := &testhelpers.FakeRouteRepository{FindByHostErr: true}
	appRepo := &testhelpers.FakeApplicationRepository{AppByNameErr: true}
	stackRepo := &testhelpers.FakeStackRepository{}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}

	fakeUI := callPush([]string{"--name", "my-new-app"}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Contains(t, fakeUI.Outputs[0], "Creating my-new-app...")
	assert.Equal(t, appRepo.CreatedApp.Name, "my-new-app")
	assert.Equal(t, appRepo.CreatedApp.Instances, 1)
	assert.Equal(t, appRepo.CreatedApp.Memory, 128)
	assert.Equal(t, appRepo.CreatedApp.BuildpackUrl, "")
	assert.Contains(t, fakeUI.Outputs[1], "OK")

	assert.Contains(t, fakeUI.Outputs[2], "Creating route my-new-app.foo.cf-app.com...")
	assert.Equal(t, routeRepo.FindByHostHost, "my-new-app")
	assert.Equal(t, routeRepo.CreatedRoute.Host, "my-new-app")
	assert.Equal(t, routeRepo.CreatedRouteDomain.Guid, "foo-domain-guid")
	assert.Contains(t, fakeUI.Outputs[3], "OK")

	assert.Contains(t, fakeUI.Outputs[4], "Binding my-new-app.foo.cf-app.com to my-new-app...")
	assert.Equal(t, routeRepo.BoundApp.Name, "my-new-app")
	assert.Equal(t, routeRepo.BoundRoute.Host, "my-new-app")
	assert.Contains(t, fakeUI.Outputs[5], "OK")

	expectedAppDir, err := os.Getwd()
	assert.NoError(t, err)

	assert.Contains(t, fakeUI.Outputs[6], "Uploading my-new-app...")
	assert.Equal(t, appRepo.UploadedApp.Guid, "my-new-app-guid")
	assert.Equal(t, zipper.ZippedDir, expectedAppDir)
	assert.Contains(t, fakeUI.Outputs[7], "OK")

	assert.Equal(t, fakeStarter.StartedApp.Name, "my-new-app")
}

func TestPushingAppWhenItDoesNotExistButRouteExists(t *testing.T) {
	domains := []cf.Domain{
		cf.Domain{Name: "foo.cf-app.com", Guid: "foo-domain-guid"},
	}
	route := cf.Route{Host: "my-new-app"}
	domainRepo := &testhelpers.FakeDomainRepository{FindByNameDomain: domains[0]}
	routeRepo := &testhelpers.FakeRouteRepository{FindByHostRoute: route}
	appRepo := &testhelpers.FakeApplicationRepository{AppByNameErr: true}
	stackRepo := &testhelpers.FakeStackRepository{}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}

	fakeUI := callPush([]string{"--name", "my-new-app"}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Contains(t, fakeUI.Outputs[0], "Creating my-new-app...")
	assert.Equal(t, appRepo.CreatedApp.Name, "my-new-app")
	assert.Equal(t, appRepo.CreatedApp.Instances, 1)
	assert.Equal(t, appRepo.CreatedApp.Memory, 128)
	assert.Equal(t, appRepo.CreatedApp.BuildpackUrl, "")
	assert.Contains(t, fakeUI.Outputs[1], "OK")

	assert.Contains(t, fakeUI.Outputs[2], "Using route my-new-app.foo.cf-app.com")
	assert.Equal(t, routeRepo.FindByHostHost, "my-new-app")

	assert.Contains(t, fakeUI.Outputs[3], "Binding my-new-app.foo.cf-app.com to my-new-app...")
	assert.Equal(t, routeRepo.BoundApp.Name, "my-new-app")
	assert.Equal(t, routeRepo.BoundRoute.Host, "my-new-app")
	assert.Contains(t, fakeUI.Outputs[4], "OK")

	expectedAppDir, err := os.Getwd()
	assert.NoError(t, err)

	assert.Contains(t, fakeUI.Outputs[5], "Uploading my-new-app...")
	assert.Equal(t, appRepo.UploadedApp.Guid, "my-new-app-guid")
	assert.Equal(t, zipper.ZippedDir, expectedAppDir)
	assert.Contains(t, fakeUI.Outputs[6], "OK")

	assert.Equal(t, fakeStarter.StartedApp.Name, "my-new-app")
}

func TestPushingAppWithCustomFlags(t *testing.T) {
	domain := cf.Domain{Name: "bar.cf-app.com", Guid: "bar-domain-guid"}
	domainRepo := &testhelpers.FakeDomainRepository{FindByNameDomain: domain}
	routeRepo := &testhelpers.FakeRouteRepository{FindByHostErr: true}
	appRepo := &testhelpers.FakeApplicationRepository{AppByNameErr: true}
	stackRepo := &testhelpers.FakeStackRepository{FindByNameStack: cf.Stack{Name: "customLinux", Guid: "custom-linux-guid"}}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{ZippedBuffer: bytes.NewBufferString("Zip File!")}

	fakeUI := callPush([]string{
		"--name", "my-new-app",
		"--domain", "bar.cf-app.com",
		"--host", "my-hostname",
		"--instances", "3",
		"--memory", "2G",
		"--buildpack", "https://github.com/heroku/heroku-buildpack-play.git",
		"--path", "/Users/pivotal/workspace/my-new-app",
		"--stack", "customLinux",
		"--no-start", "",
	}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Contains(t, fakeUI.Outputs[0], "Using stack customLinux.")
	assert.Equal(t, stackRepo.FindByNameName, "customLinux")

	assert.Contains(t, fakeUI.Outputs[1], "Creating my-new-app...")
	assert.Equal(t, appRepo.CreatedApp.Name, "my-new-app")
	assert.Equal(t, appRepo.CreatedApp.Instances, 3)
	assert.Equal(t, appRepo.CreatedApp.Memory, 2048)
	assert.Equal(t, appRepo.CreatedApp.Stack.Guid, "custom-linux-guid")
	assert.Equal(t, appRepo.CreatedApp.BuildpackUrl, "https://github.com/heroku/heroku-buildpack-play.git")
	assert.Contains(t, fakeUI.Outputs[2], "OK")

	assert.Contains(t, fakeUI.Outputs[3], "Creating route my-hostname.bar.cf-app.com...")
	assert.Equal(t, domainRepo.FindByNameName, "bar.cf-app.com")
	assert.Equal(t, routeRepo.CreatedRoute.Host, "my-hostname")
	assert.Equal(t, routeRepo.CreatedRouteDomain.Guid, "bar-domain-guid")
	assert.Contains(t, fakeUI.Outputs[4], "OK")

	assert.Contains(t, fakeUI.Outputs[5], "Binding my-hostname.bar.cf-app.com to my-new-app...")
	assert.Equal(t, routeRepo.BoundApp.Name, "my-new-app")
	assert.Equal(t, routeRepo.BoundRoute.Host, "my-hostname")
	assert.Contains(t, fakeUI.Outputs[6], "OK")

	assert.Contains(t, fakeUI.Outputs[7], "Uploading my-new-app...")
	assert.Equal(t, appRepo.UploadedApp.Guid, "my-new-app-guid")
	assert.Equal(t, zipper.ZippedDir, "/Users/pivotal/workspace/my-new-app")
	assert.Equal(t, appRepo.UploadedZipBuffer, zipper.ZippedBuffer)
	assert.Contains(t, fakeUI.Outputs[8], "OK")

	assert.Equal(t, fakeStarter.StartedApp.Name, "")
}

func TestPushingAppWithMemoryInMegaBytes(t *testing.T) {
	domain := cf.Domain{Name: "bar.cf-app.com", Guid: "bar-domain-guid"}
	domainRepo := &testhelpers.FakeDomainRepository{FindByNameDomain: domain}
	routeRepo := &testhelpers.FakeRouteRepository{}
	appRepo := &testhelpers.FakeApplicationRepository{AppByNameErr: true}
	stackRepo := &testhelpers.FakeStackRepository{}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}

	callPush([]string{
		"--name", "my-new-app",
		"--memory", "256M",
	}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Equal(t, appRepo.CreatedApp.Memory, 256)
}

func TestPushingAppWithMemoryWithoutUnit(t *testing.T) {
	domain := cf.Domain{Name: "bar.cf-app.com", Guid: "bar-domain-guid"}
	domainRepo := &testhelpers.FakeDomainRepository{FindByNameDomain: domain}
	routeRepo := &testhelpers.FakeRouteRepository{}
	appRepo := &testhelpers.FakeApplicationRepository{AppByNameErr: true}
	stackRepo := &testhelpers.FakeStackRepository{}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}

	callPush([]string{
		"--name", "my-new-app",
		"--memory", "512",
	}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Equal(t, appRepo.CreatedApp.Memory, 512)
}

func TestPushingAppWithInvalidMemory(t *testing.T) {
	domain := cf.Domain{Name: "bar.cf-app.com", Guid: "bar-domain-guid"}
	domainRepo := &testhelpers.FakeDomainRepository{FindByNameDomain: domain}
	routeRepo := &testhelpers.FakeRouteRepository{}
	appRepo := &testhelpers.FakeApplicationRepository{AppByNameErr: true}
	stackRepo := &testhelpers.FakeStackRepository{}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}

	callPush([]string{
		"--name", "my-new-app",
		"--memory", "abcM",
	}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Equal(t, appRepo.CreatedApp.Memory, 128)
}

func TestPushingAppWhenItAlreadyExists(t *testing.T) {
	domainRepo := &testhelpers.FakeDomainRepository{}
	routeRepo := &testhelpers.FakeRouteRepository{}
	existingApp := cf.Application{Name: "existing-app", Guid: "existing-app-guid"}
	appRepo := &testhelpers.FakeApplicationRepository{AppByName: existingApp}
	stackRepo := &testhelpers.FakeStackRepository{}
	fakeStarter := &FakeAppStarter{}
	zipper := &testhelpers.FakeZipper{}

	fakeUI := callPush([]string{"--name", "existing-app"}, fakeStarter, zipper, appRepo, domainRepo, routeRepo, stackRepo)

	assert.Contains(t, fakeUI.Outputs[0], "Uploading existing-app...")
	assert.Equal(t, appRepo.UploadedApp.Guid, "existing-app-guid")
	assert.Contains(t, fakeUI.Outputs[1], "OK")
}

func callPush(args []string,
	starter ApplicationStarter,
	zipper cf.Zipper,
	appRepo api.ApplicationRepository,
	domainRepo api.DomainRepository,
	routeRepo api.RouteRepository,
	stackRepo api.StackRepository) (fakeUI *testhelpers.FakeUI) {

	fakeUI = new(testhelpers.FakeUI)
	ctxt := testhelpers.NewContext("push", args)
	cmd := NewPush(fakeUI, starter, zipper, appRepo, domainRepo, routeRepo, stackRepo)
	reqFactory := &testhelpers.FakeReqFactory{LoginSuccess: true, SpaceSuccess: true}
	testhelpers.RunCommand(cmd, ctxt, reqFactory)

	return
}
