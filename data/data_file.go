package data

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"r3/cache"
	"r3/config"
	"r3/data/data_image"
	"r3/db"
	"r3/handler"
	"r3/schema"
	"r3/tools"
	"r3/types"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	newFileUnnamed = "[UNNAMED]"

	// finds: '17' from file names such as 'my_file_(17).jpg'
	regexRenameSchema = regexp.MustCompile(`_\((\d+)\)`)
)

func MayAccessFile(loginId int64, attributeId uuid.UUID) error {
	cache.Schema_mx.RLock()
	defer cache.Schema_mx.RUnlock()

	attribute, exists := cache.AttributeIdMap[attributeId]
	if !exists || !schema.IsContentFiles(attribute.Content) {
		return errors.New("not a file attribute")
	}

	if !authorizedAttribute(loginId, attributeId, types.AccessRead) {
		return errors.New(handler.ErrUnauthorized)
	}
	return nil
}

// returns path to downloadable file, a specific version, its directory or thumbnail
func GetFilePathDir(fileId uuid.UUID) string {
	return filepath.Join(config.File.Paths.Files, fileId.String()[:3])
}
func GetFilePathThumb(fileId uuid.UUID) string {
	return filepath.Join(config.File.Paths.Files, fileId.String()[:3],
		fmt.Sprintf("%s.webp", fileId.String()))
}
func GetFilePathVersion(fileId uuid.UUID, version int64) string {
	return filepath.Join(config.File.Paths.Files, fileId.String()[:3],
		fmt.Sprintf("%s_%d", fileId.String(), version))
}

// attempts to store file upload
func SetFile(ctx context.Context, loginId int64, attributeId uuid.UUID, fileId uuid.UUID,
	fileSourcePart *multipart.Part, fileSourcePath pgtype.Text, fileSourceString pgtype.Text, isNewFile bool) error {

	cache.Schema_mx.RLock()
	attribute, exists := cache.AttributeIdMap[attributeId]
	if !exists || !schema.IsContentFiles(attribute.Content) {
		cache.Schema_mx.RUnlock()
		return handler.ErrSchemaUnknownAttribute(attributeId)
	}
	cache.Schema_mx.RUnlock()

	// check for access permissions, unless it´s a system task (login ID = -1)
	if loginId != -1 && !authorizedAttribute(loginId, attributeId, types.AccessWrite) {
		return errors.New(handler.ErrUnauthorized)
	}

	// if existing file: check latest version and currently assigned records
	var recordIds []int64
	var version int64 = 0
	if !isNewFile {
		if err := db.Pool.QueryRow(ctx, fmt.Sprintf(`
			SELECT v.version+1, (
				SELECT ARRAY_AGG(r.record_id)
				FROM instance_file."%s" AS r
				WHERE r.file_id = v.file_id
			)
			FROM instance.file_version AS v
			WHERE v.file_id = $1
			ORDER BY v.version DESC
			LIMIT 1
		`, schema.GetFilesTableName(attributeId)),
			fileId).Scan(&version, &recordIds); err != nil {

			return err
		}
	}

	if err := tools.PathCreateIfNotExists(GetFilePathDir(fileId), 0700); err != nil {
		return err
	}

	fileName := ""
	filePath := GetFilePathVersion(fileId, version)

	// move/copy file from its source to its final target file path
	if fileSourcePart != nil {

		// write file from multipart form
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(file, fileSourcePart); err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		fileName = fileSourcePart.FileName()

	} else if fileSourcePath.Valid {

		// move file from file path
		stat, err := os.Stat(fileSourcePath.String)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			return fmt.Errorf("file path '%s' is a directory", fileSourcePath.String)
		}
		if err := tools.FileMove(fileSourcePath.String, filePath, false); err != nil {
			return err
		}
		fileName = filepath.Base(fileSourcePath.String)

	} else if fileSourceString.Valid {

		// write file from string
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		bufWriter := bufio.NewWriter(file)
		if _, err := bufWriter.WriteString(fileSourceString.String); err != nil {
			return err
		}
		if err := bufWriter.Flush(); err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("failed to set file, no file source defined")
	}

	// check size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	fileSizeKb := int64(fileInfo.Size() / 1024)

	if attribute.Length != 0 && fileSizeKb > int64(attribute.Length) {
		return errors.New("file size limit reached")
	}

	// get file hash
	hash, err := tools.GetFileHash(filePath)
	if err != nil {
		return err
	}

	// create/update thumbnail - failure should not block progress
	data_image.CreateThumbnail(fileId, filepath.Ext(fileName), filePath,
		GetFilePathThumb(fileId), false)

	// store file meta data in database
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := FileApplyVersion_tx(ctx, tx, isNewFile, attributeId, attribute.RelationId,
		fileId, hash, fileName, fileSizeKb, version, recordIds, loginId); err != nil {

		return err
	}
	return tx.Commit(ctx)
}

