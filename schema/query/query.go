package query

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
	"github.com/jackc/pgx/v5/pgtype"
)

func Get_tx(ctx context.Context, tx pgx.Tx, entity schema.DbEntity, id uuid.UUID, filterIndex int, filterPosition int, filterSide int) (types.Query, error) {

	var q types.Query
	q.Joins = make([]types.QueryJoin, 0)
	q.Filters = make([]types.QueryFilter, 0)
	q.Orders = make([]types.QueryOrder, 0)
	q.Lookups = make([]types.QueryLookup, 0)
	q.Choices = make([]types.QueryChoice, 0)

	if !slices.Contains(schema.DbAssignedQuery, entity) {
		return q, errors.New("unknown query parent entity")
	}

	// sub query (via query filter) requires composite key
	filterClause := ""
	if entity == "query_filter_query" {
		filterClause = fmt.Sprintf(`
			AND query_filter_index    = %d
			AND query_filter_position = %d
			AND query_filter_side     = %d
		`, filterIndex, filterPosition, filterSide)
	}

	err := tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, relation_id, fixed_limit
		FROM app.query
		WHERE %s_id = $1
		%s
	`, entity, filterClause), id).Scan(&q.Id, &q.RelationId, &q.FixedLimit)

	if err != nil && err != pgx.ErrNoRows {
		return q, err
	}

	// query does not exist for this entity, return empty query
	if err == pgx.ErrNoRows {
		return q, nil
	}

	// retrieve joins
	rows, err := tx.Query(ctx, `
		SELECT relation_id, attribute_id, index_from, index, connector,
			apply_create, apply_update, apply_delete
		FROM app.query_join
		WHERE query_id = $1
		ORDER BY position ASC
	`, q.Id)
	if err != nil {
		return q, err
	}
	defer rows.Close()

	for rows.Next() {
		var j types.QueryJoin

		if err := rows.Scan(&j.RelationId, &j.AttributeId, &j.IndexFrom, &j.Index,
			&j.Connector, &j.ApplyCreate, &j.ApplyUpdate, &j.ApplyDelete); err != nil {

			return q, err
		}
		q.Joins = append(q.Joins, j)
	}

	// retrieve filters
	q.Filters, err = getFilters_tx(ctx, tx, q.Id, pgtype.UUID{})
	if err != nil {
		return q, err
	}

	// retrieve orderings
	rows, err = tx.Query(ctx, `
		SELECT attribute_id, index, ascending
		FROM app.query_order
		WHERE query_id = $1
		ORDER BY position ASC
	`, q.Id)
	if err != nil {
		return q, err
	}
	defer rows.Close()

	for rows.Next() {
		var o types.QueryOrder

		if err := rows.Scan(&o.AttributeId, &o.Index, &o.Ascending); err != nil {
			return q, err
		}
		q.Orders = append(q.Orders, o)
	}

	// retrieve lookups
	rows, err = tx.Query(ctx, `
		SELECT pg_index_id, index
		FROM app.query_lookup
		WHERE query_id = $1
		ORDER BY index ASC
	`, q.Id)
	if err != nil {
		return q, err
	}
	defer rows.Close()

	for rows.Next() {
		var l types.QueryLookup

		if err := rows.Scan(&l.PgIndexId, &l.Index); err != nil {
			return q, err
		}
		q.Lookups = append(q.Lookups, l)
	}

	// retrieve choices
	rows, err = tx.Query(ctx, `
		SELECT id, name
		FROM app.query_choice
		WHERE query_id = $1
		ORDER BY position ASC
	`, q.Id)
	if err != nil {
		return q, err
	}
	defer rows.Close()

	for rows.Next() {
		var c types.QueryChoice

		if err := rows.Scan(&c.Id, &c.Name); err != nil {
			return q, err
		}
		q.Choices = append(q.Choices, c)
	}

	for i, c := range q.Choices {
		c.Filters, err = getFilters_tx(ctx, tx, q.Id, pgtype.UUID{Bytes: c.Id, Valid: true})
		if err != nil {
			return q, err
		}

		c.Captions, err = caption.Get_tx(ctx, tx, "query_choice", c.Id, []string{"queryChoiceTitle"})
		if err != nil {
			return q, err
		}
		q.Choices[i] = c
	}
	return q, nil
}

func Set_tx(ctx context.Context, tx pgx.Tx, entity schema.DbEntity, entityId uuid.UUID, filterIndex int,
	filterPosition int, filterSide int, query types.Query) error {

	if !slices.Contains(schema.DbAssignedQuery, entity) {
		return fmt.Errorf("unknown query parent entity '%s'", entity)
	}

	var err error
	createNew := false
	noBaseRelation := !query.RelationId.Valid
	subQuery := entity == "query_filter_query"

	// check if its a new query, old query (for the same entity) still needs to be checked as it could have been remade
	if query.Id == uuid.Nil {
		query.Id, err = uuid.NewV4()
		if err != nil {
			return err
		}
		createNew = true
	}

	// check whether a query for the parent entity already exists
	var queryIdExisting pgtype.UUID

	if !subQuery {
		if err := tx.QueryRow(ctx, fmt.Sprintf(`
			SELECT id
			FROM app.query
			WHERE %s_id = $1
		`, entity), entityId).Scan(&queryIdExisting); err != nil && err != pgx.ErrNoRows {
			return err
		}
	} else {
		if err := tx.QueryRow(ctx, `
			SELECT id
			FROM app.query
			WHERE query_filter_query_id = $1
			AND   query_filter_index    = $2
			AND   query_filter_position = $3
			AND   query_filter_side     = $4
		`, entityId, filterIndex, filterPosition, filterSide).Scan(&queryIdExisting); err != nil && err != pgx.ErrNoRows {
			return err
		}
	}

	if !queryIdExisting.Valid {
		// query does not exist, create
		createNew = true
	} else {
		// query exists - delete if it was remade (different ID) or is not required anymore (query without a base relation)
		if query.Id.String() != queryIdExisting.String() || noBaseRelation {
			if _, err := tx.Exec(ctx, `
				DELETE FROM app.query
				WHERE id = $1
			`, queryIdExisting); err != nil {
				return err
			}
			createNew = true
		}
	}

	if noBaseRelation {
		// no query needed
		return nil
	}

	// create or update query
	if createNew {
		if !subQuery {
			if _, err := tx.Exec(ctx, fmt.Sprintf(`
				INSERT INTO app.query (id, relation_id, fixed_limit, %s_id)
				VALUES ($1,$2,$3,$4)
			`, entity), query.Id, query.RelationId, query.FixedLimit, entityId); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `
				INSERT INTO app.query (id, relation_id, fixed_limit, query_filter_query_id,
					query_filter_index, query_filter_position, query_filter_side)
				VALUES ($1,$2,$3,$4,$5,$6,$7)
			`, query.Id, query.RelationId, query.FixedLimit, entityId,
				filterIndex, filterPosition, filterSide); err != nil {

				return err
			}
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE app.query
			SET relation_id = $1, fixed_limit = $2
			WHERE id = $3
		`, query.RelationId, query.FixedLimit, query.Id); err != nil {
			return err
		}
	}

	// reset joins
	if _, err := tx.Exec(ctx, `
		DELETE FROM app.query_join
		WHERE query_id = $1
	`, query.Id); err != nil {
		return err
	}

	for position, j := range query.Joins {

		if !slices.Contains(types.QueryJoinConnectors, j.Connector) {
			return errors.New("invalid join connector")
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO app.query_join (
				query_id, relation_id, attribute_id, position, index_from,
				index, connector, apply_create, apply_update, apply_delete
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, query.Id, j.RelationId, j.AttributeId, position, j.IndexFrom,
			j.Index, j.Connector, j.ApplyCreate, j.ApplyUpdate,
			j.ApplyDelete); err != nil {

			return err
		}
	}

	// reset filters
	if _, err := tx.Exec(ctx, `
		DELETE FROM app.query_filter
		WHERE query_id = $1
	`, query.Id); err != nil {
		return err
	}
	if err := setFilters_tx(ctx, tx, query.Id, pgtype.UUID{}, query.Filters, 0); err != nil {
		return err
	}

	// reset ordering
	if _, err := tx.Exec(ctx, `
		DELETE FROM app.query_order
		WHERE query_id = $1
	`, query.Id); err != nil {
		return err
	}

	for position, o := range query.Orders {

		if _, err := tx.Exec(ctx, `
			INSERT INTO app.query_order (
				query_id, attribute_id, position, index, ascending
			)
			VALUES ($1,$2,$3,$4,$5)
		`, query.Id, o.AttributeId, position, o.Index, o.Ascending); err != nil {
			return err
		}
	}

	// reset lookups
	if _, err := tx.Exec(ctx, `
		DELETE FROM app.query_lookup
		WHERE query_id = $1
	`, query.Id); err != nil {
		return err
	}

	for _, l := range query.Lookups {

		if _, err := tx.Exec(ctx, `
			INSERT INTO app.query_lookup (query_id, pg_index_id, index)
			VALUES ($1,$2,$3)
		`, query.Id, l.PgIndexId, l.Index); err != nil {
			return err
		}
	}

	// reset choices
	if _, err := tx.Exec(ctx, `
		DELETE FROM app.query_choice
		WHERE query_id = $1
	`, query.Id); err != nil {
		return err
	}

	for position, c := range query.Choices {

		if c.Id == uuid.Nil {
			c.Id, err = uuid.NewV4()
			if err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO app.query_choice (id, query_id, name, position)
			VALUES ($1,$2,$3,$4)
		`, c.Id, query.Id, c.Name, position); err != nil {
			return err
		}

		// set choice filters
		// use position offset to separate filters from optional choice filters
		//  (necessary as query ID + position is used as PK
		positionOffset := (position + 1) * 100

		if err := setFilters_tx(ctx, tx, query.Id, pgtype.UUID{Bytes: c.Id, Valid: true},
			c.Filters, positionOffset); err != nil {

			return err
		}
		if err := caption.Set_tx(ctx, tx, c.Id, c.Captions); err != nil {
			return err
		}
	}
	return nil
}

func getFilters_tx(ctx context.Context, tx pgx.Tx, queryId uuid.UUID, queryChoiceId pgtype.UUID) ([]types.QueryFilter, error) {

	var filters = make([]types.QueryFilter, 0)
	params := make([]interface{}, 0)
	params = append(params, queryId)

	nullCheck := "AND query_choice_id IS NULL"
	if queryChoiceId.Valid {
		nullCheck = "AND query_choice_id = $2"
		params = append(params, queryChoiceId.Bytes)
	}

	// get filters
	type typeFilterPos struct {
		filter   types.QueryFilter
		position int
	}
	filterPos := make([]typeFilterPos, 0)

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT connector, operator, index, position
		FROM app.query_filter
		WHERE query_id = $1
		%s
		ORDER BY position ASC
	`, nullCheck), params...)
	if err != nil {
		return filters, err
	}
	defer rows.Close()

	for rows.Next() {
		var fp typeFilterPos

		if err := rows.Scan(&fp.filter.Connector, &fp.filter.Operator, &fp.filter.Index, &fp.position); err != nil {
			return filters, err
		}
		filterPos = append(filterPos, fp)
	}

	for _, fp := range filterPos {

		fp.filter.Side0, err = getFilterSide_tx(ctx, tx, queryId, fp.filter.Index, fp.position, 0)
		if err != nil {
			return filters, err
		}
		fp.filter.Side1, err = getFilterSide_tx(ctx, tx, queryId, fp.filter.Index, fp.position, 1)
		if err != nil {
			return filters, err
		}
		filters = append(filters, fp.filter)
	}
	return filters, nil
}
func getFilterSide_tx(ctx context.Context, tx pgx.Tx, queryId uuid.UUID, filterIndex int, filterPosition int, side int) (types.QueryFilterSide, error) {
	var s types.QueryFilterSide
	var err error

	if err := tx.QueryRow(ctx, `
		SELECT attribute_id, attribute_index, attribute_nested, brackets,
			collection_id, column_id, content, field_id, now_offset, preset_id,
			role_id, variable_id, query_aggregator, value
		FROM app.query_filter_side
		WHERE query_id              = $1
		AND   query_filter_index    = $2
		AND   query_filter_position = $3
		AND   side                  = $4
	`, queryId, filterIndex, filterPosition, side).Scan(&s.AttributeId, &s.AttributeIndex,
		&s.AttributeNested, &s.Brackets, &s.CollectionId, &s.ColumnId, &s.Content,
		&s.FieldId, &s.NowOffset, &s.PresetId, &s.RoleId, &s.VariableId,
		&s.QueryAggregator, &s.Value); err != nil {

		return s, err
	}

	if s.Content == "subQuery" {
		s.Query, err = Get_tx(ctx, tx, "query_filter_query", queryId, filterIndex, filterPosition, side)
		if err != nil {
			return s, err
		}
	} else {
		s.Query.RelationId = pgtype.UUID{}
	}
	return s, nil
}

