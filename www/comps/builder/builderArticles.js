import MyBuilderCaption      from './builderCaption.js';
import MyArticles            from '../articles.js';
import {getDependentModules} from '../shared/builder.js';
import {copyValueDialog}     from '../shared/generic.js';
export {MyBuilderArticles as default};

let MyBuilderArticlesItem = {
	name:'my-builder-articles-item',
	components:{MyBuilderCaption},
	template:`<tbody>
		<tr>
			<td>
				<div class="row gap">
					<my-button image="save.png"
						@trigger="set"
						:active="!readonly && hasChanges"
						:caption="isNew ? capGen.button.create : ''"
						:captionTitle="isNew ? capGen.button.create : capGen.button.save"
					/>
					<my-button image="delete.png"
						v-if="!isNew"
						@trigger="delAsk"
						:active="!readonly"
						:cancel="true"
						:captionTitle="capGen.button.delete"
					/>
				</div>
			</td>
			<td><input class="long" v-model="name" :disabled="readonly" :placeholder="isNew ? capApp.new : ''" /></td>
			<td>
				<my-button image="visible1.png"
					@trigger="copyValueDialog(article.name,article.id,article.id)"
					:active="!isNew"
				/>
			</td>
			<td>
				<my-builder-caption
					v-model="captions.articleTitle"
					:language="builderLanguage"
					:readonly="readonly"
				/>
			</td>
			<td>
				<my-button image="edit.png"
					@trigger="showContent = !showContent"
					:caption="capGen.button.edit"
				/>
			</td>
			<td>
				<!-- article body pop up window -->
				<div class="app-sub-window under-header" v-if="showContent" @mousedown.self="showContent = false">
					<div class="contentBox builder-articles-body shade popUp">
						<div class="top lower">
							<div class="area">
								<img class="icon" src="images/question.png" />
								<h1>{{ article.name }}</h1>
							</div>
							
							<div class="area">
								<span>{{ capGen.title }}</span>
								<my-builder-caption
									v-model="captions.articleTitle"
									:language="builderLanguage"
									:readonly="readonly"
								/>
								<my-button image="languages.png"
									@trigger="$emit('nextLanguage')"
									:active="module.languages.length > 1"
									:caption="builderLanguage"
								/>
							</div>
							
							<div class="area">
								<my-button image="save.png"
									@trigger="set"
									:active="hasChanges"
									:caption="capGen.button.save"
								/>
								<my-button image="cancel.png"
									@trigger="showContent = false"
									:cancel="true"
									:captionTitle="capGen.button.close"
								/>
							</div>
						</div>
						<div class="content grow no-padding builder-articles-body-richtext">
							<my-builder-caption
								v-model="captions.articleBody"
								:contentName="''"
								:language="builderLanguage"
								:readonly="readonly"
								:richtext="true"
							/>
						</div>
					</div>
				</div>
			</td>
		</tr>
	</tbody>`,
	props:{
		builderLanguage:{ type:String,  required:true },
		module:         { type:Object,  required:true },
		readonly:       { type:Boolean, required:true },
		article:        { type:Object,  required:false,
			default() { return{
				id:null,
				name:'',
				captions:{
					articleTitle:{},
					articleBody:{}
				}
			}}
		}
	},
	emits:['nextLanguage'],
	data() {
		return {
			captions:JSON.parse(JSON.stringify(this.article.captions)),
			name:this.article.name,
			
			// states
			showContent:false
		};
	},
	computed:{
		hasChanges:(s) => s.name !== s.article.name
			|| JSON.stringify(s.captions) !== JSON.stringify(s.article.captions),
		
		// simple states
		isNew:(s) => s.article.id === null,
		
		// stores
		capApp:(s) => s.$store.getters.captions.builder.articles,
		capGen:(s) => s.$store.getters.captions.generic
	},
	mounted() {
		window.addEventListener('keydown',this.handleHotkeys);
	},
	unmounted() {
		window.removeEventListener('keydown',this.handleHotkeys);
	},
	methods:{
		// externals
		copyValueDialog,
		
		// actions
		handleHotkeys(e) {
			if(e.key === 'Escape' && this.showContent)
				this.showContent = false;
			
			if(e.ctrlKey && e.key === 's') {
				e.preventDefault();
				
				if(this.hasChanges)
					this.set();
			}
			
			if(e.ctrlKey && e.key === 'q') {
				e.preventDefault();
				
				this.$emit('nextLanguage');
			}
		},
		
		// backend calls
		delAsk() {
			this.$store.commit('dialog',{
				captionBody:this.capApp.dialog.delete,
				buttons:[{
					cancel:true,
					caption:this.capGen.button.delete,
					exec:this.del,
					image:'delete.png'
				},{
					caption:this.capGen.button.cancel,
					image:'cancel.png'
				}]
			});
		},
		del() {
			ws.send('article','del',{id:this.article.id},true).then(
				() => this.$root.schemaReload(this.module.id),
				this.$root.genericError
			);
		},
		set() {
			ws.send('article','set',{
				id:this.article.id,
				moduleId:this.module.id,
				name:this.name,
				captions:this.captions
			},true).then(
				() => {
					if(this.isNew) {
						this.name     = '';
						this.captions = {
							articleTitle:{},
							articleBody:{}
						};
					}
					this.$root.schemaReload(this.module.id);
				},
				this.$root.genericError
			);
		}
	}
};

let MyBuilderArticles = {
	name:'my-builder-articles',
	components:{
		MyArticles,
		MyBuilderArticlesItem
	},
	template:`<div class="builder-articles contentBox grow">
		<div class="top lower">
			<div class="area nowrap">
				<img class="icon" src="images/question.png" />
				<h1 class="title">{{ capApp.title }}</h1>
			</div>
		</div>
		
		<div class="builder-articles-wrap">
			<div class="builder-articles-edit content default-inputs" v-if="module">
				<table>
					<thead>
						<tr>
							<th>{{ capGen.actions }}</th>
							<th>{{ capGen.name }}</th>
							<th>{{ capGen.id }}</th>
							<th>{{ capGen.title }}</th>
							<th colspan="2">{{ capApp.body }}</th>
						</tr>
					</thead>
					
					<!-- new article -->
					<my-builder-articles-item
						@nextLanguage="$emit('nextLanguage')"
						:builderLanguage="builderLanguage"
						:module="module"
						:readonly="readonly"
					/>
					
					<!-- existing articles -->
					<my-builder-articles-item
						v-for="art in module.articles"
						@nextLanguage="$emit('nextLanguage')"
						:article="art"
						:builderLanguage="builderLanguage"
						:key="art.id"
						:module="module"
						:readonly="readonly"
					/>
				</table>
			</div>
			
			<div class="builder-articles-assign content default-inputs" v-if="module">
				<h2>{{ capApp.titleAssign }}</h2>
				
				<table>
					<tbody>
						<!-- article target -->
						<tr>
							<td>{{ capApp.assignTo }}</td>
							<td>
								<select v-model="assignTarget" @change="reset">
									<option value="module">{{ capApp.option.assignModule }}</option>
									<option value="form">{{ capApp.option.assignForm }}</option>
								</select>
							</td>
						</tr>
						<tr v-if="assignTarget === 'form'">
							<td>{{ capGen.form }}</td>
							<td>
								<select v-model="formIdAssignTo" @change="reset" :disabled="readonly">
									<option :value="null">-</option>
									<option v-for="f in module.forms" :value="f.id">{{ f.name }}</option>
								</select>
							</td>
						</tr>
						
						<!-- add article -->
						<tr v-if="assignTarget !== 'form' || formIdAssignTo !== null">
							<td>{{ capApp.addArticle }}</td>
							<td>
								<select v-model="articleIdAdd" @change="articleAdd($event.target.value)" :disabled="readonly">
									<option :value="null">-</option>
									<option v-for="a in module.articles.filter(v => !articleIdsAssigned.includes(v.id))" :value="a.id">{{ a.name }}</option>
									<optgroup
										v-for="mod in getDependentModules(module).filter(v => v.id !== module.id && v.articles.length !== 0)"
										:label="mod.name"
									>
										<option v-for="a in mod.articles.filter(v => !articleIdsAssigned.includes(v.id))" :value="a.id">
											{{ mod.name + ': ' + a.name }}
										</option>
									</optgroup>
								</select>
							</td>
						</tr>
					</tbody>
				</table>
				
				<div class="actions">
					<my-button image="save.png"
						@trigger="assign"
						:active="hasChanges"
						:caption="capGen.button.save"
					/>
					<my-button image="open.png"
						@trigger="showPreview = true"
						:caption="capGen.preview"
					/>
				</div>
				
				<!-- assigned articles list -->
				<h3 v-if="articleIdsAssigned.length !== 0">{{ capApp.titleAssigned }}</h3>
				<draggable handle=".dragAnchor" group="articles" itemKey="id" animation="100"
					:fallbackOnBody="true"
					:list="articleIdsAssigned"
				>
					<template #item="{element,index}">
				    	<div class="builder-article-line">
							<img v-if="!readonly" class="action dragAnchor" src="images/drag.png" />
							
							<span v-if="articleIdMap[element].moduleId === id">{{ articleIdMap[element].name }}</span>
							<span v-else>{{ moduleIdMap[articleIdMap[element].moduleId].name + ': ' + articleIdMap[element].name }}</span>
							
							<my-button image="cancel.png"
								@trigger="articleRemove(element)"
								:naked="true"
							/>
						</div>
					</template>
				</draggable>
			</div>
			
			<!-- articles preview -->
			<div class="app-sub-window under-header" v-if="showPreview" @mousedown.self="showPreview = false">
				<my-articles class="builder-articles-preview float"
					@close="showPreview = false"
					:form="formIdAssignTo !== null ? formIdMap[formIdAssignTo] : null"
					:isFloat="true"
					:languageForce="builderLanguage"
					:moduleId="module.id"
				/>
			</div>
		</div>
	</div>`,
	emits:['nextLanguage'],
	props:{
		builderLanguage:{ type:String,  required:true },
		id:             { type:String,  required:true },
		readonly:       { type:Boolean, required:true }
	},
	mounted() {
		this.$store.commit('keyDownHandlerAdd',{fnc:this.assign,key:'s',keyCtrl:true});
	},
	unmounted() {
		this.$store.commit('keyDownHandlerDel',this.assign);
	},
	data() {
		return {
			// inputs
			articleIdAdd:null,     // article to add
			assignTarget:'module', // module/form
			formIdAssignTo:null,   // form to add article to (if target is 'form')
			
			// states
			articleIdsAssigned:[],
			articleIdsAssignedOrg:[], // to compare for changes
			showPreview:false
		};
	},
	computed:{
		hasChanges:(s) => JSON.stringify(s.articleIdsAssigned) !== JSON.stringify(s.articleIdsAssignedOrg),
		
		// stores
		module:      (s) => typeof s.moduleIdMap[s.id] === 'undefined' ? false : s.moduleIdMap[s.id],
		moduleIdMap: (s) => s.$store.getters['schema/moduleIdMap'],
		formIdMap:   (s) => s.$store.getters['schema/formIdMap'],
		articleIdMap:(s) => s.$store.getters['schema/articleIdMap'],
		capApp:      (s) => s.$store.getters.captions.builder.articles,
		capGen:      (s) => s.$store.getters.captions.generic
	},
	watch:{
		module:{
			handler() { this.reset(); },
			immediate:true
		}
	},
	methods:{
		// externals
		getDependentModules,
		
		// states
		reset() {
			this.articleIdsAssigned    = [];
			this.articleIdsAssignedOrg = [];
			
			if(this.assignTarget === 'module') {
				this.articleIdsAssigned    = JSON.parse(JSON.stringify(this.module.articleIdsHelp));
				this.articleIdsAssignedOrg = JSON.parse(JSON.stringify(this.module.articleIdsHelp));
				return;
			}
			if(this.assignTarget === 'form' && this.formIdAssignTo !== null) {
				let f = this.formIdMap[this.formIdAssignTo];
				this.articleIdsAssigned    = JSON.parse(JSON.stringify(f.articleIdsHelp));
				this.articleIdsAssignedOrg = JSON.parse(JSON.stringify(f.articleIdsHelp));
				return;
			}
		},
		
		// actions
		articleAdd(id) {
			this.articleIdsAssigned.push(id);
			this.articleIdAdd = null;
		},
		articleRemove(id) {
			let pos = this.articleIdsAssigned.indexOf(id);
			if(pos !== -1)
				this.articleIdsAssigned.splice(pos,1);
		},
		
		// backend calls
		assign() {
			ws.send('article','assign',{
				articleIds:this.articleIdsAssigned,
				target:this.assignTarget,
				targetId:this.assignTarget === 'module' ? this.id : this.formIdAssignTo
			},true).then(
				() => {
					this.$root.schemaReload(this.id);
					this.articleIdsAssignedOrg = JSON.parse(JSON.stringify(this.articleIdsAssigned));
				},
				this.$root.genericError
			);
		}
	}
};
