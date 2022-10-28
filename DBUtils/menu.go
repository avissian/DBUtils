package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/avissian/gocli"
	"github.com/pterm/pterm"
)

type sortS struct {
	sort int
	name string
}

func globalMenu() {
	cli := gocli.MkCLI(pterm.FgGreen.Sprint("Выбор БД"))
	cli.AddOption("", "", func(args []string) string { return "" })

	// Добавление в меню databases из конфига в соответствии с sort
	sortArr := make([]sortS, len(config.Databases))
	idx := 0
	for key, val := range config.Databases {
		sortArr[idx].name = key
		sortArr[idx].sort = val.Sort
		idx++
	}
	sort.Slice(sortArr, func(i int, j int) bool { return sortArr[i].sort < sortArr[j].sort })

	// отрисовка меню
	for idx, val := range sortArr {
		cli.AddOption(strconv.Itoa(idx+1), val.name, func(args []string) string {
			dbName := cli.Options[args[0]].Help
			subMenu(dbName, config.Databases[dbName])
			pterm.FgGreen.Println(cli.Greeting)
			cli.Help(nil)
			return ""
		})
	}

	cli.AddSeparator()
	cli.AddOption("?", "Показать доступные команды", cli.Help)
	cli.AddOption("q", "Выход", cli.Exit)

	cli.DefaultOption(func(args []string) string {
		return pterm.FgRed.Sprintf("%s: команда не найдена\n", args[0])
	})

	cli.Loop("> ")
}

// меню второго уровня
func subMenu(dbName string, dbConfig dbT) {
	cli := gocli.MkCLI(pterm.FgGreen.Sprintf("База: %s", dbName))
	cli.AddOption("", "", func(args []string) string { return "" })

	cli.AddOption("w", "просмотр статуса процессов Runproc", func(args []string) string {
		return procStatDB(dbConfig)
	})
	cli.AddOption("sr", "запустить процесс", func(args []string) string {
		return startProcDB(dbConfig)
	})
	cli.AddOption("shr", "остановить процесс", func(args []string) string {
		return stopProcDB(dbConfig)
	})
	cli.AddOption("l", "список заблокированных процессов", func(args []string) string {
		return viewLocksDB(dbConfig)
	})
	cli.AddOption("v", "версия Системы \"Город\"", func(args []string) string {
		return versionDB(dbConfig)
	})
	cli.AddOption("rl", "разрешить блокировки", func(args []string) string {
		return releaseLocksDB(dbConfig)
	})
	cli.AddOption("c", "почистить очереди", func(args []string) string {
		return clearQueues(dbConfig, &cli)
	})

	cli.AddSeparator()
	cli.AddOption("?", "Показать доступные команды", cli.Help)
	cli.AddOption("e", "Назад", cli.Exit)
	cli.AddOption("q", "Выход", func(args []string) string {
		os.Exit(0)
		return ""
	})

	cli.DefaultOption(func(args []string) string {
		return pterm.FgRed.Sprintf("%s: команда не найдена", args[0])
	})

	cli.Loop("> ")

}

func clearQueues(dbConfig dbT, cli *gocli.CLI) string {
	pattern, _ := cli.Liner.Prompt(
		fmt.Sprintf(
			"Подстрока наименования очереди ('%s' если пусто): ",
			dbConfig.Queue_mask))

	if strings.Compare(pattern, "") == 0 {
		pattern = dbConfig.Queue_mask
	}
	return clearQueuesDB(dbConfig, strings.Trim(pattern, "\n"))
}
