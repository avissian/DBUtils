package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var web bool
	flag.BoolVar(&web, "web", false, "Запуск вебсервера (если не указано, то консольный режим)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [-web]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Printf("\nПример конфига, файл config.yml:\n%s\n", getConfigExample())
	}
	flag.Parse()

	config := configReader()
	if web {
		webServer(config)
	} else {
		cliGlobalMenu(config)
	}
}
