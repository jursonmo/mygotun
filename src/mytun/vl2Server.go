package main

import(
	"log"
	"net"
	"flag"
	"net/http"
	"encoding/json"
	"mylog"
	"crypto/tls"
	"fdb"
	_"net/http/pprof"
	"time"
)
/*
1、 生成服务器端的私钥
openssl genrsa -out server.key 2048
2、 生成服务器端证书
openssl req -new -x509 -key server.key -out server.pem -days3650
*/
var (
	buildTime string
	commitId string
	appVersion = "1.0.0"
	version = flag.Bool("v", true, "show version information")
	listenAddr = flag.String("listenAddr", "", "listenAddr, like 23.33.145.33:7878")
	httpAddr = flag.String("httpAddr", "", "127.0.0.1:88, check mactable, localhost:88/clientmac")
	tlsSK = flag.String("server.key", "./config/server.key", "tls server.key")
	tlsSP = flag.String("server.pem", "./config/server.pem", "tls server.pem")
	tlsEnable = flag.Bool("tls", false, "enable tls server, default false")
	pprofEnable = flag.Bool("pprof", false, "enable pprof, default false")
	ppAddr = flag.String("ppaddr", ":6060", "ppaddr , http://xxxx:6060/debug/pprof/")
	serAddr = flag.String("serAddr", "", " the addr connect to ,like 127.0.0.1:9999")
	readFwdMode =  flag.Int("rfm", 1, " readFwdMode, 1 means read one by one and forward, 2 means read big pkt and parase forward")
	br = flag.String("br", "br0"," add tun/tap to bridge")
	tuntype = flag.Int("tuntype", 1," type, 1 means tap and 0 means tun")
	tundev = flag.String("tundev","tap0"," tun dev name")
	ipstr = flag.String("ipstr", "", "set tun/tap or br ip address")
)

func HttpGetMacTable(w http.ResponseWriter, req *http.Request){
	mc := fdb.ShowClientMac()
	mcjson, err := json.MarshalIndent(mc, "","\t")
	if err != nil{
		log.Println(err)
		return
	}
	//log.Println(string(mcjson))
	w.Write(mcjson)
}

func checkError(err error, info string) bool{
	if err != nil{
		log.Println(info+": " ,err.Error())
		log.Fatal(err)
		return false
	}
	return true
}

func main(){
	var ln net.Listener
	var err error
	flag.Parse()
	mylog.InitLog(mylog.LDEBUG)
	if *version {
		log.Printf("appVersion=%s, buildTime=%s, commitId=%s\n", appVersion, buildTime, commitId)
	}
	mylog.Warning("listenAddr=%s, httpAddr =%s for check clientmac, serAddr=%s, tlsEnable =%v, br=%s, tundev=%s\n", 
			*listenAddr, *httpAddr, *serAddr, *tlsEnable, *br, *tundev)
	log.Printf("listenAddr=%s, httpAddr =%s for check clientmac, serAddr=%s, tlsEnable =%v, br=%s, tundev=%s\n", 
			*listenAddr, *httpAddr, *serAddr, *tlsEnable, *br, *tundev)

	// for show fdb mactable
	if *httpAddr != "" {
		http.HandleFunc("/clientmac", HttpGetMacTable)
		go http.ListenAndServe(*httpAddr, nil)
	}

	// for pprof
	if *pprofEnable {
		go func() {
			log.Println(http.ListenAndServe(*ppAddr, nil))
		}()
	}

	if *serAddr != "" {
		go connectSer(*serAddr)
	}
	
	if *tundev != "" {
		go createTun()
	}

	if *listenAddr == "" {
		for {
			time.Sleep(time.Minute)
		}
	}

	if *tlsEnable {
		cert, err := tls.LoadX509KeyPair(*tlsSP, *tlsSK)
		if err != nil {
			log.Println(err)
			return
 		}
		tlsconf := &tls.Config {
			Certificates: []tls.Certificate{cert},
		}
		ln, err = tls.Listen("tcp4", *listenAddr, tlsconf)
		checkError(err, "ListenTCP")
	}else {
		ln, err = net.Listen("tcp4", *listenAddr)
		checkError(err, "ListenTCP")
	}

	for {
		conn, err := ln.Accept()
		if err != nil{
			continue
		}
		go handleClient(conn)
	}
}

func connectSer(serAddr string) {
	var conn net.Conn
	var err error
	conn_th, conn_num := 1, 1
	
	reconnect:
	if *tlsEnable {
		tlsconf := &tls.Config{
 			InsecureSkipVerify: true,
 		}
 		conn, err  = tls.Dial("tcp", serAddr, tlsconf)
	}else {
		conn, err = net.Dial("tcp4", serAddr)		
	}

	if err != nil {
		log.Println(err)
		log.Printf("conn_th=%d, connect to %s time=%d \n", conn_th, serAddr, conn_num)
		time.Sleep(time.Second * 2)
		conn_num += 1
		goto reconnect
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	conn_num = 0
	handleClient(conn)
	conn_th += 1
	goto reconnect
}

func createTun() {
	open_th, open_num := 1, 1
	for {		
		tun, err := fdb.OpenTun(*br, *tundev, *tuntype, *ipstr)
		if err != nil {
			log.Println(err)
			log.Printf("open_th=%d, open tun %s fail, time=%d \n", open_th, *tundev, open_num)
			open_num += 1
			time.Sleep(time.Second)
			if open_num > 5 {
				log.Printf("quit to open tun %s, fail time =%d \n", *tundev, open_num)
				break
			}
			continue
		}
		open_num = 0		
		handleClient(tun)
		open_th += 1
	}
}

func handleClient(cio fdb.Cio) {
	c := fdb.NewClient(cio)
	go c.WriteFromChan()
	if *readFwdMode == 1 {
		c.ReadForward()
	} else {
		c.ReadForward2()
	}		
}