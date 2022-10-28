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

// Добавление в меню databases из конфига в соответствии с sort
func cliGlobalMenu(config cfgFileT) {
	// сортировка пунктов меню
	type sortS struct {
		sort int
		name string
	}
	sortArr := make([]sortS, len(config.Databases))
	idx := 0
	for key, val := range config.Databases {
		sortArr[idx].name = key
		sortArr[idx].sort = val.Sort
		idx++
	}
	sort.Slice(sortArr, func(i int, j int) bool { return sortArr[i].sort < sortArr[j].sort })

	// отрисовка меню
	cli := gocli.MkCLI(pterm.FgGreen.Sprint("Выбор БД"))
	for idx, val := range sortArr {
		cli.AddOption(strconv.Itoa(idx+1), val.name, func(args []string) string {
			dbName := cli.Options[args[0]].Help
			cliSubMenu(dbName, config.Databases[dbName])
			pterm.FgGreen.Println(cli.Greeting)
			cli.Help(nil)
			return ""
		})
	}
	// добавление зашитых команд
	cli.AddSeparator()
	cli.AddOption("?", "показать доступные команды", cli.Help)
	cli.AddOption("q", "выход", cli.Exit)

	// обработчик неверной команды
	cli.DefaultOption(func(args []string) string {
		return pterm.FgRed.Sprintf("%s: команда не найдена\n", args[0])
	})
	// обработчик случайного нажатия Enter (пустая команда)
	cli.AddOption("", "", func(args []string) string { return "" })

	cli.Loop("> ")
}

// Меню второго уровня
func cliSubMenu(dbName string, dbConfig dbT) {
	cli := gocli.MkCLI(pterm.FgGreen.Sprintf("База: %s", dbName))

	cli.AddOption("w", "просмотр статуса процессов Runproc", func(args []string) (_ string) {
		return cliStandartCommand(DBProcStat, dbConfig)
	})
	cli.AddOption("sr", "запустить процессы Runproc", func(args []string) (_ string) {
		return cliProcessCommand(DBProcStart, dbConfig)
	})
	cli.AddOption("shr", "остановить процессы Runproc", func(args []string) (_ string) {
		return cliProcessCommand(DBProcStop, dbConfig)
	})
	cli.AddOption("l", "список блокировок БД", func(args []string) (_ string) {
		return cliStandartCommand(DBViewLocks, dbConfig)
	})
	cli.AddOption("rl", "разрешить блокировки БД", func(args []string) (_ string) {
		return cliStandartCommand(DBReleaseLocks, dbConfig)
	})
	cli.AddOption("i", "информация по очередям", func(args []string) (_ string) {
		return cliQueuesCommand(DBInfoQueues, dbConfig, args, &cli)
	})
	cli.AddOption("c", "почистить очереди", func(args []string) (_ string) {
		return cliQueuesCommand(DBClearQueues, dbConfig, args, &cli)
	})
	cli.AddOption("v", "версия Системы \"Город\"", func(args []string) (_ string) {
		return cliStandartCommand(DBVersion, dbConfig)
	})

	// добавление зашитых команд
	cli.AddSeparator()
	cli.AddOption("?", "показать доступные команды", cli.Help)
	cli.AddOption("e", "назад", cli.Exit)
	cli.AddOption("q", "выход", func(args []string) (_ string) {
		os.Exit(0)
		return
	})

	// обработчик неверной команды
	cli.DefaultOption(func(args []string) string {
		return pterm.FgRed.Sprintf("%s: команда не найдена", args[0])
	})
	// обработчик случайного нажатия Enter (пустая команда)
	cli.AddOption("", "", func(args []string) string { return "" })

	cli.Loop("> ")

}

// Выполнение стандартной команды
func cliStandartCommand(
	function func(dbT, chan<- interface{}),
	dbConfig dbT) (_ string) {
	//
	c := make(chan interface{})
	go function(dbConfig, c)
	cliPrint(c)
	return
}

// Выполнение команды работы с процессами
func cliProcessCommand(
	function func(dbT, chan<- interface{}, string),
	dbConfig dbT) (_ string) {
	//
	c := make(chan interface{})
	go function(dbConfig, c, "")
	cliPrint(c)
	return
}

// Считывание ввода для подменю, вызов метода работы с БД
func cliQueuesCommand(
	function func(dbT, chan<- interface{}, string, string),
	dbConfig dbT,
	args []string,
	cli *gocli.CLI) (_ string) {
	//
	pattern := ""
	if len(args) > 1 {
		pattern = args[1]
	} else {
		pattern, _ = cli.Liner.Prompt(
			fmt.Sprintf(
				"Подстрока наименования очереди ('%%%s%%' если пусто): ",
				strings.ToUpper(dbConfig.Queue_mask)))

		fmt.Println()

		if strings.Compare(pattern, "") == 0 {
			pattern = dbConfig.Queue_mask
		}
	}

	c := make(chan interface{})
	go function(dbConfig, c, pattern, "")
	cliPrint(c)
	return
}

func cliPrint(c <-chan interface{}) {
	afterTable := false
	for val := range c {
		switch v := val.(type) {
		case TableS:
			if afterTable {
				// после таблиц добавим пустую строку
				fmt.Println()
				afterTable = false
			}
			pterm.FgLightYellow.Printf("%s\n", v.Caption)

			table := make([][]string, len(v.Rows)+1)
			table[0] = make([]string, len(v.Header))
			copy(table[0], v.Header)
			for idx, row := range v.Rows {
				table[idx+1] = make([]string, len(v.Header))
				copy(table[idx+1], row)
			}

			pterm.DefaultTable.WithHasHeader().WithData(table).Render()
			afterTable = true
		case error:
			pterm.PrintOnError(v)
		case nil: // nil - это отсутствие ошибки, пропускаем
		default:
			pterm.FgDefault.Printf("I don't know about type %T!\n", v)
		}
	}
}
