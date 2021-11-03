package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type VaultBackup struct {
	client  *vault.Client
	paths   []string
	secrets map[string]string
	output  string
}

func NewBackup() (*VaultBackup, error) {
	config := vault.DefaultConfig()

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize vault client")
	}
	return &VaultBackup{
		client: client,
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

			b.store(secrets)

			continue
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

	if secret == nil || secret.Data["data"] == nil {
		log.Printf("[ERROR] no version found for '%s'. \n", path)
		return map[string]string{}, nil
	}

	data := secret.Data["data"].(map[string]interface{})

	values := make(map[string]string, len(data))

	for k, v := range data {
		values[fmt.Sprintf("%s/%s", path, k)] = fmt.Sprintf("%v", v)
	}

	return values, nil
}

func (b *VaultBackup) format() ([]byte, error) {
	switch b.output {
	case "kv":
		var buf []byte
		for k, v := range b.secrets {
			buf = append(buf, fmt.Sprintf("%s = %s\n", k, v)...)
		}

		return buf, nil
	case "json":
		return json.Marshal(b.secrets)
	case "yaml":
	case "yml":
		return yaml.Marshal(b.secrets)
	}

	return nil, errors.New("unsupported format")
}

func main() {
	client, err := NewBackup()
	if err != nil {
		log.Fatal(err)
	}

	var paths string

	flag.StringVar(&client.output, "output", "json", "output format. one of: json|yaml|kv")
	flag.StringVar(&paths, "paths", "", "comma-separated base path. must end with /")
	flag.Parse()

	client.paths = strings.Split(paths, ",")

	if err := client.walk("", client.paths); err != nil {
		log.Fatal(err)
	}

	out, err := client.format()
	if err != nil {
		log.Fatal("[ERROR] error formating the output (%v). \n", err)
	}

	log.Println(string(out))

}
