package main

import (
	"encoding/json"
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/generatepem"
)

func main() {
	var err error
	c := generatepem.Config{}
	if err := envconfig.Process("", &c); err != nil {
		log.Fatal(err)
	}
	caPems, serverPems, err := generatepem.Generate(c)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Pems:%s", dumpJSON(map[string]any{"serverPems": serverPems, "caPems": caPems}))
}

func dumpJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