// stores database changes for uploaded/updated files
func FileApplyVersion_tx(ctx context.Context, tx pgx.Tx, isNewFile bool, attributeId uuid.UUID,
	relationId uuid.UUID, fileId uuid.UUID, fileHash string, fileName string,
	fileSizeKb int64, fileVersion int64, recordIds []int64, loginId int64) error {

	if isNewFile {
		// store file reference
		if _, err := tx.Exec(ctx, `
			INSERT INTO instance.file (id, ref_counter) VALUES ($1,0)
		`, fileId); err != nil {
			return err
		}
	}

	// store file version reference
	loginNull := pgtype.Int4{
		Int32: int32(loginId),
		Valid: loginId != -1,
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instance.file_version (
			file_id,version,login_id,hash,size_kb,date_change)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, fileId, fileVersion, loginNull, fileHash, fileSizeKb, tools.GetTimeUnix()); err != nil {
		return err
	}

	// skip change log if new file or file is not attached to any record
	// new file change logs are stored when record is saved
	if isNewFile || len(recordIds) == 0 {
		return nil
	}

	cache.Schema_mx.RLock()
	relation, exists := cache.RelationIdMap[relationId]
	if !exists {
		cache.Schema_mx.RUnlock()
		return handler.ErrSchemaUnknownRelation(relationId)
	}
	cache.Schema_mx.RUnlock()

	if !relationUsesLogging(relation.RetentionCount, relation.RetentionDays) {
		return nil
	}

	logAttributes := []types.DataSetAttribute{
		{
			AttributeId:   attributeId,
			AttributeIdNm: pgtype.UUID{},
			Value: types.DataSetFileChanges{
				FileIdMapChange: map[uuid.UUID]types.DataSetFileChange{
					fileId: {
						Action:  "update",
						Name:    fileName,
						Version: fileVersion,
					},
				},
			},
		},
	}
	logAttributeFileIndexes := []int{0}
	logValuesOld := []interface{}{nil}

	for _, recordId := range recordIds {
		if err := setLog_tx(ctx, tx, relationId, logAttributes,
			logAttributeFileIndexes, false, logValuesOld, recordId,
			loginId); err != nil {

			return err
		}
	}
	return nil
}

