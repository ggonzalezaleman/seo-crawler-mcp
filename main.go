package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	checkConfig := flag.Bool("check-config", false, "Validate config and print effective config")
	flag.Parse()

	if *showVersion {
		fmt.Printf("seo-crawler-mcp %s\n", version)
		os.Exit(0)
	}

	if *checkConfig {
		fmt.Println("Config validation not yet implemented")
		os.Exit(0)
	}

	// MCP server startup will go here
	fmt.Fprintln(os.Stderr, "seo-crawler-mcp: MCP server not yet implemented")
	os.Exit(1)
}
