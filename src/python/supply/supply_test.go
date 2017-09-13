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
		var originalPath string

		BeforeEach(func() {
			pythonInstallDir = filepath.Join(depDir, "python")
			ioutil.WriteFile(filepath.Join(buildDir, "runtime.txt"), []byte("\n\n\npython-3.4.2\n\n\n"), 0644)

			versions = []string{"3.4.2"}
			originalPath = os.Getenv("PATH")
		})

		AfterEach(func() {
			os.Setenv("PATH", originalPath)
		})

		Context("runtime.txt sets Python version 3", func() {
			It("installs Python version 3", func() {
				mockManifest.EXPECT().AllDependencyVersions("python").Return(versions)
				mockManifest.EXPECT().InstallDependency(libbuildpack.Dependency{Name: "python", Version: "3.4.2"}, pythonInstallDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")
				Expect(supplier.InstallPython()).To(Succeed())
				Expect(os.Getenv("PATH")).To(Equal(fmt.Sprintf("%s:%s", filepath.Join(depDir, "bin"), originalPath)))
				Expect(os.Getenv("PYTHONPATH")).To(Equal(filepath.Join(depDir)))
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
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib", "pkgconfig"), "pkgconfig")

			Expect(supplier.InstallPip()).To(Succeed())
		})
	})

	Describe("InstallPipPop", func() {
		It("Installs pip-pop", func() {
			mockManifest.EXPECT().InstallOnlyVersion("pip-pop", "/tmp/pip-pop")
			mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip", "install", "pip-pop", "--exists-action=w", "--no-index", "--find-links=/tmp/pip-pop")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(filepath.Join(depDir, "python"), "bin"), "bin")
			Expect(supplier.InstallPipPop()).To(Succeed())
		})
	})

	Describe("HandlePylibmc", func() {
		AfterEach(func() {
			os.Setenv("LIBMEMCACHED", "")
		})

		Context("when the app uses pylibmc", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "pylibmc").Return(nil)
			})
			It("installs libmemcache", func() {
				memcachedDir := filepath.Join(depDir, "libmemcache")
				mockManifest.EXPECT().InstallOnlyVersion("libmemcache", memcachedDir)
				mockStager.EXPECT().WriteEnvFile("LIBMEMCACHED", memcachedDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib"), "lib")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "sasl2"), "lib")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "pkgconfig"), "pkgconfig")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "include"), "include")
				Expect(supplier.HandlePylibmc()).To(Succeed())
				Expect(os.Getenv("LIBMEMCACHED")).To(Equal(memcachedDir))
			})
		})
		Context("when the app does not use pylibmc", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "pylibmc").Return(fmt.Errorf("not found"))
			})

			It("does not install libmemcache", func() {
				Expect(supplier.HandlePylibmc()).To(Succeed())
				Expect(os.Getenv("LIBMEMCACHED")).To(Equal(""))
			})
		})
	})

	Describe("HandleFfi", func() {
		AfterEach(func() {
			os.Setenv("LIBFFI", "")
		})

		Context("when the app uses ffi", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "argon2-cffi", "bcrypt", "cffi", "cryptography", "django[argon2]", "Django[argon2]", "django[bcrypt]", "Django[bcrypt]", "PyNaCl", "pyOpenSSL", "PyOpenSSL", "requests[security]", "misaka").Return(nil)
			})

			It("installs ffi", func() {
				ffiDir := filepath.Join(depDir, "libffi")
				mockManifest.EXPECT().AllDependencyVersions("libffi").Return([]string{"1.2.3"})
				mockManifest.EXPECT().InstallOnlyVersion("libffi", ffiDir)
				mockStager.EXPECT().WriteEnvFile("LIBFFI", ffiDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib"), "lib")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib", "pkgconfig"), "pkgconfig")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib", "libffi-1.2.3", "include"), "include")
				Expect(supplier.HandleFfi()).To(Succeed())
				Expect(os.Getenv("LIBFFI")).To(Equal(ffiDir))
			})
		})
		Context("when the app does not use libffi", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "argon2-cffi", "bcrypt", "cffi", "cryptography", "django[argon2]", "Django[argon2]", "django[bcrypt]", "Django[bcrypt]", "PyNaCl", "pyOpenSSL", "PyOpenSSL", "requests[security]", "misaka").Return(fmt.Errorf("not found"))
			})

			It("does not install libffi", func() {
				Expect(supplier.HandleFfi()).To(Succeed())
				Expect(os.Getenv("LIBFFI")).To(Equal(""))
			})
		})
	})

	Describe("RewriteShebangs", func() {
		BeforeEach(func() {
			os.MkdirAll(filepath.Join(depDir, "bin"), 0755)
			Expect(ioutil.WriteFile(filepath.Join(depDir, "bin", "somescript"), []byte("#!/usr/bin/python\n\n\n"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(depDir, "bin", "anotherscript"), []byte("#!//bin/python\n\n\n"), 0755)).To(Succeed())
		})
		It("changes them to #!/usr/bin/env python", func() {
			Expect(supplier.RewriteShebangs()).To(Succeed())

			fileContents, err := ioutil.ReadFile(filepath.Join(depDir, "bin", "somescript"))
			Expect(err).ToNot(HaveOccurred())

			secondFileContents, err := ioutil.ReadFile(filepath.Join(depDir, "bin", "anotherscript"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(fileContents)).To(HavePrefix("#!/usr/bin/env python"))
			Expect(string(secondFileContents)).To(HavePrefix("#!/usr/bin/env python"))
		})
	})

	Describe("RunPip", func() {
		It("Runs and outputs pip", func() {
			// FIXME test indent (and cleanup?)
			mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip", "install", "-r", "requirements.txt", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir))
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "python", "bin"), "bin")
			Expect(supplier.RunPip()).To(Succeed())
		})
	})

	Describe("CreateDefaultEnv", func() {
		It("writes an env file for PYTHONPATH", func() {
			mockStager.EXPECT().WriteEnvFile("PYTHONPATH", depDir)
			mockStager.EXPECT().WriteEnvFile("LIBRARY_PATH", filepath.Join(depDir, "lib"))
			mockStager.EXPECT().WriteEnvFile("PYTHONHASHSEED", "random")
			mockStager.EXPECT().WriteEnvFile("PYTHONUNBUFFERED", "1")
			mockStager.EXPECT().WriteEnvFile("LANG", "en_US.UTF-8")
			mockStager.EXPECT().WriteEnvFile("PYTHONHOME", filepath.Join(depDir, "python"))
			mockStager.EXPECT().WriteProfileD(gomock.Any(), gomock.Any())
			Expect(supplier.CreateDefaultEnv()).To(Succeed())
		})

		It("writes the profile.d", func() {
			mockStager.EXPECT().WriteEnvFile(gomock.Any(), gomock.Any()).AnyTimes()
			mockStager.EXPECT().WriteProfileD("python.sh", fmt.Sprintf(`export LANG=${LANG:-en_US.UTF-8}
export PYTHONHASHSEED=${PYTHONHASHSEED:-random}
export PYTHONPATH=%s
export PYTHONHOME=%s
export PYTHONUNBUFFERED=1`, depDir, filepath.Join(depDir, "python")))
			Expect(supplier.CreateDefaultEnv()).To(Succeed())
		})
	})
})
