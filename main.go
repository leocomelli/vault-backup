package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type fnEncode func(interface{}) string

// VaultBackup is all ths information required to make a backup
type VaultBackup struct {
	client     *vault.Client
	paths      []string
	secrets    map[string]string
	output     string
	encode     string
	filename   string
	pathPrefix string
}

var encode = map[string]fnEncode{
	"plain": func(v interface{}) string {
		return fmt.Sprintf("%v", v)
	},
	"base64": func(v interface{}) string {
		return b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v", v)))
	},
}

// NewBackup creates a new backup
func NewBackup() (*VaultBackup, error) {
	config := vault.DefaultConfig()
	tlsInsecure := vault.TLSConfig{Insecure: true}
	config.ConfigureTLS(&tlsInsecure)
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize vault client")
	}
	return &VaultBackup{
		client: client,
		encode: "plain",
	}, nil
}

func (b *VaultBackup) store(src map[string]string) error {
	if err := mergo.Merge(&b.secrets, src); err != nil {
		return err
	}
	return nil
}

func (b *VaultBackup) walk(parent string, paths []string) {
	for _, p := range paths {
		if p != "" {
			p = fmt.Sprintf("%s%s", parent, p)
		}

		if p != "" && !strings.HasSuffix(p, "/") {
			log.Printf("- reading %s", p)

			secrets, err := b.read(fmt.Sprintf("secret/%s", p))
			if err != nil {
				log.Printf("[ERROR] unable to read secret '%s' (%v). \n", p, err)
			}

			if err := b.store(secrets); err != nil {
				log.Printf("[ERROR] unabled to merge the secrets (%v)", err)
			}

			continue
		}

		s, err := b.client.Logical().List(fmt.Sprintf("secret/%s", p))
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
}

func (b *VaultBackup) readJson(filename string) (map[string]interface{}, error) {

	var jsonMap map[string]interface{}
	jsonFile, err := os.Open(filename)

	if err != nil {
		fmt.Println("error:", err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &jsonMap)

	return jsonMap, nil
}

func (b *VaultBackup) writeSecrets(secrets map[string]interface{}) error {
	secretMap := make(map[string]interface{})
	secretMapWrap := make(map[string]interface{})
	currentPath := ""
	secretPath := ""
	keyLength := 0
	secretNumber := 0

	for key, element := range secrets {

		secretPath = key[0:strings.LastIndex(key, "/")]
		//fmt.Println("Path:", secretPath)
		keyLength = len(key)

		secretMap[key[strings.LastIndex(key, "/")+1:keyLength]] = element.(string)

		if currentPath != secretPath && currentPath != "" {
			secretNumber++
			fmt.Printf("Write secret %d\n", secretNumber)
			secretMapWrap["data"] = secretMap
			fmt.Printf("%v\n", secretMapWrap)
			// call write method
			_, err := b.client.Logical().Write(b.pathPrefix+currentPath, secretMapWrap)

			if err != nil {
				return err
			}

		}

		currentPath = secretPath
	}

	return nil
}

func (b *VaultBackup) read(path string) (map[string]string, error) {
	secret, err := b.client.Logical().Read(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read secret '%s'\n", path)
	}

	if secret == nil {
		log.Printf("[ERROR] no version found for '%s'. \n", path)
		return map[string]string{}, nil
	}

	data := secret.Data

	values := make(map[string]string, len(data))

	for k, v := range data {
		values[fmt.Sprintf("%s/%s", path, k)] = encode[b.encode](v)
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
		buf, err := json.Marshal(b.secrets)
		if err != nil {
			return nil, err
		}

		var out bytes.Buffer
		err = json.Indent(&out, buf, "", "\t")
		if err != nil {
			return nil, err
		}

		return out.Bytes(), nil
	case "yaml":
	case "yml":
		return yaml.Marshal(b.secrets)
	}

	return nil, errors.New("unsupported format")
}

func (b *VaultBackup) write() error {
	out, err := b.format()
	if err != nil {
		return err
	}

	return os.WriteFile(b.filename, out, 0600)
}

func main() {
	client, err := NewBackup()
	if err != nil {
		log.Fatal(err)
	}

	var (
		process      string
		paths        string
		base64, help bool
	)

	flag.StringVar(&process, "process", "backup", "Process type backup|write.")
	flag.StringVar(&client.output, "output", "json", "output format. one of: json|yaml|kv")
	flag.StringVar(&paths, "paths", "", "comma-separated base path. must end with /")
	flag.BoolVar(&base64, "base64", false, "encode secret value as base64")
	flag.StringVar(&client.filename, "filename", "vault.backup", "output filename")
	flag.StringVar(&client.pathPrefix, "prefix", "", "Vault path Prefix")
	flag.BoolVar(&help, "help", false, "show this help output")
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	if process == "write" {
		client.readJson(client.filename)
		jsonS, err := client.readJson("vault.backup.json")

		if err != nil {
			log.Fatal("Can't read Json to Write secrets")
		}

		err = client.writeSecrets(jsonS)
		if err != nil {
			log.Fatalf("%v", err)
		}
		log.Println("Writing secrets done! ;)")

	} else {
		client.paths = strings.Split(paths, ",")
		if base64 {
			client.encode = "base64"
		}

		client.walk("", client.paths)

		if err = client.write(); err != nil {
			log.Fatal(err)
		}

		log.Println("Backup done! ;)")
	}

}
