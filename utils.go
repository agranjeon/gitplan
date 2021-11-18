package main

import (
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gookit/color"
	"golang.org/x/crypto/ssh"
)

// Get the command that is provided in the CLI
// Can be either commit or consume
func getCommand() string {
	for key, val := range os.Args {
		if key == 1 {
			return val
		}
	}
	return ""
}

// Retrieve a named parameter provided in the CLI
// It can be formatted in two ways
// -param "value" or -param="value"
func getParam(name string) string {
	if len(os.Args) < 3 {
		panic("Not enough arguments")
	}
	args := os.Args[2:]

	lookedForKey := "-" + name
	prevValue := ""
	for _, value := range args {
		if strings.Contains(value, "=") {
			s := strings.Split(value, "=")
			param, v := s[0], s[1]
			if param == lookedForKey {
				return v
			}
			prevValue = value
			continue
		}

		if prevValue == lookedForKey {
			return value
		}
		prevValue = value
	}

	color.Error.Printf("Param %v was not found\n", name)

	return ""
}

// Send a desktop notification
// If status is false, it means the notification tells an error
func Notify(message string, status bool) {
	if _, err := os.Stat(".gitplan/assets"); os.IsNotExist(err) {
		os.MkdirAll(".gitplan/assets", 0755)
	}
	if _, err := os.Stat(".gitplan/assets/YEP.png"); os.IsNotExist(err) {
		file, err := Asset("assets/YEP.png")
		if err == nil {
			os.WriteFile(".gitplan/assets/YEP.png", file, 0755)
		}
	}
	if _, err := os.Stat(".gitplan/assets/NOP.png"); os.IsNotExist(err) {
		file, err := Asset("assets/NOP.png")
		if err == nil {
			os.WriteFile(".gitplan/assets/NOP.png", file, 0755)
		}
	}
	image := ".gitplan/assets/YEP.png"
	if !status {
		image = ".gitplan/assets/NOP.png"
	}
	err := beeep.Notify("Gitplan", message, image)
	if err != nil {
		panic(err)
	}
}

// Format the date param from CLI which is formatted "+{value}{unit}" (for example +2hours)
// to have an UNIX timestamp
func formatDate(date string) string {
	reg, _ := regexp.Compile("[+]([0-9]+)(hours|hour|minutes|minute)")
	match := reg.FindAllSubmatch([]byte(date), 2)

	amount, _ := strconv.ParseInt(string(match[0][1]), 10, 64)
	unit := string(match[0][2])

	now := time.Now()
	imDone := time.Duration(amount)
	if unit == "hours" || unit == "hour" {
		imDone = imDone * time.Hour
	} else {
		imDone = imDone * time.Minute
	}
	finalDate := now.Add(imDone)

	return strconv.FormatInt(finalDate.Unix(), 10)
}

// Generate public keys from private key file and password
// Password can be an empty string
func GenerateAuth(privateKeyFile string, password string) (*gitssh.PublicKeys, error) {
	var signer ssh.Signer
	var err error = nil
	sshKey, err := ioutil.ReadFile(privateKeyFile)
	if len(password) == 0 {
		signer, err = ssh.ParsePrivateKey([]byte(sshKey))
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(sshKey), []byte(password))
	}
	if err != nil {
		return nil, err
	}

	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	auth := &gitssh.PublicKeys{User: "git", Signer: signer, HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
		HostKeyCallback: hostKeyCallback,
	}}

	return auth, nil
}
