package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"proxyagent/initconf"
	"proxyagent/proxy"
	"time"
)

var serinfo initconf.SerInfo
var devName string
var url string
var httpPort int
var proxyPort int

func changeIPHandler(w http.ResponseWriter, r *http.Request){
	err := initconf.RePPPoE(devName)
	if err != nil{
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%s", err)))
		return
	}
	w.Write([]byte("change"))
	log.Println("got a request to change ip. change ip success.")
	serinfo.UpdateSelfFromURL(url)
}

func LisenHttpSer(){
	http.HandleFunc("/", changeIPHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
}

func main(){
	var username, password string

	flag.StringVar(&url, "url", "","http://selinux.org:8000/node/<uuid>")
	flag.StringVar(&devName, "poedev", "ppp0","PPPoE Device Name.")
	flag.StringVar(&username, "poeuser", "", "PPPoE Username.")
	flag.StringVar(&password, "poepasswd", "", "PPPoE Password.")
	flag.IntVar(&httpPort, "httpport", 8000, "listening port of http server")
	flag.IntVar(&proxyPort, "proxyport", 9000, "listening port of proxy server")

	flag.Parse()
	if len(url) < 8{
		flag.PrintDefaults()
		return
	}

	log.Printf("starting RePPPoE PPPoE Device: %s.\n", devName)
	if len(username) >0 || len(password) > 0{
		log.Printf("config pppoe file username: %s, password: %s, devname: %s.\n", username, password, devName)
		pppoConf := initconf.NewPPPoEConf(devName)
		pppoConf.User = username
		pppoConf.Pswd = password
		pppoConf.Save()
	}

	initconf.RePPPoE(devName)

	log.Printf("init server info from url: %s.\n", url)
	serinfo, err := initconf.NewSerInfo(url)
	if err != nil{
		fmt.Println(err)
		return
	}
	go func(){
		for{
			time.Sleep(time.Duration(1)*time.Minute)
			serinfo.UpdateSelfFromURL(url)
			log.Printf("get server info from url: %s.\n", url)
		}
	}()

	log.Printf("listen http server address 0.0.0.0:%d.\n", httpPort)
	go LisenHttpSer()

	log.Printf("listen proxy server address 0.0.0.0:%d.\n", proxyPort)
	err = proxy.HTTPServer(fmt.Sprintf(":%d", proxyPort))
	fmt.Println(err)
}