package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qtgolang/SunnyNet/SunnyNet"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	sunny            *SunnyNet.Sunny
	isMainAccount    bool
	mainAccountTried bool
	packetCount      int
	isConnected      bool
	activeConn       SunnyNet.ConnWebSocket
	UserLevel        int // 0:未授权, 1:普通, 2:高级

	// --- 珍宝模块 ---
	activeRules        []PurchaseRule
	rulesMutex         sync.RWMutex
	rulesPrinted       bool
	ZhenBaoAutoRefresh bool

	// --- 菜园基础数据 ---
	gardenData       VeggieConfigData
	veggieTimestamps [4]int64
	vtMutex          sync.RWMutex
	harvestingPos    map[string]bool // 💡 用于黄金树异步任务锁
	lastGardenHex    string
	veggieMutex      sync.Mutex

	// --- 🔍 扫菜模块 (新增加) ---
	scanConfig        ScanTaskConfig // 专门存放扫菜勾选的作物和名单
	scanMutex         sync.RWMutex   // 扫菜专用的锁
	isScanningVeggie  bool           // 扫菜开关状态
	stopScanLoop      chan struct{}  // 停止扫菜巡逻的信号
	isScanLoopRunning bool           // 扫菜巡逻是否正在执行

	// --- 菜园挂机任务控制 ---
	gardenTask          GardenTaskConfig
	gdMutex             sync.RWMutex
	stopGardenLoop      chan struct{}
	isGardenLoopRunning bool
	isEatingMeat        bool
	isCheckingVeggie    bool

	// --- 远程配置 ---
	remoteConfig RemoteConfig
	rcMutex      sync.RWMutex

	// --- 其他任务 ---
	isTaskRunning bool

	// --- 跨区切磋 ---
	pkTargetUID    uint32
	pkSourceUID    uint32
	pkTargetSrvMin uint16 // 目标起始区服
	pkTargetSrvMax uint16 // 目标结束区服
	pkSourceSrvMin uint16 // 源起始区服
	pkSourceSrvMax uint16 // 源结束区服
	isPKActive     bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// ------------------- 数据结构定义 -------------------

type Attribute struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

type ZhenBaoItem struct {
	UID         string      `json:"uid"`
	District    string      `json:"district"`
	Price       int64       `json:"price"`
	ListTime    int64       `json:"listTime"` // 修改为 int64，直接传秒级时间戳
	IsLocked    bool        `json:"isLocked"` // 新增：后端直接下发判定结果
	Type        string      `json:"itemType"`
	PurchaseCmd string      `json:"purchaseCmd"`
	Attributes  []Attribute `json:"attributes"`
}

type PurchaseRule struct {
	Name             string
	TargetQuality    string
	TargetCategories []string
	TargetAttr       string
	MinAttrValue     float64
	MaxPrice         int64
}

// TaskParam 对应前端发送的任务对象
type TaskParam struct {
	ID     string                 `json:"id"`
	Config map[string]interface{} `json:"config"`
}

// NeighborInfo 定义邻居信息结构
type NeighborInfo struct {
	UID        string
	MeatType   string
	MeatStatus string
	pvp        int
	HasEgg     string
}

type VeggieInfo struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
}

type Veggie struct {
	UID        string `json:"uid"`        // 目标玩家 UID 的 16 进制 (例如: "024FA5FA")
	Name       string `json:"name"`       // 蔬菜名称 (例如: "变异南瓜")
	VeggieType string `json:"veggieType"` // 蔬菜品种 ID (例如: "01CD")
	MatureTime int64  `json:"matureTime"` // 💡 关键修改：成熟的 Unix 时间戳 (秒)
	Pos        string `json:"pos"`        // 蔬菜在地里的位置特征码 (例如: "AABB")
}

type VeggieConfigData struct {
	Veggies  map[string]VeggieInfo `json:"veggies"`
	Patterns map[string]string     `json:"patterns"`
}

type GardenTaskConfig struct {
	CollectVeggie bool   `json:"collectVeggie"`
	PlantVeggie   bool   `json:"plantVeggie"`
	VeggieType    string `json:"veggieType"`
	BuySeeds      bool   `json:"buySeeds"`
	CollectMeat   bool   `json:"collectMeat"`
	EatMeat       bool   `json:"eatMeat"`
	EatNeighbors  bool   `json:"eatNeighbors"`
	EatGuilds     bool   `json:"eatGuilds"`
	EatRankings   bool   `json:"eatRankings"`
	ShareEgg      bool   `json:"shareEgg"`
}

type ScanTaskConfig struct {
	InterestedVeggies []string `json:"interestedVeggies"` // 想要搜的菜 ID 列表 (如 ["01CD", "01CC"])
	SelectedUids      []string `json:"selectedUids"`      // 目标人员 UID 列表 (如 ["024FA5FA", ...])
}

type RemoteConfig struct {
	Version string `json:"version"`

	// 对应 JSON 中的 "gamble"
	Gamble struct {
		MonthlyGuildReward struct {
			BetPrize    string `json:"bet_prize"`
			TicketPrize string `json:"ticket_prize"`
		} `json:"monthlyGuildReward"`
	} `json:"gamble"`

	// 对应 JSON 中的 "garden_veggie"
	GardenVeggie struct {
		PlantVeggie map[string]string `json:"plantVeggie"`
	} `json:"garden_veggie"`
}

type TargetUser struct {
	UID  int64  `json:"uid"`
	Name string `json:"name"`
}

type TargetsConfig struct {
	Version string                  `json:"version"`
	AllData map[string][]TargetUser `json:"all_data"`
}

// ------------------- 全局配置与全量映射 -------------------

var (
	purchasedUIDs = make(map[string]bool)
	uidMutex      sync.Mutex
)

var colorMap = map[string]string{
	"0202": "青色", "0203": "橙色", "0204": "紫色",
	"0205": "蓝色", "0206": "绿色", "0207": "白色", "0201": "红色",
}

var categoryMap = map[string]string{
	// 遗质
	"恐龙化石": "遗质4号位", "法老木乃伊": "遗质4号位", "粽子": "遗质4号位",
	// 机械
	"留声机": "机械5号位", "蒸汽机": "机械5号位", "座钟": "机械5号位",
	// 礼器
	"圣甲虫护身符": "礼器6号位", "四羊方尊": "礼器6号位", "重金皇冠": "礼器6号位",
	// 器皿
	"琉璃杯": "器皿1号位", "陶罐": "器皿1号位", "青铜器": "器皿1号位",
	// 乐器
	"编钟": "乐器2号位", "号角": "乐器2号位", "里拉琴": "乐器2号位",
	// 雕塑
	"兵俑": "雕塑3号位", "维纳斯": "雕塑3号位", "长信宫灯": "雕塑3号位",
}

var codeToName = map[string]string{
	"00AC4591": "琉璃杯", "00AC4592": "琉璃杯", "00AC4593": "琉璃杯", "00AC4594": "琉璃杯", "00AC4595": "琉璃杯",
	"00AC4596": "陶罐", "00AC4597": "陶罐", "00AC4598": "陶罐", "00AC4599": "陶罐", "00AC459A": "陶罐",
	"00AC459B": "青铜器", "00AC459C": "青铜器", "00AC459D": "青铜器", "00AC459E": "青铜器", "00AC459F": "青铜器",
	"00AC45A0": "编钟", "00AC45A1": "编钟", "00AC45A2": "编钟", "00AC45A3": "编钟", "00AC45A4": "编钟",
	"00AC45A5": "号角", "00AC45A6": "号角", "00AC45A7": "号角", "00AC45A8": "号角", "00AC45A9": "号角",
	"00AC45AA": "里拉琴", "00AC45AB": "里拉琴", "00AC45AC": "里拉琴", "00AC45AD": "里拉琴", "00AC45AE": "里拉琴",
	"00AC45AF": "兵俑", "00AC45B0": "兵俑", "00AC45B1": "兵俑", "00AC45B2": "兵俑", "00AC45B3": "兵俑",
	"00AC45B4": "维纳斯", "00AC45B5": "维纳斯", "00AC45B6": "维纳斯", "00AC45B7": "维纳斯", "00AC45B8": "维纳斯",
	"00AC45B9": "长信宫灯", "00AC45BA": "长信宫灯", "00AC45BB": "长信宫灯", "00AC45BC": "长信宫灯", "00AC45BD": "长信宫灯",
	"00AC45BE": "恐龙化石", "00AC45BF": "恐龙化石", "00AC45C0": "恐龙化石", "00AC45C1": "恐龙化石", "00AC45C2": "恐龙化石",
	"00AC45C3": "法老木乃伊", "00AC45C4": "法老木乃伊", "00AC45C5": "法老木乃伊", "00AC45C6": "法老木乃伊", "00AC45C7": "法老木乃伊",
	"00AC45C8": "粽子", "00AC45C9": "粽子", "00AC45CA": "粽子", "00AC45CB": "粽子", "00AC45CC": "粽子",
	"00AC45CD": "留声机", "00AC45CE": "留声机", "00AC45CF": "留声机", "00AC45D0": "留声机", "00AC45D1": "留声机",
	"00AC45D2": "蒸汽机", "00AC45D3": "蒸汽机", "00AC45D4": "蒸汽机", "00AC45D5": "蒸汽机", "00AC45D6": "蒸汽机",
	"00AC45D7": "座钟", "00AC45D8": "座钟", "00AC45D9": "座钟", "00AC45DA": "座钟", "00AC45DB": "座钟",
	"00AC45DC": "圣甲虫护身符", "00AC45DD": "圣甲虫护身符", "00AC45DE": "圣甲虫护身符", "00AC45DF": "圣甲虫护身符", "00AC45E0": "圣甲虫护身符",
	"00AC45E1": "四羊方尊", "00AC45E2": "四羊方尊", "00AC45E3": "四羊方尊", "00AC45E4": "四羊方尊", "00AC45E5": "四羊方尊",
	"00AC45E6": "重金皇冠", "00AC45E7": "重金皇冠", "00AC45E8": "重金皇冠", "00AC45E9": "重金皇冠", "00AC45EA": "重金皇冠",
}

var attrToName = map[string]string{
	"01": "战斗限时增加", "02": "战斗限时增加", "03": "战斗限时增加", "04": "战斗限时增加", "05": "战斗限时增加",
	"06": "研究速度增加", "07": "研究速度增加", "08": "研究速度增加", "09": "研究速度增加", "0A": "研究速度增加",
	"0B": "射程增加", "0C": "射程增加", "0D": "射程增加", "0E": "射程增加", "0F": "射程增加",
	"10": "移动速度", "11": "移动速度", "12": "移动速度", "13": "移动速度", "14": "移动速度",
	"15": "金币加成", "16": "金币加成", "17": "金币加成", "18": "金币加成", "19": "金币加成",
	"1A": "放置奖励增加", "1B": "放置奖励增加", "1C": "放置奖励增加", "1D": "放置奖励增加", "1E": "放置奖励增加",
	"1F": "铁镐恢复速度增加", "20": "铁镐恢复速度增加", "21": "铁镐恢复速度增加", "22": "铁镐恢复速度增加", "23": "铁镐恢复速度增加",
	"24": "菜市场卖货价格增加", "25": "菜市场卖货价格增加", "26": "菜市场卖货价格增加", "27": "菜市场卖货价格增加", "28": "菜市场卖货价格增加",
	"29": "矿石获得量增加", "2A": "矿石获得量增加", "2B": "矿石获得量增加", "2C": "矿石获得量增加", "2D": "矿石获得量增加",
	"2E": "减伤(PVE)", "2F": "减伤(PVE)", "30": "减伤(PVE)", "31": "减伤(PVE)", "32": "减伤(PVE)",
	"33": "Boss伤害减免", "34": "Boss伤害减免", "35": "Boss伤害减免", "36": "Boss伤害减免", "37": "Boss伤害减免",
	"38": "普通敌人伤害减免", "39": "普通敌人伤害减免", "3A": "普通敌人伤害减免", "3B": "普通敌人伤害减免", "3C": "普通敌人伤害减免",
	"3D": "击飞流生命", "3E": "击飞流生命", "3F": "击飞流生命", "40": "击飞流生命", "41": "击飞流生命",
	"42": "异常流生命", "43": "异常流生命", "44": "异常流生命", "45": "异常流生命", "46": "异常流生命",
	"47": "普攻流生命", "48": "普攻流生命", "49": "普攻流生命", "4A": "普攻流生命", "4B": "普攻流生命",
	"4C": "核爆流生命", "4D": "核爆流生命", "4E": "核爆流生命", "4F": "核爆流生命", "50": "核爆流生命",
	"51": "流血流生命", "52": "流血流生命", "53": "流血流生命", "54": "流血流生命", "55": "流血流生命",
	"56": "反击流生命", "57": "反击流生命", "58": "反击流生命", "59": "反击流生命", "5A": "反击流生命",
	"5B": "召唤流生命", "5C": "召唤流生命", "5D": "召唤流生命", "5E": "召唤流生命", "5F": "召唤流生命",
	"65": "普通攻击加成(后排)", "66": "普通攻击加成(后排)", "67": "普通攻击加成(后排)", "68": "普通攻击加成(后排)", "69": "普通攻击加成(后排)",
	"6A": "技能伤害加成(后排)", "6B": "技能伤害加成(后排)", "6C": "技能伤害加成(后排)", "6D": "技能伤害加成(后排)", "6E": "技能伤害加成(后排)",
	"6F": "宠物普通攻击加成(后排)", "70": "宠物普通攻击加成(后排)", "71": "宠物普通攻击加成(后排)", "72": "宠物普通攻击加成(后排)", "73": "宠物普通攻击加成(后排)",
	"74": "反伤抵抗", "75": "反伤抵抗", "76": "反伤抵抗", "77": "反伤抵抗", "78": "反伤抵抗",
	"79": "增伤(PVE)", "7A": "增伤(PVE)", "7B": "增伤(PVE)", "7C": "增伤(PVE)", "7D": "增伤(PVE)",
	"7E": "Boss伤害加成", "7F": "Boss伤害加成",
	"CC80": "Boss伤害加成", "CC81": "Boss伤害加成", "CC82": "Boss伤害加成",
	"CC83": "普通敌人伤害量增加", "CC84": "普通敌人伤害量增加", "CC85": "普通敌人伤害量增加", "CC86": "普通敌人伤害量增加", "CC87": "普通敌人伤害量增加",
	"CC88": "击飞流攻击力", "CC89": "击飞流攻击力", "CC8A": "击飞流攻击力", "CC8B": "击飞流攻击力", "CC8C": "击飞流攻击力",
	"CC8D": "异常流攻击力", "CC8E": "异常流攻击力", "CC8F": "异常流攻击力", "CC90": "异常流攻击力", "CC91": "异常流攻击力",
	"CC92": "普攻流攻击力", "CC93": "普攻流攻击力", "CC94": "普攻流攻击力", "CC95": "普攻流攻击力", "CC96": "普攻流攻击力",
	"CC97": "核爆流攻击力", "CC98": "核爆流攻击力", "CC99": "核爆流攻击力", "CC9A": "核爆流攻击力", "CC9B": "核爆流攻击力",
	"CC9C": "流血流攻击力", "CC9D": "流血流攻击力", "CC9E": "流血流攻击力", "CC9F": "流血流攻击力", "CCA0": "流血流攻击力",
	"CCA1": "反击流攻击力", "CCA2": "反击流攻击力", "CCA3": "反击流攻击力", "CCA4": "反击流攻击力", "CCA5": "反击流攻击力",
	"CCA6": "召唤流攻击力", "CCA7": "召唤流攻击力", "CCA8": "召唤流攻击力", "CCA9": "召唤流攻击力", "CCAA": "召唤流攻击力",
	"CCB0": "普通攻击加成(前排)", "CCB1": "普通攻击加成(前排)", "CCB2": "普通攻击加成(前排)", "CCB3": "普通攻击加成(前排)", "CCB4": "普通攻击加成(前排)",
	"CCB5": "技能伤害加加成(前排)", "CCB6": "技能伤害加成(前排)", "CCB7": "技能伤害加成(前排)", "CCB8": "技能伤害加成(前排)", "CCB9": "技能伤害加成(前排)",
	"CCBA": "宠物普通攻击加成(前排)", "CCBB": "宠物普通攻击加成(前排)", "CCBC": "宠物普通攻击加成(前排)", "CCBD": "宠物普通攻击加成(前排)", "CCBE": "宠物普通攻击加成(前排)",
	"CCBF": "反伤", "CCC0": "反伤", "CCC1": "反伤", "CCC2": "反伤", "CCC3": "反伤",
	"CCC9": "精准(前排)", "CCCA": "精准(前排)", "CCCB": "精准(前排)", "CCCC": "精准(前排)", "CCCD": "精准(前排)",
	"CCCE": "忽视闪避", "CCCF": "忽视闪避", "CCD0": "忽视闪避", "CCD1": "忽视闪避", "CCD2": "忽视闪避",
	"CCD3": "宠物伤害量加成(前排)", "CCD4": "宠物伤害量加成(前排)", "CCD5": "宠物伤害量加成(前排)", "CCD6": "宠物伤害量加成(前排)", "CCD7": "宠物伤害量加成(前排)",
	"CCD8": "PVP吸血", "CCD9": "PVP吸血", "CCDA": "PVP吸血", "CCDB": "PVP吸血", "CCDC": "PVP吸血",
	"CCE2": "增伤(PVP)", "CCE3": "增伤(PVP)", "CCE4": "增伤(PVP)", "CCE5": "增伤(PVP)", "CCE6": "增伤(PVP)",
	"CCE7": "角色技能伤害加成", "CCE8": "角色技能伤害加成", "CCE9": "角色技能伤害加成", "CCEA": "角色技能伤害加成", "CCEB": "角色技能伤害加成",
	"CCEC": "对击飞流伤害加成", "CCED": "对击飞流伤害加成", "CCEE": "对击飞流伤害加成", "CCEF": "对击飞流伤害加成", "CCF0": "对击飞流伤害加成",
	"CCF1": "对异常流伤害加成", "CCF2": "对异常流伤害加成", "CCF3": "对异常流伤害加成", "CCF4": "对异常流伤害加成", "CCF5": "对异常流伤害加成",
	"CCF6": "对普攻流伤害加成", "CCF7": "对普攻流伤害加成", "CCF8": "对普攻流伤害加成", "CCF9": "对普攻流伤害加成", "CCFA": "对普攻流伤害加成",
	"CCFB": "对核爆流伤害加成", "CCFC": "对核爆流伤害加成", "CCFD": "对核爆流伤害加成", "CCFE": "对核爆流伤害加成", "CCFF": "对核爆流伤害加成",
	"CD0100": "对流血流伤害加成", "CD0101": "对流血流伤害加成", "CD0102": "对流血流伤害加成", "CD0103": "对流血流伤害加成", "CD0104": "对流血流伤害加成",
	"CD0105": "对反击流伤害加成", "CD0106": "对反击流伤害加成", "CD0107": "对反击流伤害加成", "CD0108": "对反击流伤害加成", "CD0109": "对反击流伤害加成",
	"CD010A": "对召唤流伤害加成", "CD010B": "对召唤流伤害加成", "CD010C": "对召唤流伤害加成", "CD010D": "对召唤流伤害加成", "CD010E": "对召唤流伤害加成",
	"CD010F": "背包技能伤害加成(后排)", "CD0110": "背包技能伤害加成(后排)", "CD0111": "背包技能伤害加成(后排)", "CD0112": "背包技能伤害加成(后排)", "CD0113": "背包技能伤害加成(后排)",
	"CD0114": "精准(后排)", "CD0115": "精准(后排)", "CD0116": "精准(后排)", "CD0117": "精准(后排)", "CD0118": "精准(后排)",
	"CD0119": "幸运一击抵抗", "CD011A": "幸运一击抵抗", "CD011B": "幸运一击抵抗", "CD011C": "幸运一击抵抗", "CD011D": "幸运一击抵抗",
	"CD011E": "宠物伤害量加成(后排)", "CD011F": "宠物伤害量加成(后排)", "CD0120": "宠物伤害量加成(后排)", "CD0121": "宠物伤害量加成(后排)", "CD0122": "宠物伤害量加成(后排)",
	"CD0123": "PVP吸血抵抗", "CD0124": "PVP吸血抵抗", "CD0125": "PVP吸血抵抗", "CD0126": "PVP吸血抵抗", "CD0127": "PVP吸血抵抗",
	"CD0128": "普通攻击伤害减免", "CD0129": "普通攻击伤害减免", "CD012A": "普通攻击伤害减免", "CD012B": "普通攻击伤害减免", "CD012C": "普通攻击伤害减免",
	"CD012D": "减伤(PVP)", "CD012E": "减伤(PVP)", "CD012F": "减伤(PVP)", "CD0130": "减伤(PVP)", "CD0131": "减伤(PVP)",
	"CD0132": "角色技能伤害抵抗", "CD0133": "角色技能伤害抵抗", "CD0134": "角色技能伤害抵抗", "CD0135": "角色技能伤害抵抗", "CD0136": "角色技能伤害抵抗",
	"CD0137": "受击飞流伤害减免", "CD0138": "受击飞流伤害减免", "CD0139": "受击飞流伤害减免", "CD013A": "受击飞流伤害减免", "CD013B": "受击飞流伤害减免",
	"CD013C": "受异常流伤害减免", "CD013D": "受异常流伤害减免", "CD013E": "受异常流伤害减免", "CD013F": "受异常流伤害减免", "CD0140": "受异常流伤害减免",
	"CD0141": "受普攻流伤害减免", "CD0142": "受普攻流伤害减免", "CD0143": "受普攻流伤害减免", "CD0144": "受普攻流伤害减免", "CD0145": "受普攻流伤害减免",
	"CD0146": "受核爆流伤害减免", "CD0147": "受核爆流伤害减免", "CD0148": "受核爆流伤害减免", "CD0149": "受核爆流伤害减免", "CD014A": "受核爆流伤害减免",
	"CD014B": "受流血流伤害减免", "CD014C": "受流血流伤害减免", "CD014D": "受流血流伤害减免", "CD014E": "受流血流伤害减免", "CD014F": "受流血流伤害减免",
	"CD0150": "受反击流伤害减免", "CD0151": "受反击流伤害减免", "CD0152": "受反击流伤害减免", "CD0153": "受反击流伤害减免", "CD0154": "受反击流伤害减免",
	"CD0155": "受召唤流伤害减免", "CD0156": "受召唤流伤害减免", "CD0157": "受召唤流伤害减免", "CD0158": "受召唤流伤害减免", "CD0159": "受召唤流伤害减免",
}

