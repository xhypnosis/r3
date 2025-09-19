import {getStringFilled} from '../shared/generic.js';
import srcBase64Icon     from '../shared/image.js';
import {getUnixFormat}   from '../shared/time.js';
import {getCaption}      from '../shared/language.js';
export {MyAdminScheduler as default};

let MyAdminScheduler = {
	name:'my-admin-scheduler',
	template:`<div class="admin-scheduler contentBox grow">
	
		<div class="top">
			<div class="area">
				<img class="icon" src="images/clock.png" />
				<h1>{{ menuTitle }}</h1>
			</div>
		</div>
		<div class="top lower">
			<div class="area">
				<my-button image="save.png"
					@trigger="set"
					:active="hasChanges"
					:caption="capGen.button.save"
				/>
				<my-button image="refresh.png"
					@trigger="get"
					:caption="capGen.button.refresh"
				/>
			</div>
		</div>
		
		<div class="content no-padding">

			<!-- mirror mode notice -->
			<p class="message error" v-if="mirrorMode">
				{{ capApp.mirrorMode }}
			</p>
		
			<!-- cluster master schedules -->
			<div class="content">
				<my-label image="cluster.png" :caption="capApp.systemTasks" :large="true" />
				<br />
				
				<table class="generic-table bright default-inputs shade">
					<thead>
						<tr>
							<th>{{ capGen.name }}</th>
							<th>{{ capApp.intervalSeconds }}</th>
							<th>{{ capApp.dateAttempt }}</th>
							<th>{{ capApp.dateSuccess }}</th>
							<th colspan="2">{{ capGen.active }}</th>
						</tr>
					</thead>
					<tbody>
						<template v-for="(s,i) in schedulersInput.filter(v => v.taskName !== '')">
							<tr v-if="s.clusterMasterOnly">
								<td>{{ displayName(s.taskName) }}</td>
								<td><input class="short" v-model.number="schedulersInput[i].intervalValue" /></td>
								<td>{{ displayTime(s.dateAttempt) }}</td>
								<td>{{ displayTime(s.dateSuccess) }}</td>
								<td>
									<my-bool
										v-if="!schedulersInput[i].activeOnly"
										@update:modelValue="schedulersInput[i].active = $event"
										:modelValue="!mirrorMode || !tasksDisabledMirrorMode.includes(s.taskName) ? schedulersInput[i].active : false"
										:readonly="mirrorMode && tasksDisabledMirrorMode.includes(s.taskName)"
									/>
								</td>
								<td>
									<my-button image="clock.png"
										@trigger="runSystemTask(s.taskName)"
										:active="!mirrorMode || !tasksDisabledMirrorMode.includes(s.taskName) ? schedulersInput[i].active : false"
										:caption="capApp.button.runNow"
									/>
								</td>
							</tr>
						</template>
					</tbody>
				</table>
			</div>
			
			<!-- cluster node schedules -->
			<div class="content">
				<my-label image="server.png" :caption="capApp.systemTasksNode" :large="true" />
				<br />
				
				<table class="generic-table bright default-inputs shade">
					<thead>
						<tr>
							<th colspan="2">{{ capGen.name }}</th>
							<th>{{ capApp.intervalSeconds }}</th>
							<th>{{ capApp.dateAttempt }}</th>
							<th>{{ capApp.dateSuccess }}</th>
							<th colspan="2">{{ capGen.active }}</th>
						</tr>
					</thead>
					<tbody>
						<template v-for="(s,i) in schedulersInput.filter(v => v.taskName !== '')">
							<template v-if="!s.clusterMasterOnly">
								<tr>
									<td class="minimum">
										<my-button
											@trigger="expandScheduler(i)"
											:image="schedulersExpanded.includes(i) ? 'triangleDown.png' : 'triangleRight.png'"
											:naked="true"
										/>
									</td>
									<td>{{ displayName(s.taskName) }}</td>
									<td><input class="short" v-model.number="schedulersInput[i].intervalValue" /></td>
									<td>{{ displayTime(s.dateAttempt) }}</td>
									<td>{{ displayTime(s.dateSuccess) }}</td>
									<td>
										<my-bool
											v-if="!schedulersInput[i].activeOnly"
											v-model="schedulersInput[i].active"
										/>
									</td>
									<td>
										<my-button image="clock.png"
											@trigger="runSystemTask(s.taskName)"
											:active="schedulers[i].active"
											:caption="capApp.button.runNow"
											:captionTitle="capApp.button.runNowHint"
										/>
									</td>
								</tr>
								<tr v-if="schedulersExpanded.includes(i)" v-for="meta in s.nodeMeta">
									<td></td>
									<td colspan="2">{{ meta.name }}</td>
									<td>{{ displayTime(meta.dateAttempt) }}</td>
									<td>{{ displayTime(meta.dateSuccess) }}</td>
									<td colspan="3"></td>
								</tr>
							</template>
						</template>
					</tbody>
				</table>
			</div>
			
			<!-- module schedules -->
			<div class="content" v-if="hasAppSchedules">
				<my-label image="builder.png" :caption="capApp.functions" :large="true" />
				<br />
				
				<table class="generic-table bright shade">
					<thead>
						<tr>
							<th>{{ capGen.application }}</th>
							<th>{{ capGen.name }}</th>
							<th>{{ capApp.interval }}</th>
							<th>{{ capApp.dateAttempt }}</th>
							<th>{{ capApp.dateSuccess }}</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="s in pgFunctionSchedulers">
							<td>
								<div class="row gap centered">
									<img class="module-icon" :src="srcBase64Icon(moduleIdMap[pgFunctionIdMap[s.pgFunctionId].moduleId].iconId,'images/module.png')" />
									<span>{{ displayModuleName(s.pgFunctionId) }}</span>
								</div>
							</td>
							<td>{{ displayFunctionName(s.pgFunctionId) }}</td>
							<td>{{ displaySchedule(s.pgFunctionId,s.pgFunctionScheduleId) }}</td>
							<td>{{ displayTime(s.dateAttempt) }}</td>
							<td>{{ displayTime(s.dateSuccess) }}</td>
							<td>
								<my-button image="clock.png"
									@trigger="runPgFunction(s.pgFunctionId,s.pgFunctionScheduleId)"
									:caption="capApp.button.runNow"
								/>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</div>`,
	props:{
		menuTitle:{ type:String, required:true }
	},
	data() {
		return {
			schedulers:[],
			schedulersInput:[],    // changes to schedulers
			schedulersExpanded:[], // indexes of schedules that show all nodes
			tasksDisabledMirrorMode:['adminMails','backupRun','mailAttach','mailRetrieve','mailSend','restExecute']
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
		hasAppSchedules:(s) => {
			for(let e of s.schedulers) {
				if(e.taskName === '')
					return true;
			}
			return false;
		},

		// simple
		hasChanges:          (s) => JSON.stringify(s.schedulers) !== JSON.stringify(s.schedulersInput),
		pgFunctionSchedulers:(s) => s.schedulers.filter(v => v.taskName === '' && v.intervalType !== 'once'),
		
		// stores
		moduleIdMap:    (s) => s.$store.getters['schema/moduleIdMap'],
		pgFunctionIdMap:(s) => s.$store.getters['schema/pgFunctionIdMap'],
		capApp:         (s) => s.$store.getters.captions.admin.scheduler,
		capGen:         (s) => s.$store.getters.captions.generic,
		mirrorMode:     (s) => s.$store.getters.mirrorMode,
		settings:       (s) => s.$store.getters.settings
	},
	methods:{
		// externals
		getCaption,
		getStringFilled,
		getUnixFormat,
		srcBase64Icon,
		
		// presentation
		displayTime(unixTime) {
			return unixTime === 0 ? '-' : this.getUnixFormat(unixTime,`${this.settings.dateFormat} H:i:S`);
		},
		displayName(name) {
			return this.capApp.names[name] === undefined ? name : this.capApp.names[name];
		},
		displayModuleName(pgFunctionId) {
			const m = this.moduleIdMap[this.pgFunctionIdMap[pgFunctionId].moduleId];
			return this.getCaption('moduleTitle',m.id,m.id,m.captions,m.name);
		},
		displayFunctionName(pgFunctionId) {
			const f = this.pgFunctionIdMap[pgFunctionId];
			return this.getCaption('pgFunctionTitle',f.moduleId,f.id,f.captions,f.name);
		},
		displaySchedule(pgFunctionId,pgFunctionScheduleId) {
			let f = this.pgFunctionIdMap[pgFunctionId];
			let s = null;
			
			for(let i = 0, j = f.schedules.length; i < j; i++) {
				if(f.schedules[i].id === pgFunctionScheduleId) {
					s = f.schedules[i];
					break;
				}
			}
			
			if(s === null)
				return '';
			
			let parts    = [];
			let typeName = '';
			
			switch(s.intervalType) {
				case 'days':    typeName = this.capApp.intervalTypeDays;    break;
				case 'hours':   typeName = this.capApp.intervalTypeHours;   break;
				case 'minutes': typeName = this.capApp.intervalTypeMinutes; break;
				case 'months':  typeName = this.capApp.intervalTypeMonths;  break;
				case 'seconds': typeName = this.capApp.intervalTypeSeconds; break;
				case 'weeks':   typeName = this.capApp.intervalTypeWeeks;   break;
				case 'years':   typeName = this.capApp.intervalTypeYears;   break;
			}
			
			parts.push(this.capApp.scheduleLine
				.replace('{TYPE}',typeName)
				.replace('{VALUE}',s.intervalValue)
			);
			
			switch(s.intervalType) {
				case 'months': parts.push(this.capApp.scheduleLineDayMonths.replace('{DAY}',s.atDay)); break;
				case 'weeks':  parts.push(this.capApp.scheduleLineDayWeeks.replace('{DAY}',s.atDay));  break;
				case 'years':  parts.push(this.capApp.scheduleLineDayYears.replace('{DAY}',s.atDay));  break;
			}
			
			if(['days','weeks','months','years'].includes(s.intervalType))
				parts.push(this.capApp.scheduleLineTime
					.replace('{HH}',this.getStringFilled(s.atHour,2,'0'))
					.replace('{MM}',this.getStringFilled(s.atMinute,2,'0'))
					.replace('{SS}',this.getStringFilled(s.atSecond,2,'0'))
				);
			
			return parts.join(', ');
		},
		expandScheduler(i) {
			let pos = this.schedulersExpanded.indexOf(i);
			
			if(pos === -1) this.schedulersExpanded.push(i);
			else           this.schedulersExpanded.splice(pos,1);
		},
		
		// backend calls
		get() {
			ws.send('scheduler','get',{},true).then(
				res => {
					this.schedulers      = res.payload;
					this.schedulersInput = JSON.parse(JSON.stringify(this.schedulers));
				},
				this.$root.genericError
			);
		},
		runPgFunction(pgFunctionId,pgFunctionScheduleId) {
			ws.send('task','run',{
				clusterMasterOnly:true,
				pgFunctionId:pgFunctionId,
				pgFunctionScheduleId:pgFunctionScheduleId
			},true).then(
				() => {},
				this.$root.genericError
			);
		},
		runSystemTask(name,clusterMasterOnly) {
			ws.send('task','run',{
				clusterMasterOnly:clusterMasterOnly,
				taskName:name
			},true).then(
				() => {},
				this.$root.genericError
			);
		},
		set() {
			if(!this.hasChanges)
				return;
			
			let requests = [];
			for(let i = 0, j = this.schedulersInput.length; i < j; i++) {
				let s = this.schedulersInput[i];
				
				if(s.taskName === '' || JSON.stringify(s) === JSON.stringify(this.schedulers[i]))
					continue;
				
				requests.push(ws.prepare('task','set',{
					active:s.active,
					interval:s.intervalValue,
					name:s.taskName
				}));
			}
			
			ws.sendMultiple(requests,true).then(
				() => {
					this.get();
					
					ws.send('task','informChanged',{},false).then(
						() => {},
						this.$root.genericError
					);
				},
				this.$root.genericError
			);
		}
	}
};