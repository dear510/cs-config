<template>
  <div class="app-container">
    <aside class="sidebar">
      <div class="brand">
        <span class="logo">🎮</span>
        <div class="brand-text">
          <h2>次神助手</h2>
          <small>v2.6.1 </small>
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
          @click="userLevel >= 1 ? activeTab = 'zhenbao' : alert('❌ 该功能需要等级1及以上权限！')"
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
              <input 
                v-model="searchKeyword" 
                class="dark-input" 
                placeholder="输入关键词" 
                @click.stop="showDropdown = !showDropdown" 
                @input="showDropdown = true"
              >
              <div v-if="showDropdown && filteredAttributes.length > 0" class="search-dropdown" @click.stop>
                <div 
                  v-for="name in filteredAttributes" 
                  :key="name" 
                  @click="selectAttribute(name)" 
                  class="dropdown-item"
                >
                  {{ name }}
                </div>
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
      <header class="status-bar">
        <div class="status-control" @click="toggleMonitor" :class="{ 'is-paused': !isCapturing }">
          <span class="dot" :class="{ pulsing: isCapturing }"></span>
          <span class="status-text">{{ isCapturing ? '正在监控系统' : '监控已停止' }}</span>
          <small class="port-hint">PORT: 2025</small>
        </div>
        <div class="header-actions">
          <!-- 日常任务页面的操作按钮 -->
          <template v-if="activeTab === 'daily'">
            <button class="clear-btn" @click="dailyLogs = []">🗑️ 清空日志</button>
          </template>
          
          <!-- 珍宝拦截页面的操作按钮 -->
          <template v-else-if="activeTab === 'zhenbao'">
            <span class="stats-label">数据流</span>
            <span class="stats-count">{{ zhenbaoLogs.length }}</span>
            <button class="clear-btn" @click="zhenbaoLogs = []; hitLogs = []; sysLogs = [];">🗑️ 清空记录</button>
          </template>
        </div>
      </header>
      
      <section v-if="activeTab === 'daily'" class="page-view animate-fade">
        <div class="daily-layout">
          <div class="tasks-columns-container">
            <div v-for="category in taskCategories" :key="category.id" class="task-column">
              
              <div class="column-header" style="display: flex; justify-content: space-between; align-items: center; width: 100%; background: transparent !important; padding: 0 8px; box-sizing: border-box; border-bottom: 1px solid rgba(255,255,255,0.05); height: 40px;">
                <div style="display: flex; align-items: center; gap: 6px;">
                  <span class="column-dot"></span>
                  <span style="font-weight: bold; font-size: 13px; color: #007aff;">{{ category.title }}</span>
                  <!-- 显示该列所需的权限等级 -->
                  <span v-if="getColumnRequiredLevel(category.id) > userLevel" class="column-level-tag">
                    需等级{{ getColumnRequiredLevel(category.id) }}
                  </span>
                </div>

                <div v-if="category.id === 'garden'" class="garden-afk-wrapper" @click.stop>
                  <label v-if="currentConfigTask && currentConfigTask.config" class="custom-check mini-afk-label">
                    <input 
                      type="checkbox" 
                      v-model="currentConfigTask.config.gardenAFK" 
                      @change="handleGardenAFKChange"
                      :disabled="userLevel < 3"
                    >
                    <span class="afk-text">自动挂机</span>
                  </label>
                </div>
              </div>
              
              <div class="column-body">
                <div 
                  v-for="task in category.tasks" 
                  :key="task.id" 
                  :class="['task-card-mini', task.selected ? 'is-selected' : '', userLevel < getColumnRequiredLevel(category.id) ? 'locked-task' : '']"
                  @click="userLevel >= getColumnRequiredLevel(category.id) ? openConfig(task) : alert(`❌ 该功能需要等级${getColumnRequiredLevel(category.id)}及以上权限！`)"
                >
                  <div class="task-main">
                    <div class="task-check">
                      <span v-if="userLevel < getColumnRequiredLevel(category.id)">🔒</span>
                      <span v-else>{{ task.selected ? '●' : '○' }}</span>
                    </div>
                    <div class="task-info">
                      <div class="task-name" :class="{ 'locked-text': userLevel < getColumnRequiredLevel(category.id) }">
                        {{ task.name }}
                      </div>
                      <div class="task-desc" :class="{ 'locked-text': userLevel < getColumnRequiredLevel(category.id) }">
                        {{ task.desc }}
                      </div>
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

            <div class="radar-wrapper" style="display: flex; gap: 12px; align-items: center;">
              
              <div class="radar-item">
                <transition name="pop-up">
                  <div v-if="isRadarPanelOpen" class="radar-popover">
                    <div class="popover-header">
                      <span>📡 扫菜情报</span>
                      <span class="close-mini" @click.stop="isRadarPanelOpen = false">×</span>
                    </div>
                    <div class="popover-body">
                      <!-- 修改这里：保持原有结构，只调整时间样式 -->
                      <div v-for="(item, index) in scanResults" :key="item.uid + item.pos" class="mini-item">
                        <div class="item-header">
                          <div class="mini-info">
                            <span class="m-name">{{ item.name }}</span>
                            <!-- 时间放到右边 -->
                            <span class="m-time">{{ item.fixedTime }}</span>
                          </div>
                          <button class="delete-item-btn" @click.stop="removeScanItem(index)" title="删除此条">×</button>
                        </div>
                        <div class="m-desc">UID: {{ item.uid }} | 昵称: {{ item.nickname }}</div>
                      </div>
                      
                      <div v-if="scanResults.length === 0" class="mini-empty">🌱 暂无变异情报...</div>

                      <!-- 推送区域 - 只在有数据时显示 -->
                      <div v-if="scanResults.length > 0" class="push-section">
                        <div class="push-header">
                          <span class="push-icon">📱</span>
                          <span class="push-title">微信推送</span>
                        </div>
                        
                        <!-- 群组选择 - 美化版本 -->
                        <div class="group-selector-modern">
                          <div class="select-wrapper">
                            <select v-model="selectedPushGroup" class="modern-select">
                              <option value="cs901linda">🌍 901-920区</option>
                              <option value="cs921linda">🌏 921-940区</option>
                              <option value="cs941linda">🌎 941-960区</option>
                              <option value="cs900linda">🌍 961-980区</option>
                              <option value="cs981linda">🌏 981-1000区</option>
                            </select>
                            <div class="select-arrow-modern">▼</div>
                          </div>
                        </div>
                        
                        <!-- 推送按钮 -->
                        <button class="push-btn" @click.stop="pushToWeChat" :disabled="isPushing">
                          <span v-if="!isPushing">📤 发送</span>
                          <span v-else>⏳ 推送中... ({{ pushProgress }})</span>
                        </button>
                        
                        <!-- 推送结果提示 -->
                        <div v-if="pushResult" class="push-result" :class="{ 'success': pushSuccess, 'error': !pushSuccess }">
                          {{ pushResult }}
                        </div>
                      </div>
                    </div>
                  </div>
                </transition>
                <div 
                  class="radar-toggle-btn" 
                  :class="{ 'pulse': isScanningVeggie }"
                  @click.stop="isRadarPanelOpen = !isRadarPanelOpen; isPKPanelOpen = false"
                  title="查看巡逻情报"
                >
                  🌱
                </div>
              </div>

              <div class="pk-wrapper">
                <transition name="pop-up">
                  <div v-if="isPKPanelOpen" class="radar-popover pk-popover">
                      <div class="popover-header">
                          <div style="display: flex; align-items: center; gap: 8px;">
                              <span>🥊 跨服切磋配置</span>
                              <button class="stop-pk-btn" v-if="isPKEnabled" @click="stopPKReplacement">🛑 停止</button>
                          </div>
                          <span class="close-mini" @click.stop="isPKPanelOpen = false">×</span>
                      </div>

                      <div class="popover-body">
                          <div class="pk-search-bar">
                              <input v-model="pkSearchQuery" placeholder="🔍 搜索云端名单..." class="pk-input" />
                          </div>
                          
                          <div class="pk-list-scroll">
                              <div v-for="user in filteredPKList" :key="user.uid" class="pk-mini-item">
                                  <div class="pk-user-info">
                                      <span class="pk-region-tag">{{ user.region }}</span>
                                      <span class="pk-user-name">{{ user.name }}</span>
                                  </div>
                                  <div class="pk-item-actions">
                                      <button class="mini-tag-btn target" @click.stop="setAsTarget(user)">定为目标</button>
                                      <button class="mini-tag-btn source" @click.stop="setAsSource(user)">定为源</button>
                                  </div>
                              </div>
                              <div v-if="filteredPKList.length === 0" class="mini-empty">未找到匹配人员</div>
                          </div>

                          <div class="pk-footer-form">
                              <div class="input-section">
                                  <label>目标 (切磋对象):</label>
                                  <div class="dual-input">
                                      <input v-model="targetUser.uid" placeholder="UID" class="pk-input-s" />
                                      <input v-model="targetUser.server" placeholder="区服" class="pk-input-xs" />
                                  </div>
                              </div>

                              <div class="pk-divider">⇄</div>

                              <div class="input-section">
                                  <label>源 (好友列表替换对象):</label>
                                  <div class="dual-input">
                                      <input v-model="sourceUser.uid" placeholder="UID" class="pk-input-s" />
                                      <input v-model="sourceUser.server" placeholder="区服" class="pk-input-xs" />
                                  </div>
                              </div>

                              <button class="pk-confirm-btn" :class="{ 'btn-active': isPKEnabled }" @click="applyPK">
                                  {{ isPKEnabled ? '🔄 更新替换规则' : '🔥 开启拦截替换' }}
                              </button>
                          </div>
                      </div>
                  </div>
                </transition>
                
                <div 
                  class="radar-toggle-btn pk-action-btn" 
                  :class="{ 'pk-active': isPKEnabled, 'locked': userLevel < 1 }"
                  @click.stop="userLevel >= 1 ? togglePKPanel() : alert('❌ 该功能需要等级1及以上权限！')" 
                  title="跨服切磋"
                >
                  🥊
                </div>
              </div>

            </div>
          </div>
        </div>
      </section>

      <section v-if="activeTab === 'zhenbao'" class="page-view animate-fade">
        <!-- 权限判断 -->
        <div v-if="userLevel < 1" class="permission-denied">
          <span class="lock-icon">🔒</span>
          <h3>权限不足</h3>
          <p>珍宝拦截需要等级1及以上权限</p>
          <p class="current-level">当前等级: {{ userLevel }}</p>
        </div>

        <template v-else>
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
        </template>
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

                  <div v-if="currentConfigTask.config.scanVeggie" class="veggie-select-container" style="position: relative; margin-right: 20px;">
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
                    <button class="action-btn" @click="openTargetModal">
                      <template v-if="currentConfigTask.config.selectedUids?.length > 0">
                        ⚙️ 已选 ({{ currentConfigTask.config.selectedUids.length }}) 人
                      </template>
                      <template v-else>
                        ⚙️ 配置目标
                      </template>
                    </button>  
                  </div>
                </div>

                <div class="dungeon-config-item" style="display: flex; align-items: center; margin-top: 8px;">
                  <label class="custom-check" style="margin-left: 0;">
                    <input 
                      type="checkbox" 
                      v-model="currentConfigTask.config.stealVeggie"
                    > 
                    <span>偷菜 (需配合自动挂机)</span>
                  </label>
                  
                  <!-- 如果偷菜被选中但没有扫菜情报时的提示 -->
                  <span 
                    v-if="currentConfigTask.config.stealVeggie && scanResults.length === 0" 
                    class="warning-tip"
                    style="margin-left: 12px; font-size: 11px; color: #ff9500;"
                  >
                    ⚠️ 暂无扫菜情报
                  </span>
                  
                  <!-- 如果偷菜被选中且有扫菜情报时的提示 -->
                  <span 
                    v-if="currentConfigTask.config.stealVeggie && scanResults.length > 0" 
                    class="success-tip"
                    style="margin-left: 12px; font-size: 11px; color: #2ecc71;"
                  >
                    ✅ {{ scanResults.length }} 个目标待偷
                  </span>
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
                      <span>召唤2w抽各档奖励</span>
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

              <div class="config-group-title" style="margin-top:15px;" v-if="userLevel >= 5">奖励领取</div>
              <div class="config-content-indent" v-if="userLevel >= 5">
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
                      <span>(敬请期待)</span>
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

    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showTargetModal" class="modal-overlay" @click.self="showTargetModal = false">
          <div class="modal-content target-modal-container">
              <div class="modal-header">
                  <h3>🎯 目标名单 (已选: {{ currentConfigTask.config.selectedUids?.length || 0 }})</h3>
                  <button class="close-x" @click="showTargetModal = false">×</button>
              </div>

              <div class="modal-filter-row">
                  <div class="custom-select-wrapper">
                      <select v-model="selectedRegion" class="dark-select">
                          <option v-for="(list, region) in remoteTargets.all_data" :key="region" :value="region">
                              {{ region }}
                          </option>
                      </select>
                      <span class="select-arrow">▼</span>
                  </div>

                  <div class="search-wrapper">
                      <span class="search-icon">🔍</span>
                      <input 
                          v-model="searchQuery" 
                          class="dark-input search-input" 
                          placeholder="搜索昵称或 UID..."
                      >
                  </div>

                  <button class="action-btn toggle-btn" @click="toggleAllFiltered">
                      <span class="btn-icon">🔘</span>
                      全选/反选
                  </button>
              </div>

              <div class="target-list-scroll">
                  <div v-if="isLoading" class="list-status">载入云端名单中...</div>
                  
                  <div 
                      v-for="user in filteredList" 
                      :key="user.uid" 
                      class="target-item"
                      :class="{ 'is-selected': currentConfigTask.config.selectedUids?.includes(user.uid) }"
                      @click="toggleTarget(user.uid)"
                  >
                      <div class="checkbox-box">
                          <div v-if="currentConfigTask.config.selectedUids?.includes(user.uid)" class="inner-dot"></div>
                      </div>
                      
                      <span class="t-info">
                          {{ user.name }} <span class="t-uid-dim">({{ user.uid }})</span>
                      </span>
                  </div>

                  <div v-if="!isLoading && filteredList.length === 0" class="list-status">未找到匹配目标</div>
              </div>

              <div class="modal-footer">
                  <button class="save-btn" @click="showTargetModal = false">完成配置</button>
              </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <transition name="fade">
      <div 
        v-if="isRadarPanelOpen" 
        class="radar-overlay" 
        @click="isRadarPanelOpen = false"
      ></div>
    </transition>

    <div v-if="showUpdateModal" class="auth-overlay">
      <div class="auth-card" style="max-width: 480px;">
        <div class="auth-header">
          <span class="auth-icon">⚙️</span>
          <h3>v2.6 更新说明</h3>
        </div>
        <div class="auth-body" style="text-align: left; max-height: 480px; overflow-y: auto; font-size: 13px; line-height: 1.6;">
          
          <div style="background: rgba(255, 149, 0, 0.1); border: 1px solid #ff9500; border-radius: 6px; padding: 10px; margin-bottom: 15px;">
            <p style="color: #ff9500; font-weight: bold; margin-bottom: 5px;">⚠️ 珍宝拦截重要提醒：</p>
            <p style="color: #eee; margin: 0; font-size: 12px;">因拦截效率极高，启动前请务必确认生效规则。规则错误可能导致橙钻瞬间清空，请谨慎操作！</p>
          </div>

          <p style="color: #007aff; font-weight: bold; margin: 10px 0 5px 0;">【竞技板块 · 跨服切磋 (Beta)】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>⚔️ <b>跨服切磋</b>：支持跨服挑战任意玩家。</li>
            <li style="background: rgba(0, 122, 255, 0.1); padding: 8px; border-radius: 4px; margin: 5px 0; font-size: 12px; border-left: 3px solid #007aff;">
              <b>操作指南：</b><br>
              1. 在模块中输入【目标玩家UID】及【所属区服】。<br>
              2. 在好友列表中选定一位【源替换好友】。<br>
              3. 启动任务后，点开好友列表点击该好友，即可定向切磋目标玩家。
              4. 目前云端名单仅更新至1-2200区, 之后大区的id信息作者将尽快完善。
            </li>
          </ul>

          <p style="color: #34c759; font-weight: bold; margin: 15px 0 5px 0;">【情报模块 · 全服扫菜】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>🔍 <b>精准定位</b>：支持自定义区服名单及目标菜品进行全量扫描。</li>
            <li>🌱 <b>实时展示</b>：扫描结果将实时汇总于右下角【嫩芽】图标，支持一键复制。</li>
            <li style="color: #ff3b30; font-size: 12px; margin-top: 5px;">
              <b>⚠️ 风控提醒：</b> 目前仅支持【单次执行】，不支持自动挂机。建议扫描间隔 <b>10分钟以上</b>，严禁高频访问以防触发次神检测警告。
            </li>
          </ul>

          <p style="color: #af52de; font-weight: bold; margin: 15px 0 5px 0;">【未来规划】</p>
          <ul style="list-style: none; padding-left: 5px; color: #ccc;">
            <li>🍎 <b>全自动偷菜</b>：功能开发中，敬请期待...</li>
          </ul>

        </div>
        <button class="primary-btn" @click="showUpdateModal = false" style="margin-top: 20px; width: 100%;">确认进入助手</button>
      </div>
    </div>

    <!-- 修改授权弹窗部分，移除CDK输入 -->
    <div v-if="!isAuthorized" class="auth-overlay">
      <div class="auth-card">
        <div class="auth-icon">🔒</div>
        <h2>系统未授权</h2>
        
        <!-- 显示MD5加密后的机器码 -->
        <div class="machine-id-box" @click="copyMachineID">
          <span class="label">机器码:</span>
          <code class="code">{{ hashedMachineID || '计算中...' }}</code>
        </div>
        
        <div class="auth-tips">
          <p>请将上面的机器码发送给管理员开通权限</p>
          <p style="font-size: 11px; color: #666; margin-top: 5px;">点击机器码即可复制</p>
        </div>
        
        <button class="primary-btn" @click="checkAuth" style="margin-top: 20px;">
          🔄 重新验证
        </button>
      </div>
    </div>

    <!-- 证书安装引导弹窗 -->
    <Transition name="fade">
      <div v-if="showCertGuide && isAuthorized" class="auth-overlay" @click.self="showCertGuide = false">
        <div class="auth-card" style="max-width: 500px;">
          <div class="auth-icon">🔐</div>
          <h2>首次使用配置</h2>
          
          <div v-if="!certInstallResult" class="cert-guide-content">
            <p style="color: #ccc; margin-bottom: 20px; font-size: 14px;">
              次神助手需要通过代理证书拦截游戏数据包，这是正常的安全配置。
            </p>
            
            <div style="background: #1a1a1a; border-radius: 10px; padding: 15px; margin-bottom: 20px; text-align: left;">
              <p style="color: #ffaa00; margin-bottom: 8px;">⚙️ 自动安装（推荐）</p>
              <p style="color: #888; font-size: 13px; margin-bottom: 15px;">点击下方按钮自动下载并安装证书</p>
              
              <button 
                class="primary-btn" 
                @click="installCertificate" 
                :disabled="certInstalling"
                style="margin-top: 0;"
              >
                {{ certInstalling ? '⏳ 安装中...' : '📥 一键安装证书' }}
              </button>
            </div>
            
            <div style="border-top: 1px solid #222; padding-top: 15px;">
              <p style="color: #666; font-size: 12px; margin-bottom: 10px;">如果自动安装失败，请手动操作：</p>
              <button class="modal-btn secondary" @click="showManualGuide = !showManualGuide" style="width: 100%; padding: 8px;">
                {{ showManualGuide ? '👆 隐藏指南' : '📖 查看手动安装指南' }}
              </button>
              
              <div v-if="showManualGuide" style="margin-top: 15px; text-align: left; background: #0a0a0a; padding: 15px; border-radius: 8px;">
                <p style="color: #007aff; margin-bottom: 8px; font-weight: bold;">Windows 手动安装：</p>
                <ol style="color: #ccc; font-size: 12px; padding-left: 20px;">
                  <li>打开浏览器访问 <code style="background: #000; padding: 2px 5px;">http://127.0.0.1:2025</code></li>
                  <li>点击下载证书 (CA certificate)</li>
                  <li>双击下载的 .der 文件</li>
                  <li>选择“安装证书” → “本地计算机” → “受信任的根证书颁发机构”</li>
                  <li>重启电脑</li>
                </ol>
                
                <p style="color: #007aff; margin: 15px 0 8px; font-weight: bold;">Mac 手动安装：</p>
                <ol style="color: #ccc; font-size: 12px; padding-left: 20px;">
                  <li>打开浏览器访问 <code style="background: #000; padding: 2px 5px;">http://127.0.0.1:2025</code></li>
                  <li>下载证书文件</li>
                  <li>打开“钥匙串访问”应用</li>
                  <li>将证书拖入“系统”钥匙串</li>
                  <li>双击证书，展开“信任”，选择“始终信任”</li>
                  <li>重启电脑</li>
                </ol>
              </div>
            </div>
            
            <div v-if="certError" style="margin-top: 20px; color: #ff4757; font-size: 13px; background: rgba(255,71,87,0.1); padding: 10px; border-radius: 6px;">
              ❌ {{ certError }}
            </div>
          </div>
          
          <!-- 安装成功界面 -->
          <div v-else class="cert-success">
            <div style="font-size: 48px; color: #2ed573; margin-bottom: 15px;">✓</div>
            <h3 style="color: #2ed573;">安装成功！</h3>
            <p style="color: #ccc; margin: 15px 0;">{{ certInstallResult.message }}</p>
            
            <div v-if="certInstallResult.needRestart" style="display: flex; gap: 10px; margin-top: 20px;">
              <button class="modal-btn secondary" @click="showCertGuide = false" style="flex: 1;">稍后重启</button>
              <button class="primary-btn" @click="window.go.main.App.RestartComputer()" style="flex: 1;">立即重启</button>
            </div>
            <button v-else class="primary-btn" @click="showCertGuide = false" style="margin-top: 20px;">开始使用</button>
          </div>
          
          <div style="margin-top: 20px; font-size: 11px; color: #555;">
            关闭监控后会自动清除代理设置，不影响正常上网
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch, computed, nextTick, onUnmounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime'
import { 
  UpdateRules, GetAttributeNames, GetCategoryNames, 
  ToggleCapture, ManualBuy, CheckCurrentAuth, GetMachineID, GetHashedMachineID,
  // 假设你在 app.go 中新增了下面这个函数
  ExecuteDailyTasks, GetRemoteTargets
} from '../wailsjs/go/main/App'

