package supply

import (
	"fmt"
	"io"
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
	WriteEnvFile(string, string) error
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
	Execute(string, io.Writer, io.Writer, string, ...string) error
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

	if err := s.InstallPython(); err != nil {
		s.Log.Error("Could not install python: %v", err)
		return err
	}

	if err := s.InstallPip(); err != nil {
		s.Log.Error("Could not install pip: %v", err)
		return err
	}

	if err := s.InstallPipPop(); err != nil {
		s.Log.Error("Could not install pip pop: %v", err)
		return err
	}

	if err := s.HandlePylibmc(); err != nil {
		s.Log.Error("Error checking Pylibmc: %v", err)
		return err
	}

	if err := s.RunPip(); err != nil {
		s.Log.Error("Could not install pip packages: %v", err)
		return err
	}

	// if err := s.CreateDefaultEnv(); err != nil {
	// 	return err
	// }

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

		s.PythonVersion = strings.TrimSpace(strings.NewReplacer("\\r", "", "\\n", "").Replace(string(userDefinedVersion)))
		s.Log.Debug("***Version info: (%s)", s.PythonVersion)
	}

	if s.PythonVersion != "" {
		versions := s.Manifest.AllDependencyVersions("python")
		shortPythonVersion := strings.TrimLeft(s.PythonVersion, "python-")
		s.Log.Debug("***Version info: (%s) (%s)", s.PythonVersion, shortPythonVersion)
		ver, err := libbuildpack.FindMatchingVersion(shortPythonVersion, versions)
		if err != nil {
			return err
		}
		dep.Name = "python"
		dep.Version = ver
		s.Log.Debug("***Version info: %s, %s, %s", dep.Name, s.PythonVersion, dep.Version)
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

	if err := os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Join(s.Stager.DepDir(), "bin"), os.Getenv("PATH"))); err != nil {
		return err
	}
	if err := os.Setenv("PYTHONPATH", filepath.Join(s.Stager.DepDir())); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) InstallPipPop() error {
	tempPath := filepath.Join("/tmp", "pip-pop")
	if err := s.Manifest.InstallOnlyVersion("pip-pop", tempPath); err != nil {
		return err
	}

	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip", "install", "pip-pop", "--exists-action=w", "--no-index", fmt.Sprintf("--find-links=%s", tempPath)); err != nil {
		s.Log.Debug("******Path val: %s", os.Getenv("PATH"))
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin"); err != nil {
		return err
	}
	return nil
}

func (s *Supplier) HandlePylibmc() error {
	memcachedDir := filepath.Join(s.Stager.DepDir(), "libmemcache")
	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip-grep", "-s", "requirements.txt", "pylibmc"); err == nil {
		if err := s.Manifest.InstallOnlyVersion("libmemcache", memcachedDir); err != nil {
			return err
		}
		os.Setenv("LIBMEMCACHED", memcachedDir)
		s.Stager.WriteEnvFile("LIBMEMCACHED", memcachedDir)
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib"), "lib")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "sasl2"), "lib")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "pkgconfig"), "pkgconfig")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "include"), "include")
	}

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

	for _, dir := range []string{"bin", "lib", "include"} {
		if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", dir), dir); err != nil {
			return err
		}
	}
	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "lib", "pkgconfig"), "pkgconfig"); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) RunPip() error {
	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip", "install", "-r", "requirements.txt", "--exists-action=w", fmt.Sprintf("--src=%s/src", s.Stager.DepDir())); err != nil {
		return err
	}
	return nil
}

// func (s *Supplier) symlinkAll(names []string) error {
// 	for _, name := range names {
// 		installDir := filepath.Join(s.Stager.DepDir(), name)

// 		for _, dir := range []string{"bin", "lib", "include", "pkgconfig", "lib/pkgconfig"} {
// 			exists, err := libbuildpack.FileExists(filepath.Join(installDir, dir))
// 			if err != nil {
// 				return err
// 			}
// 			if exists {
// 				if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(installDir, dir), path.Base(dir)); err != nil {
// 					return err
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

func (s *Supplier) CreateDefaultEnv() error {
	// if err := os.Setenv("PYTHONPATH", filepath.Join(s.Stager.DepDir())); err != nil {
	// 	return err
	// }
	// if err := os.Setenv("PYTHONHOME", filepath.Join(s.Stager.DepDir(), "python")); err != nil {
	// 	return err
	// }

	// if err := s.Stager.WriteEnvFile("PYTHONPATH", filepath.Join(s.Stager.DepDir())); err != nil {
	// 	return err
	// }
	// if err := s.Stager.WriteEnvFile("PYTHONHOME", filepath.Join(s.Stager.DepDir(), "python")); err != nil {
	// 	return err
	// }

	return nil
}
