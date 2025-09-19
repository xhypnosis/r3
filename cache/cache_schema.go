// Application cache
// Used during regular operation for fast lookups.
// Is NOT used while manipulating the schema.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"r3/config/module_meta"
	"r3/log"
	"r3/schema/api"
	"r3/schema/article"
	"r3/schema/attribute"
	"r3/schema/clientEvent"
	"r3/schema/collection"
	"r3/schema/form"
	"r3/schema/icon"
	"r3/schema/jsFunction"
	"r3/schema/loginForm"
	"r3/schema/menuTab"
	"r3/schema/module"
	"r3/schema/pgFunction"
	"r3/schema/pgIndex"
	"r3/schema/pgTrigger"
	"r3/schema/preset"
	"r3/schema/relation"
	"r3/schema/role"
	"r3/schema/searchBar"
	"r3/schema/variable"
	"r3/schema/widget"
	"r3/tools"
	"r3/types"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/maps"
)

var (
	// schema cache access and state
	Schema_mx sync.RWMutex

	// schema cache
	moduleIdMapJson = make(map[uuid.UUID]json.RawMessage)  // ID map of module definition as JSON
	moduleIdMapMeta = make(map[uuid.UUID]types.ModuleMeta) // ID map of module meta data

	// cached entities for regular use during normal operation
	ModuleIdMap        = make(map[uuid.UUID]types.Module)      // all modules by ID
	ModuleApiNameMapId = make(map[string]map[string]uuid.UUID) // all API IDs by module+API name
	RelationIdMap      = make(map[uuid.UUID]types.Relation)    // all relations by ID
	AttributeIdMap     = make(map[uuid.UUID]types.Attribute)   // all attributes by ID
	RoleIdMap          = make(map[uuid.UUID]types.Role)        // all roles by ID
	PgFunctionIdMap    = make(map[uuid.UUID]types.PgFunction)  // all PG functions by ID
	ApiIdMap           = make(map[uuid.UUID]types.Api)         // all APIs by ID
	ClientEventIdMap   = make(map[uuid.UUID]types.ClientEvent) // all client events by ID
)

func GetModuleIdMapMeta() map[uuid.UUID]types.ModuleMeta {
	Schema_mx.RLock()
	defer Schema_mx.RUnlock()
	return moduleIdMapMeta
}
func GetModuleCacheJson(moduleId uuid.UUID) (json.RawMessage, error) {
	Schema_mx.RLock()
	defer Schema_mx.RUnlock()

	json, exists := moduleIdMapJson[moduleId]
	if !exists {
		return []byte{}, fmt.Errorf("module %s does not exist in schema cache", moduleId)
	}
	return json, nil
}
func LoadModuleIdMapMeta_tx(ctx context.Context, tx pgx.Tx) error {
	moduleIdMapMetaNew, err := module_meta.GetIdMap_tx(ctx, tx)
	if err != nil {
		return err
	}
	Schema_mx.Lock()
	defer Schema_mx.Unlock()

	// apply deletions if relevant
	for id, _ := range moduleIdMapMeta {
		if _, exists := moduleIdMapMetaNew[id]; !exists {
			delete(ModuleIdMap, id)
			delete(moduleIdMapJson, id)
		}
	}

	// set new meta data
	moduleIdMapMeta = moduleIdMapMetaNew
	return nil
}

// load all modules into the schema cache
func LoadSchema_tx(ctx context.Context, tx pgx.Tx) error {
	return UpdateSchema_tx(ctx, tx, maps.Keys(moduleIdMapMeta), true)
}

// update module schema cache
func UpdateSchema_tx(ctx context.Context, tx pgx.Tx, moduleIds []uuid.UUID, initialLoad bool) error {
	var err error

	if err := updateSchemaCache_tx(ctx, tx, moduleIds); err != nil {
		return err
	}

	// renew caches, affected by potentially changed modules (preset records, login access)
	renewIcsFields()
	if err := renewPresetRecordIds_tx(ctx, tx); err != nil {
		return err
	}

	// create JSON copy of schema cache for fast retrieval
	for _, id := range moduleIds {
		Schema_mx.Lock()
		moduleIdMapJson[id], err = json.Marshal(ModuleIdMap[id])
		Schema_mx.Unlock()
		if err != nil {
			return err
		}
	}

	if initialLoad {
		return nil
	}

	// update change date for updated modules
	now := tools.GetTimeUnix()
	if err := module_meta.SetDateChange_tx(ctx, tx, moduleIds, now); err != nil {
		return err
	}

	// update module meta cache
	Schema_mx.Lock()
	for _, id := range moduleIds {
		meta, exists := moduleIdMapMeta[id]
		if !exists {
			meta, err = module_meta.Get_tx(ctx, tx, id)
			if err != nil {
				return err
			}
		}
		meta.DateChange = now
		moduleIdMapMeta[id] = meta
	}
	Schema_mx.Unlock()
	return nil
}

