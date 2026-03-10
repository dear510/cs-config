<template>
  <div class="app-container">
    <aside class="sidebar">
      <div class="brand">
        <span class="logo">🎮</span>
        <div class="brand-text">
          <h2>次神助手</h2>
          <small>v2.5.1 </small>
        </div>
      </div>

      <nav class="menu-list">
        <div 
          :class="['menu-item', activeTab === 'daily' ? 'active' : '']" 
          @click="activeTab = 'daily'"
        >
          <span class="icon">📅</span> 
          <div class="menu-info">
            <span class="menu-title">日常任务</span>
            <span class="menu-desc">自动化流水线</span>
          </div>
        </div>
        <div 
          :class="['menu-item', activeTab === 'zhenbao' ? 'active' : '', userLevel < 1 ? 'locked-item' : '']" 
          @click="userLevel >= 1 ? activeTab = 'zhenbao' : alert('❌ 该功能仅限高级用户使用！')"
        >
          <span class="icon">{{ userLevel < 1 ? '🔒' : '💎' }}</span>
          <div class="menu-info">
            <span class="menu-title">珍宝拦截</span>
            <span class="menu-desc">{{ userLevel < 1 ? '待开发...' : '极速自动秒杀' }}</span>
          </div>
        </div>
      </nav>

      <div v-if="activeTab === 'zhenbao'" class="config-wrapper animate-fade">
        <div class="config-section">
          <div class="section-title">拦截规则配置</div>
          <div class="input-card">

            <div class="form-group">
              <label>目标品质 (可选)</label>
              <select v-model="newRule.quality" class="dark-input">
                <option value="">-- 不限品质 --</option>
                <option v-for="q in qualityOptions" :key="q" :value="q">{{ q }}</option>
              </select>
            </div>
            <div class="form-group search-select-container">
              <label>目标属性 (不填为不限)</label>
              <input v-model="searchKeyword" class="dark-input" placeholder="输入关键词" @focus="showDropdown = true" @blur="setTimeout(() => showDropdown = false, 200)">
              <div v-if="showDropdown && filteredAttributes.length > 0" class="search-dropdown">
                <div v-for="name in filteredAttributes" :key="name" @click="selectAttribute(name)" class="dropdown-item">{{ name }}</div>
              </div>
            </div>
            <div class="form-group">
              <label>感兴趣部位</label>
              <div class="category-grid">
                <label v-for="cat in categoryOptions" :key="cat" class="check-item">
                  <input type="checkbox" :value="cat" v-model="newRule.targetCategories">
                  <span>{{ cat }}</span>
                </label>
              </div>
            </div>
            <div class="row">
              <div class="form-group">
                <label>最小数值%</label>
                <input v-model.number="newRule.min" type="number" step="0.1" class="dark-input">
              </div>
              <div class="form-group">
                <label>最高预算(游戏币)</label>
                <input v-model.number="newRule.price" type="number" class="dark-input">
              </div>
            </div>

            <div class="auto-refresh-wrapper">
              <label class="refresh-checkbox">
                <input type="checkbox" v-model="zhenbaoConfig.autoRefresh" @change="syncAutoRefresh">
                <div class="checkbox-box"></div>
                <span class="refresh-text">🕛 12:00 自动刷新交易行</span>
              </label>
            </div>

            <button @click="addRule" class="primary-btn" :disabled="!newRule.keyword && !newRule.price && newRule.targetCategories.length === 0">添加拦截规则</button>
          </div>
        </div>

        <div class="rule-list-container">
          <div class="section-title">生效中规则 ({{ rules.length }})</div>
          <div v-for="(r, i) in rules" :key="i" class="rule-item">
            <div class="rule-info">
              <div class="rule-main">🔍 {{ r.keyword }} ≥ {{ r.min }}%</div>
              <div class="rule-sub">{{ r.quality || '不限品质' }} | {{ r.price || '无限金额' }}</div>
            </div>
            <button @click="removeRule(i)" class="delete-btn">×</button>
          </div>
        </div>
      </div>

      <div class="sidebar-footer">
        <div class="time-display">
          <span class="label">北京时间:</span>
          <span class="time-value">{{ beijingTime }}</span>
        </div>
        <div class="mini-badge">
          ⚠️ <span class="badge-text">禁止商用，仅供学习交流</span>
        </div>
      </div>
    </aside>

    <main class="main-content">
      
      <section v-if="activeTab === 'daily'" class="page-view animate-fade">
        <header class="status-bar">
          <div class="status-control" @click="toggleMonitor" :class="{ 'is-paused': !isCapturing }">
            <span class="dot" :class="{ pulsing: isCapturing }"></span>
            <span class="status-text">{{ isCapturing ? '正在监控系统' : '监控已停止' }}</span>
            <small class="port-hint">PORT: 2025</small>
          </div>
          <div class="header-actions">
            <button class="clear-btn" @click="dailyLogs = []">🗑️ 清空日志</button>
          </div>
        </header>

        <div class="daily-layout">
          <div class="tasks-columns-container">
            <div v-for="category in taskCategories" :key="category.id" class="task-column">
              
              <div class="column-header" style="display: flex; justify-content: space-between; align-items: center; width: 100%; background: transparent !important; padding: 0 8px; box-sizing: border-box; border-bottom: 1px solid rgba(255,255,255,0.05); height: 40px;">
                <div style="display: flex; align-items: center; gap: 6px;">
                  <span class="column-dot"></span>
                  <span style="font-weight: bold; font-size: 13px; color: #007aff;">{{ category.title }}</span>
                </div>

                <div v-if="category.id === 'garden'" class="garden-afk-wrapper" @click.stop>
                  <label v-if="currentConfigTask && currentConfigTask.config" class="custom-check mini-afk-label">
                    <input 
                      type="checkbox" 
                      v-model="currentConfigTask.config.gardenAFK" 
                      @change="handleGardenAFKChange"
                    >
                    <span class="afk-text">自动挂机</span>
                  </label>
                </div>
              </div>
              
              <div class="column-body">
                <div 
                  v-for="task in category.tasks" 
                  :key="task.id" 
                  :class="['task-card-mini', task.selected ? 'is-selected' : '']"
                  @click="openConfig(task)"
                >
                  <div class="task-main">
                    <div class="task-check">{{ task.selected ? '●' : '○' }}</div>
                    <div class="task-info">
                      <div class="task-name">{{ task.name }}</div>
                      <div class="task-desc">{{ task.desc }}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="daily-console">
            <div class="console-header">执行进度日志</div>
            <div class="console-body" ref="dailyLogContainer">
              <div v-if="dailyLogs.length === 0" class="console-empty">等待任务启动...</div>
              <div v-for="(log, i) in dailyLogs" :key="i" :class="['log-entry', log.type]">
                <span class="log-timestamp">[{{ log.time }}]</span>
                <span class="log-message">{{ log.message }}</span>
              </div>
            </div>
          </div>

          <div class="action-footer">
            <button class="run-all-btn" @click="handleRunTasks" :disabled="isTaskRunning">
              <span v-if="!isTaskRunning">▶ 立即开始执行选中任务</span>
              <span v-else>⏳ 正在执行选中任务 (请勿关闭)...</span>
            </button>
          </div>
        </div>
      </section>

      <section v-if="activeTab === 'zhenbao'" class="page-view animate-fade">
        <header class="status-bar">
          <div class="status-control" @click="toggleMonitor" :class="{ 'is-paused': !isCapturing }">
            <span class="dot" :class="{ pulsing: isCapturing }"></span>
            <span class="status-text">{{ isCapturing ? '正在监控系统' : '监控已停止' }}</span>
            <small class="port-hint">PORT: 2025</small>
          </div>
          <div class="header-actions">
            <span class="stats-label">数据流</span>
            <span class="stats-count">{{ zhenbaoLogs.length }}</span>
            <button class="clear-btn" @click="zhenbaoLogs = []; hitLogs = []; sysLogs = [];">🗑️ 清空记录</button>
          </div>
        </header>

        <section class="log-viewport console-container">
          <div v-if="activeRuleSummary || hitLogs.length > 0" class="sticky-top-info">
            <div v-if="activeRuleSummary" class="console-card rule-summary-card"><pre>{{ activeRuleSummary }}</pre></div>
            <div v-for="(hit, i) in hitLogs" :key="'hit-'+i" class="console-card hit-card"><div class="hit-msg">{{ hit }}</div></div>
          </div>

          <div v-for="(item, i) in zhenbaoLogs" :key="'item-'+i" class="console-card item-with-action">
            <div class="card-info">
              <div class="user-meta">
                <span class="uid-tag">卖家UID: {{ item.uid || '未知' }}</span>
                <span class="uid-tag">区服: {{ item.district || '未知区服' }}</span>
                <span class="uid-tag">部位: {{ item.itemType || '未知部位' }}</span>
              </div>
              <div class="console-row"><span class="console-label">价格：</span><span class="console-value gold-text">{{ item.price }}</span></div>
              <div v-for="(attr, idx) in item.attributes" :key="idx" class="console-row" :style="{ color: getSoftColor(attr.color) }">
                <span class="console-label">词条{{ idx + 1 }}：</span><span class="console-value">{{ attr.name }} {{ attr.value.toFixed(2) }}%</span>
              </div>
            </div>
            <div class="card-action">
              <button 
                class="manual-buy-btn" 
                :class="{ 
                  'is-purchased': item.purchased, 
                  'is-exhibiting': item.isLocked && !item.purchased,
                  'is-cooling': isGlobalCooling && !item.isLocked && !item.purchased 
                }" 
                :disabled="item.purchased || item.isLocked || isGlobalCooling" 
                @click="handleManualBuy(item)"
              >
                <span v-if="item.purchased">已发送</span>
                <span v-else-if="item.isLocked">公示中</span>
                <span v-else-if="isGlobalCooling">CD中</span>
                <span v-else>秒杀</span>
              </button>
            </div>
            <div class="console-divider"></div>
          </div>
          
          <div v-if="zhenbaoLogs.length === 0 && sysLogs.length === 0" class="empty-state-container">
            <div class="scanner-wrapper">
              <div class="scanner-ring"></div>
              <div class="scanner-dot"></div>
            </div>
            <div class="scanning-text">
              正在监听网卡数据流<span class="dot-ani">...</span>
            </div>
            <div class="scanning-sub">WAITING FOR PACKETS ON PORT 2025</div>
          </div>
        </section>
      </section>

      <div v-if="showConfigModal" class="modal-overlay" @click.self="showConfigModal = false">
        <div class="modal-card animate-fade">
          <div class="modal-header">
            <div class="modal-title-group">
              <span class="modal-main-title">{{ currentConfigTask?.name }}</span>
              <span class="modal-sub-title">任务配置详情</span>
            </div>
            <button class="close-x" @click="showConfigModal = false">×</button>
          </div>
          
          <div class="modal-body">
            <div v-if="currentConfigTask?.id === 'dungeon_all'" class="config-section-wrapper">
              <div class="config-group-title">资源提取</div>
              <div class="config-content-indent">
                <div class="config-row-inline">
                <label class="custom-check"><input type="checkbox" v-model="currentConfigTask.config.res.getKeys"> <span>领钥匙</span></label>
                <label class="custom-check"><input type="checkbox" v-model="currentConfigTask.config.res.getPicks"> <span>领稿子</span></label>
              </div>
              </div>

              <div class="config-group-title" style="margin-top:15px;">副本通关设置</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div v-for="(val, key) in currentConfigTask.config.clears" :key="key" class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="val.active">
                      <span>{{ {boss:'领主', gold:'金币', goblin:'哥布林', lamp:'神灯', west:'西游', gear:'龙宫', demon:'熔炉'}[key] }}</span>
                    </label>
                    <div class="input-group" v-if="val.active">
                      <template v-if="key !== 'lamp' && key !== 'west'">
                        关卡: <input type="number" v-model.number="val.level" class="mini-num" min="1">
                      </template>
                      次数: <input type="number" v-model.number="val.count" class="mini-num">
                    </div>
                  </div>
                </div>
            </div>
            </div>

            <div v-if="currentConfigTask?.id === 'guild_sign'" class="config-section-wrapper">
              <div class="config-group-title">操作选项</div>
              
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.joinGuild"> 
                      <span>加入工会</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.joinGuild">
                      工会号: <input type="number" v-model.number="currentConfigTask.config.guildNum" class="mini-num" min="1">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.guildSign"> 
                      <span>工会每日签到</span>
                    </label>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.bargain"> 
                      <span>自动砍价</span>
                    </label>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.boss"> 
                      <span>Boss</span>
                    </label>
                    
                    <div class="input-group" v-if="currentConfigTask.config.boss">
                      关卡: <input type="number" v-model.number="currentConfigTask.config.bossLevel" class="mini-num" min="1">
                      次数: <input type="number" v-model.number="currentConfigTask.config.bossCount" class="mini-num" min="1">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.medicine"> 
                      <span>购买药水</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.medicine">
                      瓶数: <input type="number" v-model.number="currentConfigTask.config.medCount" class="mini-num" min="1">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.registerGuildPK"> 
                      <span>报名会战</span>
                    </label>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.quitGuild"> 
                      <span>退出当前工会</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'red_diamond'" class="config-section-wrapper">
              <div class="config-group-title">红宝石提取</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.shopRuby"> 
                      <span>商城领红宝石</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.limitRuby"> 
                      <span>限购每日免费红宝石</span>
                    </label>
                  </div>
                </div>
              </div>

              <div class="config-group-title" style="margin-top:15px;">召唤选项</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.freeSummon"> 
                      <span>每日免费召唤</span>
                    </label>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.equipDraw"> 
                      <span>装备召唤 (350抽)</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.equipDraw">
                      次数: <input type="number" v-model.number="currentConfigTask.config.equipDrawCount" class="mini-num" min="1">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.skillDraw"> 
                      <span>技能召唤 (350抽)</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.skillDraw">
                      次数: <input type="number" v-model.number="currentConfigTask.config.skillDrawCount" class="mini-num" min="1">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.petDraw"> 
                      <span>宠物召唤 (350抽)</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.petDraw">
                      次数: <input type="number" v-model.number="currentConfigTask.config.petDrawCount" class="mini-num" min="1">
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'reward_tasks'" class="config-section-wrapper">
              <div class="config-group-title">日常任务奖励</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.dailyQuest"> 
                      <span>日常任务全领</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.card"> 
                      <span>每日领卡</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.mail"> 
                      <span>领取邮件</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.treasure"> 
                      <span>寻宝任务</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.skyCity"> 
                      <span>天空之城任务</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.backtrack"> 
                      <span>回溯任务</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.treasureMap"> 
                      <span>珍宝藏宝图</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.mineAccelerate"> 
                      <span>挖矿研究加速(4次) & 免费蓝宝石(10个)</span>
                    </label>
                  </div>
                </div>
              </div>

              <div class="config-group-title" style="margin-top:15px;">周期/限时活动奖励</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.promoCode"> 
                      <span>兑换码</span>
                    </label>
                    <input 
                      v-if="currentConfigTask.config.promoCode"
                      type="text" 
                      v-model="currentConfigTask.config.promoCodeContent" 
                      placeholder="输入兑换码"
                      class="promo-input"
                    >
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.weeklyTicket"> 
                      <span>回溯周(票子)</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.weeklyRuby"> 
                      <span>召唤周(红宝石)</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.weeklyMine"> 
                      <span>挖矿周(饼干稿子)</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.speedup"> 
                      <span>狂飙排名奖励</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.mineReward"> 
                      <span>挖矿排名奖励</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.spendReward"> 
                      <span>连充夺宝</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.collectFlowerSeed"> 
                      <span>领花种子</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'garden_meat'" class="config-section-wrapper">
              <div class="config-group-title">日常操作</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.collectMeat"> 
                      <span>收肉</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.eatMeat"> 
                      <span>吃肉</span>
                    </label>

                    <label class="custom-check" style="margin-left: 12px;" v-if="currentConfigTask.config.eatMeat">
                      <input type="checkbox" v-model="currentConfigTask.config.eatNeighbors"> 
                      <span>吃邻居</span>
                    </label>

                    <label class="custom-check" style="margin-left: 12px;" v-if="currentConfigTask.config.eatMeat">
                      <input type="checkbox" v-model="currentConfigTask.config.eatGuilds"> 
                      <span>吃工会</span>
                    </label>

                    <label class="custom-check" style="margin-left: 12px;" v-if="currentConfigTask.config.eatMeat">
                      <input type="checkbox" v-model="currentConfigTask.config.eatRankings"> 
                      <span>吃排行榜</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'garden_veggie'" class="config-section-wrapper">
              <div class="config-group-title">单次操作</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.collectVeggie"> 
                      <span>菜地当前状态检测（浇水除虫收菜）</span>
                    </label>
                  </div>

                  <div class="dungeon-config-item veggie-row">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.plantVeggie"> 
                      <span>种菜</span>
                    </label>

                    <label class="custom-check" style="margin-left: 12px;" v-if="currentConfigTask.config.plantVeggie">
                      <input type="checkbox" v-model="currentConfigTask.config.buySeeds"> 
                      <span>购买种子</span>
                    </label>

                    <div v-if="currentConfigTask.config.plantVeggie" class="veggie-select-container">
                      <div 
                        class="dark-input veggie-trigger" 
                        @click="showVeggieDropdown = !showVeggieDropdown"
                      >
                        {{ veggieMap[currentConfigTask.config.veggieType] || '选择作物' }}
                        <span class="arrow-icon">▼</span>
                      </div>
                      
                      <div v-if="showVeggieDropdown" class="search-dropdown veggie-dropdown">
                        <div 
                          v-for="(name, key) in veggieMap" 
                          :key="key" 
                          @click="selectVeggie(key)" 
                          class="dropdown-item"
                        >
                          {{ name }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="config-group-title">高级功能</div>
              <div class="config-content-indent">
                <div class="dungeon-config-item scan-veggie-row" style="display: flex; align-items: center; gap: 10px;">
                  
                  <label class="custom-check">
                    <input type="checkbox" v-model="currentConfigTask.config.scanVeggie"> 
                    <span>扫菜</span>
                  </label>

                  <div v-if="currentConfigTask.config.scanVeggie" class="veggie-select-container" style="position: relative;">
                    <div class="dark-input veggie-trigger" @click.stop="showScanVeggieDropdown = !showScanVeggieDropdown">
                      <span class="veggie-text-preview">{{ selectedVeggieNames || '选择作物(多选)' }}</span>
                      <span class="arrow-icon">▼</span>
                    </div>
                    
                    <div v-if="showScanVeggieDropdown" class="search-dropdown veggie-dropdown">
                      <div 
                        v-for="(info, key) in remoteVeggies" 
                        :key="key" 
                        @click.stop="toggleInterestedVeggie(key)" 
                        class="dropdown-item multi-item"
                      >
                        <input type="checkbox" :checked="currentConfigTask.config.interestedVeggies?.includes(key)">
                        <span>{{ info.name }}</span>
                      </div>
                    </div>
                  </div>

                  <div v-if="currentConfigTask.config.scanVeggie" class="target-config-area">
                    <button class="action-btn" @click="openTargetModal">⚙️ 配置目标</button>
                    <span class="count-badge" v-if="currentConfigTask.config.selectedUids?.length">
                      ({{ currentConfigTask.config.selectedUids.length }}人)
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'garden_egg'" class="config-section-wrapper">
              <div class="config-group-title">日常操作</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.shareEgg"> 
                      <span>营地吆喝(多点3次蛋)</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'weekly_reward'" class="config-section-wrapper">
              <div class="config-group-title">奖励领取</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.groupMineReward"> 
                      <span>集体挖矿</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.skycityRewards"> 
                      <span>天空之城</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.seashell"> 
                      <span>海王贝壳（含领取竞猜奖励）</span>
                    </label>
                  </div>
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.redDiamondDrawReward"> 
                      <span>召唤2w抽</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'guild_pk'" class="config-section-wrapper">
              <div class="config-group-title" style="margin-top:15px;">上分操作</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div v-for="(val, key) in currentConfigTask.config.clears" :key="key" class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="val.active">
                      <span>{{ {boss:'领主', gold:'金币', goblin:'哥布林', lamp:'神灯'}[key] }}</span>
                    </label>
                    <div class="input-group" v-if="val.active">
                      <template v-if="key !== 'lamp'">
                        关卡: <input type="number" v-model.number="val.level" class="mini-num" min="1">
                      </template>
                      次数: <input type="number" v-model.number="val.count" class="mini-num">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.petDraw"> 
                      <span>抽宠物 (350抽)</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.petDraw">
                      次数: <input type="number" v-model.number="currentConfigTask.config.petDrawCount" class="mini-num" min="1">
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.ticketDraw"> 
                      <span>抽回溯票子 (10张)</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.ticketDraw">
                      次数: <input type="number" v-model.number="currentConfigTask.config.ticketDrawCount" class="mini-num" min="1">
                    </div>
                  </div>
                </div>
              </div>

              <div class="config-group-title" style="margin-top:15px;" v-if="userLevel >= 2">奖励领取</div>
              <div class="config-content-indent" v-if="userLevel >= 2">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input 
                        type="checkbox" 
                        v-model="currentConfigTask.config.guildDailyReward"
                      > 
                      <span>领取每日奖励</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'race'" class="config-section-wrapper">
              <div class="config-group-title" style="margin-top:15px;">狂飙（默认出战角色为哪吒+小王子）</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.rush"> 
                      <span>跑100米</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.rush">
                      次数: <input type="number" v-model.number="currentConfigTask.config.rushCount" class="mini-num" min="1">
                    </div>
                  </div>
                </div>
              </div>

              <div class="config-group-title" style="margin-top:15px;">冒险家</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.goldHammer"> 
                      <span>金锤子</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.goldHammer">
                      数量: 
                      <input 
                        type="number" 
                        v-model.number="currentConfigTask.config.goldHammerNum" 
                        class="mini-num" 
                        @input="handleNum($event, 'goldHammerNum', 50)"
                      >
                      次数: 
                      <input 
                        type="number" 
                        v-model.number="currentConfigTask.config.goldHammerCount" 
                        class="mini-num" 
                        min="1"
                      >
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.silverHammer"> 
                      <span>银锤子</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.silverHammer">
                      数量: 
                      <input 
                        type="number" 
                        v-model.number="currentConfigTask.config.silverHammerNum" 
                        class="mini-num" 
                        @input="handleNum($event, 'silverHammerNum', 50)"
                      >
                      次数: 
                      <input 
                        type="number" 
                        v-model.number="currentConfigTask.config.silverHammerCount" 
                        class="mini-num" 
                        min="1"
                      >
                    </div>
                  </div>

                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.wings"> 
                      <span>翅膀</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.wings">
                      数量: 
                      <input 
                        type="number" 
                        v-model.number="currentConfigTask.config.wingsNum" 
                        class="mini-num" 
                        @input="handleNum($event, 'wingsNum', 5)"
                      >
                      次数: 
                      <input 
                        type="number" 
                        v-model.number="currentConfigTask.config.wingsCount" 
                        class="mini-num" 
                        min="1"
                      >
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="currentConfigTask?.id === 'gamble'" class="config-section-wrapper">
              <div class="config-group-title" style="margin-top:15px;">竞猜</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.rush"> 
                      <span>(模版勿用)</span>
                    </label>
                    <div class="input-group" v-if="currentConfigTask.config.rush">
                      次数: <input type="number" v-model.number="currentConfigTask.config.rushCount" class="mini-num" min="1">
                    </div>
                  </div>
                </div>
              </div>

              <div class="config-group-title" style="margin-top:15px;">领奖/兑奖</div>
              <div class="config-content-indent">
                <div class="dungeon-grid">
                  <div class="dungeon-config-item">
                    <label class="custom-check">
                      <input type="checkbox" v-model="currentConfigTask.config.monthlyGuildReward"> 
                      <span>月赛领奖</span>
                    </label>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button class="modal-btn secondary" @click="cancelSelection">取消勾选</button>
            <button class="modal-btn primary" @click="confirmSelection">确认并保存</button>
          </div>
        </div>
      </div>
    </main>

    <div v-if="showUpdateModal" class="auth-overlay">
      <div class="auth-card" style="max-width: 480px;">
        <div class="auth-header">
          <span class="auth-icon">🚀</span>
          <h3>v2.5 更新说明</h3>
        </div>
        <div class="auth-body" style="text-align: left; max-height: 480px; overflow-y: auto; font-size: 13px; line-height: 1.6;">
          
          <div style="background: rgba(255, 149, 0, 0.1); border: 1px solid #ff9500; border-radius: 6px; padding: 10px; margin-bottom: 15px;">
            <p style="color: #ff9500; font-weight: bold; margin-bottom: 5px;">⚠️ 珍宝拦截重要提醒：</p>
            <p style="color: #eee; margin: 0; font-size: 12px;">珍宝拦截功能已全面开放！因拦截效率极高，<b>为防止误操作导致财产损失，启动前请务必确认生效规则。</b> 规则错误可能导致橙钻瞬间清空，请谨慎操作！</p>
          </div>

          <p style="color: #007aff; font-weight: bold; margin: 10px 0 5px 0;">【副本板块 · 强力支撑】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>🔑 <b>资源获取</b>：新增一键领取【龙宫】及【熔炉】钥匙。</li>
            <li>⚔️ <b>关卡推进</b>：支持自动挑战龙宫、熔炉关卡。</li>
            <li style="font-size: 11px; color: #888;">*受游戏机制限制，挑战需逐关进行，无法跨关跳过。龙宫、熔炉关卡一次只能通关一次。</li>
          </ul>

          <p style="color: #34c759; font-weight: bold; margin: 15px 0 5px 0;">【菜园维护 · 史诗级更新】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>🤖 <b>全能挂机</b>：新增 24h 全自动挂机功能。全面实现：收菜 → 种菜 → 自动吃肉 → 自动收肉。</li>
            <li>🌱 <b>漏洞修复</b>：收菜功能修复。</li>
          </ul>

          <p style="color: #af52de; font-weight: bold; margin: 15px 0 5px 0;">【珍宝模块 · 体验优化】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>💎 <b>全员开放</b>：珍宝拦截功能不再设限，面向全量用户。</li>
            <li>📊 <b>视觉重构</b>：优化词条显示，修复个别价格显示异常问题。</li>
            <li>⚖️ <b>状态区分</b>：清晰标注“可秒杀”与“公示中”，抢购更直观。</li>
          </ul>

          <p style="color: #ff3b30; font-weight: bold; margin: 15px 0 5px 0;">【工会与日常】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>🏴 <b>工会会战</b>：新增工会会战自动报名功能。</li>
            <li>⛏️ <b>任务修复</b>：修复挖矿研究加速仅生效一次的问题，现已支持持续加速。</li>
          </ul>

        </div>
        <button class="primary-btn" @click="showUpdateModal = false" style="margin-top: 20px; width: 100%;">确认进入助手</button>
      </div>
    </div>

    <div v-if="!isAuthorized" class="auth-overlay">
      <div class="auth-card">
        <div class="auth-icon">🔒</div>
        <h2>系统未授权</h2>
        <div class="machine-id-box" @click="copyMachineID">
          <span class="label">我的机器码:</span>
          <code class="code">{{ machineID }}</code>
        </div>
        <div class="auth-tips">请将机器码发送给管理开通权限</div>
        <button class="primary-btn" @click="checkAuth">重新验证</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch, computed, nextTick, onUnmounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime'
import { 
  UpdateRules, GetAttributeNames, GetCategoryNames, 
  ToggleCapture, ManualBuy, CheckCurrentAuth, GetMachineID,
  // 假设你在 app.go 中新增了下面这个函数
  ExecuteDailyTasks 
} from '../wailsjs/go/main/App'

// --- 导航与 UI 状态 ---
const activeTab = ref('daily') // 默认显示日常任务
const isAuthorized = ref(false)
const machineID = ref('')

// --- 日常任务相关变量 ---
const isTaskRunning = ref(false)
const dailyLogs = ref([])
const dailyLogContainer = ref(null)

// 弹窗控制
const showConfigModal = ref(false)
// 修改前：const currentConfigTask = ref(null)
// 修改后：
const currentConfigTask = ref({
  config: {
    gardenAFK: false
  }
});

const beijingTime = ref("")

// 更新时间的函数
const updateBeijingTime = () => {
  const now = new Date()
  
  // 将当前时间转换为 UTC 时间，再加 8 小时得到北京时间
  const utc = now.getTime() + (now.getTimezoneOffset() * 60000)
  const bjDate = new Date(utc + (3600000 * 8))
  
  // 格式化：2024-05-20 18:00:05
  const year = bjDate.getFullYear()
  const month = String(bjDate.getMonth() + 1).padStart(2, '0')
  const day = String(bjDate.getDate()).padStart(2, '0')
  const hours = String(bjDate.getHours()).padStart(2, '0')
  const minutes = String(bjDate.getMinutes()).padStart(2, '0')
  const seconds = String(bjDate.getSeconds()).padStart(2, '0')
  
  beijingTime.value = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

let timer = null

onMounted(() => {
  updateBeijingTime()
  timer = setInterval(updateBeijingTime, 1000) // 每秒更新一次
})

onUnmounted(() => {
  if (timer) clearInterval(timer) // 销毁组件时清除定时器，防止内存泄漏
})


const handleNum = (event, key, max) => {
  // 1. 获取用户当前输入的值
  // 注意：即便 v-model 绑定了数字，event.target.value 拿到的初始值也是字符串
  let val = parseInt(event.target.value);

  // 2. 这里的 config 是响应式对象
  const config = currentConfigTask.value.config;

  // 3. 核心逻辑判断
  if (val > max) {
    val = max;
  } else if (val < 1) {
    // 如果输入为空或者是负数，设为 1
    val = 1;
  }

  // 4. 强制写回数据模型
  config[key] = val;

  // 5. 【关键】强制刷新 DOM 显示
  // 即使 Vue 数据变了，有时浏览器还没渲染，这行代码能确保输入框里显示的数字立刻变回 max
  event.target.value = val;
};

const taskCategories = reactive([
  {
    title: '日常任务',
    id: 'basic',
    tasks: [
      { 
        id: 'dungeon_all', // 任务ID
        name: '副本', 
        desc: '提取钥匙/稿子&各副本通关', 
        selected: false,
        // 核心：弹窗内要显示的复杂配置
        config: {
          res: { getKeys: true, getPicks: true }, // 资源提取
          clears: {
            boss: { active: false, level: 184, count: 4 },    // 领主
            gold: { active: false, level: 246, count: 4 },    // 金币
            goblin: { active: false, level: 150, count: 4 },  // 哥布林
            lamp: { active: false, count: 4 },    // 神灯
            west: {active: false, count: 1},      // 西游
            gear: {active: false, level: 80, count: 1},      // 龙宫
            demon: {active: false, level: 80, count: 1},      // 熔炉
          }
        }
      },
      { 
        id: 'guild_sign', 
        name: '工会', 
        desc: '进退工会/签到/砍价/打boss/买药水', 
        selected: false,
        config: { guildNum: 204685, guildSign: true, bargain: true, boss: false, bossLevel: 216, bossCount: 4, medCount: 3 }
      },
      { 
        id: 'red_diamond', 
        name: '召唤', 
        desc: '领取免费红宝石/日常召唤', 
        selected: false,
        config: { shopRuby: true, limitRuby: true, freeSummon: true, equipDrawCount: 10, skillDrawCount: 10, petDrawCount: 10}
      },
      { 
        id: 'reward_tasks', 
        name: '每日任务', 
        desc: '一键领取各类日常任务奖励', 
        selected: false,
        config: { card: true, mail: true, treasureMap: true, weeklyRuby: false, mineAccelerate: true, spendReward: true, weeklyMine: true}
      },
    ]
  },
  {
    title: '菜园维护',
    id: 'garden',
    tasks: [
      { 
        id: 'garden_meat', 
        name: '肉类', 
        desc: '吃肉/收肉', 
        selected: false,
        config: {collectMeat: true, eatMeat: true, eatNeighbors: true, eatGuilds: true, eatRankings: true} // 预留配置
      },
      { 
        id: 'garden_veggie', 
        name: '菜类', 
        desc: '收菜/种菜/浇水/除虫/偷菜', 
        selected: false,
        config: {collectVeggie: true, buySeeds: true} // 预留配置
      },
      { 
        id: 'garden_egg', 
        name: '蛋类', 
        desc: '吆喝', 
        selected: false,
        config: {} // 预留配置
      },
    ]
  },
  {
    title: '周期活动',
    id: 'activity',
    tasks: [
      { 
        id: 'weekly_reward', 
        name: '其他活动', 
        desc: '饼干/天空之城/海王/宝库/幻化', 
        selected: false,
        config: {}
      },
      { 
        id: 'guild_pk', 
        name: '工会对决', 
        desc: '工会对决上分辅助', 
        selected: false,
        config: {
          petDrawCount: 1,      // 抽宠物次数
          ticketDrawCount: 1,   // 抽票子次数 
          // 💡 必须新增下面这个 clears 对象
          clears: {
            boss: { active: false, level: 185, count: 4 },
            gold: { active: false, level: 246, count: 4 },
            goblin: { active: false, level: 150, count: 4 },
            lamp: { active: false, count: 4 }
          }
        }
      },
      { 
        id: 'race', 
        name: '竞技类', 
        desc: '狂飙/冒险家上分辅助', 
        selected: false,
        config: {rushCount: 30, goldHammerNum: 50, goldHammerCount: 1, silverHammerNum:50, silverHammerCount: 1, wingsNum: 5, wingsCount: 1}
      },
      { 
        id: 'gamble', 
        name: '竞猜类', 
        desc: '海王/月赛/武道会', 
        selected: false,
        config: {}
      },
    ]
  }
])

// --- 珍宝拦截相关变量 ---
const isCapturing = ref(true)
const rules = ref([]) 
const searchKeyword = ref('') 
const showDropdown = ref(false)
const attributeOptions = ref([])
const categoryOptions = ref([])
const qualityOptions = ref(['神话', '超越'])
const zhenbaoLogs = ref([]) 
const sysLogs = ref([])     
const activeRuleSummary = ref("")
const hitLogs = ref([])
const zhenbaoConfig = reactive({
  autoRefresh: false
})

// 用户选中的置顶目标属性
const priorityAttributes = ref([]);

// 排序逻辑：根据命中目标属性的数量进行排序
const sortedZhenbaoLogs = computed(() => {
  // slice() 防止直接修改原数组导致渲染死循环
  return zhenbaoLogs.value.slice().sort((a, b) => {
    // 1. 优先按照购买状态：未买的在上
    if (a.purchased !== b.purchased) return a.purchased ? 1 : -1;

    // 2. 计算权重（命中目标属性的数量）
    const getWeight = (item) => {
      if (!item.attributes || priorityAttributes.value.length === 0) return 0;
      return item.attributes.filter(attr => 
        priorityAttributes.value.includes(attr.name)
      ).length;
    };

    const weightA = getWeight(a);
    const weightB = getWeight(b);

    if (weightA !== weightB) return weightB - weightA; // 权重高的在前

    // 3. 权重一样时，按时间倒序（最新的在前）
    return 0; 
  });
});

// 柔和颜色映射
const getSoftColor = (colorName) => {
  const colorMap = {
    '红色': '#ff6b6b', '青色': '#11e4e4', '橙色': '#ff9f43',
    '紫色': '#c262ea', '蓝色': '#74b9ff', '绿色': '#b5f2b5',
    '白色': '#dcdde1', '超越': '#cba6f7', '神话': '#f5c2e7',
  }
  return colorMap[colorName] || '#cdd6f4'
}

// 1. 在 data 或 ref 中定义
const showVeggieDropdown = ref(false);

const veggieMap = {
  'luobo': '萝卜',
  'xiaomai': '小麦',
  'baicai': '白菜',
  'huluobo': '胡萝卜',
  'yumi': '玉米',
  'nangua': '南瓜',
  'zhishujie': '植树节种子'
};

// 2. 选择作物的函数
const selectVeggie = (key) => {
  currentConfigTask.value.config.veggieType = key;
  showVeggieDropdown.value = false;
};

// --- 扫菜配置相关变量 ---
const showScanVeggieDropdown = ref(false) // 控制作物多选下拉
const showTargetModal = ref(false)        // 控制目标配置弹窗
const remoteVeggies = ref({})             // 存放从 Go 获取的作物字典
const allTargets = ref([])                // 存放从 JSON 获取的完整玩家名单
const targetSearchQuery = ref('')         // 玩家搜索框
const tempSelectedUids = ref([])          // 弹窗内的临时勾选列表

// 获取已选作物的名字（用于界面显示）
const selectedVeggieNames = computed(() => {
  const ids = currentConfigTask.value?.config?.interestedVeggies || []
  if (ids.length === 0) return ""
  // remoteVeggies 是从 Go 获取的 { "1": {name:"白菜"}, "2":{name:"萝卜"} }
  return ids.map(id => remoteVeggies.value[id]?.name || id).join(', ')
})

// 弹窗内的过滤列表
const filteredTargetList = computed(() => {
  const query = targetSearchQuery.value.toLowerCase()
  if (!query) return allTargets.value
  return allTargets.value.filter(item => 
    item.name.toLowerCase().includes(query) || 
    String(item.uid).includes(query)
  )
})

// 切换感兴趣的作物（多选）
const toggleInterestedVeggie = (key) => {
  if (!currentConfigTask.value.config.interestedVeggies) {
    currentConfigTask.value.config.interestedVeggies = []
  }
  const arr = currentConfigTask.config.interestedVeggies
  const index = arr.indexOf(key)
  if (index > -1) arr.splice(index, 1)
  else arr.push(key)
}

// 目标配置弹窗逻辑
const openTargetModal = () => {
  // 进入弹窗前，将正式配置中的 UID 拷贝到临时变量
  tempSelectedUids.value = [...(currentConfigTask.value.config.selectedUids || [])]
  showTargetModal.value = true
}

const toggleTempTarget = (uid) => {
  const index = tempSelectedUids.value.indexOf(uid)
  if (index > -1) tempSelectedUids.value.splice(index, 1)
  else tempSelectedUids.value.push(uid)
}

const saveTargetConfig = () => {
  currentConfigTask.value.config.selectedUids = [...tempSelectedUids.value]
  showTargetModal.value = false
  addDailyLog(`已更新扫描名单: ${tempSelectedUids.value.length} 人`, "info")
}

const handleGardenAFKChange = async () => {
  console.log("🖱️ 自动挂机开关状态改变");
  await nextTick();

  try {
    // 1. 定位菜园大类
    const gardenCategory = taskCategories.find(c => c.id === 'garden');
    if (!gardenCategory) return;

    // 2. 获取当前总开关状态
    const isEnabled = currentConfigTask.value?.config?.gardenAFK || false;

    // 3. 【核心优化】：如果用户关闭了总开关，自动取消所有子项勾选
    if (!isEnabled) {
      console.log("🧹 检测到关闭，正在重置 UI 勾选状态...");
      gardenCategory.tasks.forEach(task => {
        // 取消大类任务的勾选状态（即左侧列表的勾选框）
        task.selected = false;

        // 遍历 config，将所有布尔值设为 false
        if (task.config) {
          Object.keys(task.config).forEach(key => {
            if (typeof task.config[key] === 'boolean' && key !== 'gardenAFK') {
              task.config[key] = false;
            }
          });
          // 如果有特殊字段需要重置，也可以在这里处理
          // 例如：task.config.veggieType = 'xiaomai';
        }
      });
      console.log("✅ UI 状态已清空");
    }

    // 4. 合并当前最新的配置（此时已是清空后的配置）发给后端
    const combinedConfig = {};
    gardenCategory.tasks.forEach(task => {
      if (task.config) {
        Object.assign(combinedConfig, JSON.parse(JSON.stringify(task.config)));
      }
    });

    // 5. 调用后端同步
    if (window.go?.main?.App?.ToggleGardenLoop) {
      await window.go.main.App.ToggleGardenLoop(isEnabled, combinedConfig);
      
      // 可选：给个前端提示
      if (!isEnabled) {
        addDailyLog("🛑 已停止自动挂机并重置所有选项", "warning");
      }
    }
  } catch (err) {
    console.error("❌ 联动清空失败:", err);
  }
};

const newRule = reactive({
  keyword: '',
  quality: '',
  min: 0,
  price: 1500,
  targetCategories: []
})

const userLevel = ref(0); // 0:未授权, 1:普通, 2:高级
const zhenbaoAutoRefresh = ref(false);

async function checkAuth() {
    const level = await window.go.main.App.CheckCurrentAuth();
    if (level > 0) {
        isAuthorized.value = true;
        userLevel.value = level;
    } else {
        isAuthorized.value = false;
    }
}

// --- 日常任务逻辑 ---

const addDailyLog = (message, type = 'default') => {
  const time = new Date().toLocaleTimeString()
  dailyLogs.value.push({ time, message, type })
  // 自动滚动
  nextTick(() => {
    if (dailyLogContainer.value) {
      dailyLogContainer.value.scrollTop = dailyLogContainer.value.scrollHeight
    }
  })
}

// 修改 script setup 中的 handleRunTasks
const handleRunTasks = async () => {
  // 构造包含 ID 和 Config 的对象数组
  const selectedTasks = []
  taskCategories.forEach(cat => {
    cat.tasks.forEach(task => {
      if (task.selected) {
        selectedTasks.push({
          id: task.id,
          config: JSON.parse(JSON.stringify(task.config)) // 深拷贝配置，防止引用干扰
        })
      }
    })
  })

  if (selectedTasks.length === 0) {
    alert("请至少勾选一个任务！")
    return
  }

  isTaskRunning.value = true
  dailyLogs.value = []
  addDailyLog("▶ 自动化流水线已启动...", "info")

  try {
    // 注意：这里传递的是对象数组，后端 Go 结构体需要对应匹配
    await ExecuteDailyTasks(selectedTasks)
  } catch (err) {
    addDailyLog(`❌ 启动失败: ${err}`, "error")
    isTaskRunning.value = false
  }
}

// 打开配置弹窗
const openConfig = (task) => {
  if (isTaskRunning.value) return 
  currentConfigTask.value = task
  showConfigModal.value = true
}

// 确认配置：保存并勾选该任务
const confirmSelection = () => {
  currentConfigTask.value.selected = true
  showConfigModal.value = false
  addDailyLog(`已启用: ${currentConfigTask.value.name}`, "info")
}

// 取消选择：关闭弹窗并取消勾选
const cancelSelection = () => {
  currentConfigTask.value.selected = false
  showConfigModal.value = false
}

const showUpdateModal = ref(localStorage.getItem('lastVersion') !== '2.4')

// 点击“我知道了”后，下次打开就不再显示
const closeUpdateModal = () => {
  showUpdateModal.value = false
  localStorage.setItem('lastVersion', '2.4')
}

// --- 珍宝拦截逻辑 ---

const filteredAttributes = computed(() => {
  const all = attributeOptions.value || []
  if (!searchKeyword.value) return all
  const searchLower = searchKeyword.value.toLowerCase()
  return all.filter(name => name.toLowerCase().includes(searchLower))
})

const selectAttribute = (name) => {
  newRule.keyword = name
  searchKeyword.value = name 
  showDropdown.value = false
}

watch(rules, (newVal) => {
  UpdateRules(JSON.parse(JSON.stringify(newVal)))
}, { deep: true })

// --- 生命周期与事件监听 ---
onMounted(async () => {
  // --- 【核心修复】第一步：立即同步初始化菜园配置指向 ---
  const immediateInit = () => {
    if (taskCategories && taskCategories.length > 0) {
      const gardenCategory = taskCategories.find(c => c.id === 'garden');
      if (gardenCategory && gardenCategory.tasks.length > 0) {
        // 这里的赋值确保了 currentConfigTask 不为 null
        currentConfigTask.value = gardenCategory.tasks[0];
        console.log("🚀 [立即初始化] 菜园配置已绑定:", currentConfigTask.value.name);
      }
    }
  };
  immediateInit(); 

  /**
   * 1. 注册授权成功事件监听
   */
  if (window.runtime) {
    window.runtime.EventsOn("auth_success", (data) => {
      console.log("收到授权信息:", data);
      isAuthorized.value = true;
      if (data && typeof data === 'object') {
        // expiryDate.value = data.expiry || ""; // 确保你有定义这个 ref
        userLevel.value = data.level || 1;
        if (data.mID) {
          machineID.value = data.mID.replace(/[\r\n]/g, "").trim();
        }
      }
    });

    window.runtime.EventsOn("task_progress", (data) => {
      if (data.message) addDailyLog(data.message, data.type || 'default');
      if (data.finished === true) isTaskRunning.value = false;
    });

    window.runtime.EventsOn("game_status", (msg) => {
      if (msg.includes("当前生效拦截规则")) {
        activeRuleSummary.value = msg;
        hitLogs.value = [];
      } else if (msg.includes("🎯 命中")) {
        hitLogs.value.unshift(msg);
        if (hitLogs.value.length > 20) hitLogs.value.pop();
      } else {
        sysLogs.value.unshift(msg);
      }
    });

    window.runtime.EventsOn("zhenbao_log", (item) => {
      item.purchased = false;
      zhenbaoLogs.value.unshift(item);
      if (zhenbaoLogs.value.length > 1000) zhenbaoLogs.value.pop();
    });
  }

  /**
   * 4. 权限与基础数据加载
   */
  try {
    const rawId = await window.go.main.App.GetMachineID();
    if (rawId) machineID.value = rawId.replace(/[\r\n]/g, "").trim();

    const level = await window.go.main.App.CheckCurrentAuth();
    if (level > 0) {
      isAuthorized.value = true;
      userLevel.value = level;
    }
  } catch (err) {
    console.error("授权校验异常:", err);
  }

  // 5. 拉取珍宝拦截相关的属性配置
  try {
    const [attrs, cats] = await Promise.all([
      window.go.main.App.GetAttributeNames(), 
      window.go.main.App.GetCategoryNames()
    ]);
    if (attrs) attributeOptions.value = attrs;
    if (cats) {
      const order = { "器皿1号位": 1, "乐器2号位": 2, "雕塑3号位": 3, "遗质4号位": 4, "机械5号位": 5, "礼器6号位": 6 };
      categoryOptions.value = cats.sort((a, b) => (order[a] || 99) - (order[b] || 99));
    }
  } catch (e) {
    console.error("基础数据拉取失败:", e);
  }

  /**
   * 🌟 6. 新增：拉取菜园作物配置与远程玩家名单
   */
  try {
    // 拉取 Go 端 a.gardenData.Veggies (来自你的 garden_veggie.json)
    const gData = await window.go.main.App.GetGardenData();
    if (gData && gData.veggies) {
      remoteVeggies.value = gData.veggies;
      console.log("✅ 菜园品种同步成功:", Object.keys(remoteVeggies.value).length);
    }

    // 拉取远程玩家大名单 (targets.json)
    const tData = await window.go.main.App.GetRemoteTargets();
    if (tData && tData.data) {
      allTargets.value = tData.data;
      console.log("✅ 扫描名单同步成功:", allTargets.value.length);
    }
  } catch (err) {
    console.error("❌ 菜园/名单数据同步失败:", err);
  }

  // --- 兜底逻辑 ---
  setTimeout(() => {
    if (!currentConfigTask.value) immediateInit();
  }, 500);

  // 7. 时间更新逻辑
  updateBeijingTime();
  const timeIntv = setInterval(updateBeijingTime, 1000);
  
  onUnmounted(() => {
    if (timeIntv) clearInterval(timeIntv);
  });
});

// --- 其他操作方法 ---

const copyMachineID = () => {
  navigator.clipboard.writeText(machineID.value)
  alert("机器码已复制。")
}

const syncAutoRefresh = async () => {
  try {
    // 确保你的 app.go 里定义了 UpdateZhenbaoAutoRefresh 函数
    await window.go.main.App.UpdateZhenbaoAutoRefresh(zhenbaoConfig.autoRefresh)
    console.log("✅ 自动刷新状态同步成功:", zhenbaoConfig.autoRefresh)
  } catch (err) {
    console.error("❌ 同步失败:", err)
  }
}

// const checkAuth = () => { window.location.reload() }

const addRule = () => {
  if (newRule.quality || newRule.keyword || newRule.price > 0 || newRule.targetCategories.length > 0) {
    rules.value.push(JSON.parse(JSON.stringify(newRule)))
    newRule.keyword = ''; newRule.quality = ''; searchKeyword.value = ''; newRule.targetCategories = []
  }
}

const removeRule = (index) => { rules.value.splice(index, 1) }
const isGlobalCooling = ref(false);

const handleManualBuy = (item) => {
  if (isGlobalCooling.value || item.purchased || item.isLocked) return;

  // 自动检测是 Purchase 还是 purchase
  const sendCmd = window.go.main.App.purchase || window.go.main.App.Purchase;

  if (sendCmd) {
    sendCmd(item.purchaseCmd)
      .then(() => {
        item.purchased = true;
        console.log("✅ 发送成功");
      })
      .catch(err => console.error("❌ 执行失败:", err));
  } else {
    console.error("🚨 还是找不到函数！当前 App 拥有的函数列表：", Object.keys(window.go.main.App));
  }

  isGlobalCooling.value = true;
  setTimeout(() => { isGlobalCooling.value = false; }, 2600);
};

const toggleMonitor = async () => {
  const target = !isCapturing.value
  try {
    const res = await ToggleCapture(target)
    if (res === "SUCCESS_ON" || res === "SUCCESS_OFF") isCapturing.value = target
  } catch (e) { console.error(e) }
}
</script>

<style>
/* ========== 全局与容器基础 ========== */
body, html { 
  margin: 0; padding: 0; height: 100%; 
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica; 
  background: #000; color: #fff; overflow: hidden; 
}

.app-container { display: flex; height: 100vh; background: #000; }

/* 页面切换动画 */
.animate-fade {
  animation: fadeIn 0.3s ease-out;
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ========== 侧边栏样式 (Sidebar) ========== */
.sidebar { 
  width: 280px; background: #0a0a0a; border-right: 1px solid #1a1a1a; 
  display: flex; flex-direction: column; box-sizing: border-box; 
  z-index: 10;
  --wails-draggable: drag;
}

.brand { padding: 25px; border-bottom: 1px solid #1a1a1a; }
.brand h2 { font-size: 1.1rem; margin: 0; font-weight: 600; color: #fff; }
.brand small { color: #007aff; font-size: 0.7rem; letter-spacing: 1px; }

/* 导航菜单 */
.menu-list { padding: 15px 10px; }
.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 15px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 5px;
  color: #888;
  --wails-draggable: no-drag;
}
.menu-item:hover { background: #1a1a1a; color: #fff; }
.menu-item.active {
  background: #1a1a1a;
  color: #007aff;
  box-shadow: inset 0 0 0 1px #007aff22;
}
.menu-item .icon { font-size: 1.2rem; }
.menu-title { display: block; font-size: 0.9rem; font-weight: 500; }
.menu-desc { display: block; font-size: 0.7rem; opacity: 0.6; }

/* 侧边栏配置包装器 */
.config-wrapper {
  flex-grow: 1;
  overflow-y: auto;
  padding: 0 20px 20px;
}

/* 规则列表外层容器 */
.rule-list-container {
  margin-top: 20px;
  padding: 0 10px;
}

/* 标题样式 */
.section-title {
  font-size: 13px;
  color: #888;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 1px;
  display: flex;
  align-items: center;
}

/* 单条规则卡片 */
.rule-item {
  display: flex;
  align-items: center;
  background: rgba(255, 255, 255, 0.03); /* 极淡的透明白 */
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-left: 3px solid #00ff88; /* 核心：左侧绿色激活条 */
  border-radius: 6px;
  padding: 10px 14px;
  margin-bottom: 10px;
  transition: all 0.2s ease;
  position: relative;
}

.rule-item:hover {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(0, 255, 136, 0.3);
  transform: translateX(2px); /* 悬停时轻微右移 */
}

/* 左侧信息区 */
.rule-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

/* 必须给父级加这个，下拉框才会贴着输入框显示 */
.search-select-container {
  position: relative; 
  width: 100%;
}

/* 下拉列表容器 */
.search-dropdown {
  position: absolute;
  top: calc(100% + 5px); /* 距离输入框底部 5 像素 */
  left: 0;
  width: 100%;
  background: #1a1a1a;   /* 使用纯深色，防止透明度过高看不见 */
  border: 1px solid #333;
  border-radius: 8px;
  max-height: 200px;
  overflow-y: auto;
  z-index: 9999;         /* 确保在最前面 */
  box-shadow: 0 10px 30px rgba(0,0,0,0.5);
}

/* 下拉选项字体美化 */
.dropdown-item {
  padding: 10px 14px;
  /* 关键：设置清爽的系统字体 */
  font-family: "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
  font-size: 13px;
  color: #ccc;
  cursor: pointer;
  border-bottom: 1px solid #222;
  transition: background 0.2s;
}

/* 最后一个选项去掉底边线 */
.dropdown-item:last-child {
  border-bottom: none;
}

/* 悬停效果：变蓝并加亮字体 */
.dropdown-item:hover {
  background: #007aff; 
  color: #ffffff;
}

/* 滚动条细化，不占用太多视觉空间 */
.search-dropdown::-webkit-scrollbar {
  width: 4px;
}
.search-dropdown::-webkit-scrollbar-thumb {
  background: #444;
  border-radius: 10px;
}

/* 自动刷新容器 */
.auto-refresh-wrapper {
  margin: 15px 0 10px 0;
  padding: 0 4px;
}

.refresh-checkbox {
  display: flex;
  align-items: center;
  cursor: pointer;
  user-select: none;
  gap: 8px;
}

/* 隐藏原始复选框 */
.refresh-checkbox input {
  display: none;
}

/* 自定义复选框方框 */
.checkbox-box {
  width: 14px;
  height: 14px;
  border: 1.5px solid #444;
  border-radius: 3px;
  transition: all 0.2s ease;
  position: relative;
}

/* 文本样式：缩小字号，使用次要文字颜色 */
.refresh-text {
  font-size: 12px;
  color: #888;
  letter-spacing: 0.3px;
  transition: color 0.2s;
}

/* 选中状态：方框变蓝 */
.refresh-checkbox input:checked + .checkbox-box {
  background: #007aff;
  border-color: #007aff;
}

/* 选中状态：添加一个小勾勾 */
.refresh-checkbox input:checked + .checkbox-box::after {
  content: '';
  position: absolute;
  left: 4px;
  top: 1px;
  width: 4px;
  height: 7px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

/* 选中状态：文字微亮 */
.refresh-checkbox input:checked ~ .refresh-text {
  color: #ccc;
}

/* 悬停效果 */
.refresh-checkbox:hover .checkbox-box {
  border-color: #666;
}

/* 第一行：关键词和数值 */
.rule-main {
  font-size: 13px;
  font-weight: 600;
  color: #e0e0e0;
  display: flex;
  align-items: center;
  font-family: 'JetBrains Mono', 'Courier New', monospace;
}

/* 让数字和符号更好看 */
.rule-main::after {
  content: ""; /* 如果需要可以在这里加装饰 */
}

/* 第二行：副标题/过滤条件 */
.rule-sub {
  font-size: 11px;
  color: #666; /* 稍微暗一点 */
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 强调关键词和价格 */
.rule-sub::first-letter {
  text-transform: uppercase;
}

/* 删除按钮 */
.delete-btn {
  background: transparent;
  border: none;
  color: #555;
  font-size: 18px;
  cursor: pointer;
  padding: 5px;
  line-height: 1;
  transition: color 0.2s;
  border-radius: 4px;
}

.delete-btn:hover {
  color: #ff4d4d;
  background: rgba(255, 77, 77, 0.1);
}

.sidebar-footer {
  padding: 15px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(0, 0, 0, 0.2);
}

.time-display {
  display: flex;
  flex-direction: column;
  margin-bottom: 8px;
}

.time-display .label {
  font-size: 10px;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.time-display .time-value {
  font-family: 'Courier New', Courier, monospace; /* 使用等宽字体防止数字跳动 */
  font-size: 14px;
  color: #cfe2f3; /* 科技绿 */
  text-shadow: 0 0 5px rgba(0, 255, 136, 0.3);
}

.mini-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 4px;
  
  /* --- 关键：精致的小框 --- */
  border: 1px solid rgba(255, 185, 56, 0.2); /* 半透明暗金边框 */
  background: rgba(255, 185, 56, 0.05);     /* 极淡的背景色 */
  
  font-size: 0.7rem; /* 字体整体变小 */
  white-space: nowrap;
}

.badge-text {
  /* --- 关键：不再是纯白色 --- */
  color: #a0a0a0; /* 柔和的深灰色 */
  font-weight: 400;
  letter-spacing: 0.3px;
}

/* ========== 日常任务页面专用样式 ========== */
.page-view {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* 保证标题栏高度统一，不会因为有开关就变高 */
.column-header {
  min-height: 28px; 
  padding: 4px 8px;
}

.garden-afk-wrapper {
  display: flex;
  align-items: center;
}

.mini-afk-label {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  background: rgba(255, 149, 0, 0.15); /* 淡淡的橙色背景 */
  padding: 1px 6px;
  border-radius: 4px;
  border: 1px solid rgba(255, 149, 0, 0.2);
  /* 强制不换行 */
  white-space: nowrap;
}

.afk-text {
  font-size: 10px; /* 缩小字体 */
  color: #ff9500;
  font-weight: bold;
}

.mini-afk-label input[type="checkbox"] {
  width: 12px;
  height: 12px;
  margin: 0;
  accent-color: #ff9500;
}

.daily-layout {
  padding: 25px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  flex-grow: 1;
  overflow-y: auto;
}

.daily-tasks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.task-card-new {
  background: #111;
  border: 1px solid #222;
  border-radius: 12px;
  padding: 15px;
  display: flex;
  align-items: center;
  gap: 15px;
  cursor: pointer;
  transition: all 0.2s;
}
.task-card-new:hover { border-color: #444; background: #161616; }
.task-card-new.is-selected { border-color: #007aff; background: rgba(0, 122, 255, 0.05); }

.task-check-icon { font-size: 1.2rem; }
.task-name { font-size: 0.9rem; font-weight: bold; color: #eee; margin-bottom: 2px; }
.task-desc { font-size: 0.75rem; color: #666; }

/* 日常日志控制台 */
.daily-console {
  background: #000;
  border: 1px solid #1a1a1a;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  height: 300px;
}
.console-header {
  padding: 10px 15px;
  background: #0a0a0a;
  border-bottom: 1px solid #1a1a1a;
  font-size: 0.75rem;
  color: #555;
  font-weight: bold;
}
.console-body {
  flex-grow: 1;
  padding: 15px;
  overflow-y: auto;
  font-family: "Cascadia Code", "Monaco", monospace;
  font-size: 0.8rem;
}
.console-empty { color: #333; text-align: center; margin-top: 100px; }

.log-entry { margin-bottom: 6px; line-height: 1.4; display: flex; gap: 10px; }
.log-timestamp { color: #444; flex-shrink: 0; }
.log-message { color: #ccc; }
.log-entry.info .log-message { color: #cfe2f3; }
.log-entry.error .log-message { color: #ff4d4d; }

.action-footer {
  padding: 10px 0;
  display: flex;
  justify-content: center;
}
.run-all-btn {
  width: 100%;
  max-width: 400px;
  padding: 16px;
  background: #007aff;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-weight: bold;
  cursor: pointer;
  font-size: 1rem;
  transition: 0.2s;
}
.run-all-btn:hover:not(:disabled) { background: #0062cc; transform: scale(1.02); }
.run-all-btn:disabled { background: #222; color: #555; cursor: not-allowed; }

/* ========== 珍宝拦截页面样式 (保留并优化) ========== */
/* 命中消息的文字样式 */
.hit-msg {
  font-size: 11px !important;    /* 调小字号到 11px，非常精致 */
  line-height: 1.4;              /* 缩减行高 */
  color: #ff6b6b;               /* 保持红色或改成你喜欢的颜色 */
  font-family: "Consolas", monospace; /* 使用等宽字体，看起来更有“控制台”的感觉 */
}

/* 调整存放命中信息的卡片间距 */
.hit-card {
  padding: 6px 10px !important; /* 缩小上下内边距，让卡片变矮 */
  margin-bottom: 4px !important; /* 缩小卡片之间的距离 */
  border-left: 2px solid #ffa1a1; /* 添加一个小红边，区分普通日志 */
}

.main-content { flex-grow: 1; display: flex; flex-direction: column; background: #050505; }

.status-bar { 
  padding: 15px 25px; background: #0a0a0a; border-bottom: 1px solid #1a1a1a; 
  display: flex; justify-content: space-between; align-items: center; 
}

.status-control { 
  cursor: pointer; display: flex; align-items: center; gap: 12px; 
  padding: 6px 15px; border-radius: 20px; transition: 0.2s; 
}
.status-control:hover { background: #1a1a1a; }

.dot { width: 10px; height: 10px; background: #00ff00; border-radius: 50%; }
.pulsing { animation: pulse 1.5s infinite; box-shadow: 0 0 10px #00ff00; }
.is-paused .dot { background: #ff4d4d; box-shadow: none; animation: none; }

.status-text { font-weight: bold; font-size: 0.85rem; color: #eee; }
.is-paused .status-text { color: #ff4d4d; }

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* “数据流”文字标签 */
.stats-label {
  font-size: 0.65rem;
  color: #666; /* 低调的深灰 */
  font-weight: 500;
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

/* “数字”部分 */
.stats-count {
  font-family: "Cascadia Code", "Monaco", monospace; /* 数字用等宽字体更整齐 */
  font-size: 0.75rem;
  color: #cfe2f3;
  font-weight: 700;
  min-width: 20px;
  text-align: center;
}

.clear-btn {
  /* --- 基础布局 --- */
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  
  /* --- 关键：精致感设定 --- */
  background: rgba(255, 255, 255, 0.03); /* 极淡的背景 */
  border: 1px solid rgba(255, 255, 255, 0.1); /* 极细的半透明边框 */
  
  /* --- 关键：字体美化 --- */
  color: #888;          /* 柔和的深灰色，不抢眼 */
  font-size: 0.75rem;   /* 字体变小 */
  font-weight: 500;
  letter-spacing: 0.5px; /* 拉开字间距 */
}

/* 鼠标悬停效果：边框变亮，文字变白一点 */
.clear-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
  color: #ccc;
  transform: translateY(-1px); /* 轻轻上浮 */
}

/* 点击时的反馈 */
.clear-btn:active {
  transform: translateY(0);
  background: rgba(255, 255, 255, 0.05);
}

.btn-icon {
  font-size: 0.8rem;
  filter: grayscale(100%) opacity(0.6); /* 让垃圾桶图标也变灰色，不刺眼 */
}

.clear-btn:hover .btn-icon {
  filter: grayscale(0%) opacity(1); /* 悬停时图标恢复彩色 */
}
/* 拦截规则配置表单 */
.section-title { font-size: 0.7rem; color: #555; text-transform: uppercase; margin: 15px 0 10px 0; font-weight: bold; }
.input-card { background: #141414; padding: 15px; border-radius: 12px; border: 1px solid #222; margin-bottom: 15px; }
.form-group label { display: block; font-size: 0.7rem; color: #666; margin-bottom: 6px; }
.dark-input { 
  width: 100%; background: #000; border: 1px solid #222; color: #fff; 
  padding: 8px 12px; border-radius: 8px; box-sizing: border-box; font-size: 0.85rem; 
}
.dark-input:focus { border-color: #007aff; outline: none; }

.primary-btn { 
  width: 100%; background: #007aff; border: none; color: #fff; padding: 10px; 
  border-radius: 8px; cursor: pointer; font-weight: bold; margin-top: 10px;
}

.veggie-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.veggie-select-container {
  position: relative;
  width: 120px; /* 控制下拉框宽度 */
}

/* 模拟输入框的点击区域 */
.veggie-trigger {
  height: 28px;
  line-height: 28px;
  padding: 0 10px;
  font-size: 0.8rem;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 4px;
}

.arrow-icon {
  font-size: 0.6rem;
  opacity: 0.5;
}

/* 确保下拉列表在最上方 */
.veggie-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  width: 100%;
  z-index: 99;
  max-height: 200px;
  overflow-y: auto;
}

/* 核心日志控制台列表 */
.console-container {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
}

.console-card {
  background: #0a0a0a;
  border-radius: 8px;
  padding: 12px;
  border: 1px solid #1a1a1a;
  font-family: "Cascadia Code", monospace;
}

/* 容器居中 */
.empty-state-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 20px;
  min-height: 400px; /* 确保在中间 */
  opacity: 0.8;
}

/* --- 核心：雷达扫描动画 --- */
.scanner-wrapper {
  position: relative;
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.scanner-ring {
  position: absolute;
  width: 100%;
  height: 100%;
  border: 2px solid rgba(86, 241, 207, 0.1);
  border-radius: 50%;
}

.scanner-ring::after {
  content: '';
  position: absolute;
  inset: -2px;
  border: 2px solid transparent;
  border-top-color: #cfe2f3; /* 青色扫描线 */
  border-radius: 50%;
  animation: scan-rotate 2s linear infinite;
}

.scanner-dot {
  width: 6px;
  height: 6px;
  background: #cfe2f3;
  border-radius: 50%;
  box-shadow: 0 0 15px #cfe2f3;
  animation: pulse 1.5s infinite;
}

@keyframes scan-rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* --- 文字美化 --- */
.scanning-text {
  font-size: 0.9rem;
  font-weight: 500;
  color: #cfe2f3; /* 使用青色，看起来更有科技感 */
  letter-spacing: 2px;
  margin-top: 10px;
  text-shadow: 0 0 10px rgba(86, 241, 207, 0.3);
}

.scanning-sub {
  font-family: "Cascadia Code", monospace;
  font-size: 0.6rem;
  color: #444; /* 深灰色副标题 */
  letter-spacing: 1px;
}

/* 省略号动画 */
.dot-ani {
  display: inline-block;
  width: 1.5em;
  text-align: left;
  animation: dots 1.5s infinite;
}

@keyframes dots {
  0% { content: ''; }
  33% { content: '.'; }
  66% { content: '..'; }
  100% { content: '...'; }
}

.item-with-action { display: flex; flex-wrap: wrap; align-items: center; }
.card-info { flex: 1; }
.user-meta {
  margin-bottom: 8px;
  display: flex;
  gap: 10px;
}
.uid-tag {
  font-size: 0.7rem;
  background: rgba(0, 122, 255, 0.1);
  color: #cfe2f3;
  padding: 2px 6px;
  border-radius: 4px;
}

.console-row { display: flex; margin-bottom: 2px; font-size: 0.8rem; }
.console-label { width: 85px; color: #555; flex-shrink: 0; }
.gold-text { color: #f9e2af; font-weight: bold; }

/* 基础样式：透明背景，细边框 */
.manual-buy-btn {
  background: transparent;
  padding: 6px 14px;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
  cursor: pointer;
}

/* 1. 正常秒杀：亮橙色边框 */
.manual-buy-btn:not(:disabled) {
  color: #ff5722;
  border: 1.5px solid #ff5722;
}
.manual-buy-btn:not(:disabled):hover {
  background: rgba(255, 87, 34, 0.1);
  box-shadow: 0 0 8px rgba(255, 87, 34, 0.2);
}

/* 2. 展示中（公示中）：深灰蓝/钢青色边框 */
/* 注意：这里改用 isLocked 对应你的后端新逻辑 */
.manual-buy-btn.is-exhibiting {
  color: #78909c;
  border: 1.5px solid #b0bec5;
  cursor: not-allowed;
  opacity: 0.6;
}

/* 3. 已购买/已发送：清新绿色边框 */
.manual-buy-btn.is-purchased {
  color: #4caf50;
  border: 1.5px solid #4caf50;
  cursor: default;
  background: rgba(76, 175, 80, 0.05);
}

/* 4. CD冷却中：灰色虚线边框 (新增) */
.manual-buy-btn.is-cooling {
  color: #9e9e9e;
  border: 1.5px dashed #bdbdbd; /* 使用虚线表示“暂不可用” */
  cursor: wait;
  opacity: 0.5;
  animation: border-blink 1.3s infinite; /* 增加一个微弱的呼吸感 */
}

@keyframes border-blink {
  50% { opacity: 0.3; }
}


/* ========== 日常板块主布局 ========== */
/* 三列布局核心 */
/* 1. 确保三列顶部绝对平齐 */
.tasks-columns-container {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 15px;
  align-items: start; /* 👈 必须：防止列在垂直方向上居中对齐 */
}

/* 确保每一列的高度逻辑一致 */
.task-column {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid #1a1a1a;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  
  /* 让列的高度根据内容撑开，保持顶部平齐 */
  height: fit-content; 
}

.column-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 40px; /* 设定固定高度 */
  background: transparent; /* 关键：去掉背景色，防止出现色块不一致 */
  box-sizing: border-box; /* 确保 padding 不撑大盒子 */
  padding: 0 15px;
  background: rgba(255, 255, 255, 0.03);
  border-bottom: 1px solid #1a1a1a;
  font-size: 0.8rem;
  font-weight: bold;
  color: #007aff;
  gap: 8px;
}

.column-dot {
  width: 4px;
  height: 4px;
  background: #007aff;
  border-radius: 50%;
}

.column-body {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 缩小版的任务卡片 */
/* 2. 统一任务卡片的内部对齐 */
.task-card-mini {
  display: flex;
  flex-direction: column;
  padding: 12px;
  background: #0a0a0a;
  border: 1px solid #1a1a1a;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  min-height: 28px; /* 👈 建议：固定最小高度，防止描述文字行数不同导致不齐 */
}

.task-card-mini:hover {
  background: #111;
  border-color: #333;
}

.task-card-mini.is-selected {
  border-color: rgba(0, 122, 255, 0.5);
  background: rgba(0, 122, 255, 0.05);
}

.task-main {
  display: flex;
  align-items: center; /* 让图标和任务名垂直居中对齐 */
  gap: 0; /* 我们用 padding/width 控制间距 */
}

.task-check {
  width: 28px;      /* 👈 关键：给图标一个固定的宽度空间 */
  flex-shrink: 0;   /* 👈 防止被挤压 */
  display: flex;
  justify-content: flex-start; /* 图标靠左 */
  font-size: 0.9rem;
  color: #444;
}
.is-selected .task-check { color: #007aff; }

.task-info {
  flex: 1;          /* 占据剩余所有空间 */
  display: flex;
  flex-direction: column;
  justify-content: center;
  text-align: left; /* 👈 确保文字强制左对齐 */
}

/* 4. 微调文字 */
.task-name {
  font-size: 0.85rem;
  font-weight: 500;
  line-height: 1.2;
}

.task-desc {
  font-size: 0.7rem;
  color: #555;
  margin-top: 2px;
  padding-left: 0; /* 之前如果加了左内边距请去掉 */
}

/* 1. 彻底移除“资源提取”上方的分割线 */
.modal-body .config-section-wrapper:first-of-type {
  border-top: none !important;
  padding-top: 10px; /* 顶部留一点呼吸空间，但不要线 */
}

/* 2. 确保 modal-header 的线足够淡，不突兀 */
.modal-header {
  padding: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05) !important; /* 使用极淡的颜色 */
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* 3. 如果线还在，那是 config-group-title 可能自带了样式，强制去掉 */
.config-group-title {
  border-top: none !important;
  margin-top: 5px; /* 适当调小间距 */
  font-size: 0.75rem;
  color: #007aff;
  font-weight: bold;
  margin-bottom: 10px;
}

/* 4. 只有在第二个区域（副本通关设置）上方才保留分割感 */
.modal-body .config-section-wrapper:nth-of-type(2) {
  margin-top: 15px;
  padding-top: 15px;
  border-top: 1px solid rgba(255, 255, 255, 0.05); /* 这里留一根淡淡的线做区分 */
}

.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.85);
  backdrop-filter: blur(8px); display: flex; align-items: center;
  justify-content: center; z-index: 1000;
}

.modal-card {
  background: #0f0f0f;
  border: 1px solid #333;
  width: 380px; /* 稍微加宽一点点 */
  border-radius: 20px;
  box-shadow: 0 30px 60px rgba(0,0,0,0.8);
  display: flex;          /* 新增 */
  flex-direction: column; /* 新增 */
  max-height: 80vh;       /* 限制最大高度，防止撑破屏幕 */
  overflow: hidden;       /* 确保 footer 圆角不被切断 */
}

/* 核心：让 Body 承载滚动，同时保持整洁 */
.modal-body {
  flex: 1;                /* 占据剩余空间 */
  overflow-y: auto;       /* 开启滚动 */
  padding: 15px 20px;     /* 舒适的内边距 */
  scrollbar-width: thin;  /* 火狐滚动条细化 */
  scrollbar-color: #333 transparent;
}

.modal-main-title { font-size: 1rem; font-weight: bold; color: #fff; display: block; }
.modal-sub-title { font-size: 0.7rem; color: #555; text-transform: uppercase; }

.config-group-title { font-size: 0.75rem; color: #007aff; font-weight: bold; margin-bottom: 10px; }

/* === 弹窗内：资源提取行样式微调 === */
.config-row-inline {
  display: flex;
  align-items: center;
  gap: 20px;          /* 两个复选框之间的间距 */
  box-sizing: border-box; /* 确保内边距不撑开元素宽度 */
}

/* 如果你想让“副本通关设置”下面的网格也稍微往右一点，保持视觉对齐 */
.dungeon-grid {
  box-sizing: border-box;
}

/* 确保资源提取行的单个项目样式正确 */
.config-row-inline .custom-check {
  display: flex;
  align-items: center;
  gap: 8px; /* 复选框与文字之间的微小间距 */
  font-size: 0.85rem;
  color: #ccc;
  cursor: pointer;
}

/* 🚀 统一样式：让所有弹窗的二级配置内容都有相同的缩进 */
.config-content-indent {
  padding-left: 18px; /* 统一缩进距离 */
  margin-bottom: 10px;
}

/* 移除之前可能冲突的特定缩进 */
.config-row-inline, .dungeon-grid {
  padding-left: 0 !important; 
}

/* 确保所有配置组标题（蓝字部分）对齐弹窗左边缘 */
.config-group-title {
  font-size: 0.75rem;
  color: #007aff;
  font-weight: bold;
  margin-bottom: 10px;
  padding-left: 0; /* 标题不缩进，内容缩进，形成层次 */
}

.custom-check { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; cursor: pointer; color: #ccc; }
.custom-check input { accent-color: #007aff; }

.input-stepper input, .small-num-input {
  width: 45px; background: #000; border: 1px solid #333; color: #007aff;
  text-align: center; border-radius: 4px; font-size: 0.8rem;
}

/* 针对数字输入框的深度定制 */
.mini-num {
  width: 60px;           /* 宽度适中 */
  height: 24px;
  background: #0a0a0a;   /* 极深背景，告别纯白 */
  border: 1px solid #333; /* 初始深灰色边框 */
  color: #007aff;        /* 科技蓝文字 */
  text-align: center;
  border-radius: 4px;
  font-family: 'Consolas', monospace; /* 程序员风格字体 */
  font-size: 0.85rem;
  margin: 0 4px;
  transition: all 0.3s ease;
  outline: none;
}

/* 鼠标悬停或点击输入时的效果 */
.mini-num:hover, .mini-num:focus {
  border-color: #007aff; /* 边框变蓝 */
  background: #111;      /* 背景微亮 */
  box-shadow: 0 0 8px rgba(0, 122, 255, 0.3); /* 蓝色呼吸灯发光感 */
}

/* 移除 Chrome/Safari/Edge 默认的上下调节箭头（让界面更干净） */
.mini-num::-webkit-outer-spin-button,
.mini-num::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* 输入框旁边的文字（如“关卡:”、“次数:”）样式 */
.input-group {
  display: flex;
  align-items: center;
  font-size: 0.75rem;
  color: #666; /* 调暗标签文字，突出输入框 */
  gap: 2px;
}

.promo-code-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dungeon-header {
  display: flex;
  align-items: center;
  /* 确保垂直居中 */
  height: 32px; 
}

.promo-input-box {
  /* 关键修改：往右移动 25 像素，你可以根据需要调整这个值 */
  margin-left: 25px; 
  
  /* 宽度和高度 */
  width: 160px;
  height: 24px;
  
  /* 视觉样式 */
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 4px;
  padding: 0 8px;
  color: #00ffcc; /* 换成一个显眼的亮色，方便确认输入内容 */
  font-size: 12px;
  outline: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.promo-input-box:focus {
  border-color: #007aff;
  background: rgba(0, 122, 255, 0.1);
  width: 180px; /* 获取焦点时稍微拉长一点，增强交互感 */
}

/* 兼容性处理：让 checkbox 的文字不要换行 */
.custom-check {
  white-space: nowrap;
  display: flex;
  align-items: center;
}

.promo-input {
  width: 150px;
  margin-left: 20px;
  margin-right: 20px;
  background: rgba(255, 255, 255, 0.05);
  border: 2px solid rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  padding: 4px 10px;
  color: #fff;
  font-size: 12px;
  outline: none;
  transition: all 0.2s;
}

/* 1. 抹除原生外观 */
.custom-check input[type="checkbox"] {
  appearance: none;
  -webkit-appearance: none;
  width: 16px;
  height: 16px;
  background: #0a0a0a; /* 深色背景 */
  border: 1px solid #444; /* 深灰边框 */
  border-radius: 3px;
  cursor: pointer;
  position: relative;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0;
}

/* 2. 悬停效果：边框变蓝 */
.custom-check input[type="checkbox"]:hover {
  border-color: #007aff;
  box-shadow: 0 0 5px rgba(0, 122, 255, 0.2);
}

/* 3. 选中状态：背景变蓝，边框发光 */
.custom-check input[type="checkbox"]:checked {
  background: #007aff;
  border-color: #007aff;
  box-shadow: 0 0 10px rgba(0, 122, 255, 0.4);
}

/* 4. 绘制勾选的那一“撇” (使用伪元素) */
.custom-check input[type="checkbox"]:checked::after {
  content: '';
  width: 4px;
  height: 8px;
  border: solid white;
  border-width: 0 2px 2px 0; /* 绘制 L 型 */
  transform: rotate(45deg); /* 旋转 45 度变成勾 */
  position: absolute;
  top: 1px;
}

/* 5. 文字标签样式微调，让文字和 Checkbox 对齐更完美 */
.custom-check {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
  font-size: 0.85rem;
  color: #ccc;
}

.custom-check:hover span {
  color: #fff; /* 悬停时文字稍微加亮 */
}

/* 1. 增加每一项副本配置的上下间距 */
.dungeon-config-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  
  /* 🚀 关键修改：增加垂直填充，让文字和输入框不挤在一起 */
  padding: 12px 0; 
  
  /* 可选：增加一个非常淡的底边分割线，让每一行更清晰 */
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: background 0.2s;
}

/* 2. 移除最后一项的底边线，保持整洁 */
.dungeon-config-item:last-child {
  border-bottom: none;
}

/* 3. 鼠标悬停在这一行时，给一点背景高亮（交互感） */
.dungeon-config-item:hover {
  background: rgba(0, 122, 255, 0.03);
}

/* 4. 确保内部输入框组有足够的左间距 */
.input-group {
  display: flex;
  align-items: center;
  gap: 12px; /* 关卡和次数之间的间距 */
  margin-left: 10px;
}

/* 5. 刚才提到的副本选项二级缩进容器（确保已经在 HTML 里套了此类名） */
.config-content-indent {
  padding-left: 18px;
  margin-top: 10px; /* 与上方标题“副本通关设置”拉开距离 */
}

.modal-footer { display: grid; grid-template-columns: 1fr 1fr; border-top: 1px solid #1a1a1a; }
.modal-btn { padding: 15px; border: none; cursor: pointer; font-weight: bold; font-size: 0.85rem; }
.modal-btn.secondary { background: transparent; color: #555; border-right: 1px solid #1a1a1a; }
.modal-btn.primary { background: transparent; color: #007aff; }
.modal-btn:hover { background: rgba(255,255,255,0.03); }

/* ========== 授权覆盖层 (Auth) ========== */
.auth-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.9); backdrop-filter: blur(10px);
  display: flex; align-items: center; justify-content: center; z-index: 9999;
}
.auth-card {
  background: #111; border: 1px solid #222; padding: 40px;
  border-radius: 20px; text-align: center; width: 380px;
}
.auth-icon { font-size: 3rem; margin-bottom: 20px; }
.machine-id-box {
  background: #000; padding: 15px; border-radius: 10px;
  margin: 20px 0; border: 1px dashed #333; cursor: pointer;
}
.code { color: #007aff; font-family: monospace; word-break: break-all; font-size: 0.75rem; }

/* 通用细节 */
@keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.3; } 100% { opacity: 1; } }
::-webkit-scrollbar { width: 6px; }
::-webkit-scrollbar-thumb { background: #222; border-radius: 10px; }
</style>