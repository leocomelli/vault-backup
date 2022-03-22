package main

import (
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestReadJson(t *testing.T) {
	client, err := NewBackup()
	if err != nil {
		log.Fatal(err)
	}
	//jsonS := make(map[string]interface{})

	jsonS, _ := client.readJson("vault.backup.json")

	if jsonS == nil {
		t.Errorf("can't read json")
	}

	//fmt.Printf("%v", jsonS)
	currentPath := ""
	secretMap := make(map[string]interface{})
	secretMapWrap := make(map[string]interface{})
	secretPath := ""
	keyLength := 0
	secretNumber := 0

	for key, element := range jsonS {
		//fmt.Println("Key:", key, "=>", "Element:", element)
		fmt.Println()

		secretPath = key[0:strings.LastIndex(key, "/")]
		//fmt.Println("Path:", secretPath)
		keyLength = len(key)

		secretMap[key[strings.LastIndex(key, "/")+1:keyLength]] = element.(string)

		if currentPath != secretPath && currentPath != "" {
			// if secret != "" {
			// call write method
			secretNumber++
			fmt.Printf("Write secret %d \n", secretNumber)
			fmt.Println(currentPath)
			secretMapWrap["data"] = secretMap
			fmt.Printf("%v\n", secretMapWrap)
		}

		currentPath = secretPath
	}
}
