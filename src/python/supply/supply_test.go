package supply_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"python/supply"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		err          error
		buildDir     string
		depsDir      string
		depsIdx      string
		depDir       string
		supplier     *supply.Supplier
		logger       *libbuildpack.Logger
		buffer       *bytes.Buffer
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
		mockStager   *MockStager
		mockCommand  *MockCommand
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "python-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "13"
		depDir = filepath.Join(depsDir, depsIdx)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockStager = NewMockStager(mockCtrl)
		mockStager.EXPECT().BuildDir().AnyTimes().Return(buildDir)
		mockStager.EXPECT().DepDir().AnyTimes().Return(depDir)
		mockCommand = NewMockCommand(mockCtrl)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		supplier = &supply.Supplier{
			Manifest: mockManifest,
			Stager:   mockStager,
			Command:  mockCommand,
			Log:      logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("InstallPython", func() {
		var pythonInstallDir string
		var versions []string

		BeforeEach(func() {
			pythonInstallDir = filepath.Join(depDir, "python")
			ioutil.WriteFile(filepath.Join(buildDir, "runtime.txt"), []byte("python-3.4.2"), 0644)

			versions = []string{"3.4.2"}
		})

		Context("runtime.txt sets Python version 3", func() {
			It("installs Python version 3", func() {
				mockManifest.EXPECT().AllDependencyVersions("python").Return(versions)
				mockManifest.EXPECT().InstallDependency(libbuildpack.Dependency{Name: "python", Version: "3.4.2"}, pythonInstallDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")
				Expect(supplier.InstallPython()).To(Succeed())
			})
		})
	})

	Describe("InstallPip", func() {
		It("Downloads and installs setuptools", func() {
			mockManifest.EXPECT().AllDependencyVersions("setuptools").Return([]string{"2.4.6"})
			mockManifest.EXPECT().InstallOnlyVersion("setuptools", "/tmp/setuptools")
			mockCommand.EXPECT().Output("/tmp/setuptools/setuptools-2.4.6", "python", "setup.py", "install", fmt.Sprintf("--prefix=%s/python", depDir)).Return("", nil)

			mockManifest.EXPECT().AllDependencyVersions("pip").Return([]string{"1.3.4"})
			mockManifest.EXPECT().InstallOnlyVersion("pip", "/tmp/pip")
			mockCommand.EXPECT().Output("/tmp/pip/pip-1.3.4", "python", "setup.py", "install", fmt.Sprintf("--prefix=%s/python", depDir)).Return("", nil)

			pythonInstallDir := filepath.Join(depDir, "python")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "include"), "include")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "pkgconfig"), "pkgconfig")

			Expect(supplier.InstallPip()).To(Succeed())
		})
	})
})
