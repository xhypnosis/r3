import {getBuildFromVersion} from '../shared/generic.js';
export {MyAdminConfig as default};

let MyAdminConfig = {
	name:'my-admin-config',
	template:`<div class="contentBox admin-config" v-if="ready">
		<div class="top">
			<div class="area">
				<img class="icon" src="images/server.png" />
				<h1>{{ capApp.title }}</h1>
			</div>
		</div>
		<div class="top lower">
			<div class="area">
				<my-button image="save.png"
					@trigger="set"
					:active="hasChanges"
					:caption="capApp.button.apply"
				/>
				<my-button image="refresh.png"
					@trigger="configInput = JSON.parse(JSON.stringify(config))"
					:active="hasChanges"
					:caption="capGen.button.refresh"
				/>
			</div>
		</div>
		
		<div class="content no-padding">
			
			<!-- general -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/settings.png" />
					<h1>{{ capApp.titleGeneral }}</h1>
				</div>
				
				<table class="default-inputs">
					<tbody>
						<tr>
							<td>{{ capApp.appVersion }}</td>
							<td>{{ appVersion }}</td>
						</tr>
						<tr>
							<td>{{ capApp.updateCheck }}</td>
							<td v-if="updateCheckText !== capApp.updateCheckOlder">
								{{ updateCheckText }}
							</td>
							<td v-else>
								<a href="https://rei3.de/download" target="_blank">{{ updateCheckText }}</a>
							</td>
						</tr>
						<tr>
							<td>{{ capApp.licenseState }}</td>
							<td v-if="licenseValid">{{ capApp.licenseStateOk.replace('{COUNT}',this.licenseDays) }}</td>
							<td v-if="!licenseValid">{{ capApp.licenseStateNok }}</td>
						</tr>
						<tr><td colspan="2"><hr /></td></tr>
						<tr>
							<td>{{ capApp.publicHostName }}</td>
							<td>
								<div class="row gap centered">
									<input v-model="configInput.publicHostName" />
									<my-button image="question.png"
										@trigger="showHelp(capApp.publicHostNameDesc)"
									/>
								</div>
							</td>
						</tr>
						<tr>
							<td>{{ capApp.proxyUrl }}</td>
							<td>
								<div class="row gap centered">
									<input v-model="configInput.proxyUrl" />
									<my-button image="question.png"
										@trigger="showHelp(capApp.proxyUrlDesc)"
									/>
								</div>
							</td>
						</tr>
						<tr><td colspan="2"><hr /></td></tr>
						<tr><td colspan="2"><b>{{ capGen.systemModes }}</b></td></tr>
						<tr>
							<td><my-label image="serverCog.png" :caption="capApp.productionMode" /></td>
							<td>
								<my-bool-string-number
									v-model="configInput.productionMode"
									@update:modelValue="informProductionMode"
									:reversed="true"
									:readonly="configInput.builderMode === '1'"
								/>
							</td>
						</tr>
						<tr>
							<td><my-label image="builder.png" :caption="capApp.builderMode" /></td>
							<td>
								<my-bool-string-number
									v-model="configInput.builderMode"
									@update:modelValue="informBuilderMode"
									:readonly="configInput.productionMode === '1'"
								/>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			
			<!-- security -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/admin.png" />
					<h1>{{ capApp.titleLogins }}</h1>
				</div>
				
				<table class="default-inputs">
					<tbody>
						<tr>
							<td>{{ capApp.tokenExpiryHours }}</td>
							<td><input v-model="configInput.tokenExpiryHours" /></td>
						</tr>
						<tr><td colspan="2"><hr /></td></tr>
						<tr><td colspan="2"><br /><b>{{ capApp.pwTitle }}</b></td></tr>
						<tr>
							<td>{{ capApp.pwLengthMin }}</td>
							<td><input v-model="configInput.pwLengthMin" /></td>
						</tr>
						<tr>
							<td>{{ capApp.pwForceDigit }}</td>
							<td><my-bool-string-number v-model="configInput.pwForceDigit" /></td>
						</tr>
						<tr>
							<td>{{ capApp.pwForceLower }}</td>
							<td><my-bool-string-number v-model="configInput.pwForceLower" /></td>
						</tr>
						<tr>
							<td>{{ capApp.pwForceUpper }}</td>
							<td><my-bool-string-number v-model="configInput.pwForceUpper" /></td>
						</tr>
						<tr>
							<td>{{ capApp.pwForceSpecial }}</td>
							<td><my-bool-string-number v-model="configInput.pwForceSpecial" /></td>
						</tr>
					</tbody>
				</table>
			</div>
			
			<!-- login backgrounds -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/admin.png" />
					<h1>{{ capApp.titleLoginBackgrounds }}</h1>
				</div>
				
				<div class="login-bg">
					<div class="preview clickable"
						v-for="n in loginBackgroundCount"
						@click="loginBgToggle(n-1)"
						:class="{ inactive:!loginBackgrounds.includes(n-1) }"
						:style="loginBgStyle(n-1)"
					></div>
				</div>
			</div>
			
			<!-- repository -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/box.png" />
					<h1>{{ capApp.titleRepo }}</h1>
				</div>
				
				<table class="default-inputs">
					<tbody>
						<tr>
							<td>{{ capApp.repoUrl }}</td>
							<td><input v-model="configInput.repoUrl" /></td>
						</tr>
						<tr>
							<td>{{ capGen.username }}</td>
							<td><input v-model="configInput.repoUser" /></td>
						</tr>
						<tr>
							<td>{{ capGen.password }}</td>
							<td><input type="password" v-model="configInput.repoPass" /></td>
						</tr>
						<tr>
							<td>{{ capApp.repoSkipVerify }}</td>
							<td>
								<my-bool-string-number
									v-model="configInput.repoSkipVerify"
								/>
							</td>
						</tr>
						<tr>
							<td>{{ capApp.repoFeedback }}</td>
							<td>
								<my-bool-string-number
									v-model="configInput.repoFeedback"
								/>
							</td>
						</tr>
						<tr><td colspan="2"><hr /></td></tr>
						<tr><td colspan="2"><b>{{ capApp.repoKeyManagement }}</b></td></tr>
						<tr>
							<td>{{ capApp.repoPublicKeys }}</td>
							<td>
								<div class="repo-key" v-for="(key,name) in publicKeys">
									<my-button
										:active="false"
										:caption="name"
										:naked="true"
									/>
									<div class="row gap">
										<my-button image="search.png"
											@trigger="publicKeyShow(name,key)"
										/>
										<my-button image="cancel.png"
											@trigger="publicKeyRemove(name)"
											:cancel="true"
										/>
									</div>
								</div>
							</td>
						</tr>
						<tr>
							<td>{{ capApp.repoPublicKeyAdd }}</td>
							<td>
								<div class="column gap">
									<input v-model="publicKeyInputName"
										:placeholder="capApp.repoPublicKeyInputNameHint"
									/>
									<textarea v-model="publicKeyInputValue"
										:placeholder="capApp.repoPublicKeyInputValueHint"
									></textarea>
									<div>
										<my-button image="add.png"
											@trigger="publicKeyAdd"
											:active="publicKeyInputName !== '' && publicKeyInputValue !== ''"
											:caption="capGen.button.add"
										/>
									</div>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			
			<!-- performance -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/speedmeter.png" />
					<h1>{{ capApp.titlePerformance }}</h1>
				</div>
				
				<table class="default-inputs">
					<tbody>
						<tr>
							<td>{{ capApp.dbTimeoutDataWs }}</td>
							<td><input class="short"
								v-model="configInput.dbTimeoutDataWs"
								:placeholder="capApp.dbTimeoutHint"
							/></td>
						</tr>
						<tr>
							<td>{{ capApp.dbTimeoutDataRest }}</td>
							<td><input class="short"
								v-model="configInput.dbTimeoutDataRest"
								:placeholder="capApp.dbTimeoutHint"
							/></td>
						</tr>
						<tr>
							<td>{{ capApp.dbTimeoutCsv }}</td>
							<td><input class="short"
								v-model="configInput.dbTimeoutCsv"
								:placeholder="capApp.dbTimeoutHint"
							/></td>
						</tr>
						<tr>
							<td>{{ capApp.dbTimeoutIcs }}</td>
							<td><input class="short"
								v-model="configInput.dbTimeoutIcs"
								:placeholder="capApp.dbTimeoutHint"
							/></td>
						</tr>
					</tbody>
				</table>
			</div>
			
			<!-- ICS -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/calendar.png" />
					<h1>{{ capApp.titleIcs }}</h1>
				</div>
				
				<table class="default-inputs">
					<tbody>
						<tr>
							<td>{{ capApp.icsDownload }}</td>
							<td>
								<my-bool-string-number
									v-model="configInput.icsDownload"
								/>
							</td>
						</tr>
						<tr>
							<td>{{ capApp.icsDaysPre }}</td>
							<td><input v-model="configInput.icsDaysPre" /></td>
						</tr>
						<tr>
							<td>{{ capApp.icsDaysPost }}</td>
							<td><input v-model="configInput.icsDaysPost" /></td>
						</tr>
					</tbody>
				</table>
			</div>
			
			<!-- security settings -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/lock.png" />
					<h1>{{ capApp.bruteforceTitle }}</h1>
				</div>
				
				<table class="default-inputs">
					<tbody>
						<tr>
							<td>{{ capApp.bruteforceProtection }}</td>
							<td>
								<my-bool-string-number
									v-model="configInput.bruteforceProtection"
								/>
							</td>
						</tr>
						<tr>
							<td>{{ capApp.bruteforceAttempts }}</td>
							<td><input v-model="configInput.bruteforceAttempts" /></td>
						</tr>
						<tr>
							<td>{{ capApp.bruteforceCountTracked }}</td>
							<td>{{ bruteforceCountTracked }}</td>
						</tr>
						<tr>
							<td>{{ capApp.bruteforceCountBlocked }}</td>
							<td>{{ bruteforceCountBlocked }}</td>
						</tr>
					</tbody>
				</table>

				<span v-html="capApp.bruteforceDesc"></span>
			</div>
			
			<!-- admin mails -->
			<div class="contentPart">
				<div class="contentPartHeader">
					<img class="icon" src="images/lock.png" />
					<h1>{{ capApp.adminMailsTitle }}</h1>
				</div>
				
				<p>{{ capApp.adminMailsDesc }}</p>
				<ul>
					<li v-for="l in capApp.adminMailsList">{{ l }}</li>
				</ul>
				
				<div class="column">
					<my-button image="cancel.png"
						v-for="(c,i) in adminMailContacts"
						@trigger="adminMailDel(i)"
						:caption="c"
						:naked="true"
					/>
				</div>
				<br />
				
				<div class="row gap centered default-inputs">
					<input
						v-model="adminMailInput"
						@keyup.enter="adminMailAdd"
						:placeholder="capApp.adminMailsHint"
					/>
					<my-button image="add.png"
						@trigger="adminMailAdd"
						:active="adminMailInput !== ''"
					/>
				</div>
			</div>
		</div>
	</div>`,
	props:{
		menuTitle:{ type:String, required:true }
	},
	watch:{
		config:{
			handler(v) {
				this.configInput = JSON.parse(JSON.stringify(v));
			},
			immediate:true
		}
	},
	data() {
		return {
			adminMailInput:'',
			configInput:{},
			bruteforceCountBlocked:0,
			bruteforceCountTracked:0,
			loginBackgroundCount:12,
			publicKeyInputName:'',
			publicKeyInputValue:'',
			ready:false
		};
	},
	mounted() {
		this.get();
		this.$store.commit('pageTitle',this.menuTitle);
		this.$store.commit('keyDownHandlerAdd',{fnc:this.set,key:'s',keyCtrl:true});
	},
	unmounted() {
		this.$store.commit('keyDownHandlerDel',this.set);
	},
	computed:{
		publicKeys:{
			get()  { return JSON.parse(this.configInput.repoPublicKeys); },
			set(v) { this.configInput.repoPublicKeys = JSON.stringify(v); }
		},
		updateCheckText:(s) => {
			if(s.config.updateCheckVersion === '')
				return s.capApp.updateCheckUnknown;
			
			let buildNew = s.getBuildFromVersion(s.config.updateCheckVersion);
			let buildOld = s.getBuildFromVersion(s.appVersion);
			
			if(buildNew === buildOld) return s.capApp.updateCheckCurrent;
			if(buildNew > buildOld)   return s.capApp.updateCheckOlder;
			
			return s.capApp.updateCheckNewer;
		},
		
		// simple
		adminMailContacts:(s) => s.configInput.adminMails === '' ? [] : JSON.parse(s.configInput.adminMails),
		hasChanges:       (s) => JSON.stringify(s.config) !== JSON.stringify(s.configInput),
		loginBackgrounds: (s) => JSON.parse(s.configInput.loginBackgrounds),
		
		// stores
		appVersion:  (s) => s.$store.getters['local/appVersion'],
		token:       (s) => s.$store.getters['local/token'],
		config:      (s) => s.$store.getters.config,
		license:     (s) => s.$store.getters.license,
		licenseDays: (s) => s.$store.getters.licenseDays,
		licenseValid:(s) => s.$store.getters.licenseValid,
		capApp:      (s) => s.$store.getters.captions.admin.config,
		capGen:      (s) => s.$store.getters.captions.generic
	},
	methods:{
		// externals
		getBuildFromVersion,
		
		// presentation
		loginBgStyle(n) {
			return `background-image:url('../images/backgrounds/${n}_prev.webp')`;
		},
		
		// actions
		adminMailAdd() {
			if(this.adminMailInput === '') return;
			
			let v = JSON.parse(JSON.stringify(this.adminMailContacts));
			v.push(this.adminMailInput);
			this.configInput.adminMails = JSON.stringify(v);
			this.adminMailInput = '';
		},
		adminMailDel(index) {
			let v = JSON.parse(JSON.stringify(this.adminMailContacts));
			v.splice(index,1);
			this.configInput.adminMails = JSON.stringify(v);
		},
		informBuilderMode() {
			if(this.configInput.builderMode === '0')
				return;
			
			this.$store.commit('dialog',{
				captionBody:this.capApp.dialog.builderMode,
				captionTop:this.capApp.dialog.pleaseRead
			});
		},
		informProductionMode() {
			if(this.configInput.productionMode !== '0')
				return;
			
			this.$store.commit('dialog',{
				captionBody:this.capApp.dialog.productionMode,
				captionTop:this.capApp.dialog.pleaseRead
			});
		},
		loginBgToggle(n) {
			var list = JSON.parse(this.configInput.loginBackgrounds);
			
			const pos = list.indexOf(n);
			if(pos !== -1) list.splice(pos,1);
			else           list.push(n);
			
			this.configInput.loginBackgrounds = JSON.stringify(list);
		},
		publicKeyShow(name,key) {
			this.$store.commit('dialog',{
				captionBody:key,
				captionTop:name,
				image:'key.png',
				textDisplay:'textarea'
			});
		},
		publicKeyAdd() {
			this.publicKeys[this.publicKeyInputName] = this.publicKeyInputValue;
			this.publicKeys = this.publicKeys;
		},
		publicKeyRemove(keyName) {
			if(typeof this.publicKeys[keyName] !== 'undefined')
				delete this.publicKeys[keyName];
			
			this.publicKeys = this.publicKeys;
		},
		showHelp(msg) {
			this.$store.commit('dialog',{ captionBody:msg });
		},
		
		// backend calls
		get() {
			ws.send('bruteforce','get',{},true).then(
				res => {
					this.bruteforceCountBlocked = res.payload.hostsBlocked;
					this.bruteforceCountTracked = res.payload.hostsTracked;
					this.ready = true;
				},
				this.$root.genericError
			);
		},
		set() {
			if(!this.hasChanges) return;
			
			ws.send('config','set',this.configInput,true).then(
				() => {}, this.$root.genericError
			);
		}
	}
};