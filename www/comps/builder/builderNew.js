import {getQueryTemplate} from '../shared/query.js';
import {getNilUuid}       from '../shared/generic.js';
import {
	getTemplateArgs,
	getTemplateFnc,
	getTemplateReturn
} from '../shared/templates.js';
import MyBuilderFormInput from './builderFormInput.js';
export {MyBuilderNew as default};

let MyBuilderNew = {
	name:'my-builder-new',
	components:{ MyBuilderFormInput },
	template:`<div class="app-sub-window under-header" @mousedown.self="$emit('close')">
		<div class="contentBox builder-new float">
			<div class="top lower">
				<div class="area nowrap">
					<img class="icon" :src="titleImgSrc" />
					<h1 class="title">{{ title }}</h1>
				</div>
				<div class="area">
					<my-button image="cancel.png"
						@trigger="$emit('close')"
						:cancel="true"
					/>
				</div>
			</div>
			
			<div class="content gap default-inputs">
				<div class="row gap centered">
					<span>{{ capGen.name }}</span>
					<input spellcheck="false" v-model="inputs.name" v-focus />
				</div>
				
				<div
					v-if="typeof capApp.message[entity] !== 'undefined'"
					v-html="capApp.message[entity]"
				></div>
				
				<!-- additional options -->
				<div class="options" v-if="showOptions">
					<h2>{{ capApp.options }}</h2>
					
					<!-- form: duplicate form -->
					<template v-if="entity === 'form'">
						<div class="row centered gap">
							<span>{{ capApp.formIdDuplicate }}</span>
							<my-builder-form-input
								v-model="inputs.formIdDuplicate"
								:module="module"
							/>
						</div>
					</template>
					
					<!-- JS function: assigned form -->
					<template v-if="entity === 'jsFunction'">
						<div class="row centered gap">
							<span>{{ capApp.jsFunctionFormId }}</span>
							<select v-model="inputs.formId">
								<option :value="null">-</option>
								<option v-for="f in module.forms" :value="f.id">{{ f.name }}</option>
							</select>
						</div>
						<p v-html="capApp.jsFunctionFormIdHint"></p>
					</template>
					
					<!-- variable: assigned form -->
					<template v-if="entity === 'variable'">
						<div class="row centered gap">
							<span>{{ capApp.variableFormId }}</span>
							<select v-model="inputs.formId">
								<option :value="null">-</option>
								<option v-for="f in module.forms" :value="f.id">{{ f.name }}</option>
							</select>
						</div>
						<p v-html="capApp.variableFormIdHint"></p>
					</template>
					
					<!-- PG function: trigger/function template -->
					<template v-if="entity === 'pgFunction'">
						<div class="row centered gap">
							<span>{{ capApp.pgFunctionTemplate }}</span>
							<select v-model="inputs.template">
								<option value="">-</option>
								<option value="mailsFromSpooler">{{ capApp.template.mailsFromSpooler }}</option>
								<option value="loginSync">{{ capApp.template.loginSync }}</option>
								<option value="restAuthRequest">{{ capApp.template.restAuthRequest }}</option>
								<option value="restAuthResponse">{{ capApp.template.restAuthResponse }}</option>
								<option value="restDataResponse">{{ capApp.template.restDataResponse }}</option>
							</select>
						</div>
						<hr />
						
						<div class="row centered">
							<span>{{ capApp.pgFunctionTrigger }}</span>
							<my-bool v-model="inputs.isTrigger" />
						</div>
						<p v-html="capApp.pgFunctionTriggerHint"></p>
					</template>
					
					<!-- relation: E2EE encryption -->
					<template v-if="entity === 'relation'">
						<div class="row centered">
							<span>{{ capApp.relationEncryption }}</span>
							<my-bool v-model="inputs.encryption" />
						</div>
						<p v-html="capApp.relationEncryptionHint"></p>
					</template>
				</div>
				
				<p class="error" v-if="nameTaken">{{ capGen.error.nameTaken }}</p>
				<p class="error" v-if="nameTooLong">{{ capGen.error.nameTooLong.replace('{LEN}',nameMaxLength) }}</p>
				
				<div class="row">
					<my-button image="save.png"
						@trigger="set"
						:active="canSave"
						:caption="capGen.button.create"
					/>
				</div>
			</div>
		</div>
	</div>`,
	props:{
		entity:  { type:String, required:true },
		moduleId:{ type:String, required:true },
		presets: { type:Object, required:true } // preset values for inputs
	},
	emits:['close'],
	data() {
		return {
			inputs:{
				// all
				name:'',
				
				// form
				formIdDuplicate:null,
				
				// JS function
				formId:null,
				
				// PG function
				isTrigger:false,
				template:'',
				
				// relation
				encryption:false
			}
		};
	},
	computed:{
		nameMaxLength:(s) => {
			switch(s.entity) {
				case 'api':        return 60; break;
				case 'collection': return 64; break;
				case 'form':       return 64; break;
				case 'jsFunction': return 64; break;
				case 'module':     return 60; break;
				case 'pgFunction': return 60; break;
				case 'relation':   return 60; break;
				case 'role':       return 64; break;
				case 'searchBar':  return 64; break;
				case 'variable':   return 64; break;
				case 'widget':     return 64; break;
			}
			return 0;
		},
		nameTaken:(s) => {
			if(s.inputs.name === '')
				return false;
			
			let searchList;
			switch(s.entity) {
				case 'module':     searchList = s.modules;            break;
				case 'api':        searchList = s.module.apis;        break;
				case 'collection': searchList = s.module.collections; break;
				case 'form':       searchList = s.module.forms;       break;
				case 'jsFunction': searchList = s.module.jsFunctions; break;
				case 'pgFunction': searchList = s.module.pgFunctions; break;
				case 'relation':   searchList = s.module.relations;   break;
				case 'role':       searchList = s.module.roles;       break;
				case 'searchBar':  searchList = s.module.searchBars;  break;
				case 'variable':   searchList = s.module.variables;   break;
				case 'widget':     searchList = s.module.widgets;     break;
			}
			for(let e of searchList) {
				// only compare names of functions within the same scope (global or form)
				if(s.entity === 'jsFunction' && e.formId !== s.inputs.formId)
					continue;

				// only compare names of variables within the same scope (global or form)
				if(s.entity === 'variable' && e.formId !== s.inputs.formId)
					continue;
				
				if(e.name === s.inputs.name)
					return true;
			}
			return false;
		},
		
		// presentation
		title:(s) => {
			switch(s.entity) {
				case 'api':        return s.capApp.api;        break;
				case 'collection': return s.capApp.collection; break;
				case 'form':       return s.capApp.form;       break;
				case 'jsFunction': return s.capApp.jsFunction; break;
				case 'module':     return s.capApp.module;     break;
				case 'pgFunction': return s.capApp.pgFunction; break;
				case 'relation':   return s.capApp.relation;   break;
				case 'role':       return s.capApp.role;       break;
				case 'searchBar':  return s.capApp.searchBar;  break;
				case 'variable':   return s.capApp.variable;   break;
				case 'widget':     return s.capApp.widget;     break;
			}
			return '';
		},
		titleImgSrc:(s) => {
			switch(s.entity) {
				case 'api':        return 'images/api.png';            break;
				case 'collection': return 'images/tray.png';           break;
				case 'form':       return 'images/fileText.png';       break;
				case 'jsFunction': return 'images/codeScreen.png';     break;
				case 'module':     return 'images/module.png';         break;
				case 'pgFunction': return 'images/codeDatabase.png';   break;
				case 'relation':   return 'images/database.png';       break;
				case 'role':       return 'images/personMultiple.png'; break;
				case 'searchBar':  return 'images/search.png';         break;
				case 'variable':   return 'images/variable.png';       break;
				case 'widget':     return 'images/tiles.png';          break;
			}
			return '';
		},

		// simple
		canSave:    (s) => s.inputs.name !== '' && !s.nameTaken && !s.nameTooLong,
		nameTooLong:(s) => s.inputs.name !== '' && s.inputs.name.length > s.nameMaxLength,
		showOptions:(s) => ['form','jsFunction','pgFunction','relation','variable'].includes(s.entity),
		
		// stores
		module:     (s) => s.moduleIdMap[s.moduleId],
		modules:    (s) => s.$store.getters['schema/modules'],
		moduleIdMap:(s) => s.$store.getters['schema/moduleIdMap'],
		capApp:     (s) => s.$store.getters.captions.builder.new,
		capGen:     (s) => s.$store.getters.captions.generic
	},
	mounted() {
		// apply preset input values
		for(let k in this.inputs) {
			if(typeof this.presets[k] !== 'undefined')
				this.inputs[k] = this.presets[k];
		}
		
		this.$store.commit('keyDownHandlerSleep');
		this.$store.commit('keyDownHandlerAdd',{fnc:this.set,key:'s',keyCtrl:true});
		this.$store.commit('keyDownHandlerAdd',{fnc:this.close,key:'Escape'});
	},
	unmounted() {
		this.$store.commit('keyDownHandlerDel',this.set);
		this.$store.commit('keyDownHandlerDel',this.close);
		this.$store.commit('keyDownHandlerWake');
	},
	methods:{
		// externals
		getNilUuid,
		getQueryTemplate,
		getTemplateArgs,
		getTemplateFnc,
		getTemplateReturn,
		
		// actions
		close() { this.$emit('close'); },
		
		// backend calls
		set() {
			if(!this.canSave) return;
			
			let action = 'set';
			let request;
			let dependencyCheck = false;
			switch(this.entity) {
				case 'api':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						name:this.inputs.name,
						comment:null,
						columns:[],
						query:this.getQueryTemplate(),
						hasDelete:false,
						hasGet:true,
						hasPost:false,
						limitDef:100,
						limitMax:1000,
						verboseDef:true,
						version:1
					};
				break;
				case 'collection':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						iconId:null,
						name:this.inputs.name,
						columns:[],
						query:this.getQueryTemplate(),
						inHeader:[]
					};
				break;
				case 'form':
					if(this.inputs.formIdDuplicate !== null) {
						action = 'copy';
						request = {
							id:this.inputs.formIdDuplicate,
							moduleId:this.moduleId,
							newName:this.inputs.name
						};
						dependencyCheck = true;
					} else {
						request = {
							id:this.getNilUuid(),
							moduleId:this.moduleId,
							fieldIdFocus:null,
							presetIdOpen:null,
							iconId:null,
							name:this.inputs.name,
							noDataActions:false,
							query:this.getQueryTemplate(),
							fields:[],
							functions:[],
							states:[],
							actions:[],
							articleIdsHelp:[],
							captions:{
								formTitle:{}
							}
						};
					}
				break;
				case 'jsFunction':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						formId:this.inputs.formId,
						name:this.inputs.name,
						codeArgs:'',
						codeFunction:'',
						codeReturns:'',
						isClientEventExec:false,
						captions:{
							jsFunctionTitle:{},
							jsFunctionDesc:{}
						}
					};
				break;
				case 'module':
					request = {
						id:this.getNilUuid(),
						parentId:null,
						formId:null,
						iconId:null,
						name:this.inputs.name,
						color1:'217A4D',
						position:0,
						releaseBuild:0,
						releaseBuildApp:0,
						releaseDate:0,
						languageMain:'en_us',
						languages:['en_us'],
						dependsOn:[],
						startForms:[],
						articleIdsHelp:[],
						captions:{
							moduleTitle:{}
						}
					};
				break;
				case 'pgFunction':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						name:this.inputs.name,
						codeArgs:this.getTemplateArgs(this.inputs.template),
						codeFunction:this.getTemplateFnc(this.inputs.template,this.inputs.isTrigger),
						codeReturns:this.getTemplateReturn(this.inputs.isTrigger),
						isFrontendExec:false,
						isLoginSync:this.inputs.template === 'loginSync',
						isTrigger:this.inputs.isTrigger,
						volatility:'VOLATILE',
						schedules:[],
						captions:{
							pgFunctionTitle:{},
							pgFunctionDesc:{}
						}
					};
				break;
				case 'relation':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						name:this.inputs.name,
						comment:null,
						encryption:this.inputs.encryption,
						retentionCount:null,
						retentionDays:null,
						policies:[]
					};
				break;
				case 'role':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						content:'user',
						name:this.inputs.name,
						assignable:true,
						captions:{},
						childrenIds:[],
						accessApis:{},
						accessAttributes:{},
						accessClientEvents:{},
						accessCollections:{},
						accessMenus:{},
						accessRelations:{}
					};
				break;
				case 'searchBar':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						iconId:null,
						name:this.inputs.name,
						columns:[],
						query:this.getQueryTemplate(),
						openForm:null,
						captions:{
							searchBarTitle:{}
						}
					};
				break;
				case 'variable':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						formId:this.inputs.formId,
						name:this.inputs.name,
						comment:null,
						content:'text',
						contentUse:'default'
					};
				break;
				case 'widget':
					request = {
						id:this.getNilUuid(),
						moduleId:this.moduleId,
						formId:null,
						size:1,
						name:this.inputs.name
					};
				break;
				default: return; break;
			}
			
			let requests = [ws.prepare(this.entity,action,request)];
			
			if(dependencyCheck)
				requests.push(ws.prepare('schema','check',{moduleId:this.moduleId}));
			
			ws.sendMultiple(requests,true).then(
				res => {
					if(this.entity === 'module')
						this.$root.schemaReload(res[0].payload);
					else
						this.$root.schemaReload(this.moduleId);
					
					this.$emit('close');
				},
				this.$root.genericError
			);
		}
	}
};