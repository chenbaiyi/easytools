package controllers

import (
	"container/list"
	"fmt"
	"github.com/astaxie/beego"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"math/rand"
	"net/http"
	"sort"
	"time"
)

type ChatRoomController struct {
	beego.Controller
}

type WebsocketController struct {
	beego.Controller
}

type MsgType int
const (
	WelcomeMsg MsgType = iota
	LeaveMsg
	CommTextMsg
)

type Msg struct {
	Type MsgType
	User string
	Timestamp int64
	ContentLeft string
	ContentRight  string
	OriginalContent string
	Conn *websocket.Conn
}

type DBMsgSlice []DBMsg
type DBMsg struct {
	Type int
	UserName string
	Timestamp int64
	Content string
}

type Subscriber struct {
	Name string
	Conn *websocket.Conn
}

var (
	subscribe = make(chan Subscriber, 10)
	unsubscribe = make(chan string, 10)
	publish = make(chan Msg, 10)
	subscribers = list.New()
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	db *gorm.DB
)

func init() {
	go start()
	//gorm.Open("mysql", "root:root@(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True&loc=Local")
	var err error
	db, err = gorm.Open("mysql", "jfuser:jf.icpm.jfuser.300559&G2@(127.0.0.1:3306)/easychat?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		err = db.DB().Ping()
		if err != nil {
			beego.Error("ping db falied")
		}
		beego.Error("连接easychat数据库失败")
	}

	//自动迁移模式
	//db.AutoMigrate(&Msg{})
}

func start() {
	for {
		select {
		case sub := <-subscribe:
			if (!isUserExist(subscribers, sub.Name)) {
				subscribers.PushBack(sub)
				m := &Msg{
					Type: WelcomeMsg,
					User: sub.Name,
					OriginalContent: sub.Name + "爬进了房间",
					Timestamp: time.Now().Unix(),
					Conn: sub.Conn,
				}
				publish <- *SendMessage(m)
			}
		case msg := <-publish:
			broadcastMessage(msg)
		case unsub := <-unsubscribe:
			for sub := subscribers.Front(); sub != nil; sub = sub.Next() {
				if sub.Value.(Subscriber).Name == unsub {
					subscribers.Remove(sub)
					conn := sub.Value.(Subscriber).Conn
					if conn != nil {
						conn.Close()
					}

					m := &Msg{
						Type: LeaveMsg,
						User: unsub,
						OriginalContent: unsub + "爬出了房间",
						Timestamp: time.Now().Unix(),
						Conn: conn,
					}
					publish <- *SendMessage(m)
					break
				}
			}
		}
	}

	db.Close()
}

func (c *ChatRoomController) Get()  {
	c.TplName = "chatroom.html"
}

func (w *WebsocketController) Get() {
	conn, err := upgrader.Upgrade(w.Ctx.ResponseWriter, w.Ctx.Request, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		http.Error(w.Ctx.ResponseWriter, "Cannot setup WebSocket connection", 500)
		return
	}

	username := GetFullName()
	subscribe <- Subscriber{Name: username, Conn: conn}

	//发生异常时，通知关闭该用户
	defer func() {
		unsubscribe <- username
	}()

	var m Msg
	for {
		//开启监听已连接的用户
		_, p, err := conn.ReadMessage()
		if err != nil {
			return
		}

		m = Msg{
			Type: CommTextMsg,
			User: username,
			OriginalContent: string(p),
			Timestamp: time.Now().Unix(),
			Conn: conn,
		}
		publish <- *SendMessage(&m)
	}
}

func (d DBMsgSlice) Len() int {
	return len(d)
}

func (d DBMsgSlice) Less(i, j int) bool {
	return d[i].Timestamp < d[j].Timestamp
}

