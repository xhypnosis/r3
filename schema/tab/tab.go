package tab

import (
	"context"
	"errors"
	"fmt"
	"r3/schema"
	"r3/schema/caption"
	"r3/types"
	"slices"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
)

func Del_tx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM app.tab WHERE id = $1`, id)
	return err
}

func Get_tx(ctx context.Context, tx pgx.Tx, entity schema.DbEntity, entityId uuid.UUID) ([]types.Tab, error) {
	tabs := make([]types.Tab, 0)

	if !slices.Contains(schema.DbAssignedTab, entity) {
		return tabs, errors.New("bad entity")
	}

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT id, content_counter, state
		FROM app.tab
		WHERE %s_id = $1
		ORDER BY position ASC
	`, entity), entityId)
	if err != nil {
		return tabs, err
	}
	defer rows.Close()

	for rows.Next() {
		var t types.Tab
		if err := rows.Scan(&t.Id, &t.ContentCounter, &t.State); err != nil {
			return tabs, err
		}
		tabs = append(tabs, t)
	}

	for i, tab := range tabs {
		tabs[i].Captions, err = caption.Get_tx(ctx, tx, schema.DbTab, tab.Id, []string{"tabTitle"})
		if err != nil {
			return tabs, err
		}
	}
	return tabs, nil
}

func Set_tx(ctx context.Context, tx pgx.Tx, entity schema.DbEntity, entityId uuid.UUID, position int, tab types.Tab) (uuid.UUID, error) {
	if !slices.Contains(schema.DbAssignedTab, entity) {
		return tab.Id, errors.New("bad entity")
	}

	known, err := schema.CheckCreateId_tx(ctx, tx, &tab.Id, schema.DbTab, "id")
	if err != nil {
		return tab.Id, err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.tab
			SET position = $1, content_counter = $2, state = $3
			WHERE id = $4
		`, position, tab.ContentCounter, tab.State, tab.Id); err != nil {
			return tab.Id, err
		}
	} else {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO app.tab (id, %s_id, position, content_counter, state)
			VALUES ($1,$2,$3,$4,$5)
		`, entity), tab.Id, entityId, position, tab.ContentCounter, tab.State); err != nil {
			return tab.Id, err
		}
	}
	return tab.Id, caption.Set_tx(ctx, tx, tab.Id, tab.Captions)
}
