package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Python Buildpack", func() {
	var app *cutlass.App

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("uncached buildpack", func() {
		Context("pushing a Python 3 app with a runtime.txt", func() {
			Context("including flask", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
			})

			Context("including django with specified python version", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "django_python_3"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.5"))
					Expect(app.Stdout.String()).To(ContainSubstring("collectstatic --noinput"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
				})
			})

		})

		Context("pushing a Python app without the runtime.txt", func() {
			Context("including django but not specified python version", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "django_web_app"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))
					Expect(app.Stdout.String()).To(ContainSubstring("collectstatic --noinput"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
				})
			})

			Context("including flask without a vendor directory", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_not_vendored"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
				AssertUsesProxyDuringStagingIfPresent("flask_not_vendored")
			})

			Context("with mercurial dependencies", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "mercurial"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.Stdout.String()).To(ContainSubstring("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
			})
		})
	})
})