// ------------------- 核心逻辑 -------------------
// 将解析逻辑封装，因为每次 Close 之后重新开启都需要重新绑定
func (a *App) initSunnyConfig() {
	a.sunny.SetGoCallback(
		nil, nil,
		func(conn SunnyNet.ConnWebSocket) {
			// 连接成功
			if conn.Type() == 1 {
				if strings.Contains(conn.URL(), "ws/v1/game") {
					a.activeConn = conn // 保存连接
					a.packetCount = 0
					a.mainAccountTried = false
					a.isConnected = true
					wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
						"message": "游戏连接成功",
						"type":    "info",
					})
				}
			}

			if conn.Type() == 2 { // 客户端发送的数据包
				if a.isPKActive && strings.Contains(conn.URL(), "ws/v1/game") {
					body := conn.Body()
					if len(body) < 4 {
						return
					}

					originalBody := make([]byte, len(body))
					copy(originalBody, body)
					bodyHex := strings.ToUpper(hex.EncodeToString(body))

					// 第一步：处理 UID 替换
					oldUIDHex := fmt.Sprintf("%08X", a.pkSourceUID)
					newUIDHex := fmt.Sprintf("%08X", a.pkTargetUID)

					uidReplaced := false
					if strings.Contains(bodyHex, oldUIDHex) {
						bodyHex = strings.ReplaceAll(bodyHex, oldUIDHex, newUIDHex)
						uidReplaced = true
					} else if strings.Contains(bodyHex, newUIDHex) {
						uidReplaced = true
					}

					// 第二步：处理区服特征
					if uidReplaced {
						replacedSrv := false
						oldTotalLen := len(body)
						oldHeader := binary.LittleEndian.Uint32(body[:4])

						// 构建目标区服的模式: 04 + 特征码 + 区服hex + 05CB
						buildTargetPattern := func(srv uint16) string {
							switch {
							case srv < 128:
								return fmt.Sprintf("04%02X05CB", srv)
							case srv < 256:
								return fmt.Sprintf("04CC%02X05CB", srv)
							default:
								return fmt.Sprintf("04CD%04X05CB", srv)
							}
						}

						// 构建源区服的替换串
						buildSourcePattern := func(srv uint16) string {
							switch {
							case srv < 128:
								return fmt.Sprintf("04%02X05CB", srv)
							case srv < 256:
								return fmt.Sprintf("04CC%02X05CB", srv)
							default:
								return fmt.Sprintf("04CD%04X05CB", srv)
							}
						}

						// 遍历目标区服范围（961-980）
						for s := a.pkTargetSrvMin; s <= a.pkTargetSrvMax; s++ {
							targetPattern := buildTargetPattern(s)

							if strings.Contains(bodyHex, targetPattern) {
								sourcePattern := buildSourcePattern(a.pkSourceSrvMin) // 901

								// 计算长度变化
								oldPartLen := len(targetPattern) / 2
								newPartLen := len(sourcePattern) / 2

								// 统计出现次数
								count := strings.Count(bodyHex, targetPattern)

								fmt.Printf("【发现】目标模式 %s 出现 %d 次\n", targetPattern, count)

								if oldPartLen != newPartLen {
									deltaBytes := newPartLen - oldPartLen
									totalDelta := deltaBytes * count
									fmt.Printf("【长度变化】%s(%d字节) -> %s(%d字节), 每次变化:%+d字节, 总计:%+d字节\n",
										targetPattern, oldPartLen, sourcePattern, newPartLen, deltaBytes, totalDelta)
								}

								// 替换所有匹配的模式
								bodyHex = strings.ReplaceAll(bodyHex, targetPattern, sourcePattern)
								replacedSrv = true
								fmt.Printf("【✅ 拦截成功】目标区服 %d(%s) -> 源区服 %d(%s)\n",
									s, targetPattern, a.pkSourceSrvMin, sourcePattern)
								break
							}
						}

						if uidReplaced || replacedSrv {
							newBody, err := hex.DecodeString(bodyHex)
							if err == nil {
								newTotalLen := len(newBody)

								if newTotalLen != oldTotalLen {
									newDataLen := newTotalLen - 4
									// 1. 更新开头长度头
									binary.LittleEndian.PutUint32(newBody[:4], uint32(newDataLen))

									// 2. 用字符串方式查找 "DE001F" 并更新前面的4字节长度
									newBodyHex := strings.ToUpper(hex.EncodeToString(newBody))
									de001fPos := strings.Index(newBodyHex, "DE001F")

									if de001fPos >= 4 { // 确保前面有2字节(4个十六进制字符)可写
										// 计算新长度值 = 开头长度头值 - 19
										newSecondLen := newDataLen - 19

										// 将新长度值转换为4字节大端序十六进制字符串
										secondLenBytes := make([]byte, 2)
										binary.BigEndian.PutUint16(secondLenBytes, uint16(newSecondLen))
										secondLenHex := strings.ToUpper(hex.EncodeToString(secondLenBytes))

										// 替换 DE001F 前面的8个字符
										pos := de001fPos - 4
										newBodyHex = newBodyHex[:pos] + secondLenHex + newBodyHex[de001fPos:]

										fmt.Printf("【更新DE001F前长度】原值未知 -> 新值 %d (0x%04X), 位置:%d\n",
											newSecondLen, newSecondLen, pos/2)

										// 重新解码回字节数组
										newBody, _ = hex.DecodeString(newBodyHex)
									} else {
										fmt.Printf("【警告】未找到 DE001F 位置\n")
									}

									fmt.Printf("【更新长度头】原总长:%d(头:%d) -> 新总长:%d(头:%d), 变化:%+d字节\n",
										oldTotalLen, oldHeader, newTotalLen, newDataLen, newTotalLen-oldTotalLen)
								}

								conn.SetBody(newBody)

								newHeader := binary.LittleEndian.Uint32(newBody[:4])
								fmt.Printf("【原始包】总长:%d 长度头:%08X(显示) 实际值:%d(数据长)\n",
									oldTotalLen, oldHeader, oldHeader)
								fmt.Printf("【修改后】总长:%d 长度头:%08X(显示) 实际值:%d(数据长)\n",
									newTotalLen, newHeader, newHeader)
							}
						}
					}
				}
			}

			// 收到服务器数据
			if conn.Type() == 3 {
				bodyBytes := conn.Body()
				if len(bodyBytes) == 0 {
					return
				}
				bodyHex := strings.ToUpper(hex.EncodeToString(bodyBytes))

				// 累加数据包计数
				a.packetCount++

				// 自动解码
				// 检查条件：包量达到、未尝试过解码、URL匹配
				if a.packetCount >= 10 && !a.mainAccountTried && strings.Contains(conn.URL(), "ws/v1/game") {
					a.mainAccountTried = true
					// 解码指令 (十六进制转字节)
					decryptCmd := "1B0000000500000000C8CC388F41016B83D721F24F13E202744AF85AF298CE"
					cmdBytes, _ := hex.DecodeString(decryptCmd)
					// 发送给服务器 (Type 2 代表二进制)
					conn.SendToServer(2, cmdBytes)
					fmt.Println("🚀 已向服务器发送协议解码指令")
				}

				if a.isPKActive && strings.Contains(conn.URL(), "ws/v1/game") {
					body := conn.Body()
					if len(body) < 4 {
						return
					}

					originalBody := make([]byte, len(body))
					copy(originalBody, body)
					bodyHex := strings.ToUpper(hex.EncodeToString(body))

					// 第一步：处理 UID 替换
					oldUIDHex := fmt.Sprintf("%08X", a.pkSourceUID)
					newUIDHex := fmt.Sprintf("%08X", a.pkTargetUID)

					uidReplaced := false
					if strings.Contains(bodyHex, oldUIDHex) {
						bodyHex = strings.ReplaceAll(bodyHex, oldUIDHex, newUIDHex)
						uidReplaced = true
					} else if strings.Contains(bodyHex, newUIDHex) {
						uidReplaced = true
					}

					// 第二步：处理区服特征
					if uidReplaced {
						replacedSrv := false
						oldTotalLen := len(body)
						oldHeader := binary.LittleEndian.Uint32(body[:4])

						// 构建目标区服的模式: 04 + 特征码 + 区服hex + 05CB
						buildTargetPattern := func(srv uint16) string {
							switch {
							case srv < 128:
								return fmt.Sprintf("04%02X05CB", srv)
							case srv < 256:
								return fmt.Sprintf("04CC%02X05CB", srv)
							default:
								return fmt.Sprintf("04CD%04X05CB", srv)
							}
						}

						// 构建源区服的替换串
						buildSourcePattern := func(srv uint16) string {
							switch {
							case srv < 128:
								return fmt.Sprintf("04%02X05CB", srv)
							case srv < 256:
								return fmt.Sprintf("04CC%02X05CB", srv)
							default:
								return fmt.Sprintf("04CD%04X05CB", srv)
							}
						}

						// 遍历目标区服范围（961-980）
						for s := a.pkTargetSrvMin; s <= a.pkTargetSrvMax; s++ {
							targetPattern := buildTargetPattern(s)

							if strings.Contains(bodyHex, targetPattern) {
								sourcePattern := buildSourcePattern(a.pkSourceSrvMin) // 901

								// 计算长度变化
								oldPartLen := len(targetPattern) / 2
								newPartLen := len(sourcePattern) / 2

								// 统计出现次数
								count := strings.Count(bodyHex, targetPattern)

								fmt.Printf("【发现】目标模式 %s 出现 %d 次\n", targetPattern, count)

								if oldPartLen != newPartLen {
									deltaBytes := newPartLen - oldPartLen
									totalDelta := deltaBytes * count
									fmt.Printf("【长度变化】%s(%d字节) -> %s(%d字节), 每次变化:%+d字节, 总计:%+d字节\n",
										targetPattern, oldPartLen, sourcePattern, newPartLen, deltaBytes, totalDelta)
								}

								// 替换所有匹配的模式
								bodyHex = strings.ReplaceAll(bodyHex, targetPattern, sourcePattern)
								replacedSrv = true
								fmt.Printf("【✅ 拦截成功】目标区服 %d(%s) -> 源区服 %d(%s)\n",
									s, targetPattern, a.pkSourceSrvMin, sourcePattern)
								break
							}
						}

						if uidReplaced || replacedSrv {
							newBody, err := hex.DecodeString(bodyHex)
							if err == nil {
								newTotalLen := len(newBody)

								if newTotalLen != oldTotalLen {
									newDataLen := newTotalLen - 4
									// 1. 更新开头长度头
									binary.LittleEndian.PutUint32(newBody[:4], uint32(newDataLen))

									// 2. 用字符串方式查找 "DE001F" 并更新前面的4字节长度
									newBodyHex := strings.ToUpper(hex.EncodeToString(newBody))
									de001fPos := strings.Index(newBodyHex, "DE001F")

									if de001fPos >= 4 { // 确保前面有2字节(4个十六进制字符)可写
										// 计算新长度值 = 开头长度头值 - 19
										newSecondLen := newDataLen - 19

										// 将新长度值转换为2字节大端序十六进制字符串
										secondLenBytes := make([]byte, 2)
										binary.BigEndian.PutUint16(secondLenBytes, uint16(newSecondLen))
										secondLenHex := strings.ToUpper(hex.EncodeToString(secondLenBytes))

										// 替换 DE001F 前面的4个字符
										pos := de001fPos - 4
										newBodyHex = newBodyHex[:pos] + secondLenHex + newBodyHex[de001fPos:]

										fmt.Printf("【更新DE001F前长度】原值未知 -> 新值 %d (0x%04X), 位置:%d\n",
											newSecondLen, newSecondLen, pos/2)

										// 重新解码回字节数组
										newBody, _ = hex.DecodeString(newBodyHex)
									} else {
										fmt.Printf("【警告】未找到 DE001F 位置\n")
									}

									fmt.Printf("【更新长度头】原总长:%d(头:%d) -> 新总长:%d(头:%d), 变化:%+d字节\n",
										oldTotalLen, oldHeader, newTotalLen, newDataLen, newTotalLen-oldTotalLen)
								}

								conn.SetBody(newBody)

								newHeader := binary.LittleEndian.Uint32(newBody[:4])
								fmt.Printf("【原始包】总长:%d 长度头:%08X(显示) 实际值:%d(数据长)\n",
									oldTotalLen, oldHeader, oldHeader)
								fmt.Printf("【修改后】总长:%d 长度头:%08X(显示) 实际值:%d(数据长)\n",
									newTotalLen, newHeader, newHeader)
							}
						}
					}
				}

				// --- 珍宝交易行返回包识别 (50E20000) ---
				if strings.Contains(bodyHex, "50E20000") && strings.Contains(conn.URL(), "ws/v1/game") {
					fmt.Println("🔍 检测到珍宝交易行数据流")

					// 调用解析函数 (传入当前连接和 Hex 字符串)
					items := a.ParseZhenBao(bodyHex)
					for _, item := range items {
						// A. 自动购买（内部会自动读取 a.activeRules）
						a.AutoBuyZhenBao(conn, []ZhenBaoItem{item})

						wailsRuntime.EventsEmit(a.ctx, "zhenbao_log", item)
					}
				}

				// --- 自己营地行返回包识别 (D3E50000) ---
				if strings.Contains(bodyHex, "D3E50000") && strings.Contains(conn.URL(), "ws/v1/game") {
					fmt.Println("🔍 检测到自己营地检测返回包")
					a.veggieMutex.Lock()
					a.lastGardenHex = bodyHex // 把这一长条包存下来
					a.veggieMutex.Unlock()
				}

				// --- 指定人员营地行返回包识别 (D1E50000) ---
				if strings.Contains(bodyHex, "D1E50000") && strings.Contains(conn.URL(), "ws/v1/game") {
					// 2. 关键判断：只有在助手流程开启时才分析
					if a.isScanningVeggie {
						fmt.Println("🎯 [扫菜助手] 正在解析目标营地数据...")

						// 🔒 使用专用的 scanMutex
						a.scanMutex.RLock()
						selectedVeggies := a.scanConfig.InterestedVeggies
						a.scanMutex.RUnlock()

						if len(selectedVeggies) > 0 {
							// 执行解析并推送到前端
							a.FindTargetVeggies(bodyHex, selectedVeggies)
						}
					} else {
						fmt.Println("玩家手动查看：仅通过数据包，不执行自动化逻辑")
					}
				}

				// 匹配到邻居列表数据包
				if strings.Contains(bodyHex, "D2E5000000") {
					// 2. 关键判断：只有在助手流程开启时才分析
					if a.isEatingMeat {
						fmt.Println("检测到自动化任务：开始解析邻居列表并自动吃肉")

						// 解析邻居列表
						neighbors := a.parseNeighborList(bodyHex)

						// 查找符合条件的肉并发送吃肉请求
						a.findTargetMeats(neighbors)
					} else {
						fmt.Println("玩家手动查看：仅通过数据包，不执行自动化逻辑")
					}
				}

				return
			}

		},
		nil,
	)
}

