package main

// Массив таблиц
// swagger:response tables
type TableSlice struct {
	// таблицы
	// in: body
	Tables []TableS `json:"tables"`
}

// таблица
type TableS struct {
	// заголовок таблицы
	Caption string `json:"caption"`
	// список столбцов
	Header []string `json:"header"`
	// массив строк
	Rows [][]string `json:"rows"`
}

// Сообщение об ошибке
// swagger:response error
type ErrorS struct {
	// текст ошибки
	// in: body
	Error string `json:"error"`
}
