package common

import (
	"encoding/json"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

func LoadVaultConfigFile(filePath string) {
	jsonBody := make(map[string]string)
	vaultFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("unable to read json config file: %s", err)
	}
	err = json.Unmarshal(vaultFile, &jsonBody)
	if err != nil {
		log.Fatalf("unable to parse json config file: %s", err)
	}
	for k, v := range jsonBody {
		err = flag.CommandLine.Set(k, v)
		if err != nil {
			log.Fatalf("unable to set flag from json file: %s", err)
		}
	}
}
