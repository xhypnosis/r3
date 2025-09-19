package openForm

import (
	"context"
	"errors"
	"fmt"
	"r3/schema"
	"r3/schema/compatible"
	"r3/types"
	"slices"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Get_tx(ctx context.Context, tx pgx.Tx, entity schema.DbEntity, id uuid.UUID, formContext pgtype.Text) (f types.OpenForm, err error) {

	if !slices.Contains(schema.DbAssignedOpenForm, entity) {
		return f, errors.New("invalid open form entity")
	}

	sqlArgs := make([]interface{}, 0)
	sqlArgs = append(sqlArgs, id)

	sqlWhere := "AND context IS NULL"
	if formContext.Valid {
		sqlArgs = append(sqlArgs, formContext.String)
		sqlWhere = "AND context = $2"
	}

	err = tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT form_id_open, relation_index_open, attribute_id_apply,
			relation_index_apply, pop_up_type, max_height, max_width
		FROM app.open_form
		WHERE %s_id = $1
		%s
	`, entity, sqlWhere), sqlArgs...).Scan(&f.FormIdOpen, &f.RelationIndexOpen,
		&f.AttributeIdApply, &f.RelationIndexApply, &f.PopUpType, &f.MaxHeight,
		&f.MaxWidth)

	// open form is optional
	if err == pgx.ErrNoRows {
		return f, nil
	}

	// fix exports > 3.4: Set default value for legacy relation index
	f = compatible.FixOpenFormRelationIndexApplyDefault(f)

	return f, err
}

func Set_tx(ctx context.Context, tx pgx.Tx, entity schema.DbEntity, id uuid.UUID, f types.OpenForm, context pgtype.Text) error {

	if !slices.Contains(schema.DbAssignedOpenForm, entity) {
		return errors.New("invalid open form entity")
	}

	// fix imports < 3.4: Legacy pop-up option
	f = compatible.FixOpenFormPopUpType(f)

	// fix imports < 3.5: Relation index for applying record relationship value
	f = compatible.FixOpenFormRelationIndexApply(f)

	sqlArgs := make([]interface{}, 0)
	sqlArgs = append(sqlArgs, id)

	sqlWhere := "AND context IS NULL"
	if context.Valid {
		sqlArgs = append(sqlArgs, context.String)
		sqlWhere = "AND context = $2"
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		DELETE FROM app.open_form
		WHERE %s_id = $1
		%s
	`, entity, sqlWhere), sqlArgs...); err != nil {
		return err
	}

	if f.FormIdOpen == uuid.Nil {
		return nil
	}

	_, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO app.open_form (
			%s_id, context, form_id_open, relation_index_open, attribute_id_apply,
			relation_index_apply, pop_up_type, max_height, max_width
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, entity), id, context, f.FormIdOpen, f.RelationIndexOpen, f.AttributeIdApply,
		f.RelationIndexApply, f.PopUpType, f.MaxHeight, f.MaxWidth)

	return err
}
