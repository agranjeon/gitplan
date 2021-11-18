package main

import "github.com/gookit/color"

func main() {
	command := getCommand()
	switch command {
	case "commit":
		// check if .gitplan exists, if not, create it and clone the repository in it
		// commit to the repository, so the user can continue doing its life without worrying about his changes
		// Retrieve a diff of the commit, and save it in .gitplan/commits
		Commit()
	case "consume":
		// walk .gitplan to find if we have commit to push
		// If we have, checkout the branch, apply git diff, then git commit, git push (to have the wanted date)
		Consume()
	default:
		color.Error.Println("Unknown command")
	}
}
