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
func getRows(db *sql.DB, sql string, params ...any) (tableData TableS, err error) {
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
	// заголовок - имена столбцов
	tableData.Header = cols
	tableData.Rows = make([][]string, 0)

	for idx := 0; rows.Next(); idx++ {
		// заполняем values через указатели на них
		err = rows.Scan(pointers...)
		if err != nil {
			log.Printf("Can't Next: %v\n", err)
			return
		}
		tableData.Rows = append(tableData.Rows, make([]string, len(cols)))
		tableData.Rows[idx] = make([]string, len(cols))
		copy(tableData.Rows[idx], values)
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

// Получение статуса процессов
func DBProcStat(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()
	table, err := getRows(db, `select procshort short,
								procname "Процесс",
								case when is_active=1 then 'активный' else 'ОСТАНОВЛЕН' end "Статус"
							from kp.v$monitor_menu
							order by 1`)
	table.Caption = "Процессы RUNPROC"
	c <- table
	c <- err
}

// Вывод блокировок
func DBViewLocks(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

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
	table.Caption = "Список процессов, блокирующих накат объектов"
	c <- table
	c <- err

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
	table.Caption = "Список заблокированных таблиц"
	c <- table
	c <- err
}

// Разрешение блокировок
func DBReleaseLocks(dbConfig dbT, c chan<- interface{}) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	tableData, err := getRows(db, `select distinct
								h1.audsid,
								h1.sid,
								h1.module,
								'OK' "Результат"
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
	tableData.Caption = "Блокировки объектов"

	for idx, values := range tableData.Rows {
		_, err := db.Exec("begin kp.pk_orasys.kill_session(:1); end;", values[0])
		if err != nil {
			tableData.Rows[idx][3] = fmt.Sprint(err)
		}
	}
	c <- tableData
	//
	tableData, err = getRows(db, `select distinct
									o1.owner || '.' || o1.object_name table_name,
									l1.session_id,
									s.audsid,
									s.module,
									'OK' "Результат"
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
	tableData.Caption = "Блокировки таблиц"

	for idx, values := range tableData.Rows {
		_, err := db.Exec("begin kp.pk_orasys.kill_session(:1); end;", values[2])
		if err != nil {
			tableData.Rows[idx][4] = fmt.Sprint(err)
		}
	}
	c <- tableData
}

// Получение версии БД
func DBVersion(dbConfig dbT, c chan<- interface{}) {
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
	table.Caption = "Версия Системы \"Город\""
	c <- table
	c <- err

}

// Запуск процессов
func DBProcStart(dbConfig dbT, c chan<- interface{}, short string) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	tableData, err := getRows(db, `select procstart,
										procname,
										'Отправлена команда на запуск' \"Результат\"
								where procshort = :short or :short is null
								from kp.v$monitor_menu order by procname`,
		sql.Named("short", short),
		sql.Named("short", short))
	c <- err
	tableData.Caption = "Запуск процессов RUNPROC"

	for idx, values := range tableData.Rows {
		_, err := db.Exec(fmt.Sprintf("begin %s end;", values[0]))
		if err != nil {
			tableData.Rows[idx][2] = fmt.Sprint(err)
		}
	}
	// удалим первый столбец
	tableData.Header = tableData.Header[1:]
	for idx, val := range tableData.Rows {
		tableData.Rows[idx] = val[1:]
	}
	c <- tableData
}

// Стоп процессов
func DBProcStop(dbConfig dbT, c chan<- interface{}, short string) {
	defer close(c)

	db, err := getConnection(dbConfig.getKP())
	if err != nil {
		return
	}
	defer db.Close()

	tableData, err := getRows(db, `select procstop,
										procname,
										'Отправлена команда на остановку' \"Результат\"
									from kp.v$monitor_menu
									where procshort = :short or :short is null
									order by procname`,
		sql.Named("short", short),
		sql.Named("short", short))
	c <- err
	tableData.Caption = "Остановка процессов RUNPROC"

	for idx, values := range tableData.Rows {
		_, err := db.Exec(fmt.Sprintf("begin %s end;", values[0]))
		if err != nil {
			tableData.Rows[idx][2] = fmt.Sprint(err)
		}
	}
	// удалим первый столбец
	tableData.Header = tableData.Header[1:]
	for idx, val := range tableData.Rows {
		tableData.Rows[idx] = val[1:]
	}
	c <- tableData
}

// Очистка очередей
func DBClearQueues(dbConfig dbT, c chan<- interface{}, pattern string, adapter_id string) {
	defer close(c)

	db, err := getConnection(dbConfig.getBM())
	if err != nil {
		return
	}
	defer db.Close()

	// таблицы очередей для чистки
	tables := []string{"bm.creqs_tab", "bm.ireqs_tab"}
	rowsData := make([][]string, len(tables))

	for idx, table := range tables {
		// покажем стату количества всех записей перед чисткой
		creqsCount, err := getScalar(db, fmt.Sprintf(`select count(*) from %s`, table))
		c <- err

		stmt, err := db.Exec(
			fmt.Sprintf(`delete from %s c
					where (c.q_name like '%%' || :pattern || '%%' or :pattern is null)
					  and (c.user_data.adapter_id = :adapter_id or :adapter_id is null)`,
				table),
			sql.Named("pattern", strings.ToUpper(pattern)),
			sql.Named("pattern", strings.ToUpper(pattern)),
			sql.Named("adapter_id", adapter_id),
			sql.Named("adapter_id", adapter_id))

		if err != nil {
			c <- err
			continue
		}

		rows, _ := stmt.RowsAffected()
		rowsData[idx] = []string{table, creqsCount, fmt.Sprintf("%v", rows)}
	}

	c <- TableS{"Очистка очередей", []string{"Таблица", "Всего записей", "Удалено"}, rowsData}
}

// Информация об очередях
func DBInfoQueues(dbConfig dbT, c chan<- interface{}, pattern string, adapter_id string) {
	defer close(c)

	db, err := getConnection(dbConfig.getBM())
	if err != nil {
		return
	}
	defer db.Close()
	// таблицы очередей
	tables := []string{"bm.creqs_tab", "bm.ireqs_tab"}
	for _, tableName := range tables {
		// костыльная реализация distinct в listagg подзапросом (distinct поддерживается с oracle 19)
		table, err := getRows(
			db,
			fmt.Sprintf(
				`select sum(cnt) "Записей",
						nvl(q_name, '<null>') "Очередь",
						listagg(ids, ', ') within group(order by ids) "Список adapter_id"
				from (select count(*) cnt,
							regexp_replace(c.q_name, '^Q_', '') q_name,
							c.user_data.adapter_id ids
						from %s c
					where (c.q_name like '%%' || :pattern || '%%' or :pattern is null)
					  and (c.user_data.adapter_id = :adapter_id or :adapter_id is null)
					group by c.q_name,
						c.user_data.adapter_id)
				group by q_name
				order by 1 desc`, tableName),
			sql.Named("pattern", strings.ToUpper(pattern)),
			sql.Named("pattern", strings.ToUpper(pattern)),
			sql.Named("adapter_id", adapter_id),
			sql.Named("adapter_id", adapter_id))
		table.Caption = tableName
		c <- table
		c <- err
	}
}
