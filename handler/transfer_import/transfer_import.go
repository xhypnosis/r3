package transfer_import

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

func Handler(res http.ResponseWriter, req *http.Request) {

	res.Header().Set("Content-Type", "application/json")

	finishRequest := func(err error) {

		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			log.Error(log.ContextServer, "could not finish module import", err)
		}

		var response struct {
			Success bool `json:"success"`
		}
		response.Success = err == nil

		responseJson, err := json.Marshal(response)
		if err != nil {
			log.Error(log.ContextServer, "could not finish module import", err)
			res.Write([]byte{})
			return
		}
		res.Write(responseJson)
		return
	}

	reader, err := req.MultipartReader()
	if err != nil {
		finishRequest(err)
		return
	}

	// loop form reader until empty
	// fixed order: token first, then file
	var token string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		switch part.FormName() {
		case "token":
			buf := new(bytes.Buffer)
			buf.ReadFrom(part)
			token = buf.String()
			continue
		}

		ctx, ctxCanc := context.WithTimeout(context.Background(), db.CtxDefTimeoutTransfer)
		defer ctxCanc()

		// authenticate via token
		// check token
		login, err := login_auth.Token(ctx, token)
		if err != nil {
			finishRequest(err)
			return
		}

		if !login.Admin {
			finishRequest(errors.New(handler.ErrUnauthorized))
			return
		}

		// set file path in temporary directory
		filePath, err := tools.GetUniqueFilePath(config.File.Paths.Temp, 8999999, 9999999)
		if err != nil {
			finishRequest(err)
			return
		}

		// create file
		dest, err := os.Create(filePath)
		if err != nil {
			finishRequest(err)
			return
		}
		defer os.Remove(filePath)
		defer dest.Close()

		if _, err := io.Copy(dest, part); err != nil {
			finishRequest(err)
			return
		}

		if err := transfer.ImportFromFiles(ctx, []string{filePath}); err != nil {
			finishRequest(err)
			return
		}
	}
	finishRequest(nil)
}
