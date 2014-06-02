package main

import (
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Printf(`Usage of %s:
	 Tasks:
	   goship deploy [env] : Deploy to _env_
	`, os.Args[0])
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	switch flag.Arg(0) {
	case "deploy", "d":
		deploy(flag.Arg(1))
	default:
		usage()
	}
}
