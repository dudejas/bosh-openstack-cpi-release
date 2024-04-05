package main

import (
	"flag"
	"github.com/cloudfoundry/bosh-cpi-go/rpc"
	"github.com/cloudfoundry/bosh-golang-openstack-cpi-go/src/bosh-golang-openstack-cpi/config"
	"github.com/cloudfoundry/bosh-golang-openstack-cpi-go/src/bosh-golang-openstack-cpi/cpi"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"os"
)

var (
	configPathOpt = flag.String("configFile", "", "Path to configuration file")
	caCertPathOpt = flag.String("caCert", "", "Path to ca cert file")
)

func main() {
	logger, fs, _, uuidGen := basicDeps()
	defer logger.HandlePanic("Main")

	flag.Parse()

	config, err := config.NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error("main", "Loading config %s", err.Error())
		os.Exit(1)
	}

	cpiFactory := cpi.NewFactory(fs, uuidGen, config.Cloud.Properties.Openstack, logger)

	cli := rpc.NewFactory(logger).NewCLI(cpiFactory)

	err = cli.ServeOnce()
	if err != nil {
		logger.Error("main", "Serving once %s", err)
		os.Exit(1)
	}
}

func basicDeps() (boshlog.Logger, boshsys.FileSystem, boshsys.CmdRunner, boshuuid.Generator) {
	logger := boshlog.NewWriterLogger(boshlog.LevelDebug, os.Stderr)
	fs := boshsys.NewOsFileSystem(logger)
	cmdRunner := boshsys.NewExecCmdRunner(logger)
	uuidGen := boshuuid.NewGenerator()
	return logger, fs, cmdRunner, uuidGen
}