// --- 导航与 UI 状态 ---
const activeTab = ref('daily') // 默认显示日常任务
const isAuthorized = ref(false)
const machineID = ref('')
const hashedMachineID = ref('')

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
  // --- 时间定时器逻辑 ---
  updateBeijingTime()
  timer = setInterval(updateBeijingTime, 1000)

  // --- 下拉框全局点击逻辑 ---
  // 监听全局点击，实现点击空白处关闭下拉框
  window.addEventListener('click', closeDropdown)
})

onUnmounted(() => {
  // --- 清除定时器 ---
  if (timer) clearInterval(timer)

  // --- 移除全局点击监听 ---
  // 这一步非常重要！如果不移除，每次切换页面都会多出一个监听器，导致程序越来越卡
  window.removeEventListener('click', closeDropdown)
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
        desc: '钥匙/稿子&各副本通关', 
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
        desc: '收菜/种菜/浇水/除虫/扫菜/偷菜', 
        selected: false,
        config: {
          collectVeggie: true,
          plantVeggie: false,
          buySeeds: true,
          veggieType: 'baicai',
          scanVeggie: false,
          stealVeggie: false,
          interestedVeggies: [],
          selectedUids: []
        }
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

// 通用的初始化云端数据函数
const initCloudData = async () => {
    // 如果已经有数据了，就不重复加载
    if (Object.keys(remoteTargets.value.all_data).length > 0) return;
    
    isLoading.value = true;
    try {
        const res = await GetRemoteTargets();
        remoteTargets.value = res;
        
        // 建立 UID -> Name 映射表 (保留你原来的逻辑)
        Object.values(res.all_data).forEach(regionList => {
            regionList.forEach(user => {
                nameMap[user.uid] = user.name;
            });
        });
        
        // 设置默认大区
        const regions = Object.keys(res.all_data);
        if (regions.length > 0) selectedRegion.value = regions[0];
        
        console.log("✅ 云端数据加载成功，PK模块与扫菜模块共享就绪");
    } catch (e) {
        console.error("加载云端名单失败:", e);
    } finally {
        isLoading.value = false;
    }
};

// --- 扫菜配置相关变量 ---
const nameMap = {}; // 🌟 新增：普通对象即可，用于快速查找昵称
const remoteTargets = ref({ all_data: {} })
const selectedRegion = ref('961-980') // 默认大区
const searchQuery = ref('')
const isLoading = ref(false)
const showTargetModal = ref(false)
const remoteVeggies = ref({});
// 存储扫描结果的列表
const scanResults = ref([])
const isRadarPanelOpen = ref(false);

// 添加删除单个条目的方法
const removeScanItem = (index) => {
  scanResults.value.splice(index, 1)
  
  // 删除后同步更新后端
  if (scanResults.value.length > 0) {
    syncStealTargetsToBackend();
  } else {
    // 如果全部删除了，也同步空数组
    window.go.main.App.UpdateStealTargets([]);
  }
};

// 获取已选作物的数量（用于界面显示）
const selectedVeggieNames = computed(() => {
  const ids = currentConfigTask.value?.config?.interestedVeggies || []
  if (ids.length === 0) return "选择作物 (多选)"
  
  // 返回格式：已选 (3) 个
  return `已选 (${ids.length}) 个`
})

// 切换感兴趣的作物（多选）
const toggleInterestedVeggie = (key) => {
  // 1. 安全检查：确保 currentConfigTask.value 存在
  if (!currentConfigTask.value) return;

  // 2. 安全检查：确保 config 存在
  if (!currentConfigTask.value.config) {
    currentConfigTask.value.config = {};
  }

  // 3. 安全初始化数组
  if (!currentConfigTask.value.config.interestedVeggies) {
    currentConfigTask.value.config.interestedVeggies = [];
  }

  const arr = currentConfigTask.value.config.interestedVeggies;
  const index = arr.indexOf(key);

  if (index > -1) {
    arr.splice(index, 1);
  } else {
    arr.push(key);
  }

  // 4. 强制触发响应式更新
  currentConfigTask.value.config.interestedVeggies = [...arr];
};

// 微信推送相关变量
const selectedPushGroup = ref('cs900linda') // 默认选择通用群组
const isPushing = ref(false)
const pushProgress = ref('')
const pushResult = ref('')
const pushSuccess = ref(false)

// 微信推送扫菜结果
const pushToWeChat = async () => {
  if (scanResults.value.length === 0) {
    pushResult.value = '没有可推送的扫菜结果'
    pushSuccess.value = false
    return
  }

  isPushing.value = true
  pushProgress.value = '准备中...'
  pushResult.value = ''
  
  try {
    // 调用后端推送方法，传入扫菜结果和选择的群组
    const result = await window.go.main.App.PushVeggieResultsToWeChat(
      scanResults.value, 
      selectedPushGroup.value
    )
    
    if (result.success) {
      pushSuccess.value = true
      pushResult.value = `✅ 推送成功！共 ${result.count} 条记录`
      // 推送成功后也同步一次（确保数据一致性）
      await syncStealTargetsToBackend();
    } else {
      pushSuccess.value = false
      pushResult.value = `❌ 推送失败：${result.message || '未知错误'}`
    }
  } catch (err) {
    console.error('推送错误:', err)
    pushSuccess.value = false
    pushResult.value = `❌ 推送异常：${err.message || '未知错误'}`
  } finally {
    isPushing.value = false
    pushProgress.value = ''
    
    // 3秒后清除结果提示
    setTimeout(() => {
      pushResult.value = ''
    }, 3000)
  }
}

// 1. 打开弹窗并加载数据
const openTargetModal = async () => {
    showTargetModal.value = true;
    await initCloudData(); // 直接调用通用的加载函数
};

// 2. 实时过滤列表 (计算属性)
const filteredList = computed(() => {
    const list = remoteTargets.value.all_data[selectedRegion.value] || []
    if (!searchQuery.value) return list
    
    const query = searchQuery.value.toLowerCase()
    return list.filter(item => 
        item.name.toLowerCase().includes(query) || 
        item.uid.toString().includes(query)
    )
})

// 3. 勾选/取消勾选逻辑 (存储的是 UID 数字或字符串，取决于你后端 processGardenActionsSync 的需要)
const toggleTarget = (uid) => {
    if (!currentConfigTask.value.config.selectedUids) {
        currentConfigTask.value.config.selectedUids = []
    }
    const idx = currentConfigTask.value.config.selectedUids.indexOf(uid)
    if (idx > -1) {
        currentConfigTask.value.config.selectedUids.splice(idx, 1)
    } else {
        currentConfigTask.value.config.selectedUids.push(uid)
    }
}

// 4. 全选/取消全选 (基于当前搜索结果)
const toggleAllFiltered = () => {
    const currentUids = filteredList.value.map(i => i.uid)
    const allIn = currentUids.every(uid => currentConfigTask.value.config.selectedUids?.includes(uid))
    
    if (allIn) {
        // 全不选
        currentConfigTask.value.config.selectedUids = currentConfigTask.value.config.selectedUids.filter(
            id => !currentUids.includes(id)
        )
    } else {
        // 全选 (去重合并)
        const combined = [...(currentConfigTask.value.config.selectedUids || []), ...currentUids]
        currentConfigTask.value.config.selectedUids = [...new Set(combined)]
    }
}

const handleGardenAFKChange = async () => {
  console.log("🖱️ 自动挂机开关状态改变");
  await nextTick();

  try {
    const gardenCategory = taskCategories.find(c => c.id === 'garden');
    if (!gardenCategory) return;

    const isEnabled = currentConfigTask.value?.config?.gardenAFK || false;

    if (!isEnabled) {
      console.log("🧹 检测到关闭，正在重置 UI 勾选状态...");
      
      // 只重置菜园维护类的任务
      gardenCategory.tasks.forEach(task => {
        task.selected = false;
        
        // 重置该任务的配置
        if (task.config) {
          // 根据任务类型重置特定字段
          switch(task.id) {
            case 'garden_meat':
              task.config = {
                collectMeat: false,
                eatMeat: false,
                eatNeighbors: false,
                eatGuilds: false,
                eatRankings: false
              };
              break;
            case 'garden_veggie':
              task.config = {
                collectVeggie: false,
                plantVeggie: false,
                buySeeds: false,
                veggieType: 'baicai',
                scanVeggie: false,
                stealVeggie: false,
                interestedVeggies: [],
                selectedUids: []
              };
              break;
            case 'garden_egg':
              task.config = {
                shareEgg: false
              };
              break;
          }
        }
      });
      
      console.log("✅ 菜园任务状态已清空");
      
      // 调用后端停止自动挂机
      await window.go.main.App.ToggleGardenLoop(false, {});
      
    } else {
      // 开启自动挂机时，收集所有菜园任务的配置
      const combinedConfig = {};
      gardenCategory.tasks.forEach(task => {
        if (task.selected && task.config) {
          Object.assign(combinedConfig, JSON.parse(JSON.stringify(task.config)));
        }
      });
      
      // 检查是否有选中的任务
      const hasSelectedTasks = gardenCategory.tasks.some(t => t.selected);
      if (!hasSelectedTasks) {
        alert("⚠️ 请至少勾选一个菜园维护任务");
        currentConfigTask.value.config.gardenAFK = false;
        return;
      }

      // 🌟 新增：如果开启了偷菜功能，检查扫菜情报并同步
      if (combinedConfig.stealVeggie) {
        console.log("🔍 检测到偷菜功能开启，检查扫菜情报...");
        
        if (scanResults.value.length === 0) {
          alert("⚠️ 开启偷菜功能需要先有扫菜情报！请先执行扫菜");
          currentConfigTask.value.config.gardenAFK = false;
          return;
        }

        console.log(`📊 当前扫菜情报数量: ${scanResults.value.length}`);
        
        // 同步偷菜目标到后端
        try {
          console.log("🔄 正在同步偷菜目标到后端...");
          
          // 将前端数据转换为后端需要的 Veggie 格式
          const backendTargets = scanResults.value.map(item => ({
            uid: item.uid,  // 已经是字符串
            name: item.name,
            veggieType: "",  // 如果前端没有存储，暂时留空
            matureTime: item.matureTime,
            pos: item.pos,
            district: "",    // 可选，留空
            nickname: item.nickname
          }));
          
          console.log("📤 发送到后端的偷菜目标:", backendTargets);
          await window.go.main.App.UpdateStealTargets(backendTargets);
          console.log("✅ 偷菜目标同步成功");
        } catch (err) {
          console.error("❌ 偷菜目标同步失败:", err);
          alert("❌ 偷菜目标同步失败，请重试");
          currentConfigTask.value.config.gardenAFK = false;
          return;
        }
      }
      
      // 调用后端开启自动挂机
      console.log("🚀 调用后端开启自动挂机，配置:", combinedConfig);
      await window.go.main.App.ToggleGardenLoop(true, combinedConfig);
    }
  } catch (err) {
    console.error("❌ 自动挂机切换失败:", err);
    alert(`❌ 自动挂机切换失败: ${err.message || '未知错误'}`);
  }
};

// pk状态变量
const isPKPanelOpen = ref(false); // 控制 PK 小窗口显示
const pkSearchQuery = ref('');   // PK 专用的搜索框;
const isPKEnabled = ref(false);      // 控制图标呼吸灯

const targetUser = ref({ uid: '', server: '', name: '' });
const sourceUser = ref({ uid: '', server: '' });

const togglePKPanel = () => {
    isPKPanelOpen.value = !isPKPanelOpen.value;
    if (isPKPanelOpen.value) {
        initCloudData(); // 打开 PK 面板时确保数据已加载
        isRadarPanelOpen.value = false;
    }
};

// 停止替换的方法
const stopPKReplacement = () => {
    window.go.main.App.StopPKMode().then(() => {
        isPKEnabled.value = false;
        isPKPanelOpen.value = false;
        // 清空输入，防止误操作
        targetUser.value = { uid: '', server: '', name: '' };
        sourceUser.value = { uid: '', server: '' };
        alert("🛑 替换规则已移除，恢复正常请求");
    });
};

// 设置目标人 (从列表点)
const setAsTarget = (user) => {
    targetUser.value = {
        uid: user.uid.toString(),
        server: user.region, // 自动填入分组名如 961-980
        name: user.name
    };
};

// 设置源人 (从列表点)
const setAsSource = (user) => {
    sourceUser.value = {
        uid: user.uid.toString(),
        server: user.region
    };
};

// 列表点击默认逻辑：先填目标，如果目标有了就填源
const handleListClick = (user) => {
    if (!targetUser.value.uid) {
        setAsTarget(user);
    } else {
        setAsSource(user);
    }
};

// 执行应用
const applyPK = () => {
    if (!targetUser.value.uid || !sourceUser.value.uid || !targetUser.value.server || !sourceUser.value.server) {
        alert("⚠️ UID和区服都必须填写！(手动输入请确认区服数字)");
        return;
    }

    window.go.main.App.SetPKReplacement(
        targetUser.value.uid, 
        targetUser.value.server, 
        sourceUser.value.uid, 
        sourceUser.value.server
    ).then(() => {
        isPKEnabled.value = true;
        isPKPanelOpen.value = false;
        // alert(`✅ 替换已生效！\n目标: ${targetUser.value.uid}\n源: ${sourceUser.value.uid}`);
    });
};



// 1. 复用 remoteTargets 数据源进行扁平化处理
// 跨服 PK 需要直接看到所有人，或者按搜索过滤，不需要像弹窗那样分大区点击
const flatPKList = computed(() => {
    const list = [];
    // 注意：这里要判断 remoteTargets.value.all_data
    if (!remoteTargets.value || !remoteTargets.value.all_data) {
        console.log("数据源还没准备好");
        return list;
    }
    
    // 遍历 JSON 的 key (也就是 961-980 这种)
    Object.keys(remoteTargets.value.all_data).forEach(region => {
        const players = remoteTargets.value.all_data[region];
        players.forEach(user => {
            list.push({
                ...user,
                region: region // 把大区存下来，方便展示和过滤
            });
        });
    });
    
    console.log("扁平化后的总人数:", list.length);
    return list;
});

// 2. PK 专用过滤列表
const filteredPKList = computed(() => {
    const q = pkSearchQuery.value.toLowerCase().trim();
    if (!q) return flatPKList.value.slice(0, 30);
    
    return flatPKList.value.filter(u => {
        // 关键点：确保 u.name 和 u.uid 在你的 JSON 对象里确实存在
        return (u.name && u.name.toLowerCase().includes(q)) || 
               (u.uid && u.uid.toString().includes(q)) ||
               (u.region && u.region.includes(q));
    });
});

// 3. 选中目标人的逻辑
const selectPKTarget = (user) => {
    targetUser.value = {
        uid: user.uid.toString(),
        server: user.region, // 👈 关键：这里直接复用了 JSON 里的分组名作为区服范围
        name: user.name
    };
    // 之后用户再手动填入“被替换人”的信息即可
};

// 珍宝
const newRule = reactive({
  keyword: '',
  quality: '',
  min: 0,
  price: 1500,
  targetCategories: []
})

const userLevel = ref(0); // 0:未授权, 1:普通, 2:高级
const zhenbaoAutoRefresh = ref(false);

// 获取每列所需的权限等级
const getColumnRequiredLevel = (columnId) => {
  // 日常任务、菜园维护、周期活动都需要等级3
  if (['basic', 'garden', 'activity'].includes(columnId)) {
    return 3
  }
  return 1 // 默认等级1
}

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

// 添加扫菜同步函数
const syncStealTargetsToBackend = async () => {
  if (scanResults.value.length === 0) return;
  
  console.log("🔄 正在同步偷菜目标到后端，数量:", scanResults.value.length);
  
  try {
    // 将前端数据转换为后端需要的 Veggie 格式
    const backendTargets = scanResults.value.map(item => ({
      uid: item.uid,  // 已经是字符串
      name: item.name,
      veggieType: "",  // 如果前端没有存储，可能需要从别处获取
      matureTime: item.matureTime,
      pos: item.pos,
      district: "",    // 可选
      nickname: item.nickname
    }));
    
    await window.go.main.App.UpdateStealTargets(backendTargets);
    console.log("✅ 偷菜目标同步成功");
  } catch (err) {
    console.error("❌ 偷菜目标同步失败:", err);
  }
};

// --- 珍宝拦截逻辑 ---
const filteredAttributes = computed(() => {
  const all = attributeOptions.value || []
  if (!searchKeyword.value) return all
  const searchLower = searchKeyword.value.toLowerCase()
  return all.filter(name => name.toLowerCase().includes(searchLower))
})

const selectAttribute = (name) => {
  searchKeyword.value = name
  showDropdown.value = false // 选中后关闭
}

// 核心修复：点击外部关闭
const closeDropdown = () => {
  showDropdown.value = false
}

watch(rules, (newVal) => {
  UpdateRules(JSON.parse(JSON.stringify(newVal)))
}, { deep: true })

onMounted(async () => {
  // --- 1. 立即同步初始化菜园配置指向 ---
  const immediateInit = () => {
    if (taskCategories && taskCategories.length > 0) {
      const gardenCategory = taskCategories.find(c => c.id === 'garden');
      if (gardenCategory && gardenCategory.tasks.length > 0) {
        currentConfigTask.value = gardenCategory.tasks[0];
        console.log("🚀 [立即初始化] 菜园配置已绑定:", currentConfigTask.value.name);
      }
    }
  };
  immediateInit();

  /**
   * 2. 注册 Wails 事件监听 (包含新增的扫菜监听)
   */
  if (window.runtime) {
    // --- 🌟 核心新增：监听扫菜发现事件 ---
    window.runtime.EventsOn("on_veggie_discovered", (veggies) => {
      veggies.forEach(v => {
        // 1. 转换十进制 UID
        const decimalUid = parseInt(v.uid, 16);

        // 2. 🌟 从映射表中获取昵称
        const nickname = nameMap[decimalUid] || `用户(${decimalUid})`;

        // 3. 处理成熟时间
        const dateObj = new Date(v.matureTime); // 13位毫秒戳直接传入
        const fixedTimeStr = dateObj.toLocaleTimeString('en-GB', {
          hour12: false,
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
          timeZone: 'Asia/Shanghai'
        });

        const newRecord = {
          uid: decimalUid.toString(),
          nickname: nickname, // 🌟 存储查到的名字
          name: v.name,
          pos: v.pos,
          matureTime: v.matureTime, // 用于排序
          fixedTime: fixedTimeStr
        };

        // 4. 去重逻辑
        const existingIndex = scanResults.value.findIndex(
          item => item.uid === decimalUid.toString() && item.pos === v.pos
        );

        if (existingIndex !== -1) {
          scanResults.value[existingIndex] = newRecord;
        } else {
          scanResults.value.push(newRecord); // 先 push 进去
        }
      });

      // 5. 🌟 按照成熟时间顺序排列 (越早成熟的在越上面)
      scanResults.value.sort((a, b) => a.matureTime - b.matureTime);

      // 6. 限制数量
      if (scanResults.value.length > 50) {
        scanResults.value = scanResults.value.slice(0, 50);
      }

      // 7. 自动同步偷菜目标到后端
      syncStealTargetsToBackend();
    });

    // --- 原有监听逻辑 ---
    window.runtime.EventsOn("auth_success", (data) => {
      console.log("收到授权信息:", data);
      isAuthorized.value = true;
      if (data && typeof data === 'object') {
        userLevel.value = data.level || 1;
        if (data.mID) machineID.value = data.mID.replace(/[\r\n]/g, "").trim();
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
   * 3. 权限与基础数据加载
   */
  try {
    console.log("开始获取机器码...")
    const rawId = await GetMachineID()
    console.log("获取到的原始机器码:", rawId)
    
    if (rawId) {
      machineID.value = rawId.replace(/[\r\n]/g, "").trim()
      console.log('原始机器码(处理后):', machineID.value)
      
      // 直接调用后端计算MD5
      console.log("开始从后端获取MD5...")
      const hashedId = await GetHashedMachineID()
      hashedMachineID.value = hashedId.replace(/[\r\n]/g, "").trim()
      console.log('后端返回的MD5机器码:', hashedMachineID.value)
    } else {
      console.error("获取到的机器码为空")
    }

    console.log("开始验证权限...")
    const level = await CheckCurrentAuth()
    console.log("权限验证结果:", level)
    
    if (level > 0) {
      isAuthorized.value = true
      userLevel.value = level
      
      // 根据等级显示不同的欢迎消息
      let levelMsg = ''
      if (level >= 5) {
        levelMsg = '✨ 尊贵的等级5用户，您拥有全部功能权限'
      } else if (level >= 3) {
        levelMsg = '🌟 等级3用户，您拥有日常功能权限'
      } else {
        levelMsg = '⭐ 等级1用户，您拥有珍宝拦截和跨服切磋权限'
      }
      
      addDailyLog(levelMsg, 'success')
    } else {
      console.log("权限验证失败，用户未授权")
    }
  } catch (err) {
    console.error("授权校验异常:", err)
  }

  // 4. 拉取珍宝拦截相关的属性配置
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
   * 5. 同步菜园作物配置与远程配置
   */
  try {
      const gData = await window.go.main.App.GetGardenData();
      if (gData && gData.veggies) {
        remoteVeggies.value = gData.veggies;
        console.log("✅ 菜园品种同步成功:", Object.keys(gData.veggies).length);
      }

      const rConfig = await window.go.main.App.GetRemoteConfig();
      if (rConfig) {
        console.log("✅ 远程配置同步成功，版本:", rConfig.version);
      }
  } catch (err) {
      console.error("❌ 同步失败:", err);
  }

  // --- 兜底逻辑 ---
  setTimeout(() => {
    if (!currentConfigTask.value) immediateInit();
  }, 500);

  // 6. 时间更新逻辑
  updateBeijingTime();
  const timeIntv = setInterval(updateBeijingTime, 1000);
  
  onUnmounted(() => {
    if (timeIntv) clearInterval(timeIntv);
  });
});

// --- 其他操作方法 ---
const copyMachineID = () => {
  if (hashedMachineID.value) {
    navigator.clipboard.writeText(hashedMachineID.value)
    alert("✅ MD5机器码已复制，请发送给管理员")
  }
}

// 证书安装相关
const showCertGuide = ref(false)
const certInstalling = ref(false)
const certInstallResult = ref(null)
const certError = ref('')

// 检查代理状态
const checkProxyStatus = async () => {
    try {
        const status = await window.go.main.App.CheckProxyStatus()
        console.log('代理状态:', status)
        
        // 如果代理不可访问或证书未安装，显示引导
        if (!status.proxyAccessible || !status.certInstalled) {
            showCertGuide.value = true
        }
    } catch (err) {
        console.error('检查代理状态失败:', err)
    }
}

// 下载并安装证书
const installCertificate = async () => {
    certInstalling.value = true
    certError.value = ''
    certInstallResult.value = null
    
    try {
        const result = await window.go.main.App.DownloadAndInstallCert()
        certInstallResult.value = result
        
        if (result.success) {
            if (result.needRestart) {
                // 显示重启确认对话框
                if (confirm('✅ 证书已安装成功！\n\n需要重启电脑使证书完全生效。\n是否立即重启？')) {
                    await window.go.main.App.RestartComputer()
                }
            } else {
                // 不需要重启，直接关闭引导
                setTimeout(() => {
                    showCertGuide.value = false
                }, 2000)
            }
        } else {
            certError.value = result.message
        }
    } catch (err) {
        certError.value = err.message || '安装失败，请手动安装'
    } finally {
        certInstalling.value = false
    }
}

// 手动安装指南
const showManualGuide = ref(false)

// 在 onMounted 中添加检查
onMounted(async () => {
    // 检查代理和证书状态
    await checkProxyStatus()
})

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
  display: flex;
  flex-direction: column;
  height: 100vh; /* 占据整个视口高度 */
  overflow: hidden;
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
  flex: 1;             /* 占据剩余的所有空间 */
  overflow-y: auto;    /* 内容多了自动显示滚动条 */
  padding-right: 8px;  /* 给滚动条留点呼吸感 */
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
  height: 250px;
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

/* 统一合并 action-footer 的定义 */
.action-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
  position: relative;
  
  /* 🌟 核心修正：加大底部内边距到 50px 以上 */
  padding: 15px 20px 50px 20px; 
  
  /* 确保按钮不会因为内容多而被顶出屏幕 */
  margin-top: auto; 
  
  /* 强制该容器在 Flex 布局中不收缩 */
  flex-shrink: 0; 
  
  width: 100%;
  box-sizing: border-box;
}

.run-all-btn {
  /* 🌟 核心修正 3：改为 flex: 1 让它自适应 🌱 和 🥊 图标的空间 */
  flex: 1; 
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
  
  /* 确保文字不会因为宽度缩放而折行 */
  white-space: nowrap; 
}

/* 移动端或窄屏适配：如果横向太挤，可以允许折行 */
@media (max-width: 400px) {
  .action-footer {
    flex-wrap: wrap; /* 空间不够时，图标和按钮分两行 */
  }
}
.run-all-btn:hover:not(:disabled) { background: #0062cc; transform: scale(1.02); }
.run-all-btn:disabled { background: #222; color: #555; cursor: not-allowed; }


.radar-wrapper {
  position: relative; /* 🌟 关键：让弹窗相对于这个容器定位 */
}

/* 浮窗主体容器 */
.radar-popover {
  position: absolute;
  bottom: 60px;
  right: 0;
  width: 280px;
  /* 🌟 限制整个弹窗的最大高度，防止遮挡上方重要UI */
  max-height: 400px; 
  background: #1e272e;
  border: 1px solid #3d4e5f;
  border-radius: 12px;
  box-shadow: 0 10px 25px rgba(0,0,0,0.5);
  z-index: 999;
  display: flex;
  flex-direction: column;
  overflow: hidden; /* 保证圆角不被切掉 */
}

/* 🌟 新增：滚动区域设置 */
.popover-body {
  flex: 1;            /* 自动撑开 */
  overflow-y: auto;   /* 词条多了自动显示滚动条 */
  padding: 4px 0;     /* 稍微留白 */
}

/* 美化一下滚动条，让它看起来更高级 */
.popover-body::-webkit-scrollbar {
  width: 5px;
}
.popover-body::-webkit-scrollbar-thumb {
  background: #3d4e5f;
  border-radius: 10px;
}
.popover-body::-webkit-scrollbar-track {
  background: transparent;
}

/* 词条样式微调，增加点击反馈感 */
.mini-item {
  padding: 10px 12px;
  border-bottom: 1px solid rgba(255,255,255,0.05);
  transition: background 0.2s;
}
.mini-item:hover {
  background: rgba(46, 204, 113, 0.1); /* 鼠标悬停变浅绿 */
}

/* 气泡下面的那个小三角（指向 🌱） */
.radar-popover::after {
  content: '';
  position: absolute;
  bottom: -8px;
  right: 15px;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-top: 8px solid #1e272e;
}

/* 弹窗内部排版 */
.popover-header {
  padding: 8px 12px;
  background: #2f3640;
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #2ecc71;
  font-weight: bold;
}

.close-mini {
  cursor: pointer;
  color: #7f8c8d;
  font-size: 18px;
}


.mini-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 3px;
}

.mini-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px 0px;
  text-align: center;
}

.m-name { color: #fff; font-size: 13px; font-weight: bold; }
.m-time {
  color: #ffd700;
  font-size: 12px;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  font-weight: 500;
  opacity: 0.9;
}
.m-desc { color: #95a5a6; font-size: 11px; }

/* 3. 动画：向上弹出的轻盈感 */
.pop-up-enter-active, .pop-up-leave-active {
  transition: all 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}
.pop-up-enter-from, .pop-up-leave-to {
  opacity: 0;
  transform: translateY(15px) scale(0.9);
}
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

/* 下拉菜单项容器 */
.veggie-dropdown .dropdown-item {
  display: flex;         /* 使用弹性盒布局 */
  align-items: center;   /* 垂直居中对齐 */
  justify-content: flex-start; /* 👈 强制左侧对齐，不要居中 */
  padding: 8px 12px;     /* 适当的内边距，让内容离边缘有点距离 */
  width: 100%;           /* 占满宽度 */
  cursor: pointer;
  box-sizing: border-box;
}

/* 选中后的背景和边框 */
.veggie-dropdown .dropdown-item input[type="checkbox"]:checked {
  background-color: #007bff; /* 蓝色背景 */
  border-color: #007bff;
}

/* 核心：用伪元素画出白色对勾 */
.veggie-dropdown .dropdown-item input[type="checkbox"]:checked::after {
  content: '';
  position: absolute;
  /* 调整钩子的位置，使其居中 */
  left: 5px;
  top: 2px;
  /* 钩子的形状：长方形的两条边 */
  width: 5px;
  height: 9px;
  border: solid white;
  border-width: 0 2px 2px 0; /* 只显示右边和底边 */
  /* 旋转 45 度变成钩子 */
  transform: rotate(45deg);
}

/* 确保 Checkbox 容器是相对定位，方便钩子定位 */
.veggie-dropdown .dropdown-item input[type="checkbox"] {
  appearance: none;
  -webkit-appearance: none;
  position: relative; /* 必须有这个 */
  cursor: pointer;
  width: 18px;
  height: 18px;
  background-color: #1a1a1a;
  border: 1px solid #444;
  border-radius: 3px;
  display: inline-block;
  vertical-align: middle;
  transition: all 0.2s;
  flex-shrink: 0; /* 防止被挤压 */
  margin-right: 10px;
}

/* 文本标签 */
.veggie-dropdown .dropdown-item span {
  text-align: left;      /* 文本左对齐 */
  font-size: 12px;
  color: #ccc;           /* 浅灰色文字 */
  user-select: none;     /* 防止点击太快选中文本 */
}

/* 鼠标悬停效果 */
.veggie-dropdown .dropdown-item:hover {
  background-color: #2a2a2a; /* 悬停时行背景微亮 */
}

/* 容器布局：确保按钮和徽章垂直居中 */
.target-config-area {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 6px;
  padding: 2px 0;
}

/* 配置目标按钮自定义 */
.target-config-area .action-btn {
  background: linear-gradient(180deg, #2a2a2a 0%, #1a1a1a 100%);
  border: 1px solid #444;
  color: #ddd;
  padding: 6px 14px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

/* 按钮悬停效果：边框变亮，背景微蓝 */
.target-config-area .action-btn:hover {
  border-color: #007bff;
  color: #fff;
  background: linear-gradient(180deg, #333 0%, #222 100%);
  box-shadow: 0 0 8px rgba(0, 123, 255, 0.2);
}

/* 徽章（人数统计）样式 */
.target-config-area .count-badge {
  background-color: #002b55; /* 深蓝色底 */
  color: #00aaff;          /* 亮蓝色字 */
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: bold;
  border: 1px solid #004a99;
  letter-spacing: 0.5px;
  box-shadow: inset 0 0 4px rgba(0, 170, 255, 0.1);
}

/* 1. 基础行容器 */
.scan-veggie-row {
  display: flex !important; /* 确保强制开启弹性盒 */
  align-items: center;      /* 垂直居中 */
  gap: 12px;                /* 组件间距 */
  flex-wrap: nowrap;        /* 强制不换行 */
}

/* 2. 复选框容器 */
.custom-check {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;      /* 防止“扫菜”二字换行 */
}

/* 3. 下拉框触发器 (统一高度) */
.veggie-trigger {
  height: 32px;             /* 💡 设定固定高度 */
  min-width: 130px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  background: #000;         /* 黑色背景 */
  border: 1px solid #333;
  border-radius: 4px;
  cursor: pointer;
}

/* 4. 配置目标区域 */
.target-config-area {
  display: flex;
  align-items: center;      /* 确保按钮和(人)也在同一行 */
  gap: 8px;
  margin: 0 !important;      /* 💡 强制去掉可能存在的顶部间距 */
}

/* 5. 配置按钮 (高度必须与下拉框一致) */
.target-config-area .action-btn {
  height: 32px;             /* 💡 与下拉框对齐的关键 */
  background: #1a1a1a;
  border: 1px solid #444;
  color: #ddd;
  padding: 0 12px;          /* 左右内边距 */
  border-radius: 4px;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.target-config-area .action-btn:hover {
  border-color: #007bff;
  background: #222;
}

/* 6. 人数标签样式微调 */
.count-badge {
  color: #007bff;
  font-size: 12px;
  font-weight: bold;
  white-space: nowrap;
}

/* 修改珍宝拦截页面的日志容器样式 */
.console-container {
  flex: 1;               /* 让它占据右侧剩余的所有空间 */
  overflow-y: auto;      /* 关键：允许垂直滚动 */
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 15px;             /* 卡片之间的间距 */
  max-height: calc(100vh - 100px); /* 限制最大高度，防止撑开父容器 */
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

/* 修改最后一个词条的样式 */
.console-row:last-child {
  margin-bottom: 0;
  /* 确保最后一个词条有足够的底部间距 */
  padding-bottom: 2px;
}

/* 调整词条的行高，避免文字被切割 */
.console-row {
  display: flex; 
  margin-bottom: 4px; 
  font-size: 0.8rem;
  line-height: 1.5; /* 增加行高确保文字完整显示 */
}
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

.scan-panel {
    background: #111;
    border: 1px solid #444;
    padding: 10px;
    font-family: 'Consolas', monospace; /* 使用等宽字体，看起来像黑客终端 */
}

.panel-title {
    color: #00ff00;
    font-size: 13px;
    margin-bottom: 8px;
    border-bottom: 1px solid #333;
}

.scan-item {
    font-size: 12px;
    margin-bottom: 4px;
    border-left: 2px solid #555;
    padding-left: 8px;
}

.v-time { color: #00d4ff; margin-right: 10px; }
.v-name { color: #f1c40f; margin-right: 10px; font-weight: bold; }
.v-detail { color: #777; }
.v-tag {
    float: right;
    color: #e67e22;
    background: rgba(230, 126, 34, 0.1);
    padding: 0 4px;
    border-radius: 2px;
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

/* 1. 蒙层：全屏遮罩 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.75); /* 深色半透明 */
  backdrop-filter: blur(5px);      /* 磨砂玻璃效果，非常符合你的风格 */
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 9999;                   /* 确保在最上层 */
}

/* 2. 弹窗主体容器 */
.target-modal-container {
  width: 500px;
  max-height: 85vh;                /* 最高占屏幕 85%，防止溢出 */
  background: #121212;             /* 纯黑背景 */
  border: 1px solid #333;
  border-radius: 8px;
  display: flex;
  flex-direction: column;          /* 纵向排列：页头、过滤栏、列表、页脚 */
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.6);
  color: #eee;
}

/* 3. 页头 */
.modal-header {
  padding: 16px;
  border-bottom: 1px solid #222;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.modal-header h3 { margin: 0; font-size: 16px; }
.close-x { 
  background: none; border: none; color: #666; 
  font-size: 24px; cursor: pointer; 
}
.close-x:hover { color: #fff; }

/* 4. 过滤操作栏 */
/* 容器布局 */
.modal-filter-row {
  display: flex;
  gap: 12px;
  padding: 15px 20px;
  background: #121212; /* 稍微浅一点的深蓝色，区分背景 */
  align-items: center;
}

/* 🌟 下拉框美化 (去掉了系统原生丑边框) */
.custom-select-wrapper {
  position: relative;
  flex: 1;
}

.dark-select {
  width: 100%;
  appearance: none; /* 彻底移除原生箭头 */
  background: #121212;
  color: #ecf0f1;
  border: 1px solid #1a252f;
  padding: 10px 15px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  outline: none;
  transition: all 0.2s;
}

.dark-select:hover, .dark-select:focus {
  border-color: #3d4e5f;
  box-shadow: 0 0 8px rgba(52, 152, 219, 0.3);
}

.select-arrow {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  pointer-events: none;
  color: #8e98a2;
  font-size: 10px;
}

/* 🌟 搜索框美化 */
.search-wrapper {
  position: relative;
  flex: 2;
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: #1a252f;
  font-size: 14px;
}

.search-input {
  width: 100%;
  background: #121212;
  border: 1px solid #3d4e5f;
  color: white;
  padding: 10px 10px 10px 35px; /* 左侧留出图标位置 */
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  transition: all 0.2s;
}

.search-input:focus {
  border-color: #2ecc71;
  box-shadow: 0 0 8px rgba(46, 204, 113, 0.3);
}

/* 🌟 按钮美化 (全选/反选) */
.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
  white-space: nowrap;
}

.toggle-btn {
  background: #121212;
  color: #ecf0f1;
  border: 1px solid #3d4e5f;
}

.toggle-btn:hover {
  background: #455d75;
  border-color: #3498db;
  color: white;
}

.toggle-btn:active {
  transform: translateY(1px);
}
.search-input { flex: 1; }

/* 5. 名单滚动区域 */
.target-list-scroll {
  flex: 1;                         /* 自动撑满中间剩余空间 */
  overflow-y: auto;                /* 开启纵向滚动 */
  padding: 10px;
  background: #0a0a0a;             /* 列表区背景稍微深一点 */
}

/* 自定义滚动条 (可选，增加科技感) */
.target-list-scroll::-webkit-scrollbar { width: 6px; }
.target-list-scroll::-webkit-scrollbar-thumb { background: #333; border-radius: 3px; }

/* 6. 单个玩家条目 */
.target-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  margin-bottom: 4px;
  border-radius: 4px;
  cursor: pointer;
  background: #161616;
  border: 1px solid transparent;
  transition: 0.2s;
}
.target-item:hover { background: #222; }
.target-item.is-selected { 
  background: #001a35; 
  border-color: #007bff; 
}

/* 7. 模拟复选框 */
.checkbox-box {
  width: 16px; height: 16px; 
  border: 1px solid #444; 
  margin-right: 12px;
  display: flex; align-items: center; justify-content: center;
}
.inner-dot { width: 10px; height: 10px; background: #007bff; }

/* 8. 页脚 */
.modal-footer {
  padding: 12px;
  border-top: 1px solid #222;
  display: flex;
  justify-content: flex-end;
}
.save-btn {
  background: #007bff; color: white; border: none;
  padding: 8px 24px; border-radius: 4px; cursor: pointer;
}
.save-btn:hover { background: #0069d9; }

/* 9. 动画 */
.fade-enter-active, .fade-leave-active { transition: opacity 0.3s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

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

.action-footer {
  display: flex;
  align-items: center;  /* 垂直居中 */
  justify-content: center; /* 水平居中 */
  gap: 15px;            /* 按钮和图标之间的间距 */
  margin-top: 20px;     /* 与上方内容的间距 */
}

/* 之前给你的 🌱 按钮样式 */
.radar-toggle-btn {
  width: 46px;
  height: 46px;
  background: #2729289e;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  cursor: pointer;
  box-shadow: 0 4px 15px rgba(0,0,0,0.2);
  transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275); /* 加点弹跳动画 */
  border: 3px solid #227ce3;
  flex-shrink: 0; /* 防止图标被按钮挤扁 */
}

.radar-toggle-btn:hover {
  transform: scale(1.15) rotate(15deg);
  background: #59625d60;
}

/* 正在扫描时的呼吸灯特效 */
.radar-toggle-btn.pulse {
  animation: pulse-radar 2s infinite;
}

@keyframes pulse-radar {
  0% { box-shadow: 0 0 0 0 rgba(46, 204, 113, 0.7); }
  70% { box-shadow: 0 0 0 15px rgba(46, 204, 113, 0); }
  100% { box-shadow: 0 0 0 0 rgba(46, 204, 113, 0); }
}

/* 条目头部 */
.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-bottom: 4px;
}

/* 保持 mini-info 原有布局，但让它占满剩余空间 */
.mini-info {
  display: flex;
  justify-content: space-between;
  flex: 1;
  margin-right: 8px;
}

/* 品种名称保持原样 */
.m-name {
  color: #fff;
  font-size: 13px;
  font-weight: bold;
}

/* 时间样式调整 - 让它靠右 */
.m-time {
  color: #f1c40f;
  font-size: 12px;
  font-family: 'Consolas', monospace;
  margin-left: 12px;
}

/* 删除按钮样式 */
.delete-item-btn {
  background: rgba(255, 71, 87, 0.1);
  border: 1px solid rgba(255, 71, 87, 0.3);
  color: #ff4757;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  padding: 0;
  line-height: 1;
  flex-shrink: 0;
}

.delete-item-btn:hover {
  background: #ff4757;
  color: white;
  border-color: #ff4757;
}

/* 推送区域样式 */
.push-section {
  margin-top: 16px;
  padding: 12px;
  background: rgba(0, 122, 255, 0.05);
  border: 1px solid rgba(0, 122, 255, 0.2);
  border-radius: 8px;
}

.push-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
}

.push-icon {
  font-size: 16px;
}

.push-title {
  font-size: 13px;
  font-weight: bold;
  color: #007aff;
}

/* 美化现有的 push-btn 按钮 - 绿色渐变版 */
.push-btn {
  width: 100%;
  height: 42px;
  background: linear-gradient(135deg, #b0ff57 0%, #0f4e2e 100%);
  border: none;
  border-radius: 10px;
  color: white;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  margin: 10px 0 8px 0;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  letter-spacing: 0.5px;
}


/* 流光效果 */
.push-btn::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
  transition: left 0.5s ease;
}

.push-btn:hover::before {
  left: 100%;
}


/* 点击效果 */
.push-btn:active:not(:disabled) {
  transform: translateY(0);
  box-shadow: 0 2px 10px rgba(46, 204, 113, 0.3);
}

/* 禁用状态 */
.push-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background: linear-gradient(135deg, #9ca3af 0%, #6b7280 100%);
  box-shadow: none;
}

/* 按钮内的文字样式 */
.push-btn span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.push-btn:hover span:first-child::before {
  transform: scale(1.1);
}

/* 加载状态样式 */
.push-btn:disabled span:last-child::before {
  content: '';
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-right: 6px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 现代下拉框样式 - 替换旧的 group-selector */
.group-selector-modern {
  margin-bottom: 16px;
  width: 100%;
}

.select-wrapper {
  position: relative;
  width: 100%;
}

.modern-select {
  width: 100%;
  height: 44px;
  padding: 0 16px;
  padding-right: 40px; /* 为箭头留空间 */
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 10px;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
  cursor: pointer;
  transition: all 0.2s ease;
  outline: none;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.modern-select:hover {
  border-color: #007aff;
  background: #252525;
}

.modern-select:focus {
  border-color: #007aff;
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

/* 自定义下拉箭头 */
.select-arrow-modern {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #888;
  font-size: 12px;
  pointer-events: none;
  transition: transform 0.2s ease;
}

.select-wrapper:hover .select-arrow-modern {
  color: #007aff;
  transform: translateY(-50%) rotate(180deg);
}

/* 下拉选项样式 */
.modern-select option {
  background: #1e1e1e;
  color: #fff;
  padding: 12px;
  font-size: 14px;
}

/* 推送结果美化 */
.push-result {
  font-family: "SF Pro Text", -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", "Helvetica Neue", sans-serif !important;
  padding: 12px 16px !important;
  border-radius: 12px !important;
  font-size: 14px !important;
  font-weight: 500 !important;
  text-align: center !important;
  margin-top: 12px !important;
  animation: resultPop 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1) !important;
  backdrop-filter: blur(10px) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  gap: 8px !important;
}

/* 成功状态 */
.push-result.success {
  background: linear-gradient(135deg, #2ecc71 0%, #27ae60 100%) !important;
  color: white !important;
  border: none !important;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2) !important;
}

/* 失败状态 */
.push-result.error {
  background: linear-gradient(135deg, #ff4757 0%, #ee5a24 100%) !important;
  color: white !important;
  border: none !important;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2) !important;
}

/* 动画效果 */
@keyframes resultPop {
  0% {
    opacity: 0;
    transform: scale(0.8) translateY(-10px);
  }
  50% {
    transform: scale(1.02) translateY(0);
  }
  100% {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

/* PK 图标按钮样式 */
.pk-action-btn {
  background: linear-gradient(135deg, #1b1a1a, #2e292a) !important;
  border: 3px solid #ee5253 !important;
  font-size: 20px !important;
}

.pk-active {
  box-shadow: 0 0 15px rgba(255, 71, 87, 0.6);
  animation: pulse-red 2s infinite;
}

@keyframes pulse-red {
  0% { box-shadow: 0 0 0 0 rgba(255, 71, 87, 0.7); }
  70% { box-shadow: 0 0 0 12px rgba(255, 71, 87, 0); }
  100% { box-shadow: 0 0 0 0 rgba(255, 71, 87, 0); }
}

/* PK 内部布局 */
.pk-search-container { padding: 8px; }
.pk-input-field {
  width: 100%;
  padding: 6px 10px;
  background: #f1f2f6;
  border: 1px solid #dfe4ea;
  border-radius: 4px;
}

.pk-scroll-list {
  max-height: 200px;
  overflow-y: auto;
  border-top: 1px solid #eee;
}

.pk-list-item {
  padding: 10px;
  border-bottom: 1px solid #f1f2f6;
  cursor: pointer;
}

.pk-list-item:hover { background: #fff5f5; }
.pk-list-item.selected { background: #ffeaa7; border-left: 4px solid #232021; }

.pk-badge {
  background: #2729289e;
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  border-radius: 3px;
  margin-right: 6px;
}

.pk-manual-footer {
  display: flex;
  gap: 5px;
  padding: 8px;
  background: #f9f9f9;
}

.pk-manual-input {
  flex: 1;
  padding: 4px 8px;
  font-size: 12px;
  border: 1px solid #ccc;
}

.pk-confirm-btn {
  background: #2c2a2a;
  color: white;
  border: none;
  padding: 4px 10px;
  border-radius: 3px;
  cursor: pointer;
}

/* PK 面板专用样式 */
/* 1. 外壳锁定：固定高度，绝对不准出现外层滚动条 */
.pk-popover {
  width: 420px;
  height: 570px !important;    /* 🌟 核心：固定高度，确保到底部不留黑空 */
  background: #16181d !important;
  display: flex !important;
  flex-direction: column !important;
  overflow: hidden !important;  /* 🌟 核心：禁止最外层滑动 */
  border-radius: 16px;
  border: 1px solid #2d313d;
  bottom: 80px !important;
  right: 20px;
}

/* 2. 主体容器：像弹簧一样占满空间 */
.popover-body {
  flex: 1 !important;           /* 🌟 占满 Header 之外的所有空间 */
  display: flex !important;
  flex-direction: column !important;
  min-height: 0;                /* 必须加这行，内部滚动条才会出来 */
}

/* 3. 名单区域：这是唯一的动态拉伸层 */
.pk-list-scroll {
  flex: 1 !important;           /* 🌟 核心：它会吃掉所有黑色空白，顶住底部 */
  overflow-y: auto !important;  /* 只有这里允许滑动 */
  padding: 0 16px;
  min-height: 0;
}

/* 4. 底部表单：死死钉在最底下 */
.pk-footer-form {
  flex-shrink: 0 !important;    /* 绝对不准被名单挤扁 */
  padding: 16px;
  background: #111318;
  border-top: 1px solid #2d313d;
}

/* 5. 列表项：按钮彻底靠右 */
.pk-mini-item {
  display: flex !important;
  align-items: center !important;
  justify-content: space-between !important; /* 🌟 名字在左，按钮在右 */
  padding: 10px 14px;
  margin-bottom: 8px;
  background: #21242c;
  border-radius: 10px;
}

/* 6. 名字容器：防止长名字推走按钮 */
.pk-user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.pk-user-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis; /* 名字长了自动变省略号 */
}

/* 统一重置所有输入框的盒模型 */
.pk-popover input {
  box-sizing: border-box;
}

/* 2. 列表区域拉高 */
.pk-list-scroll {
  flex: 1;              /* 自动撑开剩余空间 */
  max-height: 240px;    /* 列表可见范围变大 */
  overflow-y: auto;
  padding: 0 16px;
}

.pk-user-name {
  font-size: 11px; /* 字体变小更精致 */
  color: #e0e0e0;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 3. 列表项间距优化 */
.pk-mini-item {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  margin-bottom: 8px;
  background: #21242c;
  border-radius: 10px;
}

/* --- 3. 核心表单区 (解决格子溢出) --- */
/* 4. 底部表单区增加留白 */
.pk-footer-form {
  padding: 10px 16px;   /* 增加上下留白 */
  background: #111318;  /* 稍微深一点的黑色作为区分 */
  border-top: 1px solid #2d313d;
}

.input-section {
  margin-bottom: 8px;
  padding: 0; /* 移除之前的 padding 防止挤压 */
}

.input-section label {
  font-size: 10px;
  color: #666;
  margin-bottom: 4px;
  display: block;
}

.dual-input {
  display: flex;
  gap: 6px;
  width: 100%;
}

.pk-input-s {
  flex: 2; /* UID 占大头 */
  background: #252934;
  border: 1px solid #3d4250;
  color: #fff;
  padding: 6px 8px;
  border-radius: 4px;
  font-size: 12px;
  min-width: 0; /* 核心：允许 flex 缩小而不溢出 */
}

.pk-input-xs {
  flex: 1; /* 区服占小头 */
  background: #252934;
  border: 1px solid #3d4250;
  color: #ff4757;
  padding: 6px 4px;
  border-radius: 4px;
  font-size: 11px;
  text-align: center;
  min-width: 0;
}

/* --- 4. 按钮美化 (拒绝丑按钮) --- */
.pk-confirm-btn {
  width: 100%;
  margin-top: 10px;
  height: 36px;
  border: none;
  border-radius: 8px;
  color: white;
  font-weight: bold;
  font-size: 13px;
  cursor: pointer;
  /* 使用线性渐变增加质感 */
  background: linear-gradient(135deg, #ff4757 0%, #ff6b81 100%);
  box-shadow: 0 4px 12px rgba(255, 71, 87, 0.3);
  transition: all 0.3s ease;
}

.pk-confirm-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 15px rgba(255, 71, 87, 0.4);
  filter: brightness(1.1);
}

/* 拦截成功后的绿色态 */
.pk-confirm-btn.btn-active {
  background: linear-gradient(135deg, #2ed573 0%, #7bed9f 100%);
  box-shadow: 0 4px 12px rgba(46, 213, 115, 0.3);
}

/* 顶部停止按钮 */
.stop-pk-btn {
  background: rgba(255, 71, 87, 0.1);
  color: #ff4757;
  border: 1px solid rgba(255, 71, 87, 0.3);
  padding: 2px 10px;
  border-radius: 20px;
  font-size: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.stop-pk-btn:hover {
  background: #ff4757;
  color: white;
}

/* --- 5. 其他微调 --- */
.pk-divider {
  text-align: center;
  font-size: 12px;
  color: #3d4250;
  margin: 4px 0;
}

.pk-search-bar { padding: 12px; }
.pk-input {
  width: 100%;
  box-sizing: border-box;
  background: #252934;
  border: 1px solid #3d4250;
  color: #fff;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  outline: none;
}

/* 1. 统一头部颜色 */
.popover-header {
  background: #16181d !important; /* 与底部开启拦截的背景色一致 */
  padding: 12px 15px;
  border-bottom: 1px solid #2d313d;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.popover-header span {
  color: #387ff3; /* 标题用绿色，代表配置状态 */
  font-weight: bold;
  font-size: 14px;
}

/* 2. 列表项美化：去掉笨重感 */
.pk-mini-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  margin-bottom: 6px;
  background: #252934; /* 深色背景 */
  border: 1px solid transparent;
  border-radius: 8px;
  transition: all 0.2s;
}

.pk-mini-item:hover {
  background: #2d3341;
  border-color: #3d4250;
}

/* 3. 游戏昵称与区服标签 */
.pk-region-tag {
  font-size: 10px;
  color: #888; /* 默认灰一点，不抢眼 */
  background: rgba(255, 255, 255, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
  margin-right: 10px;
}

/* 4. 列表内小按钮美化：告别突兀白色 */
.pk-item-actions {
  display: flex;
  gap: 5px;
}

.mini-tag-btn {
  font-size: 10px;
  border: 1px solid #444; /* 改为描边风格 */
  background: transparent;
  color: #999;
  padding: 3px 8px;
  border-radius: 5px;
  cursor: pointer;
  transition: all 0.2s;
}

/* 悬停时才发色 */
.mini-tag-btn.target:hover {
  background: #ff4757;
  border-color: #ff4757;
  color: white;
}

.mini-tag-btn.source:hover {
  background: #1e90ff;
  border-color: #1e90ff;
  color: white;
}

/* 5. 滚动条美化 */
.pk-list-scroll::-webkit-scrollbar {
  width: 4px;
}
.pk-list-scroll::-webkit-scrollbar-thumb {
  background: #3d4250;
  border-radius: 10px;
}

/* ========== 授权覆盖层 (Auth) ========== */
/* 权限相关样式 */
.column-level-tag {
  font-size: 9px;
  padding: 2px 4px;
  background: rgba(255, 71, 87, 0.2);
  color: #ff4757;
  border-radius: 3px;
  margin-left: 6px;
  white-space: nowrap;
}

.locked-task {
  opacity: 0.6;
  cursor: not-allowed !important;
}

.locked-task:hover {
  background: #0a0a0a !important;
  border-color: #1a1a1a !important;
}

.locked-text {
  color: #666 !important;
}

.afk-text.disabled {
  color: #666 !important;
  opacity: 0.5;
}

.console-placeholder {
  background: #000;
  border: 1px solid #1a1a1a;
  border-radius: 12px;
  height: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #666;
  font-size: 13px;
}

.lock-icon-small {
  font-size: 24px;
  opacity: 0.3;
}

.action-footer-placeholder {
  padding: 10px 0;
  display: flex;
  justify-content: center;
}

.action-footer-placeholder .run-all-btn.disabled {
  width: 100%;
  max-width: 400px;
  padding: 16px;
  background: #222;
  color: #555;
  border: none;
  border-radius: 12px;
  font-weight: bold;
  cursor: not-allowed;
  font-size: 1rem;
}

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