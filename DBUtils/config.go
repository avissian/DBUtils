package main

import (
	"io/ioutil"
	"log"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// Структура конфига
type cfgFileT struct {
	Databases map[string]dbT
	Webserver webserverT
}

// Общая структура databases из конфига
type dbT struct {
	Bm_login   string
	Bm_passw   string
	Queue_mask string
	Login      string
	Passw      string
	Server     string
	Port       uint16
	Sid        string
	Sort       int
}

type webserverT struct {
	Port uint16
}

// Вычитывание настроек подключения KP
func (dbConfig *dbT) getKP() (string, string, string, uint16, string) {
	return dbConfig.Login,
		dbConfig.Passw,
		dbConfig.Server,
		dbConfig.Port,
		dbConfig.Sid
}

// Вычитывание настроек подключения BM
func (dbConfig *dbT) getBM() (string, string, string, uint16, string) {
	return dbConfig.Bm_login,
		dbConfig.Bm_passw,
		dbConfig.Server,
		dbConfig.Port,
		dbConfig.Sid
}

// Чтение конфига с диска
func configReader() (config cfgFileT) {
	file, err := os.Open("config.yml")
	if err != nil {
		curDir, _ := os.Getwd()
		log.Fatalf("Can't open config file: %v in %v", err, curDir)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatalf("Can't close config file: %v", err)
		}
	}()

	data, err := ioutil.ReadAll(file)

	if err = yaml.Unmarshal([]byte(data), &config); err != nil {
		log.Fatalf("Config parse error: %v", err)
	}
	return
}

func getConfigExample() []byte {
	var config cfgFileT
	config.Databases = map[string]dbT{
		"database1": {"bm", "passw", "multi_jsc", "kp", "passw", "server", 1521, "sid", 0},
		"database2": {"bm", "passw", "multi_jsc", "kp", "passw", "server", 1522, "sid", 1}}
	configBytes, _ := yaml.Marshal(config)
	return configBytes
}
