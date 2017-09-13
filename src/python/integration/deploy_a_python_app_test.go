package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Python Buildpack", func() {
	var app *cutlass.App

	Context("uncached buildpack", func() {
		Context("pushing a Python 3 app with a runtime.txt", func() {
			Context("that has dependencies", func() {
				Context("including flask", func() {
					BeforeEach(func() {
						app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3"))
						app.SetEnv("BP_DEBUG", "1")
					})

					AfterEach(func() {})

					It("deploys", func() {
						PushAppAndConfirm(app)
						Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					})
				})
			})
		})
	})
})
