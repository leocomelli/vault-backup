package main

import (
	"fmt"
	"log"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

var paths = []string{""}

type VaultBackup struct {
	client  *vault.Client
	paths   []string
	secrets map[string]string
}

func NewBackup(paths []string) (*VaultBackup, error) {
	config := vault.DefaultConfig()

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize vault client")
	}
	return &VaultBackup{
		client: client,
		paths:  paths,
	}, nil
}

func (b *VaultBackup) store(src map[string]string) error {
	if err := mergo.Merge(&b.secrets, src); err != nil {
		return err
	}
	return nil
}

func (b *VaultBackup) walk(parent string, paths []string) error {
	for _, p := range paths {
		if p != "" {
			p = fmt.Sprintf("%s%s", parent, p)
		}

		if p != "" && !strings.HasSuffix(p, "/") {
			log.Printf("- reading %s", p)

			secrets, err := b.read(fmt.Sprintf("secret/data/%s", p))
			if err != nil {
				log.Printf("[ERROR] unable to read secret '%s' (%v). \n", p, err)
			}
			return b.store(secrets)
		}

		s, err := b.client.Logical().List(fmt.Sprintf("secret/metadata/%s", p))
		if err != nil {
			log.Printf("[ERROR] unable to list secret '%s' (%v). \n", p, err)
		}

		k, ok := s.Data["keys"]

		var keys []string
		for _, x := range k.([]interface{}) {
			keys = append(keys, fmt.Sprintf("%v", x))
		}

		if ok && len(keys) > 0 {
			b.walk(p, keys)
		}
	}

	return nil
}

func (b *VaultBackup) read(path string) (map[string]string, error) {
	secret, err := b.client.Logical().Read(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read secret '%s'\n", path)
	}

	data := secret.Data["data"].(map[string]interface{})

	values := make(map[string]string, len(data))

	for k, v := range data {
		values[fmt.Sprintf("%s/%s", path, k)] = fmt.Sprintf("%v", v)
	}

	return values, nil
}

func main() {
	client, err := NewBackup(paths)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.walk("", client.paths); err != nil {
		log.Fatal(err)
	}

	for k, v := range client.secrets {
		fmt.Println("  ", k, " = ", v)
	}
}
