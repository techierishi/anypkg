package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/techierishi/anypkg"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [import] [sum] [clean]\n", os.Args[0])
		flag.PrintDefaults()
	}

	args := os.Args[1:]
	for _, arg := range args {
		switch arg {
		case "import":
			anypkg.Import()
		case "sum":
			anypkg.Sum()
		case "clean":
			anypkg.Clean()
		}
	}
}
