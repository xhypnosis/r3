import MyAdminOauthClient from './adminOauthClient.js';
import {getUnixFormat}    from '../shared/time.js';
export {MyAdminOauthClients as default};

let MyAdminOauthClients = {
	name:'my-admin-oauth-clients',
	components:{ MyAdminOauthClient, },
	template:`<div class="admin-oauth-client contentBox grow">
		<div class="top">
			<div class="area">
				<img class="icon" src="images/lockCog.png" />
				<h1>{{ menuTitle }}</h1>
			</div>
		</div>
		<div class="top lower">
			<div class="area">
				<my-button image="add.png"
					@trigger="idOpen = 0"
					:active="licenseValid"
					:caption="capGen.button.new"
				/>
				<my-button image="refresh.png"
					@trigger="get"
					:caption="capGen.button.refresh"
				/>
			</div>
		</div>
		
		<div class="content grow">
			<div class="generic-entry-list wide">
				<div class="entry clickable"
					v-for="(c,k) in oauthClientIdMap"
					@click="idOpen = c.id"
					:key="c.id"
					:title="c.name"
				>
					<div class="lines">
						<span>{{ c.name }}</span>
						<span class="subtitle">{{ subtitle(c) }}</span>
					</div>
				</div>
			</div>
			
			<my-admin-oauth-client
				v-if="idOpen !== null"
				@close="idOpen = null;get()"
				@makeNew="idOpen = 0"
				:id="idOpen"
				:loginTemplates="loginTemplates"
				:oauthClientIdMap="oauthClientIdMap"
				:readonly="!licenseValid"
			/>
		</div>
	</div>`,
	props:{
		menuTitle:{ type:String, required:true }
	},
	data() {
		return {
			loginTemplates:[],
			oauthClientIdMap:[],
			idOpen:null
		};
	},
	computed:{
		// stores
		capApp:      (s) => s.$store.getters.captions.admin.oauthClient,
		capGen:      (s) => s.$store.getters.captions.generic,
		licenseValid:(s) => s.$store.getters.licenseValid,
		settings:    (s) => s.$store.getters.settings
	},
	mounted() {
		this.get();
		this.$store.commit('pageTitle',this.menuTitle);
	},
	methods:{
		// externals
		getUnixFormat,

		// presentation
		subtitle(c) {
			let parts = [`${this.capApp.option.flow[c.flow]}`];
			
			if(c.dateExpiry !== null)
				parts.push(`${this.capApp.dateExpiry}: ${getUnixFormat(c.dateExpiry,this.settings.dateFormat)}`);

			return parts.join(', ');
		},
		
		// backend calls
		get() {
			ws.sendMultiple([
				ws.prepare('oauthClient','get',{}),
				ws.prepare('loginTemplate','get',{byId:0})
			],true).then(
				res => {
					this.oauthClientIdMap = res[0].payload;
					this.loginTemplates   = res[1].payload;
				},
				this.$root.genericError
			);
		}
	}
};