package main

import (
	"bytes"
	"log"
	"os"

	"github.com/200sc/bebop"
)

var (
	bop = `
struct SendPacket {
	1 -> int64 Datetime;
	2 -> uint32 ConnectionID;
	2 -> string User;
	3 -> string Db;
	4 -> string Addr;
	5 -> string State;
	6 -> string Err;
	7 -> string Cmd;
	8 -> byte[] Packets;

}
`
)

func main() {
	f := bytes.NewBuffer([]byte(bop))
	bopf, s, err := bebop.ReadFile(f)
	if err != nil {
		log.Fatalf("bebop.ReadFile: %v\n", err)
	}
	log.Printf("bebop: %v\n", s)
	out, err := os.Create("mybebop.go")
	if err != nil {
		log.Fatalf("os.Create err: %v\n", err)
	}
	defer out.Close()
	settings := bebop.GenerateSettings{
		PackageName: "mybebop",
	}
	err = bopf.Generate(out, settings)
	if err != nil {
		log.Fatalf("bopf.Generate: %v\n", err)
	}
}
