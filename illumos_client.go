package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type IllumosClient struct {
	host   string
	user   string
	client *ssh.Client
}

func (c *IllumosClient) Connect() error {
	var err error = nil

	if c.client != nil {
		return nil
	}

	log.Println("Creating client")
	user, err := user.Current()
	if err != nil {
		return err
	}

	keyPath := path.Join(user.HomeDir, ".ssh", "id_rsa")
	log.Println("Loading private key from ", keyPath)
	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return err
	}

	log.Println("Parsing private key")
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	log.Println("Connecting to host: ", c.host)
	c.client, err = ssh.Dial("tcp", c.host, config)
	if err != nil {
		log.Println("Connection failed: ", err.Error())
		return err
	}

	log.Println("Connected successfully")
	return nil
}

func (c *IllumosClient) Close() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

func (c *IllumosClient) CreateDataset(dataset *Dataset) (*uuid.UUID, error) {
	err := c.Connect()
	if err != nil {
		return nil, err
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}

	defer session.Close()

	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to create UUID for new dataset.  error: %s", err)
	}

	var properties string
	if dataset.Compression != "" {
		properties = properties + " -o compression=\"" + dataset.Compression + "\""
	}

	if dataset.Quota != "" {
		properties = properties + " -o quota=\"" + dataset.Quota + "\""
	}

	cmd := "zfs create -o terraform:uuid=\"" + uuid.String() + "\"" + properties + " " + dataset.Name

	log.Println("SSH execute: " + cmd)
	err = session.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("remote command zfs create error: %s\n", err)
	}

	return &uuid, nil
}

func (c *IllumosClient) GetDataset(id uuid.UUID) (*Dataset, error) {
	err := c.Connect()
	if err != nil {
		return nil, err
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}

	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b

	var stderr bytes.Buffer
	session.Stderr = &stderr

	//zfs list -H -o name,terraform:uuid,compression,quota| jq -Rsn '{"datasets": [inputs | . / "\n" | (.[] | select (length > 0) | . / "\t") as $input | {"name": $input[0], "uuid": $input[1], "compression": $input[2], "quota": $input[3]}]}'
	cmd := fmt.Sprintf(
		"zfs list -H -o name,terraform:uuid,compression,quota"+
			"| jq -Rsn '{\"datasets\": [inputs | . / \"\n\" | (.[] | select (length > 0) | . / \"\t\") as $input"+
			"| {\"name\": $input[0], \"uuid\": $input[1], \"compression\": $input[2], \"quota\": $input[3]}]}'"+
			"| jq '.datasets[] | select(.uuid == \"%s\")'", id.String())

	log.Println(cmd)
	err = session.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("remote command zfs list.  error: %s (%s)", err, stderr.String())
	}

	outputBytes := b.Bytes()

	output := string(outputBytes)
	log.Printf("Returned data: %s", output)

	var dataset Dataset
	err = json.Unmarshal(outputBytes, &dataset)
	if err != nil {
		log.Printf("Failed to parse returned JSON: %s", err)
		return nil, err
	}

	return &dataset, nil
}

func (c *IllumosClient) UpdateDataset(dataset *Dataset, properties []string) error {
	err := c.Connect()
	if err != nil {
		return err
	}

	session, err := c.client.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()

	propertyString := ""
	for _, property := range properties {

		if len(propertyString) > 0 {
			propertyString += " "
		}

		propertyString = propertyString + property
	}

	var stderr bytes.Buffer
	session.Stderr = &stderr

	cmd := "zfs set " + propertyString + " " + dataset.Name
	log.Println(cmd)
	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("remote command zfs set.  error: %s (%s)", err, stderr.String())
	}

	return nil
}

func (c *IllumosClient) DeleteDataset(name string) error {
	err := c.Connect()
	if err != nil {
		return err
	}

	session, err := c.client.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()

	var b bytes.Buffer
	session.Stderr = &b

	err = session.Run("zfs destroy " + name)
	if err != nil {
		return err
	}

	output := b.String()

	if len(output) != 0 {
		return fmt.Errorf("unrecognized response from zfs destroy: %s", output)
	}

	return nil
}