func (d DBMsgSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func broadcastMessage(msg Msg) {
	//推送历史的十条消息
	if msg.Type == WelcomeMsg && msg.Conn != nil {
		var results DBMsgSlice
		var m Msg

		db.Table("history_msg").Where("type != 0 and type != 1").Order("timestamp desc").Limit(10).Scan(&results)

		//升序排列按照时间戳
		sort.Sort(results)

		for _, v := range results {
			m = Msg{
				Type: MsgType(v.Type),
				User: v.UserName,
				OriginalContent: v.Content,
				Timestamp: v.Timestamp,
				Conn: msg.Conn,
			}
			msg.Conn.WriteMessage(websocket.TextMessage, []byte(SendMessage(&m).ContentLeft))
		}
	}

	for sub := subscribers.Front(); sub != nil; sub = sub.Next() {
		conn := sub.Value.(Subscriber).Conn
		if conn == nil {
			continue
		}

		if (conn != msg.Conn && conn.WriteMessage(websocket.TextMessage, []byte(msg.ContentLeft)) != nil) ||
			(conn == msg.Conn && conn.WriteMessage(websocket.TextMessage, []byte(msg.ContentRight)) != nil) {
				unsubscribe <- sub.Value.(Subscriber).Name
		}
	}

	//保存到数据库
	if err := db.Table("history_msg").Create(&DBMsg{
		Type: int(msg.Type),
		UserName: msg.User,
		Timestamp: msg.Timestamp,
		Content: msg.OriginalContent,
	}).Error; err != nil {
		beego.Error("save msg to db failed, msg:", msg.OriginalContent)
	}
}

func SendMessage(m *Msg) *Msg  {
	m.ContentRight += "<li class=\"d-flex message right\">\n\t<div class=\"message-body\">\n\t<span class=\"date-time text-muted\">"
	m.ContentRight += time.Unix(m.Timestamp, 0).Format("2006-01-02 15:04:05")
	m.ContentRight += "<i class=\"zmdi zmdi-check-all text-primary\"></i></span>\n\t<div class=\"message-row d-flex align-items-center justify-content-end\">\n\t<div class=\"dropdown\">\n\t<a class=\"text-muted me-1 p-2 text-muted\" href=\"#\" data-toggle=\"dropdown\" aria-haspopup=\"true\" aria-expanded=\"false\">\n\t<i class=\"zmdi zmdi-more-vert\"></i>\n\t</a>\n\t<div class=\"dropdown-menu\">\n\t<a class=\"dropdown-item\" href=\"#\">Edit</a>\n\t<a class=\"dropdown-item\" href=\"#\">Share</a>\n\t<a class=\"dropdown-item\" href=\"#\">Delete</a>\n\t</div>\n\t</div>\n\t<div class=\"message-content border p-3\">"
	m.ContentRight += m.OriginalContent
	m.ContentRight += "</div>\n\t</div>\n\t</div>\n\t</li>"

	m.ContentLeft += "<li class=\"d-flex message\"><div class=\"mr-lg-3 me-2\"><img class=\"avatar sm rounded-circle\" src=\"/static/img/chatroom-head10.jpg\" alt=\"avatar\"></div><div class=\"message-body\"><span class=\"date-time text-muted\">"
	m.ContentLeft += (m.User + " ")
	m.ContentLeft += time.Unix(m.Timestamp, 0).Format("2006-01-02 15:04:05")
	m.ContentLeft += "</span><div class=\"message-row d-flex align-items-center\"><div class=\"message-content p-3\">"
	m.ContentLeft += m.OriginalContent
	m.ContentLeft += " </div><div class=\"dropdown\"><a class=\"text-muted ms-1 p-2 text-muted\" href=\"#\" data-toggle=\"dropdown\" aria-haspopup=\"true\" aria-expanded=\"false\"><i class=\"zmdi zmdi-more-vert\"></i></a><div class=\"dropdown-menu dropdown-menu-right\"><a class=\"dropdown-item\" href=\"#\">Edit</a><a class=\"dropdown-item\" href=\"#\">Share</a><a class=\"dropdown-item\" href=\"#\">Delete</a></div></div></div></div></li>"

	return m
}

func isUserExist(subscribers *list.List, user string) bool {
	for sub := subscribers.Front(); sub != nil; sub = sub.Next() {
		if sub.Value.(Subscriber).Name == user {
			return true
		}
	}

	return false
}

var lastName = []string{
	"赵", "钱", "孙", "李", "周", "吴", "郑", "王", "冯", "陈", "褚", "卫", "蒋",
	"沈", "韩", "杨", "朱", "秦", "尤", "许", "何", "吕", "施", "张", "孔", "曹", "严", "华", "金", "魏",
	"陶", "姜", "戚", "谢", "邹", "喻", "柏", "水", "窦", "章", "云", "苏", "潘", "葛", "奚", "范", "彭",
	"郎", "鲁", "韦", "昌", "马", "苗", "凤", "花", "方", "任", "袁", "柳", "鲍", "史", "唐", "费", "薛",
	"雷", "贺", "倪", "汤", "滕", "殷", "罗", "毕", "郝", "安", "常", "傅", "卞", "齐", "元", "顾", "孟",
	"平", "黄", "穆", "萧", "尹", "姚", "邵", "湛", "汪", "祁", "毛", "狄", "米", "伏", "成", "戴", "谈",
	"宋", "茅", "庞", "熊", "纪", "舒", "屈", "项", "祝", "董", "梁", "杜", "阮", "蓝", "闵", "季", "贾",
	"路", "娄", "江", "童", "颜", "郭", "梅", "盛", "林", "钟", "徐", "邱", "骆", "高", "夏", "蔡", "田",
	"樊", "胡", "凌", "霍", "虞", "万", "支", "柯", "管", "卢", "莫", "柯", "房", "裘", "缪", "解", "应",
	"宗", "丁", "宣", "邓", "单", "杭", "洪", "包", "诸", "左", "石", "崔", "吉", "龚", "程", "嵇", "邢",
	"裴", "陆", "荣", "翁", "荀", "于", "惠", "甄", "曲", "封", "储", "仲", "伊", "宁", "仇", "甘", "武",
	"符", "刘", "景", "詹", "龙", "叶", "幸", "司", "黎", "溥", "印", "怀", "蒲", "邰", "从", "索", "赖",
	"卓", "屠", "池", "乔", "胥", "闻", "莘", "党", "翟", "谭", "贡", "劳", "逄", "姬", "申", "扶", "堵",
	"冉", "宰", "雍", "桑", "寿", "通", "燕", "浦", "尚", "农", "温", "别", "庄", "晏", "柴", "瞿", "阎",
	"连", "习", "容", "向", "古", "易", "廖", "庾", "终", "步", "都", "耿", "满", "弘", "匡", "国", "文",
	"寇", "广", "禄", "阙", "东", "欧", "利", "师", "巩", "聂", "关", "荆", "司马", "上官", "欧阳", "夏侯",
	"诸葛", "闻人", "东方", "赫连", "皇甫", "尉迟", "公羊", "澹台", "公冶", "宗政", "濮阳", "淳于", "单于",
	"太叔", "申屠", "公孙", "仲孙", "轩辕", "令狐", "徐离", "宇文", "长孙", "慕容", "司徒", "司空"}
var firstName = []string{
	"伟", "刚", "勇", "毅", "俊", "峰", "强", "军", "平", "保", "东", "文", "辉", "力", "明", "永", "健", "世", "广", "志", "义",
	"兴", "良", "海", "山", "仁", "波", "宁", "贵", "福", "生", "龙", "元", "全", "国", "胜", "学", "祥", "才", "发", "武", "新",
	"利", "清", "飞", "彬", "富", "顺", "信", "子", "杰", "涛", "昌", "成", "康", "星", "光", "天", "达", "安", "岩", "中", "茂",
	"进", "林", "有", "坚", "和", "彪", "博", "诚", "先", "敬", "震", "振", "壮", "会", "思", "群", "豪", "心", "邦", "承", "乐",
	"绍", "功", "松", "善", "厚", "庆", "磊", "民", "友", "裕", "河", "哲", "江", "超", "浩", "亮", "政", "谦", "亨", "奇", "固",
	"之", "轮", "翰", "朗", "伯", "宏", "言", "若", "鸣", "朋", "斌", "梁", "栋", "维", "启", "克", "伦", "翔", "旭", "鹏", "泽",
	"晨", "辰", "士", "以", "建", "家", "致", "树", "炎", "德", "行", "时", "泰", "盛", "雄", "琛", "钧", "冠", "策", "腾", "楠",
	"榕", "风", "航", "弘", "秀", "娟", "英", "华", "慧", "巧", "美", "娜", "静", "淑", "惠", "珠", "翠", "雅", "芝", "玉", "萍",
	"红", "娥", "玲", "芬", "芳", "燕", "彩", "春", "菊", "兰", "凤", "洁", "梅", "琳", "素", "云", "莲", "真", "环", "雪", "荣",
	"爱", "妹", "霞", "香", "月", "莺", "媛", "艳", "瑞", "凡", "佳", "嘉", "琼", "勤", "珍", "贞", "莉", "桂", "娣", "叶", "璧",
	"璐", "娅", "琦", "晶", "妍", "茜", "秋", "珊", "莎", "锦", "黛", "青", "倩", "婷", "姣", "婉", "娴", "瑾", "颖", "露", "瑶",
	"怡", "婵", "雁", "蓓", "纨", "仪", "荷", "丹", "蓉", "眉", "君", "琴", "蕊", "薇", "菁", "梦", "岚", "苑", "婕", "馨", "瑗",
	"琰", "韵", "融", "园", "艺", "咏", "卿", "聪", "澜", "纯", "毓", "悦", "昭", "冰", "爽", "琬", "茗", "羽", "希", "欣", "飘",
	"育", "滢", "馥", "筠", "柔", "竹", "霭", "凝", "晓", "欢", "霄", "枫", "芸", "菲", "寒", "伊", "亚", "宜", "可", "姬", "舒",
	"影", "荔", "枝", "丽", "阳", "妮", "宝", "贝", "初", "程", "梵", "罡", "恒", "鸿", "桦", "骅", "剑", "娇", "纪", "宽", "苛",
	"灵", "玛", "媚", "琪", "晴", "容", "睿", "烁", "堂", "唯", "威", "韦", "雯", "苇", "萱", "阅", "彦", "宇", "雨", "洋", "忠",
	"宗", "曼", "紫", "逸", "贤", "蝶", "菡", "绿", "蓝", "儿", "翠", "烟", "小", "轩"}


func GetFullName() string {
	rand.Seed(time.Now().UnixNano()) //设置随机数种子
	var first string                 //名
	for i := 0; i <= rand.Intn(1); i++ { //随机产生2位或者3位的名
		first = fmt.Sprint(firstName[rand.Intn(len(lastName) - 1)])
	}
	//返回姓名
	return fmt.Sprintf("%s%s", fmt.Sprint(lastName[rand.Intn(len(lastName) - 1)]), first)
}