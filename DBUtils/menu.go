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

func subMenu(dbName string, db dbT) {
	cli := gocli.MkCLI(pterm.FgGreen.Sprintf("База: %s", dbName))
	cli.AddOption("", "", func(args []string) string { return "" })

	cli.AddOption("w", "просмотр статуса процессов Runproc", func(args []string) string {
		return selectExec(db, `select procname "Процесс",
                                case when Is_active=1 then 'активный' else 'остановлен' end "Статус"
                            from kp.v$monitor_menu
                            order by 1`)
	})
	cli.AddOption("sr", "запустить процесс", func(args []string) string {
		return startProc(db)
	})
	cli.AddOption("shr", "остановить процесс", func(args []string) string {
		return stopProc(db)
	})
	cli.AddOption("l", "список заблокированных процессов:", func(args []string) string {
		pterm.FgLightYellow.Println("Список процессов, блокирующих накат объектов")
		selectExec(db, `SELECT /*+ ORDERED */
                            W1.SID WAITING_SESSION,
                            H1.SID HOLDING_SESSION,
                            H1.USERNAME USERNAME,
                            H1.OSUSER OSUSER,
                            H1.MACHINE MACHINE
                        FROM DBA_KGLLOCK W,
                            DBA_KGLLOCK H,
                            V$SESSION W1,
                            V$SESSION H1
                        WHERE (((H.KGLLKMOD != '0')
                            AND (H.KGLLKMOD != '1')
                            AND ((H.KGLLKREQ = 0) OR (H.KGLLKREQ = 1)))
                            AND (((W.KGLLKMOD = 0) OR (W.KGLLKMOD= '1'))
                            AND ((W.KGLLKREQ != 0) AND (W.KGLLKREQ !='1'))))
                            AND W.KGLLKTYPE=H.KGLLKTYPE
                            AND W.KGLLKHDL=H.KGLLKHDL
                            AND W.KGLLKUSE=W1.SADDR
                            AND H.KGLLKUSE=H1.SADDR`)
		pterm.FgLightYellow.Println("Список заблокированных таблиц:")
		return selectExec(db, `select o.owner || '.' || o.object_name TABLE_NAME
									,l.session_id
									,l.oracle_username username
									,l.OS_USER_NAME
									,s.MODULE
								from dba_objects     o
									,v$locked_object l
									,v$session       s
								where o.object_id = l.object_id
								and s.sid(+) = l.SESSION_ID`)
	})
	cli.AddOption("v", "версия Системы \"Город\"", func(args []string) string {
		return selectExec(db, `SELECT version,
                                    to_char(modified, 'dd/mm/yy hh24:mi:ss') modified
                                from kp.programms
                                where type='SYSTEM'`)
	})
	cli.AddOption("rl", "разрешить блокировки", func(args []string) string {
		return locks(db)
	})
	cli.AddOption("c", "почистить очереди", func(args []string) string {

		// fmt.Print("Подстрока наименования очереди: ")
		pattern, _ := cli.Liner.Prompt(
			fmt.Sprintf(
				"Подстрока наименования очереди ('%s' если пусто): ",
				db.Queue_mask))

		if strings.Compare(pattern, "") == 0 {
			pattern = db.Queue_mask
		}
		return queues(db, strings.Trim(pattern, "\n"))
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
