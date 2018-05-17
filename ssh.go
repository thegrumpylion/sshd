package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"sync"

	"golang.org/x/crypto/ssh"
)

// SSHdListenAndServe  ssh server main
func SSHdListenAndServe(listen string, authorizedKeys []string) error {
	// generate server rsa key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return err
	}

	authorizedKeysMap := map[string]bool{}
	for _, authkey := range authorizedKeys {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authkey))
		if err != nil {
			// just skip
			log.Printf("Invalid ssh key")
			continue
		}
		authorizedKeysMap[string(pubKey.Marshal())] = true
		log.Println("Key added")
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				log.Printf("User \"%s\" authenticated with PubKey.", c.User())
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels
		go handleChannels(chans)
	}
}

func handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {

	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	ch, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel (%s)", err)
		return
	}

	cmd := exec.Command("/bin/busybox", "sh")
	cmd.Env = []string{"PATH=/bin", "PS1=(ssh) root@\\h \\w# "}

	close := func() {
		ch.Close()
		_, err := cmd.Process.Wait()
		if err != nil {
			log.Printf("Failed to exit command (%s)", err)
		}
		log.Printf("Session closed")
	}

	// Allocate a terminal for this channel
	log.Print("Creating pty...")
	cmdf, err := ExecPTY(cmd)
	if err != nil {
		log.Printf("Could not start pty (%s)", err)
		ch.Close()
		return
	}

	//pipe session to cmd and visa-versa
	var once sync.Once
	go func() {
		io.Copy(ch, cmdf)
		once.Do(close)
	}()
	go func() {
		io.Copy(cmdf, ch)
		once.Do(close)
	}()

	var wantsPTY bool = false
	go func() {
		for req := range requests {
			fmt.Println("Req:", req.Type)
			switch req.Type {
			case "shell":

			case "pty-req":
				wantsPTY = true
				req.Reply(true, nil)
			case "window-change":
				w, h := parseDims(req.Payload)
				ws := &Winsize{Width: uint16(w), Height: uint16(h)}
				SetWinsize(cmdf.Fd(), ws)
				req.Reply(true, nil)
			case "env":

			default:
				log.Println("cannot handle request:", req.Type)
				req.Reply(false, nil)
			}
		}
	}()

}

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}
