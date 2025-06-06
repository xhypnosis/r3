import { getBuildFromVersion } from "./shared/generic.js";
import { setSingle as setSettingSingle } from "./shared/settings.js";
import MyWidgets from "./widgets.js";
export { MyHome as default };

let MyHome = {
  name: "my-go-home",
  components: { MyWidgets },
  template: `<div class="home" :class="{ showWidgets:showWidgets }">
		
		<!-- application version -->
		<a target="_blank" class="version"
			v-if="!isMobile"
			:href="capGen.appWebsite"
		>{{ capGen.appName + ' ' + appVersion }}</a>
		
		<!-- instance setup wizard -->
		<div class="home-standardBox contentBox home-wizard" v-if="noModules && isAdmin">
			<div class="top lower">
				<div class="area">
					<img class="icon" src="images/settings.png" />
					<h1>{{ capApp.wizard.title }}</h1>
				</div>
			</div>
			
			<div class="content" v-html="capApp.wizard.intro"></div>
			
			<div class="content no-padding">
				<div class="tabBar">
					<div class="entry clickable"
						v-html="capApp.wizard.installBundle"
						@click="wizardTarget = 'bundle'"
						:class="{ active:wizardTarget === 'bundle' }"
					></div>
					<div class="entry clickable"
						v-html="capApp.wizard.installRepo"
						@click="wizardTarget = 'repo'"
						:class="{ active:wizardTarget === 'repo' }"
					></div>
					<div class="entry clickable"
						v-html="capApp.wizard.installFile"
						@click="wizardTarget = 'file'"
						:class="{ active:wizardTarget === 'file' }"
					></div>
				</div>
			</div>
				
			<div class="content home-wizardAction" v-if="wizardTarget === 'bundle'">
				<img class="preview" src="images/logo_core_company.png" />
				<p v-html="capApp.wizard.installBundleDesc"></p>
				<my-button
					@trigger="installPackage"
					:active="!installStarted"
					:caption="capApp.button.installBundle"
					:image="!installStarted ? 'ok.png' : 'load.gif'"
				/>
			</div>
			<div class="content home-wizardAction" v-if="wizardTarget === 'repo'">
				<img class="preview small" src="images/box.png" />
				<p v-html="capApp.wizard.installRepoDesc" />
				<my-button image="download.png"
					@trigger="goToRepo"
					:caption="capApp.button.goToRepo"
				/>
			</div>
			<div class="content home-wizardAction" v-if="wizardTarget === 'file'">
				<img class="preview small" src="images/fileZip.png" />
				<p v-html="capApp.wizard.installFileDesc" />
				<my-button image="upload.png"
					@trigger="goToApps"
					:caption="capApp.button.goToApps"
				/>
			</div>
		</div>
		
		<!-- no access message -->
		<div class="contentBox home-standardBox home-noAccess" v-if="noAccess">
			<div class="top lower">
				<div class="area">
					<img class="icon" src="images/key.png" />
					<h1>{{ capApp.noAccessTitle }}</h1>
				</div>
			</div>
			
			<div class="content">
				<span
					v-if="!isAdmin"
					v-html="capApp.noAccess"
				/>
				
				<template v-if="isAdmin">
					<span v-html="capApp.noAccessAdmin" />
					
					<div class="actions">
						<my-button image="person.png"
							@trigger="goToLogins"
							:caption="capApp.button.goToLogins"
						/>
					</div>
				</template>
			</div>
		</div>
		
		<!-- update notification -->
		<div class="contentBox scroll home-standardBox" v-if="showUpdate && !isMobile">
			<div class="top lower">
				<div class="area">
					<img class="icon" src="images/download.png" />
					<h1>{{ capApp.newVersion }}</h1>
				</div>
				<div class="area">
					<my-button image="cancel.png"
						@trigger="setSettingSingle('hintUpdateVersion',versionBuildNew)"
						:cancel="true"
					/>
				</div>
			</div>
			
			<div class="content">
				<span v-html="capApp.newVersionText.replace('{VERSION}',config.updateCheckVersion)" />
			</div>
		</div>
		
		<!-- login widgets -->
		<my-widgets v-if="showWidgets" />
	</div>`,
  data() {
    return {
      installStarted: false,
      wizardTarget: "bundle",
    };
  },
  computed: {
    noAccess: (s) => s.noNavigation && !s.noModules,
    noModules: (s) => s.modules.length === 0,
    noNavigation: (s) => s.moduleEntries.length === 0,
    //  Hypnos: 关闭更新通知
    // showUpdate:     (s) => !s.isAdmin || s.versionBuildNew <= s.settings.hintUpdateVersion
    // 	? false : s.versionBuildNew > s.getBuildFromVersion(s.appVersion),
    showWidgets: (s) => !s.noAccess && !s.noModules,
    versionBuildNew: (s) =>
      !s.isAdmin || s.config.updateCheckVersion === ""
        ? 0
        : s.getBuildFromVersion(s.config.updateCheckVersion),
    showUpdate: () => false,

    // stores
    activated: (s) => s.$store.getters["local/activated"],
    appName: (s) => s.$store.getters["local/appName"],
    appVersion: (s) => s.$store.getters["local/appVersion"],
    colorHeader: (s) => s.$store.getters["local/companyColorHeader"],
    modules: (s) => s.$store.getters["schema/modules"],
    moduleIdMap: (s) => s.$store.getters["schema/moduleIdMap"],
    iconIdMap: (s) => s.$store.getters["schema/iconIdMap"],
    capApp: (s) => s.$store.getters.captions.home,
    capGen: (s) => s.$store.getters.captions.generic,
    config: (s) => s.$store.getters.config,
    isAdmin: (s) => s.$store.getters.isAdmin,
    isMobile: (s) => s.$store.getters.isMobile,
    moduleEntries: (s) => s.$store.getters.moduleEntries,
    pwaModuleId: (s) => s.$store.getters.pwaModuleId,
    settings: (s) => s.$store.getters.settings,
  },
  mounted() {
    this.$store.commit("pageTitle", this.capApp.title);

    // forward to PWA if enabled
    if (this.pwaModuleId !== null) {
      let mod = this.moduleIdMap[this.pwaModuleId];
      let modParent =
        mod.parentId !== null ? this.moduleIdMap[mod.parentId] : mod;

      this.$router.replace(`/app/${modParent.name}/${mod.name}`);
    }
  },
  methods: {
    // externals
    getBuildFromVersion,
    setSettingSingle,

    // actions
    installPackage() {
      ws.send("package", "install", {}, true).then(() => {},
      this.$root.genericError);
      this.installStarted = true;
    },
    showHelp(top, body) {
      this.$store.commit("dialog", {
        captionBody: body,
        captionTop: top,
        image: "question.png",
        textDisplay: "richtext",
        width: 900,
        buttons: [
          {
            caption: this.capGen.button.cancel,
            image: "cancel.png",
          },
        ],
      });
    },

    // routing
    goToApps() {
      this.$router.push("/admin/modules");
    },
    goToLogins() {
      this.$router.push("/admin/logins");
    },
    goToRepo() {
      this.$router.push("/admin/repo");
    },
  },
};
