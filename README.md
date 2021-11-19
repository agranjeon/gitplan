## Gitplan
Somehow make it so you can plan your commit, to make your coworkers think you are working, while you are actually just sleeping


### Installation

Either grab the latest release:

[Linux](https://github.com/agranjeon/gitplan/releases/download/0.0.2/gitplan)

or build from source
```go
go install
```

### Requirements
- `git` must be installed globally

### Usage

Before committing, your branch must exist on remote (for now, we can't know from which branch your local branch was created)

* `commit`
This command creates a .diff file of the staged changes in `.gitplan/commits` and a .info file containing the date, branch and commit message. It also commits to the branch you're actually on, so you can keep working or doing other stuff without worrying about your changes.

```sh
git add *
gitplan commit -m "My sick commit" -date "+2hours"
```
`date` param accepts hours and minutes (I don't know why you would want to use seconds or days here)

The first time you use this command on a repository, it will ask for your private key file path and passphrase (because it might be needed to clone, fetch and push)

* `consume`

That is the command you will launch before going to take a nap. It walks the .info files in `.gitplan/commits` every 20 seconds to find commits to commit and push

for some reasons (for now) you need to have a branch that exists with the same name on the remote 
```sh
gitplan consume
```

When a commit is pushed, you receive a notification
