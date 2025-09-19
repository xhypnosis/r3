package csv_upload

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"r3/bruteforce"
	"r3/cache"
	"r3/config"
	"r3/data/data_import"
	"r3/db"
	"r3/handler"
	"r3/log"
	"r3/login/login_auth"
	"r3/tools"
	"r3/types"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
)

func Handler(w http.ResponseWriter, r *http.Request) {

	if blocked := bruteforce.Check(r); blocked {
		handler.AbortRequestNoLog(w, handler.ErrBruteforceBlock)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	reader, err := r.MultipartReader()
	if err != nil {
		handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
		return
	}

	// loop form reader until empty
	// fixed order: token, columns, lookups, joins, boolTrue, dateFormat,
	//  timezone, commaChar, ignoreHeader, file
	var token string
	var columns []types.Column
	var lookups []types.QueryLookup
	var joins []types.QueryJoin
	var boolTrue string
	var dateFormat string
	var timezone string
	var commaChar string
	var ignoreHeader bool

	res := struct {
		Count int    `json:"count"`
		Error string `json:"error"`
	}{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		switch part.FormName() {
		case "token":
			token = handler.GetStringFromPart(part)
		case "columns":
			if err := json.Unmarshal(handler.GetBytesFromPart(part), &columns); err != nil {
				handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
				return
			}
		case "joins":
			if err := json.Unmarshal(handler.GetBytesFromPart(part), &joins); err != nil {
				handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
				return
			}
		case "lookups":
			if err := json.Unmarshal(handler.GetBytesFromPart(part), &lookups); err != nil {
				handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
				return
			}
		case "boolTrue":
			boolTrue = handler.GetStringFromPart(part)
		case "dateFormat":
			dateFormat = handler.GetStringFromPart(part)
		case "timezone":
			timezone = handler.GetStringFromPart(part)
		case "commaChar":
			commaChar = handler.GetStringFromPart(part)
		case "ignoreHeader":
			ignoreHeader = handler.GetStringFromPart(part) == "true"
		}

		if part.FormName() != "file" {
			continue
		}

		ctx, ctxCanc := context.WithTimeout(context.Background(),
			time.Duration(int64(config.GetUint64("dbTimeoutCsv")))*time.Second)

		defer ctxCanc()

		// authenticate via token
		login, err := login_auth.Token(ctx, token)
		if err != nil {
			handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrUnauthorized)
			bruteforce.BadAttempt(r)
			return
		}

		// start work
		cache.Schema_mx.RLock()
		defer cache.Schema_mx.RUnlock()

		// store file in temporary directory
		filePath, err := tools.GetUniqueFilePath(config.File.Paths.Temp, 8999999, 9999999)
		if err != nil {
			handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
			return
		}

		dest, err := os.Create(filePath)
		if err != nil {
			handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
			return
		}
		defer os.Remove(filePath)
		defer dest.Close()

		if _, err := io.Copy(dest, part); err != nil {
			handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
			return
		}

		// read file
		res.Count, err = importFromCsv(ctx, filePath, login.Id, boolTrue, dateFormat,
			timezone, commaChar, ignoreHeader, columns, joins, lookups)

		if err != nil {
			err, expectedErr := handler.ConvertToErrCode(err, !login.Admin)
			res.Error = err.Error()

			if !expectedErr {
				handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
				return
			}
		}
	}

	resJson, err := json.Marshal(res)
	if err != nil {
		handler.AbortRequest(w, handler.ContextCsvUpload, err, handler.ErrGeneral)
		return
	}
	w.Write(resJson)
}

// import all lines from CSV, optionally skipping a header line
// returns to which line it got
func importFromCsv(ctx context.Context, filePath string, loginId int64, boolTrue string,
	dateFormat string, timezone string, commaChar string, ignoreHeader bool, columns []types.Column,
	joins []types.QueryJoin, lookups []types.QueryLookup) (int, error) {

	log.Info(log.ContextCsv, fmt.Sprintf("starts import from file '%s' via upload", filePath))

	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if err := db.SetSessionConfig_tx(ctx, tx, loginId); err != nil {
		return 0, err
	}

	// parse CSV file
	reader := csv.NewReader(file)
	reader.Comma, _ = utf8.DecodeRuneInString(commaChar)
	reader.Comment = '#'
	reader.FieldsPerRecord = len(columns)
	reader.TrimLeadingSpace = true
	ignoreHeaderDone := false
	importedCnt := 0

	// load user location based on timezone for datetime values
	locUser, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, err
	}

	// prepare relation index map for record lookups (unique PG indexes)
	indexMapPgIndexAttributeIds := data_import.ResolveQueryLookups(joins, lookups)

	for {
		values, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}

		if ignoreHeader && !ignoreHeaderDone {
			ignoreHeaderDone = true
			continue
		}

		log.Info(log.ContextCsv, fmt.Sprintf("is importing line %d", importedCnt+1))

		if err := importLine_tx(ctx, tx, loginId, boolTrue, dateFormat, locUser,
			values, columns, joins, lookups, indexMapPgIndexAttributeIds); err != nil {

			// still deliver number of imported lines, even though they were rolled back
			// reason: client needs to know which line was affected
			return importedCnt, err
		}
		importedCnt++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return importedCnt, nil
}

