package jwks

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/lestrrat-go/jwx/jwk"
	log "github.com/sirupsen/logrus"
)

// AuthConfig stores configuration related to JWKS
type authConfig struct {
	Domain   string `json:"domain"`
	Audience string `json:"audience"`
}

// LoadJWKS load settings from auth.json and retrieve JWKS
func LoadJWKS() map[string]interface{} {
	// Load settings
	jsonFile, err := os.Open("auth.json")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully opened auth.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config authConfig
	json.Unmarshal(byteValue, &config)

	// read remote JWKS
	jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", config.Domain)
	log.Printf("Reading JWKS from %s\n", jwksURL)

	set, err := jwk.FetchHTTP(jwksURL)
	if err != nil {
		log.Fatal(err)
	}

	// reformat to {"kid": "key"}, the format needed by Fiber's JWT middleware
	res := make(map[string]interface{})

	for iter := set.Iterate(context.TODO()); iter.Next(context.TODO()); {
		pair := iter.Pair()
		key := pair.Value.(jwk.Key)

		var raw interface{}
		if err := key.Raw(&raw); err != nil {
			log.Fatal(err)
		}
		res[key.KeyID()] = raw
	}

	return res
}
