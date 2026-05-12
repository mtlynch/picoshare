package main

import (
	"fmt"
	"os"

	"github.com/mtlynch/picoshare/build/httperrorreturn"
)

func main() {
	issues, err := httperrorreturn.CheckPaths(os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-http-error-return: %v\n", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		return
	}

	for _, issue := range issues {
		fmt.Fprintf(
			os.Stderr,
			"%s:%d:%d: %s\n",
			issue.Path,
			issue.Line,
			issue.Column,
			issue.Message,
		)
	}
	os.Exit(1)
}
