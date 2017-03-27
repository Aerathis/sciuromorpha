package main

import (
	"flag"
	"fmt"

	git "git.ripostegames.com/sciuromorpha/lib"
)

var tag = flag.String("tag", "", "Git tag to checkout")
var repoPath = flag.String("repopath", "", "Absolute path to reposoitory")
var sshPath = flag.String("sshpath", "", "Path to ssh credentials for remote git server")

func main() {
	flag.Parse()
	fmt.Println("Searching", *repoPath, "for git tag", *tag)

	repo, err := git.OpenRepository(*repoPath, *sshPath)
	if err != nil {
		panic(err)
	}
	defer repo.Free()

	err = repo.CheckoutTag(*tag)
	if err != nil {
		panic(err)
	}
}