func updateSchemaCache_tx(ctx context.Context, tx pgx.Tx, moduleIds []uuid.UUID) error {
	Schema_mx.Lock()
	defer Schema_mx.Unlock()

	log.Info(log.ContextCache, fmt.Sprintf("starting schema processing for %d module(s)", len(moduleIds)))

	mods, err := module.Get_tx(ctx, tx, moduleIds)
	if err != nil {
		return err
	}
	for _, mod := range mods {
		log.Info(log.ContextCache, fmt.Sprintf("parsing module '%s'", mod.Name))
		mod.Relations = make([]types.Relation, 0)
		mod.Forms = make([]types.Form, 0)
		mod.MenuTabs = make([]types.MenuTab, 0)
		mod.Icons = make([]types.Icon, 0)
		mod.Roles = make([]types.Role, 0)
		mod.Articles = make([]types.Article, 0)
		mod.LoginForms = make([]types.LoginForm, 0)
		mod.PgFunctions = make([]types.PgFunction, 0)
		mod.JsFunctions = make([]types.JsFunction, 0)
		mod.Collections = make([]types.Collection, 0)
		mod.Apis = make([]types.Api, 0)
		mod.ClientEvents = make([]types.ClientEvent, 0)
		mod.SearchBars = make([]types.SearchBar, 0)
		mod.Variables = make([]types.Variable, 0)
		mod.Widgets = make([]types.Widget, 0)
		ModuleApiNameMapId[mod.Name] = make(map[string]uuid.UUID)

		// get articles
		log.Info(log.ContextCache, "load articles")

		mod.Articles, err = article.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get relations
		log.Info(log.ContextCache, "load relations")

		rels, err := relation.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		for _, rel := range rels {

			// get attributes
			atrs, err := attribute.Get_tx(ctx, tx, rel.Id)
			if err != nil {
				return err
			}

			// store & backfill attribute to relation
			for _, atr := range atrs {
				AttributeIdMap[atr.Id] = atr
				rel.Attributes = append(rel.Attributes, atr)
			}

			// get indexes
			rel.Indexes, err = pgIndex.Get_tx(ctx, tx, rel.Id)
			if err != nil {
				return err
			}

			// get presets
			rel.Presets, err = preset.Get_tx(ctx, tx, rel.Id)
			if err != nil {
				return err
			}

			// store & backfill relation to module
			RelationIdMap[rel.Id] = rel
			mod.Relations = append(mod.Relations, rel)
		}

		// get forms
		log.Info(log.ContextCache, "load forms")

		mod.Forms, err = form.Get_tx(ctx, tx, mod.Id, []uuid.UUID{})
		if err != nil {
			return err
		}

		// get menu tabs
		log.Info(log.ContextCache, "load menu tabs")

		mod.MenuTabs, err = menuTab.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get icons
		log.Info(log.ContextCache, "load icons")

		mod.Icons, err = icon.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get roles
		log.Info(log.ContextCache, "load roles")

		mod.Roles, err = role.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		for _, rol := range mod.Roles {
			// store role
			RoleIdMap[rol.Id] = rol
		}

		// get login forms
		log.Info(log.ContextCache, "load login forms")

		mod.LoginForms, err = loginForm.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get triggers
		mod.PgTriggers, err = pgTrigger.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// store & backfill PG functions
		log.Info(log.ContextCache, "load PG functions")

		mod.PgFunctions, err = pgFunction.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}
		for _, fnc := range mod.PgFunctions {
			PgFunctionIdMap[fnc.Id] = fnc
		}

		// get JS functions
		log.Info(log.ContextCache, "load JS functions")

		mod.JsFunctions, err = jsFunction.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get collections
		log.Info(log.ContextCache, "load collections")

		mod.Collections, err = collection.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get APIs
		log.Info(log.ContextCache, "load APIs")

		mod.Apis, err = api.Get_tx(ctx, tx, mod.Id, uuid.Nil)
		if err != nil {
			return err
		}
		for _, a := range mod.Apis {
			ApiIdMap[a.Id] = a
			ModuleApiNameMapId[mod.Name][fmt.Sprintf("%s.v%d", a.Name, a.Version)] = a.Id
		}

		// get client events
		log.Info(log.ContextCache, "load client events")

		mod.ClientEvents, err = clientEvent.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}
		for _, ce := range mod.ClientEvents {
			ClientEventIdMap[ce.Id] = ce
		}

		// get search bars
		log.Info(log.ContextCache, "load search bars")
		mod.SearchBars, err = searchBar.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get variables
		log.Info(log.ContextCache, "load variables")

		mod.Variables, err = variable.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// get widgets
		log.Info(log.ContextCache, "load widgets")

		mod.Widgets, err = widget.Get_tx(ctx, tx, mod.Id)
		if err != nil {
			return err
		}

		// update cache map with parsed module
		ModuleIdMap[mod.Id] = mod
	}
	return nil
}
