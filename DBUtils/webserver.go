// API DBUtils
//
// Набор скриптов типовых операций с БД
//
//     Schemes: http
//     BasePath: /api
//     Version: 0.0.1
//     License: MIT http://opensource.org/licenses/MIT
//     Contact: Pavel Degtyarev <p.degtyarev@cft.ru>
//
//     Produces:
//     - application/json
//
// swagger:meta
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func webServer(config cfgFileT) {
	router := gin.Default()
	router.SetTrustedProxies(nil)
	// GET's
	router.GET("/api/:database/version", func(context *gin.Context) {
		// swagger:route GET /api/{database}/version getVersion
		//
		// Получение версии и даты наката
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webStandartCommand(DBVersion, config.Databases[context.Param("database")], context)
	})
	router.GET("/api/:database/queues", func(context *gin.Context) {
		// swagger:route GET /api/{database}/queues очереди getQueues
		//
		// Список очередей с фильтрами по параметрам
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//       + name: name
		//         in: query
		//         description: фильтрация по части имени очереди
		//         required: false
		//         type: string
		//       + name: adapter_id
		//         in: query
		//         description: фильтрация точному совпадению adapter_id (S# или P#)
		//         required: false
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webQueuesCommand(DBInfoQueues, config.Databases[context.Param("database")], context)
	})
	router.GET("/api/:database/processes", func(context *gin.Context) {
		// swagger:route GET /api/{database}/processes процессы getProcesses
		//
		// Список процессов и их состояний
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webStandartCommand(DBProcStat, config.Databases[context.Param("database")], context)
	})
	router.GET("/api/{database}/locks", func(context *gin.Context) {
		// swagger:route GET /api/{database}/locks блокировки getLocks
		//
		// Список блокировок объектов БД
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webStandartCommand(DBViewLocks, config.Databases[context.Param("database")], context)
	})
	// DELETE's
	router.DELETE("/api/{database}/queues", func(context *gin.Context) {
		// swagger:route DELETE /api/{database}/queues очереди clearQueues
		//
		// Очистка очередей с фильтрами по параметрам
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//       + name: name
		//         in: query
		//         description: фильтрация по части имени очереди
		//         required: false
		//         type: string
		//       + name: adapter_id
		//         in: query
		//         description: фильтрация точному совпадению adapter_id (S# или P#)
		//         required: false
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webQueuesCommand(DBClearQueues, config.Databases[context.Param("database")], context)
	})
	router.DELETE("/api/:database/locks", func(context *gin.Context) {
		// swagger:route DELETE /api/{database}/locks блокировки releaseLocks
		//
		// Разрешение блокировок объектов БД
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webStandartCommand(DBReleaseLocks, config.Databases[context.Param("database")], context)
	})
	// PUT's
	router.PUT("/api/:database/processes/start", func(context *gin.Context) {
		// swagger:route PUT /api/{database}/processes процессы startProcesses
		//
		// Запуск процессов
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//       + name: short
		//         in: query
		//         description: procshort
		//         required: false
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webProcessesCommand(DBProcStart, config.Databases[context.Param("database")], context)
	})
	router.PUT("/api/:database/processes/stop", func(context *gin.Context) {
		// swagger:route PUT /api/{database}/processes процессы stopProcesses
		//
		// Остановка процессов
		//
		//     Parameters:
		//       + name: database
		//         in: path
		//         description: имя БД из config.yml
		//         required: true
		//         type: string
		//       + name: short
		//         in: query
		//         description: procshort
		//         required: false
		//         type: string
		//
		//     Responses:
		//       200: tables
		//       500: error
		webProcessesCommand(DBProcStop, config.Databases[context.Param("database")], context)
	})
	if config.Webserver.Port != 0 {
		router.Run(fmt.Sprintf(":%v", config.Webserver.Port))
	} else {
		router.Run()
	}
}

// Обертка стандартной команды (без параметров)
func webStandartCommand(
	function func(dbT, chan<- interface{}),
	dbConfig dbT,
	context *gin.Context) {
	//
	c := make(chan interface{})
	go function(dbConfig, c)
	json, err := webPrint(c)
	if err != nil {
		context.JSON(http.StatusInternalServerError, json)
	} else {
		context.JSON(http.StatusOK, json)
	}
}

// Обертка работы с очередями (необязательные параметры: name, adapter_id)
func webQueuesCommand(
	function func(dbT, chan<- interface{}, string, string),
	dbConfig dbT,
	context *gin.Context) {
	//
	c := make(chan interface{})
	go function(dbConfig, c, context.Query("name"), context.Query("adapter_id"))
	json, err := webPrint(c)
	if err != nil {
		context.JSON(http.StatusInternalServerError, json)
	} else {
		context.JSON(http.StatusOK, json)
	}
}

// Обертка работы с процессом (необязательный параметр short)
func webProcessesCommand(
	function func(dbT, chan<- interface{}, string),
	dbConfig dbT,
	context *gin.Context) {
	//
	c := make(chan interface{})
	go function(dbConfig, c, context.Query("short"))
	json, err := webPrint(c)
	if err != nil {
		context.JSON(http.StatusInternalServerError, json)
	} else {
		context.JSON(http.StatusOK, json)
	}
}

func webPrint(c <-chan interface{}) (result interface{}, err error) {
	var tables TableSlice
	tables.Tables = make([]TableS, 0)
	for val := range c {
		switch v := val.(type) {
		case TableS:
			tables.Tables = append(tables.Tables, v)
		case error:
			err = v
			result = ErrorS{fmt.Sprintf("%v", v)}
			return
		case nil: // nil - это отсутствие ошибки, пропускаем
		default:
			log.Fatalf("I don't know about type %T!\n", v)
		}
	}
	result = tables
	return
}
