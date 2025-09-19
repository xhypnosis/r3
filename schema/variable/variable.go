package variable

import (
	"context"
	"r3/schema"
	"r3/types"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
)

func Del_tx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM app.variable WHERE id = $1`, id)
	return err
}

func Get_tx(ctx context.Context, tx pgx.Tx, moduleId uuid.UUID) ([]types.Variable, error) {

	variables := make([]types.Variable, 0)
	rows, err := tx.Query(ctx, `
		SELECT v.id, v.form_id, v.name, v.comment, v.content, v.content_use, v.def
		FROM      app.variable  AS v
		LEFT JOIN app.form      AS f ON f.id = v.form_id
		WHERE v.module_id = $1
		ORDER BY
			f.name ASC NULLS FIRST,
			v.name ASC
	`, moduleId)
	if err != nil {
		return variables, err
	}
	defer rows.Close()

	for rows.Next() {
		var v types.Variable
		v.ModuleId = moduleId
		if err := rows.Scan(&v.Id, &v.FormId, &v.Name, &v.Comment, &v.Content, &v.ContentUse, &v.Def); err != nil {
			return variables, err
		}
		variables = append(variables, v)
	}
	return variables, nil
}

func Set_tx(ctx context.Context, tx pgx.Tx, v types.Variable) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &v.Id, schema.DbVariable, "id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.variable
			SET name = $1, comment = $2, content = $3, content_use = $4, def = $5
			WHERE id = $6
		`, v.Name, v.Comment, v.Content, v.ContentUse, v.Def, v.Id); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.variable (id, module_id, form_id, name, comment, content, content_use, def)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, v.Id, v.ModuleId, v.FormId, v.Name, v.Comment, v.Content, v.ContentUse, v.Def); err != nil {
			return err
		}
	}
	return nil
}
