package auth


/*
This is a Go package named auth that provides functionality for SSH authentication with Git repositories 
using the Go Git library.
Let me break down what this code does:

Package Overview
The auth package defines a single function, GetSSHAuth(), which creates and returns an SSH authentication
object that can be used when interacting with Git repositories that require SSH authentication.

Function: GetSSHAuth()
This function returns:
An *ssh.PublicKeys object which can be used for SSH authentication
An error value, if any occurred during the process

How it works:
It first checks for an environment variable SSH_KEY_PATH to locate your SSH private key
If that environment variable isn't set, it falls back to the standard location for SSH keys:

Finds your home directory with os.UserHomeDir()
Points to ~/.ssh/id_rsa (the default SSH private key path)

Loads the SSH key using ssh.NewPublicKeysFromFile(), which:

Takes a username parameter ("git", which is standard for Git SSH connections)
Takes the path to your SSH private key file
Takes a password for the key (empty string "" means no password)

Returns the public key object or an error with additional context

Ex:
auth, err := auth.GetSSHAuth()
if err != nil {
    // Handle error
}

// Use the auth object with go-git operations
repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
    URL:  "git@github.com:username/repo.git",
    Auth: auth,
})
*/
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func GetSSHAuth() (*ssh.PublicKeys, error) {
	sshPath := os.Getenv("SSH_KEY_PATH")
	if sshPath == "" {
		// Default to standard SSH key location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		sshPath = filepath.Join(homeDir, ".ssh", "id_rsa")
	}

	publicKeys, err := ssh.NewPublicKeysFromFile("git", sshPath, "")
	if err != nil {
		return nil, fmt.Errorf("error loading SSH key: %v", err)
	}
	return publicKeys, nil
}
