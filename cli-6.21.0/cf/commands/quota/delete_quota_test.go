package quota_test

import (
	"github.com/cloudfoundry/cli/cf/api/quotas/quotasfakes"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/requirements/requirementsfakes"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"

	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("delete-quota command", func() {
	var (
		ui                  *testterm.FakeUI
		quotaRepo           *quotasfakes.FakeQuotaRepository
		requirementsFactory *requirementsfakes.FakeFactory
		configRepo          coreconfig.Repository
		deps                commandregistry.Dependency
	)

	updateCommandDependency := func(pluginCall bool) {
		deps.UI = ui
		deps.Config = configRepo
		deps.RepoLocator = deps.RepoLocator.SetQuotaRepository(quotaRepo)
		commandregistry.Commands.SetCommand(commandregistry.Commands.FindCommand("delete-quota").SetDependency(deps, pluginCall))
	}

	BeforeEach(func() {
		ui = &testterm.FakeUI{}
		configRepo = testconfig.NewRepositoryWithDefaults()
		quotaRepo = new(quotasfakes.FakeQuotaRepository)
		requirementsFactory = new(requirementsfakes.FakeFactory)
	})

	runCommand := func(args ...string) bool {
		return testcmd.RunCLICommand("delete-quota", args, requirementsFactory, updateCommandDependency, false, ui)
	}

	Context("when the user is not logged in", func() {
		BeforeEach(func() {
			requirementsFactory.NewLoginRequirementReturns(requirements.Failing{Message: "not logged in"})
		})

		It("fails requirements", func() {
			Expect(runCommand("my-quota")).To(BeFalse())
		})
	})

	Context("when the user is logged in", func() {
		BeforeEach(func() {
			requirementsFactory.NewLoginRequirementReturns(requirements.Passing{})
		})

		It("fails requirements when called without a quota name", func() {
			runCommand()
			Expect(ui.Outputs()).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires an argument"},
			))
		})

		Context("When the quota provided exists", func() {
			BeforeEach(func() {
				quota := models.QuotaFields{}
				quota.Name = "my-quota"
				quota.GUID = "my-quota-guid"

				quotaRepo.FindByNameReturns(quota, nil)
			})

			It("deletes a quota with a given name when the user confirms", func() {
				ui.Inputs = []string{"y"}

				runCommand("my-quota")
				Expect(quotaRepo.DeleteArgsForCall(0)).To(Equal("my-quota-guid"))

				Expect(ui.Prompts).To(ContainSubstrings(
					[]string{"Really delete the quota", "my-quota"},
				))

				Expect(ui.Outputs()).To(ContainSubstrings(
					[]string{"Deleting quota", "my-quota", "my-user"},
					[]string{"OK"},
				))
			})

			It("does not prompt when the -f flag is provided", func() {
				runCommand("-f", "my-quota")

				Expect(quotaRepo.DeleteArgsForCall(0)).To(Equal("my-quota-guid"))

				Expect(ui.Prompts).To(BeEmpty())
			})

			It("shows an error when deletion fails", func() {
				quotaRepo.DeleteReturns(errors.New("some error"))

				runCommand("-f", "my-quota")

				Expect(ui.Outputs()).To(ContainSubstrings(
					[]string{"Deleting", "my-quota"},
					[]string{"FAILED"},
				))
			})
		})

		Context("when finding the quota fails", func() {
			Context("when the quota provided does not exist", func() {
				BeforeEach(func() {
					quotaRepo.FindByNameReturns(models.QuotaFields{}, errors.NewModelNotFoundError("Quota", "non-existent-quota"))
				})

				It("warns the user when that the quota does not exist", func() {
					runCommand("-f", "non-existent-quota")

					Expect(ui.Outputs()).To(ContainSubstrings(
						[]string{"Deleting", "non-existent-quota"},
						[]string{"OK"},
					))

					Expect(ui.WarnOutputs).To(ContainSubstrings(
						[]string{"non-existent-quota", "does not exist"},
					))
				})
			})

			Context("when other types of error occur", func() {
				BeforeEach(func() {
					quotaRepo.FindByNameReturns(models.QuotaFields{}, errors.New("some error"))
				})

				It("shows an error", func() {
					runCommand("-f", "my-quota")

					Expect(ui.WarnOutputs).ToNot(ContainSubstrings(
						[]string{"my-quota", "does not exist"},
					))

					Expect(ui.Outputs()).To(ContainSubstrings(
						[]string{"FAILED"},
					))

				})
			})
		})
	})
})
