package field

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"r3/schema"
	"r3/schema/caption"
	"r3/schema/collection/consumer"
	"r3/schema/column"
	"r3/schema/compatible"
	"r3/schema/openForm"
	"r3/schema/query"
	"r3/schema/tab"
	"r3/types"
	"slices"
	"sort"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Del_tx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM app.field WHERE id = $1`, id)
	return err
}

func Get_tx(ctx context.Context, tx pgx.Tx, formId uuid.UUID) ([]interface{}, error) {
	fields := make([]interface{}, 0)

	rows, err := tx.Query(ctx, `
		SELECT f.id, f.parent_id, f.tab_id, f.icon_id, f.content, f.state,
		f.flags, f.on_mobile, a.content,
		
		-- button field
		fb.js_function_id,
		
		-- calendar field
		fn.attribute_id_date0, fn.attribute_id_date1, fn.attribute_id_color,
		fn.index_date0, fn.index_date1, fn.index_color, fn.ics, fn.gantt,
		fn.gantt_steps, fn.gantt_steps_toggle, fn.date_range0, fn.date_range1,
		fn.days, fn.days_toggle,
		
		-- chart field
		fa.chart_option,
		
		-- container field
		fc.direction, fc.justify_content, fc.align_items, fc.align_content,
		fc.wrap, fc.grow, fc.shrink, fc.basis, fc.per_min, fc.per_max,
		
		-- header field
		fh.richtext, fh.size,
		
		-- data field
		fd.attribute_id, fd.attribute_id_alt, fd.index, fd.display, fd.min,
		fd.max, fd.def, fd.regex_check, fd.js_function_id,
		
		-- data relationship field
		fr.attribute_id_nm, fr.filter_quick, fr.outside_in, fr.auto_select, (
			SELECT COALESCE(ARRAY_AGG(preset_id), '{}')
			FROM app.field_data_relationship_preset
			WHERE field_id = fr.field_id
		) AS preset_ids,
		
		-- kanban field
		fk.relation_index_data, fk.relation_index_axis_x,
		fk.relation_index_axis_y, fk.attribute_id_sort,
		
		-- list field
		fl.auto_renew, fl.csv_export, fl.csv_import, fl.layout,
		fl.filter_quick, fl.result_limit,

		-- variable field
		fv.variable_id, fv.js_function_id
		
		FROM app.field AS f
		LEFT JOIN app.field_button            AS fb ON fb.field_id = f.id
		LEFT JOIN app.field_calendar          AS fn ON fn.field_id = f.id
		LEFT JOIN app.field_chart             AS fa ON fa.field_id = f.id
		LEFT JOIN app.field_container         AS fc ON fc.field_id = f.id
		LEFT JOIN app.field_data              AS fd ON fd.field_id = f.id
		LEFT JOIN app.field_data_relationship AS fr ON fr.field_id = f.id
		LEFT JOIN app.field_header            AS fh ON fh.field_id = f.id
		LEFT JOIN app.field_kanban            AS fk ON fk.field_id = f.id
		LEFT JOIN app.field_list              AS fl ON fl.field_id = f.id
		LEFT JOIN app.field_variable          AS fv ON fv.field_id = f.id
		LEFT JOIN app.attribute               AS a  ON a.id        = fd.attribute_id
		WHERE f.form_id = $1
		ORDER BY f.position ASC
	`, formId)
	if err != nil {
		return fields, err
	}

	// prepare lookup slices
	// some field types require additional lookups (queries, column, captions, ...)
	pos := 0
	posButtonLookup := make([]int, 0)
	posCalendarLookup := make([]int, 0)
	posChartLookup := make([]int, 0)
	posDataLookup := make([]int, 0)
	posDataRelLookup := make([]int, 0)
	posHeaderLookup := make([]int, 0)
	posKanbanLookup := make([]int, 0)
	posListLookup := make([]int, 0)
	posParentLookup := make([]int, 0)
	posTabsLookup := make([]int, 0)
	posVariableLookup := make([]int, 0)
	posMapParentId := make(map[int]uuid.UUID)
	posMapTabId := make(map[int]uuid.UUID)

	for rows.Next() {
		var fieldId uuid.UUID
		var content string
		var state string
		var onMobile bool
		var atrContent sql.NullString

		var alignItems, alignContent, chartOption, def, direction, display,
			ganttSteps, justifyContent, layout, regexCheck pgtype.Text
		var autoSelect, days, grow, shrink, basis, perMin, perMax, index,
			indexDate0, indexDate1, size, relationIndexKanbanData,
			relationIndexKanbanAxisX, relationIndexKanbanAxisY,
			resultLimit pgtype.Int2
		var autoRenew, dateRange0, dateRange1, indexColor, min, max pgtype.Int4
		var attributeId, attributeIdAlt, attributeIdNm, attributeIdDate0,
			attributeIdDate1, attributeIdColor, attributeIdKanbanSort,
			fieldParentId, iconId, jsFunctionIdButton, jsFunctionIdData,
			jsFunctionIdVariable, tabId, variableId pgtype.UUID
		var csvExport, csvImport, daysToggle, filterQuick, filterQuickList,
			gantt, ganttStepsToggle, ics, outsideIn, richtext, wrap pgtype.Bool
		var defPresetIds []uuid.UUID
		var flags []string

		if err := rows.Scan(&fieldId, &fieldParentId, &tabId, &iconId, &content,
			&state, &flags, &onMobile, &atrContent, &jsFunctionIdButton,
			&attributeIdDate0, &attributeIdDate1, &attributeIdColor, &indexDate0,
			&indexDate1, &indexColor, &ics, &gantt, &ganttSteps, &ganttStepsToggle,
			&dateRange0, &dateRange1, &days, &daysToggle, &chartOption,
			&direction, &justifyContent, &alignItems, &alignContent, &wrap,
			&grow, &shrink, &basis, &perMin, &perMax, &richtext, &size,
			&attributeId, &attributeIdAlt, &index, &display, &min, &max, &def,
			&regexCheck, &jsFunctionIdData, &attributeIdNm,
			&filterQuick, &outsideIn, &autoSelect, &defPresetIds,
			&relationIndexKanbanData, &relationIndexKanbanAxisX,
			&relationIndexKanbanAxisY, &attributeIdKanbanSort, &autoRenew,
			&csvExport, &csvImport, &layout, &filterQuickList, &resultLimit,
			&variableId, &jsFunctionIdVariable); err != nil {

			rows.Close()
			return fields, err
		}

		// store parent if there
		posMapParentId[pos] = fieldParentId.Bytes
		posMapTabId[pos] = tabId.Bytes

		switch content {
		case "button":
			fields = append(fields, types.FieldButton{
				Id:           fieldId,
				TabId:        tabId,
				IconId:       iconId,
				Content:      content,
				State:        state,
				Flags:        flags,
				OnMobile:     onMobile,
				JsFunctionId: jsFunctionIdButton,
				OpenForm:     types.OpenForm{},
			})
			posButtonLookup = append(posButtonLookup, pos)
		case "calendar":
			fields = append(fields, types.FieldCalendar{
				Id:               fieldId,
				TabId:            tabId,
				IconId:           iconId,
				Content:          content,
				State:            state,
				Flags:            flags,
				OnMobile:         onMobile,
				AttributeIdDate0: attributeIdDate0.Bytes,
				AttributeIdDate1: attributeIdDate1.Bytes,
				AttributeIdColor: attributeIdColor,
				IndexDate0:       int(indexDate0.Int16),
				IndexDate1:       int(indexDate1.Int16),
				IndexColor:       indexColor,
				Ics:              ics.Bool,
				Gantt:            gantt.Bool,
				GanttSteps:       ganttSteps,
				GanttStepsToggle: ganttStepsToggle.Bool,
				DateRange0:       int64(dateRange0.Int32),
				DateRange1:       int64(dateRange1.Int32),
				Days:             int(days.Int16),
				DaysToggle:       daysToggle.Bool,
				Columns:          []types.Column{},
				Query:            types.Query{},
				OpenForm:         types.OpenForm{},
			})
			posCalendarLookup = append(posCalendarLookup, pos)
		case "chart":
			fields = append(fields, types.FieldChart{
				Id:          fieldId,
				TabId:       tabId,
				IconId:      iconId,
				Content:     content,
				State:       state,
				Flags:       flags,
				OnMobile:    onMobile,
				ChartOption: chartOption.String,
				Columns:     []types.Column{},
				Query:       types.Query{},
				Captions:    types.CaptionMap{},
			})
			posChartLookup = append(posChartLookup, pos)
		case "container":
			fields = append(fields, types.FieldContainer{
				Id:             fieldId,
				TabId:          tabId,
				IconId:         iconId,
				Content:        content,
				State:          state,
				Flags:          flags,
				OnMobile:       onMobile,
				Direction:      direction.String,
				JustifyContent: justifyContent.String,
				AlignItems:     alignItems.String,
				AlignContent:   alignContent.String,
				Wrap:           wrap.Bool,
				Grow:           int(grow.Int16),
				Shrink:         int(shrink.Int16),
				Basis:          int(basis.Int16),
				PerMin:         int(perMin.Int16),
				PerMax:         int(perMax.Int16),
				Fields:         []interface{}{},
			})
			posParentLookup = append(posParentLookup, pos)

		case "data":
			if schema.IsContentRelationship(atrContent.String) {
				fields = append(fields, types.FieldDataRelationship{
					Id:             fieldId,
					TabId:          tabId,
					IconId:         iconId,
					Content:        content,
					State:          state,
					Flags:          flags,
					OnMobile:       onMobile,
					AttributeId:    attributeId.Bytes,
					AttributeIdAlt: attributeIdAlt,
					AttributeIdNm:  attributeIdNm,
					Index:          int(index.Int16),
					Display:        display.String,
					AutoSelect:     int(autoSelect.Int16),
					Min:            min,
					Max:            max,
					RegexCheck:     regexCheck,
					JsFunctionId:   jsFunctionIdData,
					Def:            def.String,
					DefPresetIds:   defPresetIds,
					FilterQuick:    filterQuick.Bool,
					OutsideIn:      outsideIn.Bool,
					Columns:        []types.Column{},
					Query:          types.Query{},
					OpenForm:       types.OpenForm{},
					Captions:       types.CaptionMap{},

					// legacy
					CollectionIdDef: pgtype.UUID{},
					ColumnIdDef:     pgtype.UUID{},
				})
				posDataRelLookup = append(posDataRelLookup, pos)
			} else {
				fields = append(fields, types.FieldData{
					Id:             fieldId,
					TabId:          tabId,
					IconId:         iconId,
					Content:        content,
					State:          state,
					Flags:          flags,
					OnMobile:       onMobile,
					AttributeId:    attributeId.Bytes,
					AttributeIdAlt: attributeIdAlt,
					Index:          int(index.Int16),
					Display:        display.String,
					Def:            def.String,
					Min:            min,
					Max:            max,
					RegexCheck:     regexCheck,
					JsFunctionId:   jsFunctionIdData,
					Captions:       types.CaptionMap{},

					// legacy
					CollectionIdDef: pgtype.UUID{},
					ColumnIdDef:     pgtype.UUID{},
				})
				posDataLookup = append(posDataLookup, pos)
			}

		case "header":
			fields = append(fields, types.FieldHeader{
				Id:       fieldId,
				TabId:    tabId,
				IconId:   iconId,
				Content:  content,
				State:    state,
				Flags:    flags,
				OnMobile: onMobile,
				Richtext: richtext.Bool,
				Size:     int(size.Int16),
				Captions: types.CaptionMap{},
			})
			posHeaderLookup = append(posHeaderLookup, pos)

		case "kanban":
			fields = append(fields, types.FieldKanban{
				Id:                 fieldId,
				TabId:              tabId,
				IconId:             iconId,
				Content:            content,
				State:              state,
				Flags:              flags,
				OnMobile:           onMobile,
				RelationIndexData:  int(relationIndexKanbanData.Int16),
				RelationIndexAxisX: int(relationIndexKanbanAxisX.Int16),
				RelationIndexAxisY: relationIndexKanbanAxisY,
				AttributeIdSort:    attributeIdKanbanSort,
				OpenForm:           types.OpenForm{},
				Columns:            []types.Column{},
				Query:              types.Query{},
			})
			posKanbanLookup = append(posKanbanLookup, pos)

		case "list":
			fields = append(fields, types.FieldList{
				Id:           fieldId,
				TabId:        tabId,
				IconId:       iconId,
				Content:      content,
				State:        state,
				Flags:        flags,
				OnMobile:     onMobile,
				Columns:      []types.Column{},
				AutoRenew:    autoRenew,
				CsvExport:    csvExport.Bool,
				CsvImport:    csvImport.Bool,
				Layout:       layout.String,
				FilterQuick:  filterQuickList.Bool,
				Query:        types.Query{},
				Captions:     types.CaptionMap{},
				OpenForm:     types.OpenForm{},
				OpenFormBulk: types.OpenForm{},
				ResultLimit:  int(resultLimit.Int16),
			})
			posListLookup = append(posListLookup, pos)

		case "tabs":
			fields = append(fields, types.FieldTabs{
				Id:       fieldId,
				TabId:    tabId,
				IconId:   iconId,
				Content:  content,
				State:    state,
				Flags:    flags,
				OnMobile: onMobile,
				Captions: types.CaptionMap{},
				Tabs:     []types.Tab{},
			})
			posTabsLookup = append(posTabsLookup, pos)
			posParentLookup = append(posParentLookup, pos)

		case "variable":
			fields = append(fields, types.FieldVariable{
				Id:           fieldId,
				VariableId:   variableId,
				JsFunctionId: jsFunctionIdVariable,
				IconId:       iconId,
				Content:      content,
				State:        state,
				Flags:        flags,
				OnMobile:     onMobile,
				Captions:     types.CaptionMap{},
			})
			posVariableLookup = append(posVariableLookup, pos)
		}
		pos++
	}
	rows.Close()

	// lookup button fields: open form, captions
	for _, pos := range posButtonLookup {
		var field = fields[pos].(types.FieldButton)

		field.OpenForm, err = openForm.Get_tx(ctx, tx, schema.DbField, field.Id, pgtype.Text{})
		if err != nil {
			return fields, err
		}
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle"})
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup calendar fields: open form, query, columns
	for _, pos := range posCalendarLookup {
		var field = fields[pos].(types.FieldCalendar)

		field.OpenForm, err = openForm.Get_tx(ctx, tx, schema.DbField, field.Id, pgtype.Text{})
		if err != nil {
			return fields, err
		}
		field.Query, err = query.Get_tx(ctx, tx, schema.DbField, field.Id, 0, 0, 0)
		if err != nil {
			return fields, err
		}
		field.Columns, err = column.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		field.Collections, err = consumer.Get_tx(ctx, tx, schema.DbField, field.Id, "fieldFilterSelector")
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup chart fields: query, columns
	for _, pos := range posChartLookup {
		var field = fields[pos].(types.FieldChart)

		field.Query, err = query.Get_tx(ctx, tx, schema.DbField, field.Id, 0, 0, 0)
		if err != nil {
			return fields, err
		}
		field.Columns, err = column.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle"})
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup data fields: default value collection, captions
	for _, pos := range posDataLookup {
		var field = fields[pos].(types.FieldData)

		field.DefCollection, err = consumer.GetOne_tx(ctx, tx, schema.DbField, field.Id, "fieldDataDefault")
		if err != nil {
			return fields, err
		}
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle", "fieldHelp"})
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup data relationship fields: open form, query, columns, efault value collection, captions
	for _, pos := range posDataRelLookup {
		var field = fields[pos].(types.FieldDataRelationship)

		field.OpenForm, err = openForm.Get_tx(ctx, tx, schema.DbField, field.Id, pgtype.Text{})
		if err != nil {
			return fields, err
		}
		field.Query, err = query.Get_tx(ctx, tx, schema.DbField, field.Id, 0, 0, 0)
		if err != nil {
			return fields, err
		}
		field.Columns, err = column.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		field.DefCollection, err = consumer.GetOne_tx(ctx, tx, schema.DbField, field.Id, "fieldDataDefault")
		if err != nil {
			return fields, err
		}
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle", "fieldHelp"})
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup header fields: captions
	for _, pos := range posHeaderLookup {
		var field = fields[pos].(types.FieldHeader)

		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle"})
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup kanban fields: open form, query, columns, consumed collections
	for _, pos := range posKanbanLookup {
		var field = fields[pos].(types.FieldKanban)

		field.OpenForm, err = openForm.Get_tx(ctx, tx, schema.DbField, field.Id, pgtype.Text{})
		if err != nil {
			return fields, err
		}
		field.Query, err = query.Get_tx(ctx, tx, schema.DbField, field.Id, 0, 0, 0)
		if err != nil {
			return fields, err
		}
		field.Columns, err = column.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		field.Collections, err = consumer.Get_tx(ctx, tx, schema.DbField, field.Id, "fieldFilterSelector")
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup list fields: open form, query, columns, consumed collections
	for _, pos := range posListLookup {
		var field = fields[pos].(types.FieldList)

		field.OpenForm, err = openForm.Get_tx(ctx, tx, schema.DbField, field.Id, pgtype.Text{})
		if err != nil {
			return fields, err
		}
		field.OpenFormBulk, err = openForm.Get_tx(ctx, tx, schema.DbField, field.Id, pgtype.Text{String: "bulk", Valid: true})
		if err != nil {
			return fields, err
		}
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle"})
		if err != nil {
			return fields, err
		}
		field.Query, err = query.Get_tx(ctx, tx, schema.DbField, field.Id, 0, 0, 0)
		if err != nil {
			return fields, err
		}
		field.Columns, err = column.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		field.Collections, err = consumer.Get_tx(ctx, tx, schema.DbField, field.Id, "fieldFilterSelector")
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup tabs fields: get tabs
	for _, pos := range posTabsLookup {
		var field = fields[pos].(types.FieldTabs)
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle"})
		if err != nil {
			return fields, err
		}
		field.Tabs, err = tab.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// lookup variable fields: open form, query, columns, captions
	for _, pos := range posVariableLookup {
		var field = fields[pos].(types.FieldVariable)

		field.Query, err = query.Get_tx(ctx, tx, schema.DbField, field.Id, 0, 0, 0)
		if err != nil {
			return fields, err
		}
		field.Columns, err = column.Get_tx(ctx, tx, schema.DbField, field.Id)
		if err != nil {
			return fields, err
		}
		field.Captions, err = caption.Get_tx(ctx, tx, schema.DbField, field.Id, []string{"fieldTitle", "fieldHelp"})
		if err != nil {
			return fields, err
		}
		fields[pos] = field
	}

	// get sorted keys for field positions with parent ID
	orderedPos := make([]int, 0, len(posMapParentId))
	for k := range posMapParentId {
		orderedPos = append(orderedPos, k)
	}
	sort.Ints(orderedPos)

	// initialize function for recursive execution
	var getChildren func(parentId uuid.UUID, tabId uuid.UUID) []interface{}
	getChildren = func(parentId uuid.UUID, tabId uuid.UUID) []interface{} {
		children := make([]interface{}, 0)

		for _, pos := range orderedPos {
			if posMapParentId[pos] != parentId || posMapTabId[pos] != tabId {
				continue
			}

			// no parent field
			if !slices.Contains(posParentLookup, pos) {
				children = append(children, fields[pos])
				continue
			}

			// tabs field
			if slices.Contains(posTabsLookup, pos) {
				field := fields[pos].(types.FieldTabs)

				for i, tab := range field.Tabs {
					field.Tabs[i].Fields = getChildren(field.Id, tab.Id)
				}

				children = append(children, field)
				continue
			}

			// container field
			field := fields[pos].(types.FieldContainer)
			field.Fields = getChildren(field.Id, uuid.Nil)
			children = append(children, field)
		}
		return children
	}

	// recursively resolve all fields with their children
	return getChildren(uuid.Nil, uuid.Nil), nil
}
func GetCalendar_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID) (types.FieldCalendar, error) {

	var f types.FieldCalendar
	f.Id = fieldId

	err := tx.QueryRow(ctx, `
		SELECT attribute_id_date0, attribute_id_date1, index_date0, index_date1,
			date_range0, date_range1, days, days_toggle
		FROM app.field_calendar
		WHERE ics
		AND gantt = FALSE
		AND field_id = $1
	`, fieldId).Scan(&f.AttributeIdDate0, &f.AttributeIdDate1, &f.IndexDate0,
		&f.IndexDate1, &f.DateRange0, &f.DateRange1, &f.Days, &f.DaysToggle)

	if err != nil {
		return f, err
	}

	f.OpenForm, err = openForm.Get_tx(ctx, tx, schema.DbField, f.Id, pgtype.Text{})
	if err != nil {
		return f, err
	}
	f.Query, err = query.Get_tx(ctx, tx, schema.DbField, f.Id, 0, 0, 0)
	if err != nil {
		return f, err
	}
	f.Columns, err = column.Get_tx(ctx, tx, schema.DbField, f.Id)
	if err != nil {
		return f, err
	}
	f.Collections, err = consumer.Get_tx(ctx, tx, schema.DbField, f.Id, "fieldFilterSelector")
	if err != nil {
		return f, err
	}
	return f, nil
}

func Set_tx(ctx context.Context, tx pgx.Tx, formId uuid.UUID, parentId pgtype.UUID, tabId pgtype.UUID,
	fields []interface{}, fieldIdMapQuery map[uuid.UUID]types.Query) error {

	for pos, fieldIf := range fields {

		fieldJson, err := json.Marshal(fieldIf)
		if err != nil {
			return err
		}

		var f types.Field
		if err := json.Unmarshal(fieldJson, &f); err != nil {
			return err
		}

		// check for special case: data relationship field
		var isDataRel = false
		if f.Content == "data" {
			fieldData, valid := fieldIf.(map[string]interface{})
			if !valid {
				return errors.New("field interface is not map string interface")
			}
			if _, ok := fieldData["outsideIn"].(bool); ok {
				isDataRel = true
			}
		}

		// fix imports < 3.10: New field flags
		f.Flags = compatible.FixNilFieldFlags(f.Flags)

		// fix imports < 3.11: Migrate options to field flags
		f.Flags, err = compatible.FixFieldOptionsToFlags(f, isDataRel, fieldJson)
		if err != nil {
			return err
		}

		fieldId, err := setGeneric_tx(ctx, tx, formId, parentId, tabId, f, pos)
		if err != nil {
			return err
		}

		switch f.Content {
		case "button":
			var f types.FieldButton
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setButton_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}
		case "calendar":
			var f types.FieldCalendar
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setCalendar_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			fieldIdMapQuery[fieldId] = f.Query

		case "chart":
			var f types.FieldChart
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setChart_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}
			fieldIdMapQuery[fieldId] = f.Query

		case "container":
			var f types.FieldContainer
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setContainer_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}

			// update container children
			if err := Set_tx(ctx, tx, formId, pgtype.UUID{Bytes: fieldId, Valid: true},
				pgtype.UUID{}, f.Fields, fieldIdMapQuery); err != nil {

				return err
			}

		case "data":
			var f types.FieldData
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setData_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}
			if isDataRel {
				var f types.FieldDataRelationship
				if err := json.Unmarshal(fieldJson, &f); err != nil {
					return err
				}
				if err := setDataRelationship_tx(ctx, tx, fieldId, f); err != nil {
					return err
				}
				fieldIdMapQuery[fieldId] = f.Query
			}

		case "header":
			var f types.FieldHeader
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setHeader_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}

		case "kanban":
			var f types.FieldKanban
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setKanban_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			fieldIdMapQuery[fieldId] = f.Query

		case "list":
			var f types.FieldList
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setList_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}
			fieldIdMapQuery[fieldId] = f.Query

		case "tabs":
			var f types.FieldTabs
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if len(f.Tabs) == 0 {
				return fmt.Errorf("tabs field '%s' has 0 tabs", fieldId)
			}

			// insert/update/delete tabs
			idsKeep := make([]uuid.UUID, 0)
			for i, t := range f.Tabs {
				t.Id, err = tab.Set_tx(ctx, tx, schema.DbField, fieldId, i, t)
				if err != nil {
					return err
				}

				if err := Set_tx(ctx, tx, formId,
					pgtype.UUID{Bytes: fieldId, Valid: true},
					pgtype.UUID{Bytes: t.Id, Valid: true},
					t.Fields, fieldIdMapQuery); err != nil {

					return err
				}
				idsKeep = append(idsKeep, t.Id)
			}
			if _, err := tx.Exec(ctx, `
				DELETE FROM app.tab
				WHERE field_id = $1
				AND id <> ALL($2)
			`, f.Id, idsKeep); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}

		case "variable":
			var f types.FieldVariable
			if err := json.Unmarshal(fieldJson, &f); err != nil {
				return err
			}
			if err := setVariable_tx(ctx, tx, fieldId, f); err != nil {
				return err
			}
			if err := caption.Set_tx(ctx, tx, fieldId, f.Captions); err != nil {
				return err
			}
			fieldIdMapQuery[fieldId] = f.Query

		default:
			return errors.New("unknown field content")
		}
	}
	return nil
}

func setGeneric_tx(ctx context.Context, tx pgx.Tx, formId uuid.UUID, parentId pgtype.UUID,
	tabId pgtype.UUID, f types.Field, position int) (uuid.UUID, error) {

	known, err := schema.CheckCreateId_tx(ctx, tx, &f.Id, schema.DbField, "id")
	if err != nil {
		return f.Id, err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field
			SET parent_id = $1, tab_id = $2, icon_id = $3, state = $4,
				flags = $5, on_mobile = $6, position = $7
			WHERE id = $8
		`, parentId, tabId, f.IconId, f.State, f.Flags, f.OnMobile, position, f.Id); err != nil {
			return f.Id, err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field (id, form_id, parent_id, tab_id,
				icon_id, content, state, flags, on_mobile, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, f.Id, formId, parentId, tabId, f.IconId, f.Content, f.State, f.Flags, f.OnMobile, position); err != nil {
			return f.Id, err
		}
	}
	return f.Id, nil
}
func setButton_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldButton) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldButton, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_button
			SET js_function_id = $1
			WHERE field_id = $2
		`, f.JsFunctionId, fieldId); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_button (field_id, js_function_id)
			VALUES ($1,$2)
		`, fieldId, f.JsFunctionId); err != nil {
			return err
		}
	}

	// set open form
	return openForm.Set_tx(ctx, tx, schema.DbField, fieldId, f.OpenForm, pgtype.Text{})
}
func setCalendar_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldCalendar) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldCalendar, "field_id")
	if err != nil {
		return err
	}

	// fix imports < 3.5: Default view
	f.Days = compatible.FixCalendarDefaultView(f.Days)

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_calendar
			SET attribute_id_date0 = $1, attribute_id_date1 = $2,
				attribute_id_color = $3, index_date0 = $4, index_date1 = $5,
				index_color = $6, gantt = $7, gantt_steps = $8,
				gantt_steps_toggle = $9, ics = $10, date_range0 = $11,
				date_range1 = $12, days = $13, days_toggle = $14
			WHERE field_id = $15
		`, f.AttributeIdDate0, f.AttributeIdDate1, f.AttributeIdColor,
			f.IndexDate0, f.IndexDate1, f.IndexColor, f.Gantt, f.GanttSteps,
			f.GanttStepsToggle, f.Ics, f.DateRange0, f.DateRange1, f.Days,
			f.DaysToggle, fieldId); err != nil {

			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_calendar (
				field_id, attribute_id_date0, attribute_id_date1,
				attribute_id_color, index_date0, index_date1, index_color,
				gantt, gantt_steps, 	gantt_steps_toggle, ics, date_range0,
				date_range1, days, days_toggle
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		`, fieldId, f.AttributeIdDate0, f.AttributeIdDate1, f.AttributeIdColor,
			f.IndexDate0, f.IndexDate1, f.IndexColor, f.Gantt, f.GanttSteps,
			f.GanttStepsToggle, f.Ics, f.DateRange0, f.DateRange1, f.Days,
			f.DaysToggle); err != nil {

			return err
		}
	}

	// set open form
	if err := openForm.Set_tx(ctx, tx, schema.DbField, fieldId, f.OpenForm, pgtype.Text{}); err != nil {
		return err
	}

	// set collection consumer
	if err := consumer.Set_tx(ctx, tx, schema.DbField, fieldId, "fieldFilterSelector", f.Collections); err != nil {
		return err
	}

	// set columns
	return column.Set_tx(ctx, tx, schema.DbField, fieldId, f.Columns)
}
func setChart_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldChart) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldChart, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_chart
			SET chart_option = $1
			WHERE field_id = $2
		`, f.ChartOption, fieldId); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_chart (field_id, chart_option)
			VALUES ($1,$2)
		`, fieldId, f.ChartOption); err != nil {
			return err
		}
	}
	return column.Set_tx(ctx, tx, schema.DbField, fieldId, f.Columns)
}
func setContainer_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldContainer) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldContainer, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_container
			SET direction = $1, justify_content = $2, align_items = $3,
				align_content = $4, wrap = $5, grow = $6, shrink = $7, basis = $8,
				per_min = $9, per_max = $10
			WHERE field_id = $11
		`, f.Direction, f.JustifyContent, f.AlignItems, f.AlignContent, f.Wrap, f.Grow, f.Shrink,
			f.Basis, f.PerMin, f.PerMax, fieldId); err != nil {

			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_container (
				field_id, direction, justify_content, align_items,
				align_content, wrap, grow, shrink, basis, per_min, per_max
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		`, fieldId, f.Direction, f.JustifyContent, f.AlignItems, f.AlignContent, f.Wrap,
			f.Grow, f.Shrink, f.Basis, f.PerMin, f.PerMax); err != nil {

			return err
		}
	}
	return nil
}
func setData_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldData) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldData, "field_id")
	if err != nil {
		return err
	}

	// fix imports < 3.0: Migrate legacy definitions
	if f.CollectionIdDef.Valid {
		f.DefCollection.CollectionId = f.CollectionIdDef.Bytes
		f.DefCollection.ColumnIdDisplay = f.ColumnIdDef
		f.DefCollection.Flags = make([]string, 0)
	}

	// fix imports < 3.3: Migrate display option to attribute content use
	f.Display, err = compatible.MigrateDisplayToContentUse_tx(ctx, tx, f.AttributeId, f.Display)
	if err != nil {
		return err
	}
	if f.AttributeIdAlt.Valid {
		_, err = compatible.MigrateDisplayToContentUse_tx(ctx, tx, f.AttributeIdAlt.Bytes, f.Display)
		if err != nil {
			return err
		}
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_data
			SET attribute_id = $1, attribute_id_alt = $2, index = $3,
				def = $4, display = $5, min = $6, max = $7, regex_check = $8,
				js_function_id = $9
			WHERE field_id = $10
		`, f.AttributeId, f.AttributeIdAlt, f.Index, f.Def, f.Display, f.Min,
			f.Max, f.RegexCheck, f.JsFunctionId, fieldId); err != nil {

			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_data (
				field_id, attribute_id, attribute_id_alt, index, def,
				display, min, max, regex_check, js_function_id
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, fieldId, f.AttributeId, f.AttributeIdAlt, f.Index, f.Def,
			f.Display, f.Min, f.Max, f.RegexCheck, f.JsFunctionId); err != nil {

			return err
		}
	}

	// set collection consumer
	return consumer.Set_tx(ctx, tx, schema.DbField, fieldId, "fieldDataDefault", []types.CollectionConsumer{f.DefCollection})
}
func setDataRelationship_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldDataRelationship) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldDataRelationship, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_data_relationship
			SET attribute_id_nm = $1, filter_quick = $2, outside_in = $3, auto_select = $4
			WHERE field_id = $5
		`, f.AttributeIdNm, f.FilterQuick, f.OutsideIn, f.AutoSelect, fieldId); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_data_relationship (field_id, attribute_id_nm, filter_quick, outside_in, auto_select)
			VALUES ($1,$2,$3,$4,$5)
		`, fieldId, f.AttributeIdNm, f.FilterQuick, f.OutsideIn, f.AutoSelect); err != nil {
			return err
		}
	}

	// set default preset IDs
	if _, err := tx.Exec(ctx, `
		DELETE FROM app.field_data_relationship_preset
		WHERE field_id = $1
	`, fieldId); err != nil {
		return err
	}

	for _, presetId := range f.DefPresetIds {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_data_relationship_preset (field_id, preset_id)
			VALUES ($1,$2)
		`, fieldId, presetId); err != nil {
			return err
		}
	}

	// set open form
	if err := openForm.Set_tx(ctx, tx, schema.DbField, fieldId, f.OpenForm, pgtype.Text{}); err != nil {
		return err
	}
	return column.Set_tx(ctx, tx, schema.DbField, fieldId, f.Columns)
}
func setHeader_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldHeader) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldHeader, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_header
			SET richtext = $1, size = $2
			WHERE field_id = $3
		`, f.Richtext, f.Size, fieldId); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_header (field_id, richtext, size)
			VALUES ($1,$2,$3)
		`, fieldId, f.Richtext, f.Size); err != nil {
			return err
		}
	}
	return nil
}
func setKanban_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldKanban) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldKanban, "field_id")
	if err != nil {
		return err
	}

	if f.RelationIndexData == f.RelationIndexAxisX {
		return errors.New("a separate relation must be chosen for Kanban columns")
	}
	if f.RelationIndexAxisY.Valid && int(f.RelationIndexAxisY.Int16) == f.RelationIndexAxisX {
		return errors.New("relations for Kanban columns & rows must be different")
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_kanban
			SET relation_index_data = $1, relation_index_axis_x = $2,
				relation_index_axis_y = $3, attribute_id_sort = $4
			WHERE field_id = $5
		`, f.RelationIndexData, f.RelationIndexAxisX, f.RelationIndexAxisY,
			f.AttributeIdSort, fieldId); err != nil {

			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_kanban (
				field_id, relation_index_data, relation_index_axis_x,
				relation_index_axis_y, attribute_id_sort
			)
			VALUES ($1,$2,$3,$4,$5)
		`, fieldId, f.RelationIndexData, f.RelationIndexAxisX,
			f.RelationIndexAxisY, f.AttributeIdSort); err != nil {

			return err
		}
	}

	// set open form
	if err := openForm.Set_tx(ctx, tx, schema.DbField, fieldId, f.OpenForm, pgtype.Text{}); err != nil {
		return err
	}

	// set collection consumer
	if err := consumer.Set_tx(ctx, tx, schema.DbField, fieldId, "fieldFilterSelector", f.Collections); err != nil {
		return err
	}

	// set columns
	return column.Set_tx(ctx, tx, schema.DbField, fieldId, f.Columns)
}
func setList_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldList) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldList, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_list
			SET auto_renew = $1, csv_export = $2, csv_import = $3, layout = $4,
				filter_quick = $5, result_limit = $6
			WHERE field_id = $7
		`, f.AutoRenew, f.CsvExport, f.CsvImport, f.Layout, f.FilterQuick, f.ResultLimit, fieldId); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_list (
				field_id, auto_renew, csv_export, csv_import,
				layout, filter_quick, result_limit
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, fieldId, f.AutoRenew, f.CsvExport, f.CsvImport, f.Layout, f.FilterQuick, f.ResultLimit); err != nil {
			return err
		}
	}

	// set open forms
	if err := openForm.Set_tx(ctx, tx, schema.DbField, fieldId, f.OpenForm, pgtype.Text{}); err != nil {
		return err
	}
	if err := openForm.Set_tx(ctx, tx, schema.DbField, fieldId, f.OpenFormBulk, pgtype.Text{String: "bulk", Valid: true}); err != nil {
		return err
	}

	// set collection consumer
	if err := consumer.Set_tx(ctx, tx, schema.DbField, fieldId, "fieldFilterSelector", f.Collections); err != nil {
		return err
	}

	// set columns
	return column.Set_tx(ctx, tx, schema.DbField, fieldId, f.Columns)
}
func setVariable_tx(ctx context.Context, tx pgx.Tx, fieldId uuid.UUID, f types.FieldVariable) error {

	known, err := schema.CheckCreateId_tx(ctx, tx, &fieldId, schema.DbFieldVariable, "field_id")
	if err != nil {
		return err
	}

	if known {
		if _, err := tx.Exec(ctx, `
			UPDATE app.field_variable
			SET variable_id = $1, js_function_id = $2
			WHERE field_id = $3
		`, f.VariableId, f.JsFunctionId, fieldId); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app.field_variable (field_id, variable_id, js_function_id)
			VALUES ($1,$2,$3)
		`, fieldId, f.VariableId, f.JsFunctionId); err != nil {
			return err
		}
	}

	// set columns
	return column.Set_tx(ctx, tx, schema.DbField, fieldId, f.Columns)
}
