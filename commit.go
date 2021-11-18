package main

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"

	"github.com/gookit/color"
	"github.com/sony/sonyflake"
)

// Commit changes and prepare files for planned commit
func Commit() {
	r, _ := git.PlainOpen(".")
	customR, err := checkOrCreateGitplanWorkdir(r)
	if err != nil {
		color.Error.Println("Something is wrong with .gitplan dir")
		color.Error.Println(err.Error())
		panic(err)
	}
	diff, err := commitExistingBranch(r)

	if err != nil || diff == nil {
		color.Error.Println("Something went wrong committing your changes")
		panic(err)
	}
	currentBranch, err := getCurrentBranch(r)
	if err != nil {
		color.Error.Println("Could not get current branch name: ")
		panic(err)
	}

	prepareCommitHiddenBranch(customR, diff, currentBranch)
}

// Commit to the existing branch to let the user do other things
func commitExistingBranch(r *git.Repository) ([]byte, error) {
	worktree, _ := r.Worktree()
	/*
		Unmodified         StatusCode = ' '
		Untracked          StatusCode = '?'
		Modified           StatusCode = 'M'
		Added              StatusCode = 'A'
		Deleted            StatusCode = 'D'
		Renamed            StatusCode = 'R'
		Copied             StatusCode = 'C'
		UpdatedButUnmerged StatusCode = 'U'
	*/
	status, _ := worktree.Status()
	hasChanges := false
	for _, s := range status {
		if (s.Staging != git.Untracked) && (s.Staging != git.Unmodified) {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		color.Error.Println("Nothing to commit, make sure to git add your modifications")

		return nil, nil
	}
	message := getParam("m")
	if message == "" {
		color.Error.Println("Should maybe provide a message for the commit")

		return nil, nil
	}
	cmd := exec.Command("git", "diff", "--staged")
	diff, _ := cmd.Output()

	_, err := worktree.Commit(message, &git.CommitOptions{})

	return diff, err
}

// Save the diff file in .gitplan/commits/{id}.diff
// Save the info in .gitplan/commits/{id}.info
// Info file contains the date (as an UNIX timestamp), the branch name and the commit message
func prepareCommitHiddenBranch(r *git.Repository, diff []byte, branchName string) {
	flake := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, _ := flake.NextID()
	message := getParam("m")
	date := getParam("date")

	date = formatDate(date)

	info := date + "\n" + branchName + "\n" + message
	if _, err := os.Stat(".gitplan/commits"); os.IsNotExist(err) {
		os.Mkdir(".gitplan/commits", 0755)
	}
	strId := strconv.FormatUint(id, 10)
	os.WriteFile(".gitplan/commits/"+strId+".info", []byte(info), 0755) // contains date, branche name and commit message
	os.WriteFile(".gitplan/commits/"+strId+".diff", diff, 0755)
}

func getCurrentBranch(r *git.Repository) (string, error) {
	head, err := r.Head()
	if err != nil {
		return "", err
	}

	return strings.Replace(string(head.Name()), "refs/heads/", "", 1), nil
}

// Verify that the .gitplan/repo is initialized
// if it's not, initialize it
// returns the .gitplan/repo repository
func checkOrCreateGitplanWorkdir(r *git.Repository) (*git.Repository, error) {
	if _, err := os.Stat(".gitplan/repo"); !os.IsNotExist(err) {
		newR, err := git.PlainOpen(".gitplan/repo")

		return newR, err
	}
	color.Comment.Println("Initializing .gitplan/repo folder with a copy of the repository")
	remote, _ := r.Remote("origin")
	originUrl := remote.Config().URLs[0]

	err := os.MkdirAll(".gitplan/repo", 0755)
	if err != nil {
		color.Error.Println("Could not initialize .gitplan/repo folder")
		panic(err)
	}
	color.Info.Println("Sir, we need the path to your private key file")
	scanner := bufio.NewScanner(os.Stdin)
	privateKeyFile := ""
	password := ""
	for scanner.Scan() {
		if len(privateKeyFile) == 0 {
			privateKeyFile = scanner.Text()
			color.Info.Println("Thanks you good sir, would you now mind giving the passphrase to this private key file, if there is any? Otherwise just press enter")
		} else {
			password = scanner.Text()
			color.Info.Println("Sir, you are awesome")
			break
		}
	}
	os.WriteFile(".gitplan/config", []byte(privateKeyFile+"\n"+password), 0644)

	auth, err := GenerateAuth(privateKeyFile, password)
	if err != nil {
		color.Error.Println("generate publickeys failed", err.Error())
		return nil, err
	}

	newR, err := git.PlainClone(".gitplan/repo", false, &git.CloneOptions{
		URL:      originUrl,
		Auth:     auth,
		Progress: os.Stdout,
	})

	return newR, err
}
