package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"scanPort/app/scan"
	"scanPort/app/wsConn"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var (
	wConn *wsConn.WsConnection
	//osBase  string
	version string
)

func main() {
	var (
		port = flag.Int("port", 25252, "端口号")
		h    = flag.Bool("h", false, "帮助信息")
	)
	version = "v1.0"
	flag.Parse()
	//帮助信息
	if *h == true {
		usage("scanPort version: scanPort/v2.0\n Usage: scanPort [-h] [-ip ip地址] [-n 进程数] [-p 端口号范围] [-t 超时时长] [-path 日志保存路径]\n\nOptions:\n")
		return
	}
	serverUri := "http://127.0.0.1:" + strconv.Itoa(*port) + "/scanweb"
	openErr := open(serverUri)
	if openErr != nil {
		fmt.Println(openErr, serverUri)
	}
	//绑定路由地址
	http.HandleFunc("/", indexHandle)
	http.HandleFunc("/scanweb", scanwebHandle)
	http.HandleFunc("/run", runHandle)
	http.HandleFunc("/ws", wsHandle)

	//启动服务端口
	addr := ":" + strconv.Itoa(*port)
	log.Println(" ^_^ 服务已启动...")
	log.Println(" 扫描结果会存放在./log目录下")
	http.ListenAndServe(addr, nil)
}

//首页
func indexHandle(w http.ResponseWriter, r *http.Request) {
	s := "端口扫描 " + version + " (by:lizhejie)"
	w.Write([]byte(s))
}

//端口扫描web页面
func scanwebHandle(w http.ResponseWriter, r *http.Request) {
	f, err := ioutil.ReadFile("app/ui/index.html")
	if err != nil {
		fmt.Println("read index.html fail", err)
	}
	w.Write([]byte(f))

}

//运行
func runHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")             //返回数据格式是json

	resp := map[string]interface{}{
		"code": 200,
		"msg":  "ok",
	}
	decoder := json.NewDecoder(r.Body)
	type Params struct {
		Ip      string `json:"ip"`
		Port    string `json:"port"`
		Process int    `json:"process"`
		Timeout int    `json:"timeout"`
		Debug   int    `json:"debug"`
	}
	var params Params
	decoder.Decode(&params)
	if params.Ip == "" {
		// w.WriteHeader(201),不添加http头信息。即使缺少字段依然返回状态码为200
		resp["code"] = 201
		resp["msg"] = "缺少字段 ip"
		b, _ := json.Marshal(resp)
		w.Write(b)
		return
	}

	if params.Port == "" {
		params.Port = "80"
	}
	if params.Process == 0 {
		params.Process = 10
	}
	if params.Timeout == 0 {
		params.Timeout = 100
	}
	debug := false
	if params.Debug == 0 {
		params.Debug = 1
		debug = true
	}

	//初始化
	scanIP := scan.NewScanIp(params.Timeout, params.Process, debug)
	ips, err := scanIP.GetAllIp(params.Ip)
	if err != nil {
		wConn.WriteMessage(1, []byte(fmt.Sprintf("  ip解析出错....  %v", err.Error())))
		return
	}
	//扫所有的ip
	filePath, _ := mkdir("log")
	fileName := filePath + params.Ip + "_port.txt"
	for i := 0; i < len(ips); i++ {
		ports := scanIP.GetIpOpenPort(ips[i], params.Port, wConn)
		if len(ports) > 0 {
			f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				if err := f.Close(); err != nil {
					fmt.Println(err)
				}
				continue
			}
			if _, err := f.WriteString(fmt.Sprintf("%v【%v】开放:%v \n", time.Now().Format("2006-01-02 15:04:05"), ips[i], ports)); err != nil {
				if err := f.Close(); err != nil {
					fmt.Println(err)
				}
				continue
			}
		}
	}
	open(fileName)
	b, _ := json.Marshal(resp)
	w.Write(b)
	return
}

//ws服务
func wsHandle(w http.ResponseWriter, r *http.Request) {
	wsUp := websocket.Upgrader{
		HandshakeTimeout: time.Second * 5,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
	}
	wsSocket, err := wsUp.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}
	wConn = wsConn.New(wsSocket)
	for {
		data, err := wConn.ReadMessage()
		if err != nil {
			wConn.Close()
			return
		}
		if err := wConn.WriteMessage(data.MessageType, data.Data); err != nil {
			wConn.Close()
			return
		}
	}
}

func usage(str string) {
	fmt.Fprintf(os.Stderr, str)
	flag.PrintDefaults()
}
func mkdir(path string) (string, error) {
	delimiter := "/"
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	filePtah := dir + delimiter + path + delimiter
	err := os.MkdirAll(filePtah, 0777)
	if err != nil {
		return "", err
	}
	return filePtah, nil
}
func open(uri string) error {
	var commands = map[string]string{
		"windows": "start",
		"darwin":  "open",
		"linux":   "xdg-open",
	}
	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("%s platform ？？？", runtime.GOOS)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "start ", uri)
		//cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	} else {
		cmd = exec.Command(run, uri)
	}
	return cmd.Start()
}
