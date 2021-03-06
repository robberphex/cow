package main

import (
	// "flag"

	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"

	// "runtime/pprof"
	"sync"
	"syscall"

	"github.com/pkg/errors"
)

// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var (
	quit     chan struct{}
	relaunch bool
)

// This code is from goagain
func lookPath() (argv0 string, err error) {
	argv0, err = exec.LookPath(os.Args[0])
	if nil != err {
		return
	}
	if _, err = os.Stat(argv0); nil != err {
		return
	}
	return
}

func setRlimitNofile() error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return errors.Wrap(err, "Error Getting Rlimit")
	}
	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return errors.Wrap(err, "Error Setting Rlimit")
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return errors.Wrap(err, "Error Getting Rlimit")
	}
	logrus.Infof("Rlimit Final, Max: %d, Cur: %d", rLimit.Max, rLimit.Cur)
	return nil
}

func main() {
	setRlimitNofile()
	quit = make(chan struct{})
	// Parse flags after load config to allow override options in config
	cmdLineConfig := parseCmdLineConfig()
	if cmdLineConfig.PrintVer {
		printVersion()
		os.Exit(0)
	}

	parseConfig(cmdLineConfig.RcFile, cmdLineConfig)

	initSelfListenAddr()
	initLog()
	initAuth()
	initSiteStat()
	initPAC() // initPAC uses siteStat, so must init after site stat

	initStat()

	initParentPool()

	/*
		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				Fatal(err)
			}
			pprof.StartCPUProfile(f)
		}
	*/

	if config.Core > 0 {
		runtime.GOMAXPROCS(config.Core)
	}

	go sigHandler()
	go runSSH()
	if config.EstimateTimeout {
		go runEstimateTimeout()
	} else {
		info.Println("timeout estimation disabled")
	}

	var wg sync.WaitGroup
	wg.Add(len(listenProxy))
	for _, proxy := range listenProxy {
		go proxy.Serve(&wg, quit)
	}

	wg.Wait()

	if relaunch {
		info.Println("Relunching cow...")
		// Need to fork me.
		argv0, err := lookPath()
		if nil != err {
			errl.Println(err)
			return
		}

		err = syscall.Exec(argv0, os.Args, os.Environ())
		if err != nil {
			errl.Println(err)
		}
	}
	debug.Println("the main process is , exiting...")
}
