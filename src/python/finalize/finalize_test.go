package finalize_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"python/finalize"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=finalize.go --destination=mocks_test.go --package=finalize_test

var _ = Describe("Finalize", func() {
	var (
		err                error
		buildDir           string
		depsDir            string
		depsIdx            string
		finalizer          *finalize.Finalizer
		logger             *libbuildpack.Logger
		buffer             *bytes.Buffer
		mockCtrl           *gomock.Controller
		mockManifest       *MockManifest
		mockCommand        *MockCommand
		mockManagePyFinder *MockManagePyFinder
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "python-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "9"
		Expect(os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)).To(Succeed())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockCommand = NewMockCommand(mockCtrl)
		mockManagePyFinder = NewMockManagePyFinder(mockCtrl)

		args := []string{buildDir, "", depsDir, depsIdx}
		stager := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		finalizer = &finalize.Finalizer{
			Stager:         stager,
			Manifest:       mockManifest,
			Log:            logger,
			Command:        mockCommand,
			ManagePyFinder: mockManagePyFinder,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())

		os.Setenv("DISABLE_COLLECTSTATIC", "")
	})

	Describe("HandleCollectStatic", func() {
		Context("When DISABLE_COLLECTSTATIC is set", func() {
			BeforeEach(func() {
				os.Setenv("DISABLE_COLLECTSTATIC", "1")
			})
			It("does nothing", func() {
				Expect(finalizer.HandleCollectstatic()).To(Succeed())
			})
		})
		Context("When DISABLE_COLLECTSTATIC is not set", func() {
			BeforeEach(func() {
				os.Setenv("DISABLE_COLLECTSTATIC", "")
			})
			Context("app uses Django", func() {
				BeforeEach(func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "django", "Django").Return(nil)
					mockManagePyFinder.EXPECT().FindManagePy(buildDir).Return("/foo/bar/manage.py", nil)
				})

				It("runs collectstatic with the most top-level manage.py", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "/foo/bar/manage.py", "collectstatic", "--noinput", "--traceback").Return(nil)
					Expect(finalizer.HandleCollectstatic()).To(Succeed())
				})

				Context("when collectstatic fails", func() {
					It("prints an error", func() {
						mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "/foo/bar/manage.py", "collectstatic", "--noinput", "--traceback").Return(fmt.Errorf("oh no it failed"))
						Expect(finalizer.HandleCollectstatic()).NotTo(Succeed())
						Expect(buffer.String()).To(ContainSubstring(` !     Error while running '$ python /foo/bar/manage.py collectstatic --noinput'.`))
						Expect(buffer.String()).To(ContainSubstring(`     See traceback above for details.`))
						Expect(buffer.String()).To(ContainSubstring(`      You may need to update application code to resolve this error.`))
						Expect(buffer.String()).To(ContainSubstring(`      Or, you can disable collectstatic for this application:`))
						Expect(buffer.String()).To(ContainSubstring(`         $ cf set-env <app> DISABLE_COLLECTSTATIC 1`))
						Expect(buffer.String()).To(ContainSubstring(`      https://devcenter.heroku.com/articles/django-assets`))
					})
				})
			})

			Context("app does not use Django", func() {
				BeforeEach(func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "django", "Django").Return(fmt.Errorf("Not found"))
				})

				It("does not run anything", func() {
					Expect(finalizer.HandleCollectstatic()).To(Succeed())
				})
			})
		})
	})

})
