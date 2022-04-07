package main

import (
	yaml "gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
)

type dbT struct {
	Bm_login   string
	Bm_passw   string
	Queue_mask string
	Login      string
	Passw      string
	Server     string
	Port       uint32
	Sid        string
	Sort       int
}

type cfgFileT struct {
	Databases map[string]dbT
}

var config cfgFileT

func configReader() {
	file, err := os.Open("config.yml")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	data, err := ioutil.ReadAll(file)

	if err = yaml.Unmarshal([]byte(data), &config); err != nil {
		log.Fatalf("error: %v", err)
	}
	// log.Println(config)
}