// updates file record assignment or deletion state based on file attribute changes
func FilesApplyAttributChanges_tx(ctx context.Context,
	tx pgx.Tx, recordId int64, attributeId uuid.UUID,
	fileIdMapChange map[uuid.UUID]types.DataSetFileChange) error {

	tNameR := schema.GetFilesTableName(attributeId)

	// apply created & deleted files
	fileIdsCreated := make([]uuid.UUID, 0)
	fileIdsDeleted := make([]uuid.UUID, 0)

	for fileId, change := range fileIdMapChange {
		switch change.Action {
		case "create":
			fileIdsCreated = append(fileIdsCreated, fileId)
		case "delete":
			fileIdsDeleted = append(fileIdsDeleted, fileId)
		}
	}

	if len(fileIdsCreated) != 0 {
		if err := FilesAssignToRecord_tx(ctx, tx, attributeId, fileIdsCreated, recordId); err != nil {
			return err
		}
	}
	if len(fileIdsDeleted) != 0 {
		if err := FilesSetDeletedForRecord_tx(ctx, tx, attributeId, fileIdsDeleted, recordId); err != nil {
			return err
		}
	}

	// execute file rename actions after files were assigned to records (created files)
	// rename is dependent on other files assigned to the same record
	for fileId, change := range fileIdMapChange {
		if (change.Action == "create" || change.Action == "rename") && change.Name != "" {

			// trim spaces
			change.Name = strings.TrimSpace(change.Name)

			// check whether name is free to use within context of all assigned records
			nameCandidate := change.Name
			nameExt := filepath.Ext(nameCandidate)
			nameExists := false

			for attempts := 0; attempts < 20; attempts++ {
				if err := tx.QueryRow(ctx, fmt.Sprintf(`
					SELECT EXISTS(
						SELECT file_id
						FROM instance_file."%s"
						WHERE record_id   =  $1
						AND   file_id     <> $2
						AND   name        =  $3
						AND   date_delete IS NULL
					)
				`, tNameR), recordId, fileId, nameCandidate).Scan(&nameExists); err != nil {
					return err
				}

				if !nameExists {
					// name not taken yet, can be used
					break
				}
				nameOnly := strings.Replace(nameCandidate, nameExt, "", -1)

				// check if taken name already has rename schema
				matches := regexRenameSchema.FindStringSubmatch(nameOnly)
				if len(matches) == 2 {
					// schema used, increment counter and try again
					counter, err := strconv.ParseInt(matches[1], 10, 0)
					if err != nil {
						return err
					}

					nameCandidate = regexRenameSchema.ReplaceAllString(nameOnly,
						fmt.Sprintf("_(%d)%s", counter+1, nameExt))

					continue
				}

				// schema not used, apply it by adding counter
				nameCandidate = fmt.Sprintf("%s_(1)%s", nameOnly, nameExt)
			}

			if nameExists {
				// name is still taken, abort rename
				continue
			}

			if _, err := tx.Exec(ctx, fmt.Sprintf(`
				UPDATE instance_file."%s"
				SET   name      = $1
				WHERE file_id   = $2
				AND   record_id = $3
			`, tNameR), nameCandidate, fileId, recordId); err != nil {
				return err
			}
		}
	}
	return nil
}

func FilesAssignToRecord_tx(ctx context.Context, tx pgx.Tx,
	attributeId uuid.UUID, fileIds []uuid.UUID, recordId int64) error {

	for _, fileId := range fileIds {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO instance_file."%s" (file_id, record_id, name)
			VALUES ($1,$2,$3)
		`, schema.GetFilesTableName(attributeId)), fileId,
			recordId, newFileUnnamed); err != nil {

			return err
		}
	}
	return nil
}

func FilesSetDeletedForRecord_tx(ctx context.Context, tx pgx.Tx,
	attributeId uuid.UUID, fileIds []uuid.UUID, recordId int64) error {

	_, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE instance_file."%s"
		SET date_delete = $1
		WHERE record_id = $2
		AND   file_id   = ANY($3)
	`, schema.GetFilesTableName(attributeId)), tools.GetTimeUnix(), recordId, fileIds)
	return err
}

func FileGetLatestVersion(fileId uuid.UUID) (int64, error) {
	ctx, ctxCanc := context.WithTimeout(context.Background(), db.CtxDefTimeoutSysTask)
	defer ctxCanc()

	var version int64
	err := db.Pool.QueryRow(ctx, `
		SELECT MAX(version)
		FROM instance.file_version
		WHERE file_id = $1
	`, fileId).Scan(&version)
	return version, err
}
