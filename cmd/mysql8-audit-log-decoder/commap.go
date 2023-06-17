package main

import (
	"fmt"

	"github.com/go-mysql-org/go-mysql/mysql"
)

var (
	COMMap = map[byte]string{
		mysql.COM_SLEEP:               "COM_SLEEP",
		mysql.COM_QUIT:                "COM_QUIT",
		mysql.COM_INIT_DB:             "COM_INIT_DB",
		mysql.COM_QUERY:               "COM_QUERY",
		mysql.COM_FIELD_LIST:          "COM_FIELD_LIST",
		mysql.COM_CREATE_DB:           "COM_CREATE_DB",
		mysql.COM_DROP_DB:             "COM_DROP_DB",
		mysql.COM_REFRESH:             "COM_REFRESH",
		mysql.COM_SHUTDOWN:            "COM_SHUTDOWN",
		mysql.COM_STATISTICS:          "COM_STATISTICS",
		mysql.COM_PROCESS_INFO:        "COM_PROCESS_INFO",
		mysql.COM_CONNECT:             "COM_CONNECT",
		mysql.COM_PROCESS_KILL:        "COM_PROCESS_KILL",
		mysql.COM_DEBUG:               "COM_DEBUG",
		mysql.COM_PING:                "COM_PING",
		mysql.COM_TIME:                "COM_TIME",
		mysql.COM_DELAYED_INSERT:      "COM_DELAYED_INSERT",
		mysql.COM_CHANGE_USER:         "COM_CHANGE_USER",
		mysql.COM_BINLOG_DUMP:         "COM_BINLOG_DUMP",
		mysql.COM_TABLE_DUMP:          "COM_TABLE_DUMP",
		mysql.COM_CONNECT_OUT:         "COM_CONNECT_OUT",
		mysql.COM_REGISTER_SLAVE:      "COM_REGISTER_SLAVE",
		mysql.COM_STMT_PREPARE:        "COM_STMT_PREPARE",
		mysql.COM_STMT_EXECUTE:        "COM_STMT_EXECUTE",
		mysql.COM_STMT_SEND_LONG_DATA: "COM_STMT_SEND_LONG_DATA",
		mysql.COM_STMT_CLOSE:          "COM_STMT_CLOSE",
		mysql.COM_STMT_RESET:          "COM_STMT_RESET",
		mysql.COM_SET_OPTION:          "COM_SET_OPTION",
		mysql.COM_STMT_FETCH:          "COM_STMT_FETCH",
		mysql.COM_DAEMON:              "COM_DAEMON",
		mysql.COM_BINLOG_DUMP_GTID:    "COM_BINLOG_DUMP_GTID",
		mysql.COM_RESET_CONNECTION:    "COM_RESET_CONNECTION",
	}
)

func GetComName(com byte) string {
	s, ok := COMMap[com]
	if ok {
		return s
	}
	return fmt.Sprintf("UNKNOWN:%x", com)
}