// 在 startup 函数中
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 初始化拿到远程json
	go a.LoadRemoteCommands()
	go a.LoadRemoteGardenConfig()

	mID := a.GetMachineID()
	fmt.Println("当前设备机器码:", mID)

	// 调用你修改后的 CheckAuthOnline，它现在返回三个值：授权状态、有效期、等级
	authorized, info, level := a.CheckAuthOnline(mID)

	if authorized {
		// 定义等级名称
		levelName := "普通用户"
		if level >= 2 {
			levelName = "高级用户"
		}

		// 1. 在终端打印包含等级的信息
		fmt.Printf("✅ 授权通过 | 等级: %s (%d) | 有效期至: %s\n", levelName, level, info)

		// 2. 异步通知前端
		go func() {
			time.Sleep(5 * time.Second)
			// 将等级也传给前端，方便前端做 UI 拦截
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("用户授权通过，有效期至: %s", info),
				"type":    "info", // 这样前端会显示绿色
			})
		}()

		a.ToggleCapture(true)
		go a.StartZhenbaoTimer()
	} else {
		fmt.Printf("❌ 未授权或已过期设备: [%s] (原因: %s)\n", mID, info)
	}
}

func (a *App) HexToFloat64(hexStr string) float64 {
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	byteData, _ := hex.DecodeString(hexStr)
	if len(byteData) != 8 {
		return 0
	}
	bits := binary.BigEndian.Uint64(byteData)
	return math.Float64frombits(bits)
}

// 跨服切磋
func (a *App) SetPKReplacement(tUID, tSrv, sUID, sSrv string) {
	// 1. 如果传入参数为空，直接关闭功能（防止空跑）
	if tUID == "" || sUID == "" {
		a.isPKActive = false
		return
	}

	// 2. 解析 UID (10进制字符串转 uint32)
	tu, _ := strconv.ParseUint(tUID, 10, 32)
	su, _ := strconv.ParseUint(sUID, 10, 32)

	// 3. 提取解析逻辑：处理 "1-100" 或 "85" 这种格式
	parseRange := func(srvStr string) (min, max uint16) {
		if strings.Contains(srvStr, "-") {
			parts := strings.Split(srvStr, "-")
			vMin, _ := strconv.ParseUint(parts[0], 10, 16)
			vMax, _ := strconv.ParseUint(parts[1], 10, 16)
			// 确保 min 永远小于等于 max
			if vMin > vMax {
				return uint16(vMax), uint16(vMin)
			}
			return uint16(vMin), uint16(vMax)
		}
		val, _ := strconv.ParseUint(srvStr, 10, 16)
		return uint16(val), uint16(val)
	}

	tMin, tMax := parseRange(tSrv)
	sMin, sMax := parseRange(sSrv)

	// 赋值给结构体
	a.pkTargetUID = uint32(tu)
	a.pkSourceUID = uint32(su)
	a.pkTargetSrvMin = tMin
	a.pkTargetSrvMax = tMax
	a.pkSourceSrvMin = sMin
	a.pkSourceSrvMax = sMax

	// 开启开关
	a.isPKActive = true

	fmt.Printf("【配置生效】UID: %d -> %d, 区服范围: %d-%d -> %d-%d\n",
		a.pkSourceUID, a.pkTargetUID, a.pkTargetSrvMin, a.pkTargetSrvMax, a.pkSourceSrvMin, a.pkSourceSrvMax)
}

func (a *App) StopPKMode() error {
	a.isPKActive = false
	// 清空内部数据，防止下次开启时逻辑混乱
	a.pkSourceUID = 0
	a.pkTargetUID = 0
	return nil
}

func (a *App) ParseZhenBao(bodyHex string) []ZhenBaoItem {
	// 1. 基础预处理
	bodyHex = strings.ToUpper(strings.ReplaceAll(bodyHex, " ", ""))
	posStart := strings.Index(bodyHex, "50E20000")
	if posStart == -1 {
		return nil
	}

	// 核心修改：每次收到新的完整列表包时，清空已购记录
	uidMutex.Lock()
	purchasedUIDs = make(map[string]bool)
	uidMutex.Unlock()

	// 裁剪掉前面的包头
	bodyHex = bodyHex[posStart-8:]
	var results []ZhenBaoItem
	if len(bodyHex) < 28 {
		return nil
	}
	userID := bodyHex[12:28]

	// 2. 分割物品块
	parts := strings.Split(bodyHex, "008400D9")
	if len(parts) <= 1 {
		return nil
	}

	for _, block := range parts[1:] {
		// 🚨 防御检查：长度过短直接跳过
		if len(block) < 80 {
			continue
		}
		item := ZhenBaoItem{}

		// --- A. 精准定位：寻找第 5 个 5F (_) 锚点 ---
		currSearchPos := 0
		count5F := 0
		target5FPos := -1
		for i := 0; i < 5; i++ {
			idx := strings.Index(block[currSearchPos:], "5F")
			if idx == -1 {
				break
			}
			actualPos := currSearchPos + idx
			count5F++
			if count5F == 5 {
				target5FPos = actualPos
				break
			}
			currSearchPos = actualPos + 2
		}

		if target5FPos == -1 {
			continue
		}

		// --- B. 动态扫描：确定 ASCII 结束位置 ---
		asciiEndPos := target5FPos + 2
		for asciiEndPos+2 <= len(block) {
			charHex := block[asciiEndPos : asciiEndPos+2]
			val, err := strconv.ParseUint(charHex, 16, 8)
			// 如果不是数字 ASCII (0-9 对应 48-57)，说明已经到了二进制价格区
			if err != nil || val < 48 || val > 57 {
				break
			}
			asciiEndPos += 2
		}

		// --- C. 解析 ASCII 文本段 (原始时间戳处理) ---
		if asciiEndPos > len(block) {
			continue
		}
		asciiHex := block[2:asciiEndPos]
		decodedBytes, err := hex.DecodeString(asciiHex)
		if err == nil {
			asciiStr := string(decodedBytes)
			segments := strings.Split(asciiStr, "_")
			if len(segments) >= 3 {
				// 解析原始时间戳字符串为数字
				tsVal, _ := strconv.ParseInt(segments[0], 10, 64)
				if tsVal > 9999999999 {
					tsVal /= 1000 // 毫秒转秒
				}

				// 【关键修改点】直接存储原始时间戳
				item.ListTime = tsVal
				item.UID = segments[1]
				item.District = segments[2]

				// 终端静默调试打印
				// fmt.Printf("\n📦 [解析物品] UID: %s | 原始时间戳: %d | 转换北京时间: %s\n",
				//	item.UID, item.ListTime, time.Unix(item.ListTime, 0).Format("2006-01-02 15:04:05"))
			}
		}

		// --- D. 解析价格 (无硬编码偏移版) ---
		priceBlock := block[asciiEndPos:]
		pos0285 := strings.Index(priceBlock, "0285")
		if pos0285 != -1 {
			searchArea := priceBlock[:pos0285]
			if idxCE := strings.Index(searchArea, "01CE"); idxCE != -1 {
				if len(searchArea) >= idxCE+12 {
					priceHex := searchArea[idxCE+4 : idxCE+12]
					val, _ := strconv.ParseUint(priceHex, 16, 64)
					item.Price = int64(val)
				}
			} else if idxCD := strings.Index(searchArea, "01CD"); idxCD != -1 {
				if len(searchArea) >= idxCD+8 {
					priceHex := searchArea[idxCD+4 : idxCD+8]
					val, _ := strconv.ParseUint(priceHex, 16, 64)
					item.Price = int64(val)
				}
			}
		}

		// --- E. 生成购买指令 ---
		productID := block[:asciiEndPos]
		purchaseCmdBody := "000000B21D" + userID + "0091D9" + productID
		lengthByte := (len(purchaseCmdBody) / 2) - 3
		item.PurchaseCmd = fmt.Sprintf("%02X%s", lengthByte, purchaseCmdBody)

		// --- F. 品质与类型识别 (包含公示判定所需 Type) ---
		for code, name := range codeToName {
			if strings.Contains(block, code) {
				if len(code) >= 2 {
					lastTwo := code[len(code)-2:]
					val, _ := strconv.ParseUint(lastTwo, 16, 8)
					quality := "未知"
					if val%5 == 3 {
						quality = "神话"
					} else if val%5 == 4 {
						quality = "超越"
					}
					category := categoryMap[name]
					if category == "" {
						category = "其他"
					}
					item.Type = fmt.Sprintf("%s-%s-%s", quality, category, name)
				}
				break
			}
		}

		// --- G. 词条解析 ---
		lastPos := 0
		for i := 0; i < 5; i++ {
			marker := fmt.Sprintf("%02X83", i)
			searchRange := block[lastPos:]
			relStart := strings.Index(searchRange, marker)
			if relStart != -1 {
				start83 := lastPos + relStart
				if start83+4 >= len(block) {
					break
				}
				relBlock := block[start83+4:]
				endCB := strings.Index(relBlock, "01CB")
				if endCB != -1 {
					attrID := relBlock[2:endCB]
					attr := Attribute{}
					attr.Name = attrToName[attrID]
					if attr.Name == "" {
						attr.Name = "未知(" + attrID + ")"
					}

					absCBPos := start83 + 4 + endCB
					if len(block) >= absCBPos+24 {
						valHex := block[absCBPos+4 : absCBPos+20]
						colorCode := block[absCBPos+20 : absCBPos+24]
						attr.Value = a.HexToFloat64(valHex) * 100
						attr.Color = colorMap[colorCode]
						item.Attributes = append(item.Attributes, attr)
						lastPos = absCBPos + 24
					} else {
						lastPos = start83 + 4
					}
				}
			}
		}

		// --- H. 最终拦截校验打印 (仅对超越品质) ---
		item.IsLocked = a.isLocked(item)
		if strings.Contains(item.Type, "超越") {
			locked := a.isLocked(item) // 此时 item.ListTime 已经是 int64，可以直接传入
			fmt.Printf("⚖️ [判定锁定] 类型:%s | 结果:%v\n", item.Type, locked)
		}

		// --- I. 添加到结果数组 ---
		results = append(results, item)
	}
	return results
}

func (a *App) UpdateRules(frontendRules []map[string]interface{}) {
	a.rulesMutex.Lock()
	defer a.rulesMutex.Unlock()

	var newRules []PurchaseRule
	for _, r := range frontendRules {
		minVal, _ := r["min"].(float64)
		priceVal, _ := r["price"].(float64)
		keyword, _ := r["keyword"].(string)
		quality, _ := r["quality"].(string) // 👈 接收品质字段

		var targetCats []string
		if cats, ok := r["targetCategories"].([]interface{}); ok {
			for _, c := range cats {
				targetCats = append(targetCats, c.(string))
			}
		}

		newRules = append(newRules, PurchaseRule{
			Name:             "动态规则",
			TargetQuality:    quality, // 👈 赋值品质
			TargetCategories: targetCats,
			TargetAttr:       keyword,
			MinAttrValue:     minVal,
			MaxPrice:         int64(priceVal),
		})
	}
	a.activeRules = newRules
	a.rulesPrinted = false
}

func (a *App) AutoBuyZhenBao(conn SunnyNet.ConnWebSocket, items []ZhenBaoItem) {
	a.rulesMutex.Lock()
	currentRules := a.activeRules
	// 检查是否需要打印：有数据、有规则、且之前没打印过
	shouldPrint := !a.rulesPrinted && len(items) > 0 && len(currentRules) > 0
	if shouldPrint {
		a.rulesPrinted = true // 标记为已打印
	}
	a.rulesMutex.Unlock()

	// --- 1. 打印规则摘要（加上 shouldPrint 判断） ---
	if shouldPrint {
		var sb strings.Builder
		sb.WriteString("📋 当前生效拦截规则：\n")
		for i, r := range currentRules {
			quality := r.TargetQuality
			if quality == "" {
				quality = "不限品质"
			}

			categories := "全部位"
			if len(r.TargetCategories) > 0 {
				categories = strings.Join(r.TargetCategories, "/")
			}

			keyword := r.TargetAttr
			if keyword == "" {
				keyword = "不限属性"
			}

			line := fmt.Sprintf(" [%d] 品质:%s | 部位:%s | %s >= %.1f%% | 预算:%d\n",
				i+1, quality, categories, keyword, r.MinAttrValue, r.MaxPrice)
			sb.WriteString(line)
		}
		sb.WriteString("--------------------------------------------------")

		// 推送到前端
		wailsRuntime.EventsEmit(a.ctx, "game_status", sb.String())
	}

	// --- 2. 执行购买逻辑 (保持不变) ---
	for _, item := range items {
		uidMutex.Lock()
		already := purchasedUIDs[item.UID]
		uidMutex.Unlock()
		if already {
			continue
		}

		for i, rule := range currentRules {
			if a.isMatch(item, rule) {
				uidMutex.Lock()
				purchasedUIDs[item.UID] = true
				uidMutex.Unlock()

				go func(it ZhenBaoItem, rl PurchaseRule, index int) {
					attrName := rl.TargetAttr
					if attrName == "" {
						attrName = "任意属性"
					}

					ruleDesc := fmt.Sprintf("规则[%d]: %s ≥ %.1f%%", index+1, attrName, rl.MinAttrValue)
					hitMsg := fmt.Sprintf("🎯 命中 %s\n正在购买: %s (价格: %d)", ruleDesc, it.Type, it.Price)

					wailsRuntime.EventsEmit(a.ctx, "game_status", hitMsg)

					a.executePurchase(conn, it, rl)
					time.Sleep(3 * time.Second)
					fmt.Println(hitMsg)
				}(item, rule, i) // 将索引 i 传进去

				break
			}
		}
	}
}

