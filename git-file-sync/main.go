package main

import (
	"fmt"
	"log"
)

func main() {
	c, err := initConfig()
	if err != nil {
		log.Fatalf("initing config: %v", err)
	}
	fmt.Printf("Configuration:\n %+v", c)

	// 1. Get the list of repositories
	// 2. For each repo:
	// 	- Clone the repository
	// 	- Get final branch - could be an existing file-sync pull request or main.
	// 		- get current branch
	// 	- Make the diff.
	// 	- Create or Update the pull request if something has changed
}
