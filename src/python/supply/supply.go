package supply

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	BuildDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	// WriteEnvFile(string, string) error
	// WriteProfileD(string, string) error
	// SetStagingEnvironment() error
}

type Manifest interface {
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
	InstallDependency(libbuildpack.Dependency, string) error
	// InstallOnlyVersion(string, string) error
}

type Supplier struct {
	PythonVersion string
	Manifest      Manifest
	Stager        Stager
	Log           *libbuildpack.Logger
	Logfile       *os.File
}

func Run(s *Supplier) error {
	return nil
}

func (s *Supplier) InstallPython() error {
	var dep libbuildpack.Dependency

	runtimetxtExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "runtime.txt"))
	if err != nil {
		return err
	}

	if runtimetxtExists {
		userDefinedVersion, err := ioutil.ReadFile(filepath.Join(s.Stager.BuildDir(), "runtime.txt"))
		if err != nil {
			return err
		}

		s.PythonVersion = string(userDefinedVersion)
	}

	if s.PythonVersion != "" {
		versions := s.Manifest.AllDependencyVersions("python")
		shortPythonVersion := strings.TrimLeft(s.PythonVersion, "python-")
		ver, err := libbuildpack.FindMatchingVersion(shortPythonVersion, versions)
		if err != nil {
			return err
		}
		dep.Name = "python"
		dep.Version = ver
		// s.Log.Info("***Version info: %s, %s, %s", dep.Name, s.PythonVersion, dep.Version)
	} else {
		var err error

		dep, err = s.Manifest.DefaultVersion("python")
		if err != nil {
			return err
		}
	}

	pythonInstallDir := filepath.Join(s.Stager.DepDir(), "python")
	if err := s.Manifest.InstallDependency(dep, pythonInstallDir); err != nil {
		return err
	}

	s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
	s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")

	return nil
}