func (a *App) isLocked(item ZhenBaoItem) bool {
	if strings.Contains(item.Type, "神话") || !strings.Contains(item.Type, "超越") {
		return false
	}

	// ✅ 修复：不要使用 LoadLocation，避免 Windows 找不到时区库闪退
	loc := time.FixedZone("CST", 8*3600)

	listTime := time.Unix(item.ListTime, 0).In(loc)
	now := time.Now().In(loc)

	// 核心逻辑保持不变，但使用安全的 loc
	unlockTime := time.Date(listTime.Year(), listTime.Month(), listTime.Day(), 12, 0, 0, 0, loc)

	if listTime.Hour() < 12 {
		unlockTime = unlockTime.Add(24 * time.Hour)
	} else {
		unlockTime = unlockTime.Add(48 * time.Hour)
	}

	return now.Before(unlockTime)
}

func (a *App) isMatch(item ZhenBaoItem, rule PurchaseRule) bool {
	// --- 新增：展示期校验 ---
	if a.isLocked(item) {
		return false // 还在公示展示中，拦截购买行为
	}

	// 1. 预算校验
	if rule.MaxPrice > 0 && item.Price > rule.MaxPrice {
		return false
	}

	// 2. 品质校验 (核心新增)
	// 如果规则设定了品质（如 "神话"），且物品类型里不包含该文字，则排除
	if rule.TargetQuality != "" && !strings.Contains(item.Type, rule.TargetQuality) {
		return false
	}

	// 3. 部位校验
	if len(rule.TargetCategories) > 0 {
		matchedCat := false
		for _, cat := range rule.TargetCategories {
			if strings.Contains(item.Type, cat) {
				matchedCat = true
				break
			}
		}
		if !matchedCat {
			return false
		}
	}

	// 4. 属性数值校验
	if rule.TargetAttr == "" {
		return true
	}
	for _, attr := range item.Attributes {
		if attr.Name == rule.TargetAttr && attr.Value >= rule.MinAttrValue {
			return true
		}
	}
	return false
}

func (a *App) executePurchase(conn SunnyNet.ConnWebSocket, item ZhenBaoItem, rule PurchaseRule) {
	uidMutex.Lock()
	purchasedUIDs[item.UID] = true
	uidMutex.Unlock()

	fmt.Printf("🎯 [尝试秒杀] 规则: %s, 价格: %d\n", rule.Name, item.Price)

	cmdBytes, err := hex.DecodeString(item.PurchaseCmd)
	if err != nil {
		fmt.Println("❌ 指令解析失败:", err)
		return
	}

	success := conn.SendToServer(2, cmdBytes)

	if success {
		fmt.Println("✅ 购买指令已送达服务器")
	} else {
		fmt.Println("❌ 发送失败，可能连接已断开")
	}
}

// Purchase 是给前端手动购买调用的入口 (Wails 会自动导出为 purchase)
func (a *App) Purchase(cmdHex string) {
	// 1. 打印日志方便调试
	fmt.Printf("🖱️ [前端手动触发] 收到指令: %s\n", cmdHex)

	// 2. 解析指令
	cmdBytes, err := hex.DecodeString(cmdHex)
	if err != nil {
		fmt.Println("❌ 手动指令解析失败:", err)
		return
	}

	// 3. 找到当前的 WebSocket 连接并发送
	// 注意：你需要确保你的 App 结构体里保存了当前的 conn
	// 如果你没有保存全局 conn，你可能需要一个变量来记录最后一次活跃的 conn
	if a.activeConn != nil {
		success := a.activeConn.SendToServer(2, cmdBytes)
		if success {
			fmt.Println("✅ 手动秒杀指令已送达服务器")
		} else {
			fmt.Println("❌ 手动秒杀发送失败：连接可能已失效")
		}
	} else {
		fmt.Println("🚨 无法发送：当前没有活跃的 WebSocket 连接 (a.CurrentConn 为空)")
	}
}

// UpdateZhenbaoAutoRefresh 供前端调用，更新开关状态
func (a *App) UpdateZhenbaoAutoRefresh(status bool) {
	a.ZhenBaoAutoRefresh = status
	fmt.Printf("🔄 珍宝自动刷新开关状态已更新: %v\n", status)
}

// StartZhenbaoTimer 毫秒级精准定时
func (a *App) StartZhenbaoTimer() {
	go func() {
		beijing := time.FixedZone("CST", 8*3600)
		fmt.Println("📅 [系统] 精准定时任务已就绪，目标：每日 12:00:00.000")

		for {
			now := time.Now().In(beijing)

			// 1. 计算今天的 12:00:1
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 1, 0, beijing)

			// 2. 如果今天已经过了 12 点，就计算明天的 12 点
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// 3. 计算距离目标时刻还有多少时间
			waitDuration := time.Until(nextRun)
			fmt.Printf("⏳ 距离下次刷新还有: %v\n", waitDuration)

			// 4. 直接阻塞等待，直到时间到
			timer := time.NewTimer(waitDuration)
			<-timer.C // 这里会卡住，直到 12:00:00.000 瞬间唤醒

			// 5. 到点执行逻辑
			if a.ZhenBaoAutoRefresh && a.activeConn != nil {
				packetHex := "0E000000B01D00000022987F824100920000"
				data, _ := hex.DecodeString(packetHex)

				a.activeConn.SendToServer(2, data)

				wailsRuntime.EventsEmit(a.ctx, "game_status", "🕛 [精准任务] 12:00:00 毫秒级触发成功")
			}

			// 6. 稍微休息几秒再进入下一次循环计算，防止因微小误差导致连发
			time.Sleep(2 * time.Second)
		}
	}()
}

// ManualBuy 处理前端手动点击秒杀的请求
func (a *App) ManualBuy(purchaseCmd string) bool {
	// 1. 检查连接状态
	if a.activeConn == nil {
		fmt.Println("❌ 手动购买失败：未捕获到活跃的游戏连接")
		return false
	}

	// 2. 解析十六进制指令
	cmdBytes, err := hex.DecodeString(purchaseCmd)
	if err != nil {
		fmt.Printf("❌ 指令格式错误: %v\n", err)
		return false
	}

	// 3. 通过保存的连接发送指令 (Type 2 为二进制发送)
	success := a.activeConn.SendToServer(2, cmdBytes)

	if success {
		fmt.Printf("🎯 [手动秒杀] 指令已下达: %s\n", purchaseCmd[:10]+"...")
		return true
	} else {
		fmt.Println("❌ 手动购买发送失败，可能连接已断开")
		return false
	}
}

// GetAttributeNames 获取所有去重后的属性名称，供前端下拉框使用
func (a *App) GetAttributeNames() []string {
	nameMap := make(map[string]bool)
	var names []string
	for _, name := range attrToName {
		if !nameMap[name] {
			nameMap[name] = true
			names = append(names, name)
		}
	}
	return names
}

// GetCategoryNames 获取所有珍宝大类
func (a *App) GetCategoryNames() []string {
	// 根据你的 categoryMap 提取
	nameMap := make(map[string]bool)
	var categories []string
	for _, cat := range categoryMap {
		if !nameMap[cat] {
			nameMap[cat] = true
			categories = append(categories, cat)
		}
	}
	return categories
}

// 解析邻居列表
func (a *App) parseNeighborList(hexData string) []NeighborInfo {
	var neighbors []NeighborInfo
	pos := 0

	for pos < len(hexData) {
		// 查找UID开始的标记 (00CE)
		uidStart := strings.Index(hexData[pos:], "00CE")
		if uidStart == -1 {
			break
		}
		uidStart += pos
		neighbor := parseSingleNeighbor(hexData, uidStart)
		if neighbor.UID != "" {
			neighbors = append(neighbors, neighbor)
		}

		// 移动到下一个可能的位置
		pos = uidStart + 4
	}
	return neighbors
}

// 解析单个邻居信息
func parseSingleNeighbor(hexData string, startPos int) NeighborInfo {
	var neighbor NeighborInfo
	pos := startPos

	// 1. 安全读取 UID
	if pos+12 <= len(hexData) && hexData[pos:pos+4] == "00CE" {
		neighbor.UID = hexData[pos+4 : pos+12]
		pos += 12
	} else {
		return neighbor // 如果连UID头都对不上，直接返回空
	}

	// 2. 查找肉标记 (06xx07xx) - 增加安全保护
	pattern := `06(01|02|03)07(01|02|03)`
	re := regexp.MustCompile(pattern)
	match := re.FindStringIndex(hexData[pos:])

	// 🚨 修复核心：必须判断是否找到了匹配
	if match == nil {
		return neighbor // 没找到肉标记，提前结束，防止 match[0] 崩溃
	}

	// 记录肉标记在全局 hexData 中的绝对位置
	absoluteMatchPos := pos + match[0]

	// 检查长度是否足以读取 MeatType 和 MeatStatus (需要绝对位置往后至少 8 位)
	if absoluteMatchPos+8 <= len(hexData) {
		neighbor.MeatType = hexData[absoluteMatchPos+2 : absoluteMatchPos+4]
		neighbor.MeatStatus = hexData[absoluteMatchPos+6 : absoluteMatchPos+8]

		// 检查 pos-16 是否越界 (用于获取 pvp)
		if absoluteMatchPos >= 16 {
			neighbor.pvp = hexToInt(hexData[absoluteMatchPos-16 : absoluteMatchPos-14])
		}

		// 更新指针到肉标记之后
		pos = absoluteMatchPos + 8
	} else {
		return neighbor
	}

	// 3. 查找蛋标记 (0Cxx0Dxx)
	pattern = `0C[0-9A-F]{2}0D[0-9A-F]{2}`
	re1 := regexp.MustCompile(pattern)
	match1 := re1.FindStringIndex(hexData[pos:])

	if match1 != nil {
		eggPos := pos + match1[0]
		// 确保蛋标记后的数据够读
		if eggPos+4 <= len(hexData) {
			neighbor.HasEgg = hexData[eggPos+2 : eggPos+4]
			pos = eggPos + 8 // 更新指针
		}
	} else {
		neighbor.HasEgg = ""
	}

	return neighbor
}

// 查找目标肉类型
func (a *App) findTargetMeats(neighbors []NeighborInfo) {
	var smallMeatFound, mediumMeatFound, largeMeatFound bool

	for _, neighbor := range neighbors {
		// 检查肉的类型和状态（每种只发送一次）
		if !smallMeatFound && neighbor.MeatType == "01" && neighbor.MeatStatus == "02" {
			smallMeatFound = true
			cmdBytes, _ := hex.DecodeString("120000008417000000287A7F82410092CE" + neighbor.UID + "00")
			a.activeConn.SendToServer(2, cmdBytes)
			// fmt.Println("120000008417000000287A7F82410092CE" + neighbor.UID + "00")
			// time.Sleep(1 * time.Second)

		} else if !mediumMeatFound && neighbor.MeatType == "02" && neighbor.MeatStatus == "02" {
			mediumMeatFound = true
			cmdBytes, _ := hex.DecodeString("120000008417000000287A7F82410092CE" + neighbor.UID + "00")
			a.activeConn.SendToServer(2, cmdBytes)
			// time.Sleep(1 * time.Second)

		} else if !largeMeatFound && neighbor.MeatType == "03" && neighbor.MeatStatus == "02" {
			largeMeatFound = true
			cmdBytes, _ := hex.DecodeString("120000008417000000287A7F82410092CE" + neighbor.UID + "00")
			a.activeConn.SendToServer(2, cmdBytes)
			// time.Sleep(1 * time.Second)
		}
	}

}

// FindTargetVeggies 核心扫描逻辑
// hexData: 服务器返回的十六进制原始数据包
// selectedIds: 用户在前端勾选的蔬菜 ID 数组 (例如 ["01CD", "01CC"])
func (a *App) FindTargetVeggies(hexData string, selectedIds []string) []Veggie {
	a.gdMutex.RLock()
	config := a.gardenData
	a.gdMutex.RUnlock()

	var results []Veggie
	// 增加一个防御性打印
	if len(hexData) < 100 {
		return results
	}

	// 1. 定位 UID (特征码 00CE)
	uidIndex := strings.Index(hexData, "00CE")
	if uidIndex == -1 || uidIndex+12 > len(hexData) {
		// fmt.Println("❌ 未在包中找到 UID 引导码 00CE")
		return results
	}
	uidHex := hexData[uidIndex+4 : uidIndex+12]

	// 2. 遍历用户勾选的每一种蔬菜特征码
	for _, vID := range selectedIds {
		// 尝试从配置中获取该特征码对应的蔬菜名称等信息
		info, ok := config.Veggies[vID]
		if !ok {
			// 如果找不到配置，为了调试可以打印一下，或者给个默认名
			// fmt.Printf("⚠️ 警告: 特征码 %s 在配置库中无对应名称\n", vID)
			continue
		}

		// 3. 在包中查找所有匹配该特征码的位置
		searchPos := 0
		for {
			// 直接搜索特征码 vID
			idx := strings.Index(hexData[searchPos:], vID)
			if idx == -1 {
				break
			}

			foundPos := searchPos + idx
			// 确保找到的位置后面还有足够长度解析时间戳 (至少 56 字节)
			if foundPos+56 <= len(hexData) {

				// 转换时间戳：截取特征码后 40-56 位
				rawTimeHex := hexData[foundPos+40 : foundPos+56]
				matureTimestamp := int64(hexToInt(rawTimeHex)) + int64(info.Offset*60000)

				veggie := Veggie{
					UID:        uidHex,
					Name:       info.Name,
					VeggieType: vID,
					MatureTime: matureTimestamp,
				}

				// 核心：抓取坑位信息 (特征码前面的 4 位)
				if foundPos >= 4 {
					veggie.Pos = hexData[foundPos-4 : foundPos]
				}

				results = append(results, veggie)

				// 后台实时打印发现结果
				fmt.Printf("✅ [命中] 发现 %s! 坑位:%s 成熟时间:%s\n",
					veggie.Name,
					veggie.Pos,
					time.UnixMilli(matureTimestamp).Format("15:04:05"),
				)
			}
			// 🌟 纠错点：步进长度应为特征码 vID 的长度
			searchPos = foundPos + len(vID)
		}
	}

	// 4. 实时推送到前端
	if len(results) > 0 {
		fmt.Printf("🚀 准备推送 %d 棵菜到前端...\n", len(results))
		wailsRuntime.EventsEmit(a.ctx, "on_veggie_discovered", results)
	}

	return results
}

func (a *App) LoadRemoteGardenConfig() {
	// 💡 建议在 URL 后面加个随机数，防止 GitHub CDN 缓存导致下到旧的 JSON
	url := "https://raw.githubusercontent.com/dear510/cs-config/refs/heads/main/garden_veggie.json?t=" + strconv.FormatInt(time.Now().Unix(), 10)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)

	if err != nil {
		fmt.Println("⚠️ 远程配置拉取失败:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️ 远程文件返回错误状态码: %d\n", resp.StatusCode)
		return
	}

	// 1. 一次性读取全部内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("❌ 读取响应体失败:", err)
		return
	}

	// 2. 解析 JSON (此时 body 就在内存里，可以反复使用)
	var tempConfig VeggieConfigData
	if err := json.Unmarshal(body, &tempConfig); err != nil {
		fmt.Println("❌ JSON解析失败!")
		fmt.Println("报错内容:", err)
		fmt.Println("接收到的原始数据:", string(body))
		return
	}

	// 3. 赋值给全局变量
	a.gdMutex.Lock()
	a.gardenData = tempConfig
	a.gdMutex.Unlock()

	// 4. 打印结果
	fmt.Printf("✅ 菜园配置同步成功，当前品种数: %d\n", len(tempConfig.Veggies))
}

func (a *App) LoadRemoteCommands() {
	url := "https://raw.githubusercontent.com/dear510/cs-config/refs/heads/main/commands.json"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("❌ 无法获取远程指令集:", err)
		return
	}
	defer resp.Body.Close()

	var temp RemoteConfig
	if err := json.NewDecoder(resp.Body).Decode(&temp); err != nil {
		fmt.Println("❌ 解析远程指令失败:", err)
		return
	}

	a.rcMutex.Lock()
	a.remoteConfig = temp
	a.rcMutex.Unlock()
	fmt.Printf("✅ 远程指令集同步成功，版本: %s\n", temp.Version)
}

// 1. 提供菜园配置给前端
func (a *App) GetGardenData() VeggieConfigData {
	a.gdMutex.RLock()
	defer a.gdMutex.RUnlock()
	return a.gardenData
}

// 2. 提供远程大名单给前端 (假设你的配置存在 a.remoteConfig 里)
func (a *App) GetRemoteConfig() RemoteConfig {
	a.rcMutex.RLock()
	defer a.rcMutex.RUnlock()
	return a.remoteConfig
}

