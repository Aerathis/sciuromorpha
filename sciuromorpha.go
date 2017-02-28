package main

import (
	"flag"
	"fmt"

	git "github.com/Aerathis/sciuromorpha/lib"
)

var tag = flag.String("tag", "", "Git tag to checkout")
var repoPath = flag.String("repopath", "", "Absolute path to reposoitory")

func main() {
	flag.Parse()
	fmt.Println("Searching", *repoPath, "for git tag", *tag)

	repo, err := git.OpenRepository(*repoPath)
	if err != nil {
		panic(err)
	}
	defer repo.Free()

	repo.CheckoutTag(*tag)
}
