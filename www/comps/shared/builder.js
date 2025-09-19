import {isAttributeRelationship} from './attribute.js';
import {
	getJoinIndexMap,
	getQueryExpressions,
	getQueryFiltersProcessed,
	getRelationsJoined
} from './query.js';
import MyStore from '../../stores/store.js';

export function getValidDbCharsForRx() {
	// valid characters for DB entity names (PG functions, modules, relations, attributes, ...)
	return '[a-z0-9_]+';
};

export function getFieldHasQuery(field) {
	if(field.content === 'variable' && field.variableId !== null)
		return isAttributeRelationship(MyStore.getters['schema/variableIdMap'][field.variableId].content);

	return ['calendar','chart','kanban','list'].includes(field.content)
		? true : field.content === 'data' && isAttributeRelationship(
			MyStore.getters['schema/attributeIdMap'][field.attributeId].content
		);
};

export function getFormEntityMapRef(fieldsParent,formActions) {
	let refs           = { field:{}, formAction:{}, tab:{} };
	let ctrFields      = 0; // unique reference number for each field
	let ctrFormActions = 0; // unique reference number for each form action
	let ctrTabs        = 0; // unique reference number for each tab
	
	const collectFromFields = function(fields) {
		for(let f of fields) {
			refs.field[f.id] = ctrFields++;
			switch(f.content) {
				case 'container': collectFromFields(f.fields); break;
				case 'tabs':
					for(let t of f.tabs) {
						refs.tab[t.id] = ctrTabs++;
						collectFromFields(t.fields);
					}
				break;
			}
		}
	};
	collectFromFields(fieldsParent);

	for(const a of formActions) {
		refs.formAction[a.id] = ctrFormActions++;
	}
	return refs;
};

export function getJsFunctionsProcessed(fncs,filter) {
	let outGlobal = [];
	let outForm   = [];
	let formIdMap = MyStore.getters['schema/formIdMap'];
	filter        = filter.toLowerCase();
	
	for(let fnc of fncs) {
		if(filter !== '') {
			if(fnc.formId === null && !fnc.name.toLowerCase().includes(filter))
				continue;
			
			if(fnc.formId !== null && !fnc.name.toLowerCase().includes(filter)
				&& !formIdMap[fnc.formId].name.toLowerCase().includes(filter)) {
				
				continue;
			}
		}
		
		if(fnc.formId === null) outGlobal.push(fnc);
		else                    outForm.push(fnc);
	}
	
	// sort by form name
	outForm.sort((a,b) => (formIdMap[a.formId].name > formIdMap[b.formId].name) ? 1 : -1);
	
	// order: generic then functions assigned to a form
	return outGlobal.concat(outForm);
};

export function getDependentModules(moduleSource) {
	return MyStore.getters['schema/modules'].filter(v => v.id === moduleSource.id || moduleSource.dependsOn.includes(v.id));
};

export function getDependentRelations(moduleSource) {
	const modules = getDependentModules(moduleSource);
	let rels = [];
	for(const mod of modules) {
		rels = rels.concat(mod.relations);
	}
	return rels;
};

export function getDependentAttributes(moduleSource) {
	const modules = getDependentModules(moduleSource);
	let atrs = [];
	for(const mod of modules) {
		for(const rel of mod.relations) {
			atrs = atrs.concat(rel.attributes);
		}
	}
	return atrs;
};

export function getFunctionHelp(functionPrefix,functionObj,builderLanguage) {
	let help = `${functionObj.name}(${functionObj.codeArgs}) => ${functionObj.codeReturns}`;
	
	// add translated title/description, if available
	let cap = `${functionPrefix}FunctionTitle`;
	if(typeof functionObj.captions[cap] !== 'undefined'
		&& typeof functionObj.captions[cap][builderLanguage] !== 'undefined'
		&& functionObj.captions[cap][builderLanguage] !== '') {
		
		help += `<br /><br />${functionObj.captions[cap][builderLanguage]}`;
	}
	
	cap = `${functionPrefix}FunctionDesc`;
	if(typeof functionObj.captions[cap] !== 'undefined'
		&& typeof functionObj.captions[cap][builderLanguage] !== 'undefined'
		&& functionObj.captions[cap][builderLanguage] !== '') {
		
		help += `<br /><br />${functionObj.captions[cap][builderLanguage]}`;
	}
	return help;
};