// GetRemoteTargets 获取云端配置玩家名单
func (a *App) GetRemoteTargets() (*TargetsConfig, error) {
	url := "https://raw.githubusercontent.com/dear510/cs-config/refs/heads/main/targets.json"

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var config TargetsConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (a *App) processGardenActionsSync(hexData string) {
	var wg sync.WaitGroup // 声明一个等待组
	// --- 1. 识别当前账号 UID ---
	reUID := regexp.MustCompile(`D3E50000`)
	matchUID := reUID.FindStringIndex(hexData)
	if matchUID == nil {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
			"message": "⚠️ 未能在数据包中找到 UID",
			"type":    "error",
		})
		return
	}
	uidHex := hexData[matchUID[1]+36 : matchUID[1]+44]
	uidDecimal, _ := strconv.ParseInt(uidHex, 16, 64)

	// --- 2. 获取正则与配置 ---
	a.gdMutex.RLock()
	remotePatterns := a.gardenData.Patterns
	remoteVeggies := a.gardenData.Veggies
	a.gdMutex.RUnlock()

	matchPattern := `00CD[0-9A-F]{4}(01CE00AAE[0-9A-F]{27}|01CD01F6[0-9A-F]{24})06CF[0-9A-F]{16}`
	if p, ok := remotePatterns["veggie_match"]; ok {
		matchPattern = p
	}
	reVeg := regexp.MustCompile(matchPattern)
	matches := reVeg.FindAllStringIndex(hexData, -1)

	wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
		"message": fmt.Sprintf("🔍 账号 %d 扫描完成，发现 %d 块土地", uidDecimal, len(matches)),
		"type":    "default",
	})

	currentTime := time.Now().Unix()
	tempTimestamps := [4]int64{0, 0, 0, 0}

	// --- 3. 循环解析每块土地 ---
	for i, m := range matches {
		if i >= 4 {
			break
		}

		startIdx := m[0]
		var endIdx int
		if i+1 < len(matches) {
			endIdx = matches[i+1][0]
		} else {
			endIdx = startIdx + 500
			if endIdx > len(hexData) {
				endIdx = len(hexData)
			}
		}
		block := hexData[startIdx:endIdx]
		pos := block[4:8]

		isGoldenTree := strings.Contains(block[8:24], "01CD01F6")
		var vType string
		if isGoldenTree {
			vType = "01CD01F6"
		} else {
			vType = block[8:20]
		}

		// 匹配品种
		info, ok := remoteVeggies[vType]
		if !ok {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("❌ 未知特征码 [%s]，请更新字典", vType),
				"type":    "warning",
			})
			continue
		}

		// 提取成熟时间
		reTime := regexp.MustCompile(`06CF([0-9A-F]{16})`)
		tMatch := reTime.FindStringSubmatch(block)
		if len(tMatch) < 2 {
			continue
		}

		rawUnix, _ := strconv.ParseInt(tMatch[1], 16, 64)
		matureTime := (rawUnix / 1000) + int64(info.Offset*60)
		tempTimestamps[i] = matureTime

		// --- 4. 成熟逻辑判定 ---
		if currentTime >= matureTime {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("🚀 %s 已成熟，执行收割...", info.Name, pos),
				"type":    "default",
			})
			if isGoldenTree {
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": fmt.Sprintf("🌳 %s 已成熟，准备开始 50 次灌注...", info.Name, pos),
					"type":    "default",
				})

				wg.Add(1) // 告诉程序：有一个异步任务要等
				go func() {
					defer wg.Done() // 任务结束后，计数减 1
					a.handleGoldenTreeHarvest(uidHex, pos, info.Name)
				}()
			} else {
				harvestCmd := fmt.Sprintf("14000000991700000012345F82410092CD%sCE%s", pos, uidHex)
				cmdBytes, _ := hex.DecodeString(harvestCmd)
				if a.activeConn != nil {
					a.activeConn.SendToServer(2, cmdBytes)
				}
				time.Sleep(time.Duration(600+rand.Intn(200)) * time.Millisecond)
			}
		} else {
			timeLeft := matureTime - currentTime
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("⏳ %s 预计成熟时间为%s (%s 后成熟)", info.Name, time.Unix(matureTime, 0).Format("15:04:05"), formatDuration(timeLeft)),
				"type":    "default",
			})
		}

		// --- 5. 灾害处理 ---
		if !isGoldenTree {
			reIncident := regexp.MustCompile(`02CF(000001[0-9A-F]{10})`)
			if incMatch := reIncident.FindStringSubmatch(block); len(incMatch) > 1 {
				startTime, _ := strconv.ParseInt(incMatch[1], 16, 64)
				startTime = startTime / 1000
				if currentTime >= startTime && (currentTime-startTime) < 3600 {
					wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
						"message": fmt.Sprintf("🐛 %s 发现灾害，自动处理中...", info.Name, pos),
						"type":    "warning",
					})
					fixCmd := fmt.Sprintf("15000000981700000012345F82410093CD%sCE%s01", pos, uidHex)
					cmdBytes, _ := hex.DecodeString(fixCmd)
					if a.activeConn != nil {
						a.activeConn.SendToServer(2, cmdBytes)
					}
					time.Sleep(time.Duration(5200+rand.Intn(300)) * time.Millisecond)
				}
			}
		}
	}

	// 核心更新：同步给前端倒计时组件
	a.vtMutex.Lock()
	a.veggieTimestamps = tempTimestamps
	a.vtMutex.Unlock()
	wg.Wait()
}

// 辅助函数：处理黄金树 50 次灌注
func (a *App) handleGoldenTreeHarvest(uid, pos, name string) {
	a.vtMutex.Lock()
	if a.harvestingPos == nil {
		a.harvestingPos = make(map[string]bool)
	}
	if a.harvestingPos[pos] {
		a.vtMutex.Unlock()
		return
	}
	a.harvestingPos[pos] = true
	a.vtMutex.Unlock()

	defer func() {
		a.vtMutex.Lock()
		delete(a.harvestingPos, pos)
		a.vtMutex.Unlock()
	}()

	// 阶段 1: 发送 50 次指令
	for i := 1; i <= 50; i++ {
		part1 := fmt.Sprintf("15000000951700000076543282410093CE%sCD%s01", uid, pos)
		cmd, _ := hex.DecodeString(part1)
		if a.activeConn != nil {
			a.activeConn.SendToServer(2, cmd)
		}
		time.Sleep(time.Duration(550+rand.Intn(150)) * time.Millisecond)
	}

	// 阶段 2: 最终收尾
	part2 := fmt.Sprintf("0F000000971700000087654F82410091CD%s", pos)
	finalCmd, _ := hex.DecodeString(part2)
	if a.activeConn != nil {
		a.activeConn.SendToServer(2, finalCmd)
	}

	wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
		"message": fmt.Sprintf("✅ %s 收割完成！", name),
		"type":    "success",
	})
}

// 预告还有多久成熟
func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return "即将成熟"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%d小时%d分%d秒", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%d分%d秒", m, s)
	}
	return fmt.Sprintf("%d秒", s)
}

// ToggleGardenLoop 开启或关闭挂机
func (a *App) ToggleGardenLoop(enable bool, config map[string]interface{}) {
	// 1. 立即同步配置仓库，解决延迟问题
	// 这里的 updateGardenConfigFromMap 必须确保能正确填充 a.gardenTask
	a.updateGardenConfigFromMap(config)

	if enable {
		a.gdMutex.Lock()
		if a.isGardenLoopRunning {
			a.gdMutex.Unlock()
			// 如果已在运行，仅同步新配置并通知前端
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": "🔄 挂机配置已实时同步更新 (当前轮次结束后生效)",
				"type":    "info",
			})
			return
		}

		// 初始化启动状态
		a.isGardenLoopRunning = true
		a.stopGardenLoop = make(chan struct{})
		a.gdMutex.Unlock()

		// 启动主循环协程
		go func() {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": "🚀 菜园自动挂机已启动",
				"type":    "success",
			})

			for {
				// 2. 每一轮循环开始，加读锁获取配置快照
				a.gdMutex.RLock()
				taskStruct := a.gardenTask
				a.gdMutex.RUnlock()

				// 3. 将 Struct 转回 Map 以兼容你现有的三个任务函数
				taskMap := make(map[string]interface{})
				j, _ := json.Marshal(taskStruct)
				json.Unmarshal(j, &taskMap)

				// --- 🟢 通知前端：新一轮巡查开始 ---
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "🔄 开始新一轮自动巡查...",
					"type":    "default",
					"time":    time.Now().Format("15:04:05"),
				})

				// 4. 执行原有的任务函数
				a.runMeatTask(taskMap)
				a.runVeggieTask(taskMap)
				a.runEggTask(taskMap)

				// --- ⚪ 通知前端：本轮巡查结束 ---
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 本轮巡查任务已全部完成，进入休眠...",
					"type":    "success",
					"time":    time.Now().Format("15:04:05"),
				})

				// 5. 等待 5 分钟或停止信号
				select {
				case <-time.After(305 * time.Second):
					continue
				case <-a.stopGardenLoop:
					return
				}
			}
		}()
	} else {
		// 6. 处理关闭逻辑
		a.gdMutex.Lock()
		if a.isGardenLoopRunning {
			close(a.stopGardenLoop)
			a.isGardenLoopRunning = false
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": "🛑 菜园自动挂机已停止",
				"type":    "warning",
			})
		}
		a.gdMutex.Unlock()
	}
}

// updateGardenConfigFromMap 负责将前端传来的 Map 安全同步到 gardenTask 结构体
func (a *App) updateGardenConfigFromMap(m map[string]interface{}) {
	// 使用 json 序列化反序列化是最稳妥的转换方式，自动处理类型
	jsonData, _ := json.Marshal(m)

	a.gdMutex.Lock()
	defer a.gdMutex.Unlock()

	// 将前端传来的 map 映射到 gardenTask 结构体中
	err := json.Unmarshal(jsonData, &a.gardenTask)
	if err != nil {
		fmt.Printf("❌ 任务配置解析失败: %v\n", err)
		return
	}
	fmt.Printf("✅ [配置同步] 当前任务状态: %+v\n", a.gardenTask)
}

// 提取出来的维护核心逻辑
func (a *App) executeGardenMaintenance(config map[string]interface{}) {
	if a.activeConn == nil {
		fmt.Println("⚠️ 挂机中：检测到连接断开，跳过本次巡查")
		return
	}

	// 执行收菜、种菜、除虫检测 吃肉收肉等内容
	a.runVeggieTask(config)
	// a.runMeatTask(config)

}

// ExecuteDailyTasks 执行日常任务流水线
// 该方法会被前端 Vue 通过 window.go.main.App.ExecuteDailyTasks 调用
func (a *App) ExecuteDailyTasks(tasks []TaskParam) {
	if a.isTaskRunning {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
			"message": "⚠️ 任务流水线已在运行中",
			"type":    "error",
		})
		return
	}

	// 开启唯一的一个异步协程，负责跑完整个任务列表
	go func() {
		a.isTaskRunning = true
		defer func() { a.isTaskRunning = false }()

		// 这里的循环会按照 taskIDs 的顺序一个一个跑
		for _, id := range tasks {
			switch id.ID {
			case "dungeon_all":
				a.runDungeonTask(id.Config)
				// 每个子任务结束后，可以在这里发一个阶段性完成的消息
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 副本任务已完成",
					"type":    "info",
				})

			case "guild_sign":
				// 紧跟其后执行工会签到
				a.runGuildTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 工会任务已完成",
					"type":    "info",
				})

			case "red_diamond":
				// 召唤类任务
				a.runRedDiamondTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 召唤任务已完成",
					"type":    "info",
				})

			case "reward_tasks":
				// 召唤类任务
				a.runRewardTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 每日任务已完成",
					"type":    "info",
				})

			case "garden_meat":
				// 菜园肉类任务
				a.runMeatTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 菜园肉类操作已完成",
					"type":    "info",
				})

			case "garden_veggie":
				// 菜园菜类任务
				a.runVeggieTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 菜园菜类操作已完成",
					"type":    "info",
				})

			case "garden_egg":
				// 菜园肉类任务
				a.runEggTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 菜园蛋类操作已完成",
					"type":    "info",
				})

			case "weekly_reward":
				// 周奖励
				a.runWeeklyRewardTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 周期活动奖励已领取/兑换",
					"type":    "info",
				})

			case "guild_pk":
				// 工会对决任务
				a.runGuildPKTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 工会对决任务已完成",
					"type":    "info",
				})

			case "race":
				// 竞技类
				a.runRaceTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 竞技类任务已完成",
					"type":    "info",
				})

			case "gamble":
				// 竞猜类
				a.runGambleTask(id.Config)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
					"message": "✅ 竞猜类任务已完成",
					"type":    "info",
				})

			}

			// 任务之间建议给 2 秒左右的喘息时间，模拟真人切换界面的动作
			time.Sleep(2 * time.Second)
		}

		// ⚠️ 只有这里才发 finished: true，按钮才会变回来
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
			"message":  "✨ 所有选中任务已全部执行完毕",
			"type":     "info",
			"finished": true,
		})
	}()
}

func (a *App) runDungeonTask(config map[string]interface{}) {
	// --- 1. 资源提取 ---
	if res, ok := config["res"].(map[string]interface{}); ok {
		// 领钥匙
		if getKeys, _ := res["getKeys"].(bool); getKeys {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
				"message": "🔑 正在领取每日钥匙 (10个)...",
				"type":    "default",
			})
			keys := []string{
				"0E0000004D040000000884F4814100926500", "0E0000004D040000000884F4814100926600",
				"0E0000004D040000000884F4814100926700", "0E0000004D040000000884F4814100926800",
				"0E0000004D040000000884F4814100926500", "0E0000004D040000000884F4814100926600",
				"0E0000004D040000000884F4814100926700", "0E0000004D040000000884F4814100926800",
				"0E0000004D040000000884F4814100926C00", "0E0000004D04000000307A2F124100926D00",
			}
			for _, p := range keys {
				if a.activeConn != nil {
					keyCmd, _ := hex.DecodeString(p)
					a.activeConn.SendToServer(2, keyCmd)
				}
				// 一行搞定：随机 900ms ~ 1100ms 延迟
				time.Sleep(time.Duration(900+rand.Intn(201)) * time.Millisecond)
			}
		}
		// 领稿子
		if getPicks, _ := res["getPicks"].(bool); getPicks {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
				"message": "⛏️ 正在领取免费稿子...",
				"type":    "default",
			})
			picks := []string{"0E0000004D040000000884F4814100920100", "0E0000004D040000000884F4814100920100"}
			for _, p := range picks {
				if a.activeConn != nil {
					cmdBytes, _ := hex.DecodeString(p)
					a.activeConn.SendToServer(2, cmdBytes)
				}
				// 一行搞定：随机 900ms ~ 1100ms 延迟
				time.Sleep(time.Duration(900+rand.Intn(201)) * time.Millisecond)
			}
		}
	}

	// --- 2. 副本挑战 ---
	if clears, ok := config["clears"].(map[string]interface{}); ok {
		dungeonTypes := []string{"boss", "gold", "goblin", "lamp", "west", "gear", "demon"}
		dungeonNames := map[string]string{"boss": "领主", "gold": "金币", "goblin": "哥布林", "lamp": "神灯", "west": "西游", "gear": "龙宫", "demon": "熔炉"}

		for _, t := range dungeonTypes {
			conf, ok := clears[t].(map[string]interface{})
			if !ok || !conf["active"].(bool) {
				continue
			}

			// 处理前端传来的数字
			count := 0
			if c, ok := conf["count"].(float64); ok {
				count = int(c)
			}
			level := 0
			if l, ok := conf["level"].(float64); ok {
				level = int(l)
			}

			// 发送开始执行日志
			logMsg := ""
			if t != "lamp" && t != "west" {
				logMsg = fmt.Sprintf("🚀 开始挑战[%s] - 第 %d 关 - 共 %d 次", dungeonNames[t], level, count)
			} else {
				logMsg = fmt.Sprintf("🚀 开始挑战[%s] - 共 %d 次", dungeonNames[t], count)
			}
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
				"message": logMsg,
				"type":    "msgType",
			})

			for i := 1; i <= count; i++ {
				// 这里的 buildDungeonPacket 需要你自己实现逻辑返回 hex 字符串
				packetHex := a.buildDungeonPacket(t, level)

				if packetHex != "" && a.activeConn != nil {
					cmdBytes, _ := hex.DecodeString(packetHex)
					a.activeConn.SendToServer(2, cmdBytes)

					wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
						"message": fmt.Sprintf("  > [%s] 第 %d/%d 次已发送", dungeonNames[t], i, count),
						"type":    "msgType",
					})
				}
				time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
			}
		}
	}
}

