package types

import (
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Login struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}
type LoginAccess struct {
	RoleIds     []uuid.UUID          `json:"roleIds"`     // all assigned roles (incl. inherited)
	Api         map[uuid.UUID]Access `json:"api"`         // effective access to specific API
	Attribute   map[uuid.UUID]Access `json:"attribute"`   // effective access to specific attributes
	ClientEvent map[uuid.UUID]Access `json:"clientEvent"` // effective access to specific client events
	Collection  map[uuid.UUID]Access `json:"collection"`  // effective access to specific collection
	Menu        map[uuid.UUID]Access `json:"menu"`        // effective access to specific menus
	Relation    map[uuid.UUID]Access `json:"relation"`    // effective access to specific relations
	SearchBar   map[uuid.UUID]Access `json:"searchBar"`   // effective access to specific search bars
	Widget      map[uuid.UUID]Access `json:"widget"`      // effective access to specific widgets
}
type LoginAuthResult struct {
	// auth types: user, token, fixed token, openId
	Admin bool   `json:"admin"` // login has instance admin permissions
	Id    int64  `json:"id"`    // login ID
	Name  string `json:"name"`  // login name (unique in instance)
	Token string `json:"token"` // login token

	// auth types: user
	MfaTokens []LoginMfaToken `json:"mfaTokens"` // available MFAs, filled if user auth ok, but MFA not satisfied
	NoAuth    bool            `json:"noAuth"`    // login is without authentication (public auth with only name)

	// auth types: user, openId
	SaltKdf string `json:"saltKdf"`

	// auth types: token, fixed token
	LanguageCode string `json:"languageCode"`
}
type LoginClientEvent struct {
	// login client events exist if a login has enabled a hotkey client event
	HotkeyChar      string      `json:"hotkeyChar"`
	HotkeyModifier1 string      `json:"hotkeyModifier1"`
	HotkeyModifier2 pgtype.Text `json:"hotkeyModifier2"`
}
type LoginFavorite struct {
	Id       uuid.UUID   `json:"id"`
	FormId   uuid.UUID   `json:"formId"`   // ID of form to show
	RecordId pgtype.Int8 `json:"recordId"` // ID of record to open, NULL if no record to open
	Title    pgtype.Text `json:"title"`    // user defined title of favorite, empty if not set
}
type LoginMeta struct {
	Department    string `json:"department"`
	Email         string `json:"email"`
	Location      string `json:"location"`
	Notes         string `json:"notes"`
	Organization  string `json:"organization"`
	PhoneFax      string `json:"phoneFax"`
	PhoneLandline string `json:"phoneLandline"`
	PhoneMobile   string `json:"phoneMobile"`
	NameDisplay   string `json:"nameDisplay"`
	NameFore      string `json:"nameFore"`
	NameSur       string `json:"nameSur"`
}
type LoginMfaToken struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}
type LoginOptions struct {
	FavoriteId pgtype.UUID `json:"favoriteId"` // NOT NULL if options are valid in context of a favorite form
	FieldId    uuid.UUID   `json:"fieldId"`
	Options    string      `json:"options"`
}
type LoginPublicKey struct {
	LoginId   int64   `json:"loginId"`   // ID of login
	PublicKey string  `json:"publicKey"` // public key of login (not encrypted)
	RecordIds []int64 `json:"recordIds"` // IDs of record not yet encrypted with public key
}
type LoginRecord struct {
	Id   int64  `json:"id"`   // ID of relation record
	Name string `json:"name"` // name for relation record (based on lookup attribute)
}
type LoginRoleAssign struct {
	RoleId       uuid.UUID `json:"roleId"`
	SearchString string    `json:"searchString"` // if value matches this string, role is assigned
}
type LoginTokenFixed struct {
	Id         int64  `json:"id"`
	Name       string `json:"name"`    // to identify token user/device
	Context    string `json:"context"` // what is being used for (client, ics, totp)
	Token      string `json:"token"`
	DateCreate int64  `json:"dateCreate"`
}
type LoginWidgetGroupItem struct {
	WidgetId pgtype.UUID `json:"widgetId"` // ID of a module widget, empty if system widget is used
	ModuleId pgtype.UUID `json:"moduleId"` // ID of a module, if relevant for widget (systemModuleMenu)
	Content  string      `json:"content"`  // content of widget (moduleWidget, systemModuleMenu)
}
type LoginWidgetGroup struct {
	Title string                 `json:"title"`
	Items []LoginWidgetGroupItem `json:"items"`
}
