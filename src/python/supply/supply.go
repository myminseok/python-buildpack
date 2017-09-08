package supply

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	BuildDir() string
	DepDir() string
	// DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	// WriteEnvFile(string, string) error
	// WriteProfileD(string, string) error
	// SetStagingEnvironment() error
}

type Manifest interface {
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type Command interface {
	// Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
	// Run(cmd *exec.Cmd) error
}

type Supplier struct {
	PythonVersion string
	Manifest      Manifest
	Stager        Stager
	Command       Command
	Log           *libbuildpack.Logger
	Logfile       *os.File
}

func Run(s *Supplier) error {
	// FIXME handle errors

	s.InstallPython()
	s.InstallPip()

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

func (s *Supplier) InstallPip() error {
	for _, name := range []string{"setuptools", "pip"} {
		if err := s.Manifest.InstallOnlyVersion(name, filepath.Join("/tmp", name)); err != nil {
			return err
		}
		versions := s.Manifest.AllDependencyVersions(name)
		if output, err := s.Command.Output(filepath.Join("/tmp", name, name+"-"+versions[0]), "python", "setup.py", "install", fmt.Sprintf("--prefix=%s", filepath.Join(s.Stager.DepDir(), "python"))); err != nil {
			s.Log.Error(output)
			return err
		}
	}

	for _, dir := range []string{"bin", "lib", "include", "pkgconfig"} {
		s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", dir), dir)
	}

	return nil
}
