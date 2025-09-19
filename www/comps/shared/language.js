import MyStore from '../../stores/store.js';

export function getCaption(content,moduleId,id,captions,fallback) {
	if(MyStore.getters['schema/moduleIdMap'][moduleId] === undefined) return '';
	return getCaptionForLang(content,MyStore.getters.moduleIdMapLang[moduleId],
		id,captions,fallback,MyStore.getters['schema/moduleIdMap'][moduleId].languageMain);
};
export function getCaptionForLang(content,lang,id,captions,fallback,fallbackLang) {
	const captionsCustom = MyStore.getters.captionMapCustom[getCaptionMapName(content)];
	if(captionsCustom[id]?.[content]?.[lang] !== undefined) return captionsCustom[id][content][lang];
	if(captions?.[content]?.[lang]           !== undefined) return captions[content][lang];
	
	if(fallbackLang !== undefined)
		return getCaptionForLang(content,fallbackLang,id,captions,fallback);
	
	return fallback !== undefined ? fallback : '';
};
export function getCaptionMapName(content) {
	switch(content) {
		case 'articleBody':      // fallthrough
		case 'articleTitle':     return 'articleIdMap';     break;
		case 'attributeTitle':   return 'attributeIdMap';   break;
		case 'clientEventTitle': return 'clientEventIdMap'; break;
		case 'columnTitle':      return 'columnIdMap';      break;
		case 'fieldHelp':        // fallthrough
		case 'fieldTitle':       return 'fieldIdMap';       break;
		case 'formActionTitle':  return 'formActionIdMap';  break;
		case 'formTitle':        return 'formIdMap';        break;
		case 'jsFunctionTitle':  return 'jsFunctionIdMap';  break;
		case 'loginFormTitle':   return 'loginFormIdMap';   break;
		case 'menuTitle':        return 'menuIdMap';        break;
		case 'menuTabTitle':     return 'menuTabIdMap';     break;
		case 'moduleTitle':      return 'moduleIdMap';      break;
		case 'pgFunctionTitle':  return 'pgFunctionIdMap';  break;
		case 'queryChoiceTitle': return 'queryChoiceIdMap'; break;
		case 'roleDesc':         // fallthrough
		case 'roleTitle':        return 'roleIdMap';        break;
		case 'searchBarTitle':   return 'searchBarIdMap';   break;
		case 'tabTitle':         return 'tabIdMap';         break;
		case 'widgetTitle':      return 'widgetIdMap';      break;
	}
	return '';
};
export function getDictByLang() {
	let dict = 'simple';
	switch(MyStore.getters.settings.languageCode.substring(0,2)) {
		case 'ar': dict = 'arabic';    break;
		case 'de': dict = 'german';    break;
		case 'en': dict = 'english';   break;
		case 'es': dict = 'spanish';   break;
		case 'fr': dict = 'french';    break;
		case 'hu': dict = 'hungarian'; break;
		case 'it': dict = 'italian';   break;
		case 'ro': dict = 'romanian';  break;
	}
	// apply dictionary if supported by the system
	return MyStore.getters.searchDictionaries.includes(dict) ? dict : 'simple';
};