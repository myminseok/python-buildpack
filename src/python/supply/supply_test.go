package supply_test

import (
	"bytes"
	"io/ioutil"
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
		depsIdx      string
		depDir       string
		supplier     *supply.Supplier
		logger       *libbuildpack.Logger
		buffer       *bytes.Buffer
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
		mockStager   *MockStager
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err := ioutil.TempDir("", "python-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "13"
		depDir = filepath.Join(depsDir, depsIdx)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockStager = NewMockStager(mockCtrl)
		mockStager.EXPECT().BuildDir().AnyTimes().Return(buildDir)
		mockStager.EXPECT().DepDir().AnyTimes().Return(depDir)

		// args := []string{buildDir, "", depsDir, depsIdx}
		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		supplier = &supply.Supplier{
			Manifest: mockManifest,
			Stager:   mockStager,
			Log:      logger,
		}
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

				// Expect(buffer.String()).To(ContainSubstring("asdfg"))
			})
		})
	})
})