func (a *App) buildDungeonPacket(dType string, level int) string {
	// 1. 神灯和西游是特例
	if dType == "lamp" {
		return "0C000000A916000000309A7F82410090"
	}
	if dType == "west" {
		return "0C000000AC160000002818398F410090"
	}

	// 2. 龙宫和熔炉是第二例
	if dType == "gear" {
		header := "0D000000AD16000000307A2F12410091"
		levelHex := fmt.Sprintf("%02X", level)
		return header + levelHex
	}
	if dType == "demon" {
		header := "0D000000B216000000287A7F82410091"
		levelHex := fmt.Sprintf("%02X", level)
		return header + levelHex
	}

	// 3. 副本品种 ID
	typeID := ""
	switch dType {
	case "boss":
		typeID = "CD03E9"
	case "gold":
		typeID = "CD03EA"
	case "goblin":
		typeID = "CD03EB"
	}

	// 3. 关卡数变长逻辑
	var levelHex string
	if level >= 1 && level <= 127 {
		levelHex = fmt.Sprintf("%02X", level)
	} else if level <= 255 {
		levelHex = fmt.Sprintf("CC%02X", level)
	} else {
		low := level & 0xFF
		high := (level >> 8) & 0xFF
		levelHex = fmt.Sprintf("CD%02X%02X", low, high)
	}

	// 4. 组装 Body (不含首字节长度位)
	// 参照你给的逻辑：Body = 000000 + 固定位 + 品种ID + 关卡位 + 结尾00
	// 注意：你之前的例子 12000000... 说明前面有 3 个字节是 000000
	body := "000000AF1600000028723F82410093" + typeID + levelHex + "00"

	// 5. 计算长度：(Body字符串长度 / 2)
	// 按照你的逻辑：lengthByte = (len/2) - 3？
	// 不对，看你的例子：12000000...
	// 如果 body 是 "000000AF1600000028723F82410093CD03E9CCAA00" (长度 42 字符 = 21 字节)
	// 21 - 3 = 18。 十六进制 18 是 12。 完美对上！

	lengthByte := (len(body) / 2) - 3
	packet := fmt.Sprintf("%02X%s", lengthByte, body)

	return packet
}

func (a *App) runGuildTask(config map[string]interface{}) {
	// 1. 加入工会
	if sign, ok := config["joinGuild"].(bool); ok && sign {
		guildID := config["guildNum"].(float64)
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": fmt.Sprintf("正在加入工会: %d", int(guildID)),
			"type":    "default",
		})
		headerHex := "110000004616000000307A2F12410091CE"
		idHex := fmt.Sprintf("%08X", int(guildID))
		fullHex := headerHex + idHex
		// 发送签到封包
		cmd, _ := hex.DecodeString(fullHex)
		a.activeConn.SendToServer(2, cmd)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 2. 工会每日签到
	if sign, ok := config["guildSign"].(bool); ok && sign {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "📅 正在进行工会每日签到...",
			"type":    "default",
		})
		// 发送签到封包
		packet := "0C0000004C16000000F03D4782410090"
		sign, _ := hex.DecodeString(packet)
		a.activeConn.SendToServer(2, sign)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 3. 自动砍价处理
	if bargainActive, ok := config["bargain"].(bool); bargainActive && ok {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "🔪 正在执行工会自动砍价...",
			"type":    "default",
		})

		// 1. 获取当前 Unix 时间戳（秒）
		nowMilli := time.Now().UnixMilli()

		// 2. 直接转换为 16 位的大端序十六进制字符串
		// %016X 表示占位 16 位，不足前面补 0，大写
		timeHex := fmt.Sprintf("%016X", nowMilli)

		// 3. 组装最终封包
		// 固定前缀 + 时间戳Hex + 结尾 (根据你提供的示例，CF 后面直接跟时间戳)
		// 注意：你提供的原包 CF 后面是 0000019CAFFCEB5F，看起来像是一个长整数
		// 如果是直接替换最后16位：
		basePrefix := "15000000192E00000078F4298F410091CF"
		finalPacket := basePrefix + timeHex

		if a.activeConn != nil {
			bargainCmd, _ := hex.DecodeString(finalPacket)
			a.activeConn.SendToServer(2, bargainCmd)

			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
				"message": fmt.Sprintf("✅ 砍价已发送"),
				"type":    "msgType",
			})
		}
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 4. 挑战工会 Boss
	if bossActive, ok := config["boss"].(bool); bossActive && ok {
		// 提取次数
		count := 1
		if c, ok := config["bossCount"].(float64); ok {
			count = int(c)
		}

		// 提取关卡
		level := 1
		if l, ok := config["bossLevel"].(float64); ok {
			level = int(l)
		}

		wailsRuntime.EventsEmit(a.ctx, "daily_log", map[string]string{
			"message": fmt.Sprintf("🚀 开始挑战工会Boss - 第 %d 关 - 共 %d 次", level, count),
			"type":    "default",
		})

		// 将关卡数字转换为 2 位十六进制，例如 207 关 -> CF
		levelHex := fmt.Sprintf("%02X", level)

		for i := 1; i <= count; i++ {
			// 组装封包：将原始包末尾的 CF 替换为动态的 levelHex
			// 原始包：10000000AF16000000307A2F1241009307CC + CF + 00
			packetStr := "10000000AF16000000307A2F1241009307CC" + levelHex + "00"

			if a.activeConn != nil {
				bossCmd, _ := hex.DecodeString(packetStr)
				a.activeConn.SendToServer(2, bossCmd)

				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
					"message": fmt.Sprintf("  > [工会Boss %d关] 第 %d/%d 次已发送", level, i, count),
					"type":    "msgType",
				})
			}

			// 随机延迟 300ms-500ms，模拟正常点击
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}

	// 5. 购买药水
	if medActive, ok := config["medicine"].(bool); medActive && ok {
		count := 1
		if c, ok := config["medCount"].(float64); ok {
			count = int(c)
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": fmt.Sprintf("🧪 正在购买工会药水 - 数量: %d", count),
			"type":    "default",
		})

		// 原始封包 (3瓶时最后是 ...0300)
		// 你的描述是：910300 里 3 的位置，即倒数第三个字符
		// 这意味着最后 3 个字节是 91 03 00
		basePacket := "0D000000DA16000000B812398F410091120D000000DA16000000B812398F4100911314000000DB16000000401B398F41009413CE00030D91"

		// 将数量转为 2 位十六进制 (例如 5 变为 "05")
		countHex := fmt.Sprintf("%02X", count)

		// 拼接最后的部分：基础包 + 数量位 + 结尾的 "00"
		// 注意：如果 3 是倒数第三个字符，那么它属于倒数第二个字节 (XX 00)
		finalPacket := basePacket + countHex + "00"
		med, _ := hex.DecodeString(finalPacket)
		a.activeConn.SendToServer(2, med)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 6. 报名会战
	if sign, ok := config["registerGuildPK"].(bool); ok && sign {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "☑️ 正在报名会战...",
			"type":    "default",
		})
		// 发送签到封包
		packet := "0C0000009C1300000025263F82410090"
		sign, _ := hex.DecodeString(packet)
		a.activeConn.SendToServer(2, sign)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 7. 退出工会
	if sign, ok := config["quitGuild"].(bool); ok && sign {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "正在退出当前工会...",
			"type":    "default",
		})
		// 发送签到封包
		packet := "0C0000004716000000654A7F82410090"
		sign, _ := hex.DecodeString(packet)
		a.activeConn.SendToServer(2, sign)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}
}

func (a *App) runRedDiamondTask(config map[string]interface{}) {
	if a.ctx == nil {
		return
	}

	// 1. 商城领红宝石
	if val, ok := config["shopRuby"].(bool); ok && val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "💎 正在领取商城红宝石...", "type": "default"})
		packet := "0E000000EC0300000078F4298F41009202010E000000EC0300000078F4298F41009203010E000000EC0300000078F4298F41009204010D000000351700000060C27A8241009101"
		p, _ := hex.DecodeString(packet)
		a.activeConn.SendToServer(2, p)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)

	}

	// 2. 限购每日免费红宝石
	if val, ok := config["limitRuby"].(bool); ok && val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🎁 正在领取限购每日免费红宝石...", "type": "default"})
		packet := "0E000000EC0300000060C27A824100920101"
		p, _ := hex.DecodeString(packet)
		a.activeConn.SendToServer(2, p)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 3. 每日免费召唤 (包含3条指令)
	if val, ok := config["freeSummon"].(bool); ok && val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "✨ 正在执行每日免费召唤...", "type": "default"})
		packets := []string{
			"0E0000004D04000000285FB28C4100921000",
			"0E0000004D04000000285FB28C4100921100",
			"0E0000004D04000000285FB28C4100920F00",
		}
		for _, p := range packets {
			packet, _ := hex.DecodeString(p)
			a.activeConn.SendToServer(2, packet)
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}

	// 4. 召唤功能通用循环处理 (装备、技能、宠物)
	drawConfigs := []struct {
		key      string
		countKey string
		packet   string
		label    string
	}{
		{"equipDraw", "equipDrawCount", "1B000000321400000028607F82410092A56571756970A870726F5F64726177", "装备召唤"},
		{"skillDraw", "skillDrawCount", "1B0000003214000000287A2532410092A5736B696C6CA870726F5F64726177", "技能召唤"},
		{"petDraw", "petDrawCount", "1D0000003214000000307A2F12410092A7706172746E6572A870726F5F64726177", "宠物召唤"},
	}

	for _, cfg := range drawConfigs {
		if active, ok := config[cfg.key].(bool); ok && active {
			count := 1
			if c, ok := config[cfg.countKey].(float64); ok {
				count = int(c)
			}
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
				"message": fmt.Sprintf("🚀 开始执行%s (350抽) - 共 %d 次", cfg.label, count),
				"type":    "default",
			})
			for i := 1; i <= count; i++ {
				packet, _ := hex.DecodeString(cfg.packet)
				a.activeConn.SendToServer(2, packet)
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
					"message": fmt.Sprintf("  > [%s] 第 %d/%d 次已发送", cfg.label, i, count),
					"type":    "msgType",
				})
				time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
			}
		}
	}
}

func (a *App) runRewardTask(config map[string]interface{}) {
	if a.ctx == nil {
		return
	}

	// 1. 日常任务全领 (指令非常多，包含活跃度进度条)
	if val, _ := config["dailyQuest"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "📜 正在领取日常全额奖励...", "type": "default"})
		dailyPackets := []string{
			"110000001E14000000307A2F12410091CE030A36A9", "110000001E14000000307A2F12410091CE030A36AA",
			"110000001E14000000307A2F12410091CE030A36AB", "110000001E14000000307A2F12410091CE030A36AC",
			"110000001E14000000307A2F12410091CE030A36AD", "110000001E14000000307A2F12410091CE030A36AE",
			"110000001E14000000307A2F12410091CE030A36AF", "180000002114000000001D398F410092AA73686172655F67616D6501",
			"120000004D04000000307A2F1241009214CE030A36B0", "120000004D04000000307A2F1241009214CE030A36B1",
			"120000004D04000000307A2F1241009214CE030A36B2", "120000004D04000000307A2F1241009214CE030A36B3",
			"120000004D04000000307A2F1241009214CE030A36B4", "120000004D04000000307A2F1241009214CE030A36B5",
			"120000004D04000000307A2F1241009214CE030A36B6", "120000004D04000000307A2F1241009214CE030A36B7",
			"120000004D04000000307A2F1241009214CE030A36B8", "120000004D04000000307A2F1241009214CE030A36B9",
		}
		for _, p := range dailyPackets {
			packet, _ := hex.DecodeString(p)
			a.activeConn.SendToServer(2, packet)
			time.Sleep(time.Duration(1000+rand.Intn(201)) * time.Millisecond)
		}
	}

	// 2. 领卡任务
	if val, _ := config["card"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🔍 正在领取卡牌...", "type": "default"})
		packet, _ := hex.DecodeString("0E000000EC03000000A81F298F4100920B01")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 3. 领取邮件
	if val, _ := config["mail"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🔍 正在领取全部邮件...", "type": "default"})
		packet, _ := hex.DecodeString("0D0000003C1400000010C6408D41009100")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 4. 寻宝任务
	if val, _ := config["treasure"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🔍 正在领取寻宝任务奖励...", "type": "default"})
		packet, _ := hex.DecodeString("110000001E14000000B0CE288F410091CE030B53D1110000001E14000000B0CE288F410091CE030B53D2110000001E14000000B0CE288F410091CE030B53D3110000001E14000000B0CE288F410091CE030B53D4")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 5. 天空之城任务
	if val, _ := config["skyCity"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "☁️ 正在领取天空之城奖励...", "type": "default"})
		packet, _ := hex.DecodeString("110000001E14000000307A2F12410091CE030B4431110000001E14000000307A2F12410091CE030B4432110000001E14000000307A2F12410091CE030B4433110000001E14000000307A2F12410091CE030B4434")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 6. 回溯任务
	if val, _ := config["backtrack"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "⏳ 正在领取回溯任务奖励...", "type": "default"})
		packet, _ := hex.DecodeString("110000001E14000000307A2F12410091CE030B4819110000001E14000000307A2F12410091CE030B481A110000001E14000000307A2F12410091CE030B481B110000001E14000000307A2F12410091CE030B481C")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 7. 珍宝藏宝图
	if val, _ := config["treasureMap"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🗺️ 正在领取珍宝藏宝图...", "type": "default"})
		packet, _ := hex.DecodeString("0C000000072E000000307A2F12410090")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 8. 挖矿研究加速+蓝宝石
	if val, _ := config["mineAccelerate"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🧐 正在加速研究并领取免费蓝宝石...", "type": "default"})
		packet, _ := hex.DecodeString("0E0000004D0400000028729F824100926B00")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		p, _ := hex.DecodeString("0E0000004D04000000524A7F824100921200")
		a.activeConn.SendToServer(2, p)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 9. 兑换码
	if val, _ := config["promoCode"].(bool); val {
		// 1. 获取前端输入框的内容
		codeContent, _ := config["promoCodeContent"].(string)
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": fmt.Sprintf("🎟️ 正在兑换兑换码: %s...", codeContent), "type": "default"})
		hexString := a.BuildRedeemHex(codeContent)
		packet, _ := hex.DecodeString(hexString)
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 10. 周常及排名奖励 (这些通常是单条或短指令)
	rewards := []struct {
		key   string
		label string
		code  string
	}{
		{"weeklyTicket", "回溯周每日票子", "0E000000EC03000000C81B398F4100920701"},
		{"weeklyRuby", "召唤周每日红宝石", "0E000000EC0300000078F4298F4100920801"},
		{"weeklyMine", "挖矿周每日饼干稿子", "0E000000EC03000000307A2F1241009206010F0000004D04000000F03D4782410092CCAF00"},
		{"speedup", "狂飙排名奖励", "0D0000005D16000000F018398F410091020D0000005D16000000F018398F41009101"},
		{"mineReward", "挖矿排名奖励", "0D0000006416000000307A2F1241009101"},
	}

	for _, r := range rewards {
		if val, _ := config[r.key].(bool); val {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🎁 正在领取：" + r.label, "type": "default"})
			packet, _ := hex.DecodeString(r.code)
			a.activeConn.SendToServer(2, packet)
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}

	// 11. 连充夺宝
	if val, _ := config["spendReward"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "💰 正在领取连充夺宝奖励...", "type": "default"})
		packet, _ := hex.DecodeString("0C0000002D2E0000003019398F410090")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// 12. 领花种子
	if val, _ := config["collectFlowerSeed"].(bool); val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{"message": "🌸 正在领取种花种子...", "type": "default"})
		packet, _ := hex.DecodeString("0C0000002019000000001D398F410090")
		a.activeConn.SendToServer(2, packet)
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}
}

// BuildRedeemHex 按照你的规律构造十六进制字符串
func (a *App) BuildRedeemHex(code string) string {
	// 1. 将兑换码转为 Hex
	codeHex := hex.EncodeToString([]byte(code))

	// 2. 计算兑换码长度标识 (根据你抓到的规律，0xA0 是基数)
	// csgzjx6283 (10位) -> AA (0xA0 + 10)
	// 暑消凉意起 (15字节) -> AF (0xA0 + 15)
	codeLenFlag := fmt.Sprintf("%02X", 0xA0+len([]byte(code)))

	// 3. 拼接中间固定部分 + 长度标识 + 兑换码内容
	// 注意：这里去掉了开头的 4 字节长度，先算内容
	fixedBody := "9416000000736A7F82410091"
	mainContent := fixedBody + codeLenFlag + codeHex

	// 4. 计算总长度 (Length Byte)
	// mainContent 的长度是字符数，除以 2 才是字节数
	totalBytes := len(mainContent) / 2

	// 按照你说的逻辑：如果是 4 字节长度头，通常 hex 表现为：[长度] 00 00 00
	lengthPrefix := fmt.Sprintf("%02X000000", totalBytes)

	// 5. 组合最终 Hex 字符串
	finalHex := lengthPrefix + mainContent

	return strings.ToUpper(finalHex)
}

func (a *App) runMeatTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// --- 1. 收肉 ---
	if active, ok := config["collectMeat"].(bool); ok && active {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "🥩 正在收肉...",
			"type":    "default",
		})
		Packet := "0E0000008617000000C81B398F41009203000E0000008617000000C81B398F41009202000E0000008617000000307A2F124100920100"
		p, _ := hex.DecodeString(Packet)
		a.activeConn.SendToServer(2, p)
		// 设置较短的随机延迟，防止发包太快被服务器屏蔽
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

	// --- 2. 吃邻居肉 ---
	if active, ok := config["eatMeat"].(bool); ok && active {
		a.isEatingMeat = true // 开启全局拦截标记

		// 定义任务列表：前端 key, 打印消息, 指令 Hex
		meatTasks := []struct {
			Key     string
			Message string
			Packet  string
		}{
			{"eatNeighbors", "🍖 正在扫描 [邻居列表] 搜索肉类...", "0F0000002E1A00000036B279A1410093020164"},
			{"eatGuilds", "🛡️ 正在扫描 [工会列表] 搜索肉类...", "0F0000002E1A00000036B279A1410093030164"},
			{"eatRankings", "🏆 正在扫描 [排行榜列表] 搜索肉类...", "0F0000002E1A00000036B279A1410093040164"},
		}

		for _, task := range meatTasks {
			// 检查前端是否勾选了对应的子选项
			if selected, ok := config[task.Key].(bool); ok && selected {
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
					"message": task.Message,
					"type":    "default",
				})

				if a.activeConn != nil {
					req, _ := hex.DecodeString(task.Packet)
					a.activeConn.SendToServer(2, req)

					// 每发送一种列表，给 3 秒的处理时间
					// 此时 initSunnyConfig 收到 D2E5 包会触发 parseNeighborList 和吃肉逻辑
					time.Sleep(3 * time.Second)
				}
			}
		}

		// 所有选中的列表都扫完了
		a.isEatingMeat = false
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}

}

