package transfer_export

import (
	"context"
	"errors"
	"net/http"
	"os"
	"r3/config"
	"r3/db"
	"r3/handler"
	"r3/log"
	"r3/login/login_auth"
	"r3/tools"
	"r3/transfer"
)

var genErr = "could not finish module export"

func Handler(w http.ResponseWriter, r *http.Request) {

	// get authentication token
	token, err := handler.ReadGetterFromUrl(r, "token")
	if err != nil {
		log.Error(log.ContextServer, genErr, err)
		return
	}

	ctx, ctxCanc := context.WithTimeout(context.Background(), db.CtxDefTimeoutTransfer)
	defer ctxCanc()

	// authenticate via token
	login, err := login_auth.Token(ctx, token)
	if err != nil {
		log.Error(log.ContextServer, genErr, err)
		return
	}

	if !login.Admin {
		log.Error(log.ContextServer, genErr, errors.New(handler.ErrUnauthorized))
		return
	}

	// get module ID
	moduleId, err := handler.ReadUuidGetterFromUrl(r, "module_id")
	if err != nil {
		log.Error(log.ContextServer, genErr, err)
		return
	}

	filePath, err := tools.GetUniqueFilePath(config.File.Paths.Temp, 8999999, 9999999)
	if err != nil {
		log.Error(log.ContextServer, genErr, err)
		return
	}

	if err := transfer.ExportToFile(ctx, moduleId, filePath); err != nil {
		log.Error(log.ContextServer, genErr, err)
		return
	}
	http.ServeFile(w, r, filePath)
	if err := os.Remove(filePath); err != nil {
		log.Warning(log.ContextServer, "could not delete temporary export file", err)
	}
}