func setFilters_tx(ctx context.Context, tx pgx.Tx, queryId uuid.UUID, queryChoiceId pgtype.UUID,
	filters []types.QueryFilter, positionOffset int) error {

	for position, f := range filters {

		if !slices.Contains(types.QueryFilterConnectors, f.Connector) {
			return errors.New("invalid filter connector")
		}

		if !slices.Contains(types.QueryFilterOperators, f.Operator) {
			return errors.New("invalid filter operator")
		}

		position += positionOffset

		if _, err := tx.Exec(ctx, `
			INSERT INTO app.query_filter (query_id, query_choice_id,
				index, position, connector, operator)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, queryId, queryChoiceId, f.Index, position, f.Connector, f.Operator); err != nil {
			return err
		}

		if err := SetFilterSide_tx(ctx, tx, queryId, f.Index, position, 0, f.Side0); err != nil {
			return err
		}
		if err := SetFilterSide_tx(ctx, tx, queryId, f.Index, position, 1, f.Side1); err != nil {
			return err
		}
	}
	return nil
}
func SetFilterSide_tx(ctx context.Context, tx pgx.Tx, queryId uuid.UUID, filterIndex int,
	filterPosition int, side int, s types.QueryFilterSide) error {

	if _, err := tx.Exec(ctx, `
		INSERT INTO app.query_filter_side (
			query_id, query_filter_index, query_filter_position, side, attribute_id,
			attribute_index, attribute_nested, brackets, collection_id, column_id,
			content, field_id, now_offset, preset_id, role_id, variable_id,
			query_aggregator, value
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, queryId, filterIndex, filterPosition, side, s.AttributeId, s.AttributeIndex,
		s.AttributeNested, s.Brackets, s.CollectionId, s.ColumnId, s.Content,
		s.FieldId, s.NowOffset, s.PresetId, s.RoleId, s.VariableId,
		s.QueryAggregator, s.Value); err != nil {

		return err
	}

	if s.Content == "subQuery" {
		if err := Set_tx(ctx, tx, schema.DbQueryFilterQuery, queryId, filterIndex, filterPosition, side, s.Query); err != nil {
			return err
		}
	}
	return nil
}
