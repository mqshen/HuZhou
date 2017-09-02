package main

import (
	"math/rand"
	"time"
	"github.com/spf13/pflag"
	"github.com/mqshen/HuZhou/cmd/kub-apiserver/app/options"
	"github.com/HuZhou/apiserver/pkg/util/flag"
	"github.com/HuZhou/apiserver/pkg/util/logs"
	"fmt"
	"os"
	"github.com/HuZhou/apiserver/pkg/server"
	"github.com/mqshen/HuZhou/cmd/kub-apiserver/app"
	"github.com/mqshen/HuZhou/pkg/version/verflag"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	s := options.NewServerRunOptions()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	verflag.PrintAndExitIfRequested()

	stopCh := server.SetupSignalHandler()
	if err := app.Run(s, stopCh); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
