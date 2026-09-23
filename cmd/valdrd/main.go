package main

import (
	"fmt"

	"github.com/Sheff1981/valdr-core/config"
)

func main() {
	fmt.Printf("%s %s valdrd %s\n", config.ProjectName, config.Ticker, config.Version)
}
