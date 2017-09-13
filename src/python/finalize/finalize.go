package finalize

import (
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

type Finalizer struct {
	Stager   Stager
	Log      *libbuildpack.Logger
	Logfile  *os.File
	Manifest Manifest
	// StartScript string
}

func Run(f *Finalizer) error {

	return nil
}
