package log

import (
	"context"
	"fmt"
	"r3/db"
	"r3/tools"
	"r3/types"
	"sync"
	"sync/atomic"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type logContext int

const (
	// log contexts
	ContextApi       logContext = 10
	ContextBackup    logContext = 20
	ContextCache     logContext = 30
	ContextCluster   logContext = 40
	ContextCsv       logContext = 50
	ContextFile      logContext = 60
	ContextImager    logContext = 70
	ContextLdap      logContext = 80
	ContextMail      logContext = 90
	ContextModule    logContext = 100 // only used via instance logging functions
	ContextOauth     logContext = 110
	ContextScheduler logContext = 120
	ContextServer    logContext = 130
	ContextTransfer  logContext = 140
	ContextWebsocket logContext = 150
)

var (
	access_mx = sync.RWMutex{}
	nodeId    = pgtype.UUID{} // ID of the current node

	// simple options, accessible without lock
	debug     atomic.Bool
	outputCli atomic.Bool // write logs also to command line

	// log levels: 1 = errors, 2 = errors + warning, 3 = everything
	logContextLevel = map[logContext]int{
		ContextApi:       1,
		ContextBackup:    1,
		ContextCache:     1,
		ContextCluster:   1,
		ContextCsv:       1,
		ContextFile:      1,
		ContextImager:    1,
		ContextLdap:      1,
		ContextMail:      1,
		ContextModule:    1,
		ContextOauth:     1,
		ContextScheduler: 1,
		ContextServer:    1,
		ContextTransfer:  1,
		ContextWebsocket: 1,
	}
	logContextName = map[logContext]string{
		ContextApi:       "api",
		ContextBackup:    "backup",
		ContextCache:     "cache",
		ContextCluster:   "cluster",
		ContextCsv:       "csv",
		ContextFile:      "file",
		ContextImager:    "imager",
		ContextLdap:      "ldap",
		ContextMail:      "mail",
		ContextModule:    "module",
		ContextOauth:     "oauth",
		ContextScheduler: "scheduler",
		ContextServer:    "server",
		ContextTransfer:  "transfer",
		ContextWebsocket: "websocket",
	}
)

func Get_tx(ctx context.Context, tx pgx.Tx, dateFrom pgtype.Int8, dateTo pgtype.Int8, limit int, offset int,
	context string, byString string) ([]types.Log, int, error) {

	logs := make([]types.Log, 0)
	total := 0

	var qb tools.QueryBuilder
	qb.UseDollarSigns()
	qb.AddList("SELECT", []string{"l.level", "l.context", "l.message", "l.date_milli", "COALESCE(m.name,'-')", "n.name"})
	qb.SetFrom("instance.log AS l")
	qb.Add("JOIN", "LEFT JOIN app.module AS m ON m.id = l.module_id")
	qb.Add("JOIN", "LEFT JOIN instance_cluster.node AS n ON n.id = l.node_id")

	if context != "" {
		qb.Add("WHERE", `l.context::TEXT = {CONTEXT}`)
		qb.AddPara("{CONTEXT}", context)
	}

	if byString != "" {
		qb.Add("WHERE", `(
			l.message ILIKE {NAME} OR
			m.name    ILIKE {NAME}
		)`)
		qb.AddPara("{NAME}", fmt.Sprintf("%%%s%%", byString))
	}

	if dateFrom.Valid {
		qb.Add("WHERE", "l.date_milli >= {DATEFROM}")
		qb.AddPara("{DATEFROM}", dateFrom.Int64*1000)
	}

	if dateTo.Valid {
		qb.Add("WHERE", "l.date_milli <= {DATETO}")
		qb.AddPara("{DATETO}", dateTo.Int64*1000)
	}

	qb.Add("ORDER", "l.date_milli DESC")
	qb.SetOffset(offset)
	qb.SetLimit(limit)

	query, err := qb.GetQuery()
	if err != nil {
		return nil, 0, err
	}

	rows, err := tx.Query(ctx, query, qb.GetParaValues()...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var l types.Log
		var dateMilli int64

		if err := rows.Scan(&l.Level, &l.Context, &l.Message, &dateMilli, &l.ModuleName, &l.NodeName); err != nil {
			return nil, 0, err
		}
		l.Date = int64(dateMilli / 1000)
		logs = append(logs, l)
	}

	// get total count
	qb.UseDollarSigns()
	qb.Reset("SELECT")
	qb.Reset("ORDER")
	qb.Reset("LIMIT")
	qb.Reset("OFFSET")
	qb.Add("SELECT", "COUNT(*)")

	query, err = qb.GetQuery()
	if err != nil {
		return nil, 0, err
	}

	if err := tx.QueryRow(ctx, query, qb.GetParaValues()...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func SetDebug(state bool) {
	debug.Store(state)
}
func SetOutputCli(state bool) {
	outputCli.Store(state)
}
func SetLogLevel(context logContext, level int) {
	access_mx.Lock()
	defer access_mx.Unlock()

	if _, exists := logContextLevel[context]; exists {
		logContextLevel[context] = level
	}
}
func SetNodeId(id uuid.UUID) {
	access_mx.Lock()
	nodeId.Bytes = id
	nodeId.Valid = true
	access_mx.Unlock()
}

func Info(context logContext, message string) {
	go write(3, context, message, nil)
}
func Warning(context logContext, message string, err error) {
	go write(2, context, message, err)
}
func Error(context logContext, message string, err error) {
	go write(1, context, message, err)
}

func write(level int, logContext logContext, message string, err error) {
	access_mx.RLock()
	nodeIdLocal := nodeId
	levelActive, exists := logContextLevel[logContext]
	access_mx.RUnlock()

	if !exists {
		return
	}

	if !debug.Load() && level > levelActive {
		return
	}

	// append error message, if available
	if err != nil {
		if message != "" {
			message = fmt.Sprintf("%s, %s", message, err.Error())
		} else {
			message = err.Error()
		}
	}

	// log to CLI if available
	if outputCli.Load() {
		fmt.Printf("%s %s %s\n", tools.GetTimeSql(), logContextName[logContext], message)
	}

	// log to database if available
	if db.Pool != nil {

		// reduce message size stored in DB to at most 10k chars
		// if access to larger messages is required, use CLI
		if len(message) > 10000 {
			message = message[:10000]
		}

		ctx, ctxCanc := context.WithTimeout(context.Background(), db.CtxDefTimeoutLogWrite)
		defer ctxCanc()

		if _, err := db.Pool.Exec(ctx, `
			INSERT INTO instance.log (level, context, message, date_milli, node_id)
			VALUES ($1,$2,$3,$4,$5)
		`, level, logContextName[logContext], message, tools.GetTimeUnixMilli(), nodeIdLocal); err != nil {

			// if database logging fails, output error to CLI if available
			if outputCli.Load() {
				fmt.Printf("failed to write log to DB, error: %v\n", err)
			}
		}
	}
}
