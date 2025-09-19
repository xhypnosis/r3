package login_setting

import (
	"context"
	"errors"
	"fmt"
	"r3/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Get_tx(ctx context.Context, tx pgx.Tx, loginId pgtype.Int8, loginTemplateId pgtype.Int8) (types.Settings, error) {

	var s types.Settings
	if (loginId.Valid && loginTemplateId.Valid) || (!loginId.Valid && !loginTemplateId.Valid) {
		return s, errors.New("settings can only be retrieved for either login or login template")
	}

	entryId := loginId.Int64
	entryName := "login_id"

	if loginTemplateId.Valid {
		entryId = loginTemplateId.Int64
		entryName = "login_template_id"
	}

	err := tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT language_code, date_format, sunday_first_dow, font_size,
			borders_squared, header_captions, header_modules, spacing, dark,
			hint_update_version, mobile_scroll_form, form_actions_align,
			warn_unsaved, pattern, font_family, tab_remember, list_colored,
			list_spaced, color_classic_mode, color_header, color_header_single,
			color_menu, number_sep_decimal, number_sep_thousand, bool_as_icon,
			shadows_inputs
		FROM instance.login_setting AS ls
		WHERE %s = $1
	`, entryName), entryId).Scan(&s.LanguageCode, &s.DateFormat, &s.SundayFirstDow,
		&s.FontSize, &s.BordersSquared, &s.HeaderCaptions, &s.HeaderModules,
		&s.Spacing, &s.Dark, &s.HintUpdateVersion, &s.MobileScrollForm, &s.FormActionsAlign,
		&s.WarnUnsaved, &s.Pattern, &s.FontFamily, &s.TabRemember, &s.ListColored,
		&s.ListSpaced, &s.ColorClassicMode, &s.ColorHeader, &s.ColorHeaderSingle,
		&s.ColorMenu, &s.NumberSepDecimal, &s.NumberSepThousand, &s.BoolAsIcon,
		&s.ShadowsInputs)

	return s, err
}

func Set_tx(ctx context.Context, tx pgx.Tx, loginId pgtype.Int8, loginTemplateId pgtype.Int8, s types.Settings, isNew bool) error {

	if (loginId.Valid && loginTemplateId.Valid) || (!loginId.Valid && !loginTemplateId.Valid) {
		return errors.New("settings can only be applied for either login or login template")
	}

	entryId := loginId.Int64
	entryName := "login_id"

	if loginTemplateId.Valid {
		entryId = loginTemplateId.Int64
		entryName = "login_template_id"
	}

	if isNew {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO instance.login_setting (%s, language_code, date_format,	sunday_first_dow,
				font_size, borders_squared, header_captions, header_modules, spacing,
				dark, hint_update_version, mobile_scroll_form, form_actions_align, warn_unsaved,
				pattern, font_family, tab_remember, list_colored, list_spaced, color_classic_mode,
				color_header, color_header_single,color_menu, number_sep_decimal,
				number_sep_thousand, bool_as_icon, shadows_inputs)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,
				$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
		`, entryName), entryId, s.LanguageCode, s.DateFormat, s.SundayFirstDow, s.FontSize,
			s.BordersSquared, s.HeaderCaptions, s.HeaderModules, s.Spacing, s.Dark,
			s.HintUpdateVersion, s.MobileScrollForm, s.FormActionsAlign, s.WarnUnsaved, s.Pattern,
			s.FontFamily, s.TabRemember, s.ListColored, s.ListSpaced, s.ColorClassicMode,
			s.ColorHeader, s.ColorHeaderSingle, s.ColorMenu, s.NumberSepDecimal,
			s.NumberSepThousand, s.BoolAsIcon, s.ShadowsInputs); err != nil {

			return err
		}
	} else {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			UPDATE instance.login_setting
			SET language_code = $1, date_format = $2, sunday_first_dow = $3, font_size = $4,
				borders_squared = $5, header_captions = $6, header_modules = $7,
				spacing = $8, dark = $9, hint_update_version = $10, mobile_scroll_form = $11,
				form_actions_align = $12, warn_unsaved = $13, pattern = $14, font_family = $15,
				tab_remember = $16, list_colored = $17, list_spaced = $18, color_classic_mode = $19,
				color_header = $20, color_header_single = $21, color_menu = $22, number_sep_decimal = $23,
				number_sep_thousand = $24, bool_as_icon = $25, shadows_inputs = $26
			WHERE %s = $27
		`, entryName), s.LanguageCode, s.DateFormat, s.SundayFirstDow, s.FontSize,
			s.BordersSquared, s.HeaderCaptions, s.HeaderModules, s.Spacing, s.Dark,
			s.HintUpdateVersion, s.MobileScrollForm, s.FormActionsAlign, s.WarnUnsaved, s.Pattern,
			s.FontFamily, s.TabRemember, s.ListColored, s.ListSpaced, s.ColorClassicMode,
			s.ColorHeader, s.ColorHeaderSingle, s.ColorMenu, s.NumberSepDecimal,
			s.NumberSepThousand, s.BoolAsIcon, s.ShadowsInputs, entryId); err != nil {

			return err
		}
	}
	return nil
}
