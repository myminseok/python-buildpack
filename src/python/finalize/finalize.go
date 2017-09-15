package finalize

import (
	"fmt"
	"io"
	"os"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest interface {
	RootDir() string
}

type Stager interface {
	BuildDir() string
	DepDir() string
}

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
	// Run(cmd *exec.Cmd) error
}

type ManagePyFinder interface {
	FindManagePy(dir string) (string, error)
}

type Finalizer struct {
	Stager         Stager
	Log            *libbuildpack.Logger
	Logfile        *os.File
	Manifest       Manifest
	Command        Command
	ManagePyFinder ManagePyFinder
	// StartScript string
}

func Run(f *Finalizer) error {

	if err := f.HandleCollectstatic(); err != nil {
		f.Log.Error("Error handling collectstatic: %v", err)
		return err
	}

	return nil
}

func (f *Finalizer) HandleCollectstatic() error {
	if len(os.Getenv("DISABLE_COLLECTSTATIC")) > 0 {
		return nil
	}
	if err := f.Command.Execute(f.Stager.BuildDir(), os.Stdout, os.Stderr, "pip-grep", "-s", "requirements.txt", "django", "Django"); err != nil {
		return nil
	}

	managePyPath, err := f.ManagePyFinder.FindManagePy(f.Stager.BuildDir())
	if err != nil {
		return err
	}

	f.Log.Info("Running python %s collectstatic --noinput --traceback", managePyPath)
	//TODO: should filter out empty lines or those starting with Post-processed --OR-- Copying
	if err = f.Command.Execute(f.Stager.BuildDir(), os.Stdout, os.Stderr, "python", managePyPath, "collectstatic", "--noinput", "--traceback"); err != nil {
		f.Log.Error(fmt.Sprintf(` !     Error while running '$ python %s collectstatic --noinput'.
       See traceback above for details.

       You may need to update application code to resolve this error.
       Or, you can disable collectstatic for this application:

          $ cf set-env <app> DISABLE_COLLECTSTATIC 1

       https://devcenter.heroku.com/articles/django-assets`, managePyPath))
		//TODO: dump environment variables if $DEBUG_COLLECTSTATIC is set???
		return err
	}

	return nil
}