func (a *App) runVeggieTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// --- 第一阶段：环境刷新与同步处理 (收菜、除虫、浇水) ---
	if active, ok := config["collectVeggie"].(bool); ok && active {
		a.veggieMutex.Lock()
		a.lastGardenHex = "" // 清空旧包
		a.veggieMutex.Unlock()

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "🌱 正在刷新菜园状态...",
			"type":    "default",
		})

		a.isCheckingVeggie = true
		// 1. 发送访问营地指令 (2D1A)
		Packet := "0C0000002D1A000000401B398F410090"
		p, _ := hex.DecodeString(Packet)
		a.activeConn.SendToServer(2, p)

		// 2. 轮询等待长包数据返回 (最多等 4 秒)
		var targetHex string
		for i := 0; i < 40; i++ {
			a.veggieMutex.Lock()
			if a.lastGardenHex != "" {
				targetHex = a.lastGardenHex
				a.veggieMutex.Unlock()
				break
			}
			a.veggieMutex.Unlock()
			time.Sleep(100 * time.Millisecond)
		}

		// 3. 如果拿到了包，执行同步处理函数 (这里会跑完所有收菜和除虫 Sleep 才会返回)
		if targetHex != "" {
			a.processGardenActionsSync(targetHex)
		} else {
			fmt.Println("⚠️ 未收到菜园数据包回复")
		}
		a.isCheckingVeggie = false
	}

	// --- 第二阶段：种菜逻辑 (在上述操作全部结束后执行) ---
	if active, ok := config["plantVeggie"].(bool); ok && active {
		// 1. 检查缓存时间 (processGardenActionsSync 内部会更新 a.veggieTimestamps)
		a.vtMutex.RLock()
		now := time.Now().Unix()
		hasData, allInFuture := false, true
		for _, ts := range a.veggieTimestamps {
			if ts > 0 {
				hasData = true
			}
			if ts <= now {
				allInFuture = false
			}
		}
		a.vtMutex.RUnlock()

		if hasData && allInFuture {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": "🚫 菜园尚未成熟，跳过种菜操作",
				"type":    "default",
			})
			return
		}

		// 2. 自动购买种子逻辑
		if buyActive, ok := config["buySeeds"].(bool); ok && buyActive {
			veggieKey, _ := config["veggieType"].(string)
			seedPackets := map[string]string{
				"luobo":   "12000000DB16000000B812398F41009405CD01F50400",
				"xiaomai": "12000000DB16000000B812398F41009405CD01F60400",
				"baicai":  "12000000DB16000000B812398F41009405CD01F70400",
				"huluobo": "12000000DB16000000B812398F41009405CD01F80400",
				"yumi":    "12000000DB16000000B812398F41009405CD01F90400",
				"nangua":  "12000000DB16000000B812398F41009405CD01FA0400",
			}

			if packetHex, exists := seedPackets[veggieKey]; exists {
				wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
					"message": "🛒 正在自动补货种子...",
					"type":    "default",
				})
				p, _ := hex.DecodeString(packetHex)
				a.activeConn.SendToServer(2, p)
				time.Sleep(1500 * time.Millisecond)
			}
		}

		// 3. 执行种植流程
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "🌱 正在种菜...",
			"type":    "default",
		})
		veggieKey, _ := config["veggieType"].(string)
		a.rcMutex.RLock()
		prefix := a.remoteConfig.GardenVeggie.PlantVeggie[veggieKey]
		a.rcMutex.RUnlock()

		for i := 0; i < 4; i++ {
			slot := fmt.Sprintf("%02d", i)
			p, _ := hex.DecodeString(prefix + slot)
			a.activeConn.SendToServer(2, p)
			time.Sleep(1300 * time.Millisecond)
		}
	}

	// --- 第三阶段：扫菜巡逻逻辑 ---
	if active, ok := config["scanVeggie"].(bool); ok && active {
		// 1. 开启扫菜解析开关
		a.isScanningVeggie = true
		// 2. 解析 UID 列表 (处理 JSON 数字转 float64 的坑)
		var targetUids []int64 // 直接存 int64 更稳妥
		if uids, ok := config["selectedUids"].([]interface{}); ok {
			for _, u := range uids {
				// 🌟 关键：JSON 数字在 map[string]interface{} 中默认是 float64
				if val, ok := u.(float64); ok {
					targetUids = append(targetUids, int64(val))
				} else if valStr, ok := u.(string); ok {
					// 兼容字符串格式
					id, _ := strconv.ParseInt(valStr, 10, 64)
					if id > 0 {
						targetUids = append(targetUids, id)
					}
				}
			}
		}

		// 2. 🌟 关键修复：解析感兴趣的蔬菜 ID 列表 (用于解析包)
		var interestedVeggies []string
		if ivs, ok := config["interestedVeggies"].([]interface{}); ok {
			for _, v := range ivs {
				// 蔬菜 ID 通常是字符串 (如 "01CD")，但也可能是数字
				if valStr, ok := v.(string); ok {
					interestedVeggies = append(interestedVeggies, valStr)
				} else if valFloat, ok := v.(float64); ok {
					interestedVeggies = append(interestedVeggies, fmt.Sprintf("%04X", int64(valFloat)))
				}
			}
		}
		// 3. 将解析出来的名单存入内存，供拦截器使用
		a.scanMutex.Lock()
		a.scanConfig.SelectedUids = []string{} // 清空旧的
		for _, id := range targetUids {
			a.scanConfig.SelectedUids = append(a.scanConfig.SelectedUids, fmt.Sprint(id))
		}
		a.scanConfig.InterestedVeggies = interestedVeggies // 存入正确断言后的名单
		a.scanMutex.Unlock()

		fmt.Println("📍 待扫描 UID 列表 (十进制):", targetUids)
		fmt.Printf("🥬 感兴趣的蔬菜 ID: %v (数量: %d)\n", interestedVeggies, len(interestedVeggies))

		if len(targetUids) > 0 {
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("🔍 开始扫菜巡逻: 共有 %d 个目标", len(targetUids)),
				"type":    "default",
			})

			// 3. 依次发送访问指令
			for i, uidInt := range targetUids {
				select {
				case <-a.stopGardenLoop:
					fmt.Println("🛑 扫菜巡逻被手动中止")
					a.isScanningVeggie = false
					return
				default:
					// 🌟 修正：现在 uidInt 已经是正确的 int64 了
					// %08X 会将其转为 8 位大写的十六进制 (如 024FA5FA)
					uidHex := fmt.Sprintf("%08X", uidInt)
					packetHex := "110000002F1A00000009253F82410091CE" + uidHex

					wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
						"message": fmt.Sprintf("📡 正在扫描(%d/%d): %d 的菜园", i+1, len(targetUids), uidInt),
						"type":    "default",
					})

					p, _ := hex.DecodeString(packetHex)
					a.activeConn.SendToServer(2, p)

					// 每条包间隔，你设置的 0.5s~0.9s
					time.Sleep(1500 * time.Millisecond)
				}
			}

			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": "🏁 扫菜巡逻已全部完成",
				"type":    "success",
			})
		}

		// 扫描结束，多等几秒让最后一个包返回，再关闭解析开关
		time.Sleep(5 * time.Second)
		a.isScanningVeggie = false
	}
}

func (a *App) runEggTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// --- 1. 营地吆喝 ---
	if active, ok := config["shareEgg"].(bool); ok && active {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "📢 正在吆喝...",
			"type":    "default",
		})
		Packet := "0F0000004D04000000B03F2D8F410092CCB100"
		p, _ := hex.DecodeString(Packet)
		a.activeConn.SendToServer(2, p)
		// 设置较短的随机延迟，防止发包太快被服务器屏蔽
		time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
	}
}

func (a *App) runWeeklyRewardTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// --- 1. 集体挖矿奖励 ---
	if active, ok := config["groupMineReward"].(bool); ok && active {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "⛏️ 正在执行集体挖矿奖励领取...",
			"type":    "default",
		})
		// 基础包体（去掉最后一位字节）
		basePacket := "0E0000004E14000000301C398F410091CC"
		// 领取范围是从 C9 到 DC
		// C9 (201) -> DC (220)
		for i := 0xC9; i <= 0xDC; i++ {
			// 动态生成完整的 Hex 字符串
			hexStr := fmt.Sprintf("%s%02X", basePacket, i)

			p, err := hex.DecodeString(hexStr)
			if err != nil {
				fmt.Println("Hex转换失败:", err)
				continue
			}
			// 发送到服务器
			a.activeConn.SendToServer(2, p)
			// 设置较短的随机延迟，防止发包太快被服务器屏蔽
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "集体挖矿奖励领取完毕",
			"type":    "success",
		})
	}

	// --- 2. 天空之城翻牌奖励 ---
	if active, ok := config["skycityRewards"].(bool); ok && active {
		// 使用反引号存储原始抓包数据，方便以后维护和替换
		rawPackets := `
		0D0000003A160000003817398F41009103
		0D0000003C160000003817398F41009100
		0D0000003C160000003817398F41009101
		0D0000003C160000003817398F41009102
		0D0000003C160000003817398F41009103
		0D0000003C160000003817398F41009104
		0D0000003C160000003817398F41009105
		0D0000003C160000003817398F41009106
		0D0000003C160000003817398F41009107
		0C0000003D160000003817398F410090
		0D0000003A160000003817398F41009101
		0D0000003C160000003817398F41009100
		0D0000003C160000003817398F41009101
		0D0000003C160000003817398F41009102
		0D0000003C160000003817398F41009103
		0D0000003C160000003817398F41009104
		0D0000003C160000003817398F41009105
		0D0000003C160000003817398F41009106
		0D0000003C160000003817398F41009107
		0C0000003D160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0D0000003A160000003817398F41009103
		0D0000003C160000003817398F41009100
		0D0000003C160000003817398F41009101
		0D0000003C160000003817398F41009102
		0D0000003C160000003817398F41009103
		0D0000003C160000003817398F41009104
		0D0000003C160000003817398F41009105
		0D0000003C160000003817398F41009106
		0D0000003C160000003817398F41009107
		0D0000003C160000003817398F41009108
		0D0000003C160000003817398F41009109
		0C0000003D160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0C0000003B160000003817398F410090
		0D0000003A160000003817398F41009101
		0D0000003C160000003817398F41009100
		0D0000003C160000003817398F41009101
		0D0000003C160000003817398F41009102
		0D0000003C160000003817398F41009103
		0D0000003C160000003817398F41009104
		0D0000003C160000003817398F41009105
		0D0000003C160000003817398F41009106
		0C0000003B160000003817398F410090
		0D0000003C160000003817398F41009107
		0D0000003C160000003817398F41009108
		0C0000003B160000003817398F410090
		0D0000003C160000003817398F41009109
		0C0000003D160000003817398F410090
		0D0000003A1600000078F4298F41009102
		0D0000003C160000003817398F41009100
		0D0000003C160000003817398F41009101
		0D0000003C160000003817398F41009102
		0D0000003C160000003817398F41009103
		0D0000003C160000003817398F41009104
		0C0000003B160000003817398F410090
		0D0000003C160000003817398F41009105
        `

		// 将字符串切割成数组，并过滤掉空格/空行
		lines := strings.Split(strings.TrimSpace(rawPackets), "\n")

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "☁️ 正在执行天空之城翻奖励任务...",
			"type":    "default",
		})

		for _, hexStr := range lines {
			cleanHex := strings.TrimSpace(hexStr)
			if cleanHex == "" {
				continue
			}

			p, _ := hex.DecodeString(cleanHex)
			a.activeConn.SendToServer(2, p)

			// 翻贝壳间隔：如果你想快点，可以改成 100+rand.Intn(50)
			time.Sleep(time.Duration(300+rand.Intn(100)) * time.Millisecond)
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "✅ 天空之城翻奖励执行完毕",
			"type":    "success",
		})
	}

	// --- 3. 海王翻贝壳 ---
	if active, ok := config["seashell"].(bool); ok && active {
		// 将你提供的长串指令按顺序存入数组
		packets := []string{
			"0D000000B118000000C017398F41009131", // 竞猜奖励
			"0D0000009D180000009818398F41009100", // 翻开位置0
			"0D0000009D18000000401B398F41009101", // 翻开位置1
			"0D0000009D18000000401B398F41009102",
			"0D0000009D18000000401B398F41009103",
			"0D0000009D18000000401B398F41009104",
			"0D0000009D18000000401B398F41009105",
			"0D0000009D18000000401B398F41009106",
			"0D0000009D18000000401B398F41009107",
			"0D0000009D18000000401B398F41009108",
			"0D0000009D180000009818398F41009100", // 第二次循环起点
			"0D0000009D18000000401B398F41009101",
			"0D0000009D18000000401B398F41009102",
			"0D0000009D18000000401B398F41009103",
			"0D0000009D18000000401B398F41009104",
			"0D0000009D18000000401B398F41009105",
			"0D0000009D18000000401B398F41009106",
			"0D0000009D18000000401B398F41009107",
			"0D0000009D18000000401B398F41009108",
			"0D0000009D180000009818398F41009100", // 第三次循环起点
			"0D0000009D18000000401B398F41009101",
			"0D0000009D18000000401B398F41009102",
			"0D0000009D18000000401B398F41009103",
			"0D0000009D18000000401B398F41009104",
			"0D0000009D18000000401B398F41009105",
			"0D0000009D18000000401B398F41009106",
			"0D0000009D18000000401B398F41009107",
			"0D0000009D18000000401B398F41009108",
			"0D0000009D180000009818398F41009100", // 第四次循环起点
			"0D0000009D18000000401B398F41009101",
			"0D0000009D18000000401B398F41009102",
			"0D0000009D18000000401B398F41009103",
			"0D0000009D18000000401B398F41009104",
			"0D0000009D18000000401B398F41009105",
			"0D0000009D18000000401B398F41009106",
			"0D0000009D18000000401B398F41009107",
			"0D0000009D18000000401B398F41009108",
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "🐚 正在执行海王翻贝壳...",
			"type":    "default",
		})

		for _, hexStr := range packets {
			// 确保有些包长度不一致时也能正常解析
			p, err := hex.DecodeString(hexStr)
			if err != nil {
				fmt.Printf("⚠️ 封包解析跳过: %s, 错误: %v\n", hexStr, err)
				continue
			}

			// 发送到服务器
			a.activeConn.SendToServer(2, p)

			// 设置一个固定的微小延迟，防止发包粘连
			time.Sleep(time.Duration(300+rand.Intn(100)) * time.Millisecond)
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "✅ 海王翻贝壳执行完毕",
			"type":    "success",
		})
	}

	// --- 4. 召唤2w抽奖励 ---
	if active, ok := config["redDiamondDrawReward"].(bool); ok && active {
		// 将你提供的长串指令按顺序存入数组
		packets := []string{
			"110000001E14000000B0CE288F410091CE030B6BAF",
			"110000001E14000000287A7F82410091CE030B6BB1",
			"110000001E14000000287A7F82410091CE030B6BB2",
			"110000001E14000000287A7F82410091CE030B6BB3",
			"110000001E14000000287A7F82410091CE030B6BB4",
			"110000001E14000000287A7F82410091CE030B6BB5",
			"110000001E14000000287A7F82410091CE030B6BB6",
			"110000001E14000000287A7F82410091CE030B6BB7",
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "💎 正在领取召唤活动奖励...",
			"type":    "default",
		})

		for _, hexStr := range packets {
			// 确保有些包长度不一致时也能正常解析
			p, err := hex.DecodeString(hexStr)
			if err != nil {
				fmt.Printf("⚠️ 封包解析跳过: %s, 错误: %v\n", hexStr, err)
				continue
			}

			// 发送到服务器
			a.activeConn.SendToServer(2, p)

			// 设置一个固定的微小延迟，防止发包粘连
			time.Sleep(time.Duration(300+rand.Intn(100)) * time.Millisecond)
		}

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "✅ 召唤2w抽奖励领取完毕",
			"type":    "success",
		})
	}

	// --- 3. 抽回溯票子 (10张) ---
	if active, ok := config["ticketDraw"].(bool); ok && active {
		count := 1
		if c, ok := config["ticketDrawCount"].(float64); ok {
			count = int(c)
		}
		ticketPacket := "0D0000001C16000000B81A398F4100910A"

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": fmt.Sprintf("🎫 [PK上分] 抽取回溯票子 (10张) - 共 %d 次", count),
			"type":    "default",
		})

		for i := 1; i <= count; i++ {
			p, _ := hex.DecodeString(ticketPacket)
			a.activeConn.SendToServer(2, p)
			time.Sleep(time.Duration(400+rand.Intn(201)) * time.Millisecond)
		}
	}

}