func importLine_tx(ctx context.Context, tx pgx.Tx, loginId int64,
	boolTrue string, dateFormat string, locUser *time.Location,
	valuesString []string, columns []types.Column, joins []types.QueryJoin,
	lookups []types.QueryLookup, indexMapPgIndexAttributeIds map[int][]uuid.UUID) error {

	if len(valuesString) != len(columns) {
		return errors.New("column and value count do not match")
	}

	var err error
	valuesIn := make([]interface{}, len(valuesString))

	// apply column value overwrites
	for i, column := range columns {

		atr, exists := cache.AttributeIdMap[column.AttributeId]
		if !exists {
			return handler.CreateErrCode(handler.ErrContextApp, handler.ErrCodeAppUnknownAttribute)
		}
		if atr.Encrypted {
			return handler.CreateErrCode(handler.ErrContextCsv, handler.ErrCodeCsvEncryptedAttribute)
		}

		if valuesString[i] == "" {
			// set to NULL if string value is empty
			valuesIn[i] = nil
			continue
		}

		// parse values if set
		switch atr.Content {
		case "integer", "bigint":

			switch atr.ContentUse {
			case "date", "datetime":
				// date values are always stored as UTC at midnight
				loc := time.UTC
				format := "2006-01-02"

				switch dateFormat {
				case "Y-m-d":
					format = "2006-01-02"
				case "Y/m/d":
					format = "2006/01/02"
				case "d.m.Y":
					format = "02.01.2006"
				case "d/m/Y":
					format = "02/01/2006"
				case "m/d/Y":
					format = "01/02/2006"
				}

				// datetime values are in context of user timezone
				if atr.ContentUse == "datetime" {
					loc = locUser
					format = fmt.Sprintf("%s 15:04:05", format)
				}

				t, err := time.ParseInLocation(format, valuesString[i], loc)
				if err != nil {
					return handler.CreateErrCodeWithData(handler.ErrContextCsv, handler.ErrCodeCsvParseDateTime, struct {
						Expect string `json:"expect"`
						Value  string `json:"value"`
					}{format, valuesString[i]})
				}
				valuesIn[i] = t.Unix()

			case "time":
				// time values are always stored at UTC zero date
				// UTC date must be set in parse string, otherwise year 0 will be used
				t, err := time.Parse("2006-01-02 15:04:05 MST",
					fmt.Sprintf("1970-01-01 %s UTC", valuesString[i]))

				if err != nil {
					return handler.CreateErrCodeWithData(handler.ErrContextCsv, handler.ErrCodeCsvParseDateTime, struct {
						Expect string `json:"expect"`
						Value  string `json:"value"`
					}{"15:04:05", valuesString[i]})
				}
				valuesIn[i] = t.Unix()
			default:
				valuesIn[i], err = strconv.ParseInt(valuesString[i], 10, 64)
				if err != nil {
					return handler.CreateErrCodeWithData(handler.ErrContextCsv, handler.ErrCodeCsvParseInt, struct {
						Value string `json:"value"`
					}{valuesString[i]})
				}
			}

		case "real", "double precision":
			valuesIn[i], err = strconv.ParseFloat(valuesString[i], 64)
			if err != nil {
				return handler.CreateErrCodeWithData(handler.ErrContextCsv, handler.ErrCodeCsvParseFloat, struct {
					Value string `json:"value"`
				}{valuesString[i]})
			}

		// numeric must be handled as text as conversion to float is not 1:1
		case "numeric", "text", "uuid", "varchar":
			valuesIn[i] = valuesString[i]

		case "boolean":
			valuesIn[i] = valuesString[i] == boolTrue

		case "default":
			return handler.CreateErrCodeWithData(handler.ErrContextCsv, handler.ErrCodeCsvBadAttributeType, struct {
				Value string `json:"value"`
			}{atr.Content})
		}
	}

	_, err = data_import.FromInterfaceValues_tx(ctx, tx, loginId,
		valuesIn, columns, joins, lookups, indexMapPgIndexAttributeIds)

	return err
}
