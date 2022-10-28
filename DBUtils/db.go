package main

// Модуль работы с БД
import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"database/sql"

	_ "github.com/sijms/go-ora/v2"
)

// Подключение к БД и возврат объекта подключения
func getConnection(
	login string,
	passw string,
	server string,
	port uint16,
	sid string) (db *sql.DB, err error) {
	//
	db, err = sql.Open(
		"oracle",
		fmt.Sprintf("oracle://%s:%s@%s:%v/%s",
			url.PathEscape(login),
			url.PathEscape(passw),
			server,
			port,
			sid))
	if err != nil {
		log.Printf("Open Connection: %v\n", err)
		return
	}
	return
}

// Выполнение SQL и возврат строк в виде двумерного слайса с именами столбцов
func getRows(db *sql.DB, sql string, params ...any) (tableData [][]string, err error) {
	rows, err := db.Query(sql, params...)
	if err != nil {
		log.Printf("Can't create query: %v\n", err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		log.Printf("Can't get columns: %v\n", err)
		return
	}
	// иногда статическая типизация - это боль
	// делаем срез указателей на срез string для возможности передачи
	// в (*sql.Rows).Scan динамического числа параметров (столбцов)
	pointers := make([]interface{}, len(cols))
	values := make([]string, len(cols))
	for i := range pointers {
		pointers[i] = &values[i]
	}
	// двумерный срез для таблицы результата SQL
	tableData = make([][]string, 1)
	// заголовок - имена столбцов
	tableData[0] = cols
	idx := 0

	for rows.Next() {
		idx++
		// заполняем values через указатели на них
		err = rows.Scan(pointers...)
		if err != nil {
			log.Printf("Can't Next: %v\n", err)
			return
		}
		tableData = append(tableData, make([]string, len(cols)))
		copy(tableData[idx], values)
	}

	return
}

// Выполнение SQL и получение первой строки
func getScalar(db *sql.DB, sql string) (value string, err error) {
	rows := db.QueryRow(sql)
	err = rows.Scan(&value)
	if err != nil {
		log.Printf("Can't get scalar value: %v\n", err)
		return
	}

	return
}

// Волучение статуса процессов
func procStatDB(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()
	table, err := getRows(db, `select procname "Процесс",
							case when is_active=1 then 'активный' else 'ОСТАНОВЛЕН' end "Статус"
						from kp.v$monitor_menu
						order by 1`)
	c <- table
	c <- err
}

// Вывод блокировок
func viewLocksDB(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	c <- "Список процессов, блокирующих накат объектов"
	table, err := getRows(db, `select /*+ ordered */
						w1.sid waiting_session,
						h1.sid holding_session,
						h1.username username,
						h1.osuser osuser,
						h1.machine machine
					from dba_kgllock w,
						 dba_kgllock h,
						 v$session w1,
						 v$session h1
					where h.kgllkmod not in (0, 1)
						and h.kgllkreq in  (0, 1)
						and w.kgllkmod in (0, 1)
						and w.kgllkreq not in (0, 1)
						and w.kgllktype = h.kgllktype
						and w.kgllkhdl  = h.kgllkhdl
						and w.kgllkuse  = w1.saddr
						and h.kgllkuse  = h1.saddr`)
	c <- table
	c <- err

	c <- "Список заблокированных таблиц:"
	table, err = getRows(db, `select o.owner || '.' || o.object_name table_name,
									l.session_id,
									l.oracle_username username,
									l.os_user_name,
									s.module
								from dba_objects		o,
									 v$locked_object	l,
									 v$session			s
								where o.object_id = l.object_id
								and s.sid(+) = l.session_id`)
	c <- table
	c <- err
}

// Разрешение блокировок
func releaseLocksDB(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	tableData, err := getRows(db, `select distinct
								h1.audsid,
								h1.sid,
								h1.module
							from dba_kgllock w,
								dba_kgllock h,
								v$session w1,
								v$session h1
							where h.kgllkmod not in (0, 1)
								and h.kgllkreq in  (0, 1)
								and w.kgllkmod in (0, 1)
								and w.kgllkreq not in (0, 1)
								and w.kgllktype = h.kgllktype
								and w.kgllkhdl  = h.kgllkhdl
								and w.kgllkuse  = w1.saddr
								and h.kgllkuse  = h1.saddr`)
	c <- err

	for idx, values := range tableData {
		// пропустим заголовок
		if idx == 0 {
			continue
		}

		c <- fmt.Sprintf("Lock was detected. SID=%v, MODULE=%s", values[1], values[2])

		_, err := db.Exec("begin kp.pk_orasys.kill_session(:1); end;", values[0])
		c <- err

	}
	//
	tableData, err = getRows(db, `select distinct
									o1.owner || '.' || o1.object_name table_name,
									l1.session_id,
									s.audsid,
									s.module
								from dba_objects	o1,
									dba_objects		o2,
									v$locked_object	l1,
									v$locked_object	l2,
									v$session		s
								where o1.object_id = l1.object_id
								and o2.object_id   = l2.object_id
								and l1.session_id != l2.session_id
								and o1.object_id   = o2.object_id
								and s.sid          = l1.session_id`)
	c <- err

	for idx, values := range tableData {
		// пропустим заголовок
		if idx == 0 {
			continue
		}

		c <- fmt.Sprintf("Lock was detected. SID=%v TABLE=%s MODULE=%s", values[1], values[0], values[3])

		_, err := db.Exec("begin kp.pk_orasys.kill_session(:1); end;", values[2])
		c <- err
	}
}

// Получение версии БД
func versionDB(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()
	table, err := getRows(db, `select version,
							to_char(modified, 'dd/mm/yy hh24:mi:ss') modified
						from kp.programms
						where type='SYSTEM'`)
	c <- table
	c <- err

}

// Запуск процессов
func startProcDB(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	tableData, err := getRows(db, "select procstart, procname from kp.v$monitor_menu")
	c <- err

	for key, values := range tableData {
		// пропуск заголовка
		if key == 0 {
			continue
		}

		c <- fmt.Sprintf("Отправляем процессу \"%s\" команду на запуск", values[1])
		_, err := db.Exec(fmt.Sprintf("begin %s end;", values[0]))
		c <- err
	}
}

// Стоп процессов
func stopProcDB(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	tableData, err := getRows(db, "select procstop, procname from kp.v$monitor_menu")
	c <- err

	for key, values := range tableData {
		// пропуск заголовка
		if key == 0 {
			continue
		}

		c <- fmt.Sprintf("Отправляем процессу \"%s\" команду на остановку", values[1])
		_, err := db.Exec(fmt.Sprintf("begin %s end;", values[0]))
		c <- err
	}
}

// Очистка очередей
func clearQueuesDB(dbConfig dbT, c chan<- interface{}, pattern string) {
	defer close(c)

	db, err := getConnection(dbConfig.getBM())
	if err != nil {
		return
	}
	defer db.Close()

	// таблицы очередей для чистки
	tables := []string{"bm.creqs_tab", "bm.ireqs_tab"}
	res := make([][]string, 1+len(tables))
	res[0] = []string{"Таблица", "Всего записей", "Удалено"}

	for idx, table := range tables {
		// покажем стату количества всех записей перед чисткой
		creqsCount, err := getScalar(db, fmt.Sprintf(`select count(*) from %s`, table))
		c <- err

		stmt, err := db.Exec(
			fmt.Sprintf(`delete
							from %s c
							where c.q_name like '%%' || :pattern || '%%'`, table),
			sql.Named("pattern", strings.ToUpper(pattern)))
		if err != nil {
			c <- err
			continue
		}

		rows, _ := stmt.RowsAffected()
		res[1+idx] = []string{table, creqsCount, fmt.Sprintf("%v", rows)}
	}
	c <- res
}

// Информация об очередях
func infoQueuesDB(dbConfig dbT, c chan<- interface{}, pattern string) {
	defer close(c)

	db, err := getConnection(dbConfig.getBM())
	if err != nil {
		return
	}
	defer db.Close()
	// таблицы очередей
	tables := []string{"bm.creqs_tab", "bm.ireqs_tab"}
	for _, table := range tables {
		c <- fmt.Sprintf("Таблица %s:", table)
		// костыльная реализация distinct в listagg подзапросом (distinct поддерживается с oracle 19)
		table, err := getRows(
			db,
			fmt.Sprintf(
				`select sum(cnt) "Записей",
						q_name "Очередь",
						listagg(ids, ', ') within group(order by ids) "Список adapter_id"
				from (select count(*) cnt,
							regexp_replace(c.q_name, '^Q_', '') q_name,
							c.user_data.adapter_id ids
						from %s c
					where c.q_name like '%%' || :pattern || '%%'
					group by c.q_name,
						c.user_data.adapter_id)
				group by q_name
				order by 1 desc`, table),
			sql.Named("pattern", strings.ToUpper(pattern)))
		c <- table
		c <- err
	}
}