func (a *App) runGuildPKTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// --- 1. 副本挑战 (领主、金币、哥布林、神灯) ---
	if clears, ok := config["clears"].(map[string]interface{}); ok {
		// 这里的顺序和副本任务保持一致
		dungeonTypes := []string{"boss", "gold", "goblin", "lamp"}
		dungeonNames := map[string]string{"boss": "领主", "gold": "金币", "goblin": "哥布林", "lamp": "神灯"}

		for _, t := range dungeonTypes {
			conf, ok := clears[t].(map[string]interface{})
			if !ok || !conf["active"].(bool) {
				continue
			}

			count := 0
			if c, ok := conf["count"].(float64); ok {
				count = int(c)
			}
			level := 0
			if l, ok := conf["level"].(float64); ok {
				level = int(l)
			}

			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
				"message": fmt.Sprintf("🚀 [PK上分] 挑战[%s] - 共 %d 次", dungeonNames[t], count),
				"type":    "default",
			})

			for i := 1; i <= count; i++ {
				packetHex := a.buildDungeonPacket(t, level)
				if packetHex != "" {
					cmdBytes, _ := hex.DecodeString(packetHex)
					a.activeConn.SendToServer(2, cmdBytes)
				}
				time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
			}
		}
	}

	// --- 2. 抽宠物 (350抽) ---
	if active, ok := config["petDraw"].(bool); ok && active {
		count := 1
		if c, ok := config["petDrawCount"].(float64); ok {
			count = int(c)
		}
		// 复用之前的宠物召唤封包
		petPacket := "1D0000003214000000307A2F12410092A7706172746E6572A870726F5F64726177"

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": fmt.Sprintf("🐾 [PK上分] 执行宠物召唤 (350抽) - 共 %d 次", count),
			"type":    "default",
		})

		for i := 1; i <= count; i++ {
			p, _ := hex.DecodeString(petPacket)
			a.activeConn.SendToServer(2, p)
			time.Sleep(time.Duration(400+rand.Intn(201)) * time.Millisecond)
		}
	}

	// --- 3. 抽回溯票子 (10张) ---
	if active, ok := config["ticketDraw"].(bool); ok && active {
		count := 1
		if c, ok := config["ticketDrawCount"].(float64); ok {
			count = int(c)
		}
		ticketPacket := "0D0000001C16000000B81A398F4100910A"

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": fmt.Sprintf("🎫 [PK上分] 抽取回溯票子 (10张) - 共 %d 次", count),
			"type":    "default",
		})

		for i := 1; i <= count; i++ {
			p, _ := hex.DecodeString(ticketPacket)
			a.activeConn.SendToServer(2, p)
			time.Sleep(time.Duration(400+rand.Intn(201)) * time.Millisecond)
		}
	}

	// --- 4. 领取每日奖励 (那一大串封包) ---
	if val, ok := config["guildDailyReward"].(bool); ok && val {
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": "🎁 正在领取工会对决每日进度奖励...",
			"type":    "default",
		})

		rewardPackets := []string{
			"0E0000002823000000A81F298F4100920101", "0E0000002823000000A81F298F4100920201",
			"0E0000002823000000A81F298F4100920301", "0E0000002823000000A81F298F4100920401",
			"0E0000002823000000A81F298F4100920501", "0E0000002823000000307A2F124100920602",
			"0E0000002823000000307A2F124100920702", "0E0000002823000000307A2F124100920802",
			"0E0000002823000000307A2F124100920902", "0E0000002823000000307A2F124100920A02",
			"0E0000002823000000307A2F124100920B03", "0E0000002823000000307A2F124100920C03",
			"0E0000002823000000307A2F124100920D03", "0E0000002823000000307A2F124100920E03",
			"0E0000002823000000307A2F124100920F03", "0E0000002823000000307A2F124100921004",
			"0E0000002823000000307A2F124100921104", "0E0000002823000000307A2F124100921204",
			"0E0000002823000000307A2F124100921304", "0E0000002823000000307A2F124100921404",
			"0E0000002823000000307A2F124100921505", "0E0000002823000000307A2F124100921605",
			"0E0000002823000000307A2F124100921705", "0E0000002823000000307A2F124100921805",
			"0E0000002823000000307A2F124100921905", "0E0000002823000000307A2F124100921A06",
			"0E0000002823000000307A2F124100921B06", "0E0000002823000000307A2F124100921C06",
			"0E0000002823000000307A2F124100921D06", "0E0000002823000000307A2F124100921E06",
		}

		for _, p := range rewardPackets {
			cmdBytes, _ := hex.DecodeString(p)
			a.activeConn.SendToServer(2, cmdBytes)
			// 领取奖励的封包可以快一点
			time.Sleep(time.Duration(1500+rand.Intn(201)) * time.Millisecond)
		}
	}
}

func (a *App) runRaceTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// --- 1. 狂飙 ---
	if active, ok := config["rush"].(bool); ok && active {
		count := 1
		if c, ok := config["rushCount"].(float64); ok {
			count = int(c)
		}

		// 固定指令
		Packet := "0C000000AA16000000307A2F12410090"
		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]string{
			"message": fmt.Sprintf("🏃 [狂飙跑米] 正在狂飙%d00米", count),
			"type":    "default",
		})

		// 布阵（哪吒+小王子）
		group := "1E0000005C16000000C8996C82410091B131393031303033302C3139303130303031170000005F16000000C8996C8241009192CE012211EECE012211D1"
		d, _ := hex.DecodeString(group)
		a.activeConn.SendToServer(2, d)

		// 循环发送
		for i := 1; i <= count; i++ {
			if a.activeConn == nil {
				break
			}
			data, _ := hex.DecodeString(Packet)

			// 发送封包
			a.activeConn.SendToServer(2, data)

			// 打印进度日志 (每 1 次都打印或者按需抽样)
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("🏃‍♂️ 狂飙执行中: 第 %d00/%d00 米", i, count),
				"type":    "default",
			})

			// 模拟人工点击延迟 (300ms - 500ms 随机)
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}

	// --- 2. 金锤子 ---
	if active, ok := config["goldHammer"].(bool); ok && active {
		// 数量：每个封包里包含的数量 (1-50) -> 对应 fullPacket 的最后两位
		num := 1
		if n, ok := config["goldHammerNum"].(float64); ok {
			num = int(n)
			if num > 50 {
				num = 50
			} // 安全校验
		}

		// 次数：循环发送多少次
		count := 1
		if c, ok := config["goldHammerCount"].(float64); ok {
			count = int(c)
		}

		// 固定前缀
		frontPacket := "0E0000006114000000B0CE288F41009202"

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
			"message": fmt.Sprintf("🔭 [冒险家] 开始砸金锤子：单次 %d 个，准备执行 %d 次", num, count),
			"type":    "info",
		})

		// 循环发送
		for i := 1; i <= count; i++ {
			if a.activeConn == nil {
				break
			}

			// 核心逻辑：将 num (数量) 转为 2 位大写十六进制
			numHex := fmt.Sprintf("%02X", num)

			// 拼接完整包字符串 (前缀 + 数量Hex)
			fullPacketHex := frontPacket + numHex
			data, _ := hex.DecodeString(fullPacketHex)

			// 发送封包
			a.activeConn.SendToServer(2, data)

			// 打印进度日志 (每 1 次都打印或者按需抽样)
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("🔨 金锤子执行中: 第 %d/%d 次 (%d个/次)", i, count, num),
				"type":    "default",
			})

			// 模拟人工点击延迟 (300ms - 500ms 随机)
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}

	// --- 3. 银锤子 ---
	if active, ok := config["silverHammer"].(bool); ok && active {
		// 数量：每个封包里包含的数量 (1-50) -> 对应 fullPacket 的最后两位
		num := 1
		if n, ok := config["silverHammerNum"].(float64); ok {
			num = int(n)
			if num > 50 {
				num = 50
			} // 安全校验
		}

		// 次数：循环发送多少次
		count := 1
		if c, ok := config["silverHammerCount"].(float64); ok {
			count = int(c)
		}

		// 固定前缀
		frontPacket := "0E0000006114000000B0CE288F41009201"

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
			"message": fmt.Sprintf("🔭 [冒险家] 开始砸银锤子：单次 %d 个，准备执行 %d 次", num, count),
			"type":    "info",
		})

		// 循环发送
		for i := 1; i <= count; i++ {
			if a.activeConn == nil {
				break
			}

			// 核心逻辑：将 num (数量) 转为 2 位大写十六进制
			numHex := fmt.Sprintf("%02X", num)

			// 拼接完整包字符串 (前缀 + 数量Hex)
			fullPacketHex := frontPacket + numHex
			data, _ := hex.DecodeString(fullPacketHex)

			// 发送封包
			a.activeConn.SendToServer(2, data)

			// 打印进度日志 (每 1 次都打印或者按需抽样)
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("🔨 银锤子执行中: 第 %d/%d 次 (%d个/次)", i, count, num),
				"type":    "default",
			})

			// 模拟人工点击延迟 (300ms - 500ms 随机)
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}

	// --- 4. 翅膀 ---
	if active, ok := config["wings"].(bool); ok && active {
		// 数量：每个封包里包含的数量 (1-5) -> 对应 fullPacket 的最后两位
		num := 1
		if n, ok := config["wingsNum"].(float64); ok {
			num = int(n)
			if num > 5 {
				num = 5
			} // 安全校验
		}

		// 次数：循环发送多少次
		count := 1
		if c, ok := config["wingsCount"].(float64); ok {
			count = int(c)
		}

		// 固定前缀
		frontPacket := "0D000000BA17000000B0CE288F410091"

		wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
			"message": fmt.Sprintf("🔭 [冒险家] 开始抽翅膀：单次 %d 个，准备执行 %d 次", num, count),
			"type":    "info",
		})

		// 循环发送
		for i := 1; i <= count; i++ {
			if a.activeConn == nil {
				break
			}

			// 核心逻辑：将 num (数量) 转为 2 位大写十六进制
			numHex := fmt.Sprintf("%02X", num)

			// 拼接完整包字符串 (前缀 + 数量Hex)
			fullPacketHex := frontPacket + numHex
			data, _ := hex.DecodeString(fullPacketHex)

			// 发送封包
			a.activeConn.SendToServer(2, data)

			// 打印进度日志 (每 1 次都打印或者按需抽样)
			wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
				"message": fmt.Sprintf("🪽 抽翅膀执行中: 第 %d/%d 次 (%d个/次)", i, count, num),
				"type":    "default",
			})

			// 模拟人工点击延迟 (300ms - 500ms 随机)
			time.Sleep(time.Duration(300+rand.Intn(201)) * time.Millisecond)
		}
	}
}

func (a *App) runGambleTask(config map[string]interface{}) {
	if a.ctx == nil || a.activeConn == nil {
		return
	}

	// 1. 获取远程指令（加读锁保证并发安全）
	a.rcMutex.RLock()
	cmds := a.remoteConfig.Gamble.MonthlyGuildReward
	a.rcMutex.RUnlock()

	// 打印进度日志
	wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
		"message": "🎁 开始领取月赛奖励...",
		"type":    "default",
	})

	// 领取投注奖励 (Bet Prize)
	betData, _ := hex.DecodeString(cmds.BetPrize)
	a.activeConn.SendToServer(2, betData)
	time.Sleep(time.Duration(200+rand.Intn(201)) * time.Millisecond)

	// 领取门票奖励 (Ticket Prize)
	ticketData, _ := hex.DecodeString(cmds.TicketPrize)
	a.activeConn.SendToServer(2, ticketData)
	time.Sleep(time.Duration(200+rand.Intn(201)) * time.Millisecond)
}

// 辅助方法：向前端推送进度日志
func (a *App) emitTaskProgress(msg string, msgType string, finished bool) {
	wailsRuntime.EventsEmit(a.ctx, "task_progress", map[string]interface{}{
		"message":  msg,
		"type":     msgType,
		"finished": finished,
	})
}

func hexToInt(hexStr string) int {
	value, _ := strconv.ParseInt(hexStr, 16, 64)
	return int(value)
}

// GetMachineID 获取电脑唯一标识 (Windows版)
func (a *App) GetMachineID() string {
	var id string
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-Command", "(Get-ItemProperty 'HKLM:\\SOFTWARE\\Microsoft\\Cryptography').MachineGuid")
		out, _ := cmd.Output()
		// 过滤掉标题和空格
		id = string(out)
	} else if runtime.GOOS == "darwin" {
		// Mac 专门提取 IOPlatformSerialNumber
		cmd := exec.Command("sh", "-c", "ioreg -l | grep IOPlatformSerialNumber | awk -F'\"' '{print $4}'")
		out, _ := cmd.Output()
		id = string(out)
	}

	finalID := strings.TrimSpace(id)
	if finalID == "" {
		return "UNKNOWN_DEVICE_ID"
	}
	return finalID
}

func (a *App) CheckAuthOnline(mID string) (bool, string, int) {
	fmt.Printf("🔍 正在验证设备: [%s]\n", mID)
	const authUrl = "https://gist.githubusercontent.com/dear510/e24f215a3f5534e5dc5971f7038643c6/raw/auth.txt"

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(authUrl)
	if err != nil {
		return false, "NET_ERROR", 0
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		cloudID := parts[0]
		expireStr := parts[1]

		if cloudID == mID {
			// 验证日期
			expireDate, err := time.Parse("2006-01-02", expireStr)
			if err != nil || time.Now().After(expireDate) {
				return false, "EXPIRED", 0
			}

			// --- 核心修改：解析等级 ---
			level := 1 // 默认普通用户
			if len(parts) >= 3 {
				if l, err := strconv.Atoi(parts[2]); err == nil {
					level = l
				}
			}

			a.UserLevel = level // 存入结构体
			return true, expireStr, level
		}
	}
	return false, "NOT_FOUND", 0
}

// 修改前端调用的方法，返回等级
func (a *App) CheckCurrentAuth() int {
	mID := a.GetMachineID()
	authorized, _, level := a.CheckAuthOnline(mID)
	if !authorized {
		return 0
	}
	return level
}

func (a *App) ToggleCapture(active bool) string {
	const port = 2025
	if active {
		if a.sunny == nil {
			a.sunny = SunnyNet.NewSunny()
			a.initSunnyConfig()
		}

		// 确保无论是否第一次，都设置一次端口和配置
		a.sunny.SetPort(port)

		err := a.sunny.Start()
		if err.Error != nil {
			fmt.Println("❌ 启动失败:", err.Error)
			return "ERR_START_FAILED" // 返回明确的错误码
		}

		if runtime.GOOS == "darwin" {
			a.setMacProxy(true, port)
		} else if runtime.GOOS == "windows" {
			a.sunny.SetIEProxy()
		}
		return "SUCCESS_ON"
	} else {
		if a.sunny != nil {
			a.activeConn = nil // 清空连接引用
			a.sunny.Close()    // 👈 建议在关闭时调用 Close 释放端口，下次 Start 会更稳
			if runtime.GOOS == "darwin" {
				a.setMacProxy(false, port)
			} else if runtime.GOOS == "windows" {
				a.sunny.CancelIEProxy()
			}
		}
		return "SUCCESS_OFF"
	}
}

func (a *App) setMacProxy(enable bool, port int) {
	// 定义需要设置代理的服务列表(Mac上常见的网络名称)
	networkServices := []string{"Wi-Fi", "Ethernet", "USB 10/100/1000 LAN"}

	for _, service := range networkServices {
		if enable {
			// 设置 HTTP 代理
			exec.Command("networksetup", "-setwebproxy", service, "127.0.0.1", strconv.Itoa(port)).Run()
			// 设置 HTTPS 代理
			exec.Command("networksetup", "-setsecurewebproxy", service, "127.0.0.1", strconv.Itoa(port)).Run()
			// 确保开关是打开的
			exec.Command("networksetup", "-setwebproxystate", service, "on").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", service, "on").Run()
		} else {
			// 关闭模式：强制关掉开关
			// 注意：在 Mac 上，直接关闭 state 才是最稳妥的，不需要清除 127.0.0.1 地址
			exec.Command("networksetup", "-setwebproxystate", service, "off").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", service, "off").Run()
		}
	}
}

// ClearLogs 清空前端日志(可选：如果想在后端也做些清理)
func (a *App) ClearLogs() {
	// 这里可以执行一些后端清理逻辑
	fmt.Println("用户清空了拦截日志")
}

func (a *App) domReady(ctx context.Context) {}
func (a *App) shutdown(ctx context.Context) {
	a.activeConn = nil // 清空连接引用
	if a.sunny != nil {
		a.sunny.Close()
	}
	if runtime.GOOS == "darwin" {
		a.setMacProxy(false, 2025)
	} else if runtime.GOOS == "windows" {
		a.sunny.CancelIEProxy()
	}
}
