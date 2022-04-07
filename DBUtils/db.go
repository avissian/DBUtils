package main

import (
	"fmt"
	"io"
	"strings"

	"database/sql/driver"
	"github.com/pterm/pterm"
	go_ora "github.com/sijms/go-ora"
)

func selectExec(db dbT, sql string) string {
	conn, err := go_ora.NewConnection(fmt.Sprintf("oracle://%s:%s@%s:%v/%s", db.Login, db.Passw, db.Server, db.Port, db.Sid))
	dieOnError("Connection:", err)
	err = conn.Open()
	dieOnError("Open Connection:", err)
	defer conn.Close()

	stmt := go_ora.NewStmt(sql, conn)
	defer stmt.Close()

	rows, err := stmt.Query(nil)
	dieOnError("Can't create query:", err)

	defer rows.Close()
	cols := rows.Columns()

	values := make([]driver.Value, len(cols))
	tableData := make([][]string, 1)
	tableData[0] = cols
	idx := 0

	for {
		idx += 1
		err = rows.Next(values)
		if err != nil {
			break
		}
		tableData = append(tableData, make([]string, 0))
		for _, v := range values {
			tableData[idx] = append(tableData[idx], fmt.Sprintf("%v", v))
		}
	}
	if err != io.EOF {
		dieOnError("Can't Next", err)
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	return ""
}

func startProc(db dbT) string {
	conn, err := go_ora.NewConnection(fmt.Sprintf("oracle://%s:%s@%s:%v/%s", db.Login, db.Passw, db.Server, db.Port, db.Sid))
	dieOnError("Connection:", err)
	err = conn.Open()
	dieOnError("Open Connection:", err)
	defer conn.Close()

	stmt := go_ora.NewStmt("select procstart, procname from kp.v$monitor_menu", conn)
	defer stmt.Close()

	rows, err := stmt.Query(nil)
	dieOnError("Can't create query:", err)
	defer rows.Close()

	values := make([]driver.Value, len(rows.Columns()))

	for {
		err = rows.Next(values)
		if err != nil {
			break
		}

		pterm.FgWhite.Printf("Процессу \"%s\" отправлена команда на запуск\n", values[1])

		stmt1 := go_ora.NewStmt(fmt.Sprintf("begin %s end;", values[0]), conn)
		_, err = stmt1.Query(nil)
		dieOnError("Can't create query:", err)
		stmt1.Close()
	}
	if err != io.EOF {
		dieOnError("Can't Next", err)
	}

	return ""
}

func stopProc(db dbT) string {
	conn, err := go_ora.NewConnection(fmt.Sprintf("oracle://%s:%s@%s:%v/%s", db.Login, db.Passw, db.Server, db.Port, db.Sid))
	dieOnError("Connection:", err)
	err = conn.Open()
	dieOnError("Open Connection:", err)
	defer conn.Close()

	stmt := go_ora.NewStmt("select procstop, procname from kp.v$monitor_menu", conn)
	defer stmt.Close()

	rows, err := stmt.Query(nil)
	dieOnError("Can't create query:", err)
	defer rows.Close()

	values := make([]driver.Value, len(rows.Columns()))

	for {
		err = rows.Next(values)
		if err != nil {
			break
		}

		pterm.FgWhite.Printf("Процессу \"%s\" отправлена команда на остановку\n", values[1])

		stmt1 := go_ora.NewStmt(fmt.Sprintf("begin %s end;", values[0]), conn)
		_, err = stmt1.Query(nil)
		dieOnError("Can't create query:", err)
		stmt1.Close()
	}
	if err != io.EOF {
		dieOnError("Can't Next", err)
	}

	return ""
}

func locks(db dbT) string {
	conn, err := go_ora.NewConnection(fmt.Sprintf("oracle://%s:%s@%s:%v/%s", db.Login, db.Passw, db.Server, db.Port, db.Sid))
	dieOnError("Connection:", err)
	err = conn.Open()
	dieOnError("Open Connection:", err)
	defer conn.Close()

	stmt := go_ora.NewStmt(`select h1.audsid,
                                h1.sid,
                                h1.module
                            from dba_kgllock w,
                                dba_kgllock h,
                                v$session w1,
                                v$session h1
                            where h.kgllkmod not in ('0', '1')
                                and h.kgllkreq in (0, 1)
                                and w.kgllkmod in ('0', '1')
                                and w.kgllkreq != 0
                                and w.kgllkreq !='1'
                                and w.kgllktype = h.kgllktype
                                and w.kgllkhdl = h.kgllkhdl
                                and w.kgllkuse = w1.saddr
                                and h.kgllkuse = h1.saddr`, conn)
	defer stmt.Close()

	rows, err := stmt.Query(nil)
	dieOnError("Can't create query:", err)
	defer rows.Close()

	values := make([]driver.Value, len(rows.Columns()))

	for {
		err = rows.Next(values)
		if err != nil {
			break
		}

		sp, _ := pterm.DefaultSpinner.Start(pterm.FgWhite.Sprintf("Lock was detected. SID=%v, MODULE=%s\n", values[1], values[2]))

		stmt1 := go_ora.NewStmt(fmt.Sprintf("begin kp.pk_orasys.kill_session(%s); end;", values[0]), conn)
		_, err = stmt1.Query(nil)
		if err != nil {
			sp.Fail()
		} else {
			sp.Success()
		}
		stmt1.Close()
	}
	if err != io.EOF {
		dieOnError("Can't Next", err)
	}
	//
	stmt2 := go_ora.NewStmt(`select distinct o1.owner || '.' || o1.object_name TABLE_NAME
      ,l1.session_id
      ,s.AUDSID
      ,s.MODULE
  from dba_objects     o1
      ,dba_objects     o2
      ,v$locked_object l1
      ,v$locked_object l2
      ,v$session       s
 where o1.object_id = l1.object_id
   and o2.object_id = l2.object_id
   and l1.SESSION_ID != l2.SESSION_ID
   and o1.OBJECT_ID = o2.OBJECT_ID
   and s.sid = l1.SESSION_ID`, conn)
	defer stmt2.Close()

	rows2, err := stmt2.Query(nil)
	dieOnError("Can't create query:", err)
	defer rows2.Close()

	values = make([]driver.Value, len(rows2.Columns()))

	for {
		err = rows2.Next(values)
		if err != nil {
			break
		}

		sp, _ := pterm.DefaultSpinner.Start(pterm.FgWhite.Sprintf("Lock was detected. SID=%v TABLE=%s MODULE=%s\n", values[1], values[0], values[3]))

		stmt3 := go_ora.NewStmt(fmt.Sprintf("begin kp.pk_orasys.kill_session(%v); end;", values[2]), conn)
		_, err = stmt3.Query(nil)
		if err != nil {
			sp.Fail()
		} else {
			sp.Success()
		}
		stmt3.Close()
	}
	if err != io.EOF {
		dieOnError("Can't Next", err)
	}

	return ""
}

func queues(db dbT, pattern string) string {
	conn, err := go_ora.NewConnection(fmt.Sprintf("oracle://%s:%s@%s:%v/%s", db.Bm_login, db.Bm_passw, db.Server, db.Port, db.Sid))
	dieOnError("Connection:", err)
	err = conn.Open()
	dieOnError("Open Connection:", err)
	defer conn.Close()

	stmt := go_ora.NewStmt("delete from bm.creqs_tab c where c.q_name like '%'||:pattern||'%'", conn)
	defer stmt.Close()

	stmt.AddParam("pattern", strings.ToUpper(pattern), 0, go_ora.Input)
	res, err := stmt.Exec(nil)
	dieOnError("Exec:", err)

	rows, _ := res.RowsAffected()
	pterm.FgWhite.Printf("Удалено записей creqs: %d\n", rows)
	//

	stmt2 := go_ora.NewStmt("delete from bm.ireqs_tab c where c.q_name like '%'||:pattern||'%'", conn)
	defer stmt2.Close()

	stmt2.AddParam("pattern", strings.ToUpper(pattern), 0, go_ora.Input)
	res, err = stmt.Exec(nil)
	dieOnError("Exec:", err)

	rows, _ = res.RowsAffected()
	pterm.FgWhite.Printf("Удалено записей ireqs: %d\n", rows)

	return ""
}
