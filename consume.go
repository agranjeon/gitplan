package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gookit/color"
)

var auth *gitssh.PublicKeys

// Start the consumer that will walk .gitplan/commits to find commits to push on a given date
func Consume() {
	if _, err := os.Stat(".gitplan/commits"); os.IsNotExist(err) {
		color.Error.Println("Can't consume because there has never been any commit using gitplan")

		return
	}
	lockConsumer()
	defer removeLock()
	r, err := git.PlainOpen(".gitplan/repo")
	if err != nil {
		color.Error.Println("Can't consume")
		panic(err)
	}
	config, err := ioutil.ReadFile(".gitplan/config")
	configContent := string(config)
	c := strings.Split(configContent, "\n")
	keyFile := ""
	password := ""
	for k, v := range c {
		if k == 0 {
			keyFile = v
		} else {
			password = v
		}
	}
	auth, err = GenerateAuth(keyFile, password)
	if err != nil {
		color.Error.Println("generate publickeys failed", err.Error())
		panic(err)
	}
	color.Info.Println("You're all set to sleep and your commit will be pushed while you sleep (hopefully)")
	for {
		files, err := ioutil.ReadDir(".gitplan/commits")
		if err != nil {
			// Should never happen, but who knows
			color.Error.Println("Weird error")
			panic(err)
		}
		for _, file := range files {
			if !strings.Contains(file.Name(), ".info") {
				continue
			}
			if !shouldProcessFile(file.Name()) {
				continue
			}
			processFile(r, file.Name())
		}
		time.Sleep(time.Duration(20) * time.Second)
	}
}

// Create a .lock file to make sure only one consumer is started at a time
func lockConsumer() {
	if _, err := os.Stat(".gitplan/consumer.lock"); !os.IsNotExist(err) {
		color.Error.Println("Consumer is already started. If it's not, then you're screwed")
		os.Exit(1)
	}
	os.WriteFile(".gitplan/consumer.lock", []byte(""), 0755)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		removeLock()
		os.Exit(1)
	}()
}

// Remove the .lock file
func removeLock() {
	if _, err := os.Stat(".gitplan/consumer.lock"); os.IsNotExist(err) {
		return
	}

	os.Remove(".gitplan/consumer.lock")
}

// Check if the given file should be processed based on the date in it and current date
func shouldProcessFile(filename string) bool {
	content, err := os.ReadFile(".gitplan/commits/" + filename)
	if err != nil {
		return false
	}

	fileContent := string(content)
	date, _ := strconv.ParseInt(strings.Split(fileContent, "\n")[0], 10, 64)

	now := time.Now().Unix()

	return now > date
}

// Apply the diff file
// Add the updated file from the diff
// Commit the changes and push
// Remove the branch to ensure the next commit with the same branch name will work
// Remove the .info and .diff file that was processed
func processFile(repository *git.Repository, filename string) {
	content, err := os.ReadFile(".gitplan/commits/" + filename)
	if err != nil {
		return
	}
	defer deleteFiles(filename)
	fileContent := string(content)
	s := strings.Split(fileContent, "\n")
	branchName, message := s[1], s[2]
	worktree, err := checkoutBranch(repository, branchName)
	if err != nil && err.Error() != "worktree contains unstaged changes" {
		Notify(fmt.Sprintf("Something went wrong switching local branch: %v", err.Error()), false)
		return
	}

	diffFilename := strings.Replace(filename, ".info", ".diff", -1)
	cmd := exec.Command("git", "apply", ".gitplan/commits/"+diffFilename, "--directory=.gitplan/repo/")
	_, err = cmd.Output()
	if err != nil {
		Notify("Can't apply diff, maybe you comitted an image or something extra weird, sorry", false)
		return
	}

	err = worktree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		Notify(fmt.Sprintf("Can't add your changes: %v", err.Error()), false)
		return
	}
	_, err = worktree.Commit(message, &git.CommitOptions{})
	if err != nil {
		Notify(fmt.Sprintf("Something went wrong comitting your changes: %v", err.Error()), false)
		return
	}

	err = repository.Push(&git.PushOptions{Auth: auth})
	if err != nil {
		Notify(fmt.Sprintf("Something went wrong pushing your changes: %v", err.Error()), false)
		return
	}

	Notify(fmt.Sprintf("%v is pushed!", branchName), true)
	headRef, err := repository.Head()
	// Remove the branch after the commit has been processed
	// It will allow us to recreate a branch from remote in case there is an other commit with the same branch Name
	// If we don't do that, we're heading to big troubles, and we don't want to be in big trouble
	repository.Storer.RemoveReference(headRef.Name())
}

// Checkout the .gitplan/repo to branchname
// Fetch remote, checkout remote branch, create a new local branch
func checkoutBranch(r *git.Repository, branchName string) (*git.Worktree, error) {
	err := r.Fetch(&git.FetchOptions{Auth: auth})
	if err != nil && err.Error() != "already up-to-date" {
		//If we're already up to date, it's perfect
		color.Error.Println(err.Error())
		return nil, err
	}
	w, _ := r.Worktree()
	refs, _ := r.References()
	expectedReferenceName := plumbing.NewBranchReferenceName(branchName)
	refExists := false
	refs.ForEach(func(p *plumbing.Reference) error {
		if p.Name() == expectedReferenceName {
			refExists = true
		}
		return nil
	})
	// Check if branch already exists
	if !refExists {
		// Checkout to remote branch's commit
		err := w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewRemoteReferenceName("origin", branchName),
		})
		if err != nil && err.Error() == "reference not found" {
			color.Error.Println(err.Error())
			return nil, err
		}
	}

	// Create a local branch from the commit we checked out or the ref that exists
	err = w.Checkout(&git.CheckoutOptions{
		Branch: expectedReferenceName,
		Create: !refExists,
	})

	return w, err
}

func deleteFiles(filename string) {
	infoFile := ".gitplan/commits/" + filename
	diffFile := strings.Replace(infoFile, ".info", ".diff", -1)

	os.Remove(infoFile)
	os.Remove(diffFile)
}