export function getSqlPreview(query,columns) {
	ws.send('dataSql','get',{
		relationId:query.relationId,
		joins:getRelationsJoined(query.joins),
		expressions:getQueryExpressions(columns),
		filters:getQueryFiltersProcessed(query.filters,getJoinIndexMap(query.joins)),
		orders:query.orders,
		limit:query.fixedLimit !== 0 ? query.fixedLimit : 0
	},true).then(
		res => {
			MyStore.commit('dialog',{
				captionTop:MyStore.getters.captions.generic.sqlPreview,
				captionBody:res.payload,
				image:'database.png',
				textDisplay:'textarea',
				width:800
			});
		},
		MyStore.getters.appFunctions.genericError
	);
};

export function getValueFromJson(inputJson,nameChain,valueFallback) {
	let o = JSON.parse(inputJson);
	for(let i = 0, j = nameChain.length; i < j; i++) {
		
		if(typeof o[nameChain[i]] === 'undefined')
			return valueFallback;
		
		o = o[nameChain[i]];
	}
	return o;
};

export function setValueInJson(inputJson,nameChain,value) {
	let o = JSON.parse(inputJson);
	let s = o;
	
	for(let i = 0, j = nameChain.length; i < j; i++) {
		
		if(i+1 === j) {
			// last element reached, set value and break out
			s[nameChain[i]] = value;
			break;
		}
		
		if(typeof s[nameChain[i]] === 'undefined')
			s[nameChain[i]] = {};
		
		s = s[nameChain[i]];
	}
	return JSON.stringify(o,null,2);
};

export function getItemTitle(attributeId,index,outsideIn,attributeIdNm) {
	let atr = MyStore.getters['schema/attributeIdMap'][attributeId];
	let rel = MyStore.getters['schema/relationIdMap'][atr.relationId];
	
	if(isAttributeRelationship(atr.content) && typeof attributeIdNm !== 'undefined' && attributeIdNm !== null) {
		let atrNm = MyStore.getters['schema/attributeIdMap'][attributeIdNm];
		return `${index} ${rel.name}.${atr.name} -> ${atrNm.name}`;
	}
	return `${index} ${rel.name}.${atr.name}`;
};

export function getItemTitlePath(attributeId) {
	let atr = MyStore.getters['schema/attributeIdMap'][attributeId];
	let rel = MyStore.getters['schema/relationIdMap'][atr.relationId];
	let mod = MyStore.getters['schema/moduleIdMap'][rel.moduleId];
	return `${mod.name}.${rel.name}.${atr.name}`;
};

export function getItemTitleNoRelationship(attributeId,index) {
	let atr = MyStore.getters['schema/attributeIdMap'][attributeId];
	let rel = MyStore.getters['schema/relationIdMap'][atr.relationId];
	return `${index}) ${rel.name}.${atr.name}`;
};

export function getItemTitleColumn(column,withTitle) {
	let name;
	if(column.subQuery) name = `SubQuery`;
	else                name = getItemTitle(column.attributeId,column.index,false,null);
	
	if(withTitle && typeof column.captions.columnTitle[MyStore.getters.settings.languageCode] !== 'undefined')
		name = `${name} (${column.captions.columnTitle[MyStore.getters.settings.languageCode]})`;
	
	return name;
};

export function getItemTitleRelation(relationId,index) {
	return `${index} ${MyStore.getters['schema/relationIdMap'][relationId].name}`;
};

// locally stored builder options
export function builderOptionGet(optionName,fallbackValue) {
	return MyStore.getters['local/builderOptionMap'][optionName] !== undefined
		? MyStore.getters['local/builderOptionMap'][optionName] : fallbackValue;
};
export function builderOptionSet(optionName,value) {
	MyStore.commit('local/builderOptionSet',{
		name:optionName,
		value:value
	});
};