package initconf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var PPPoEFConfFile = "/etc/sysconfig/network-scripts/ifcfg-ppp0"
var PPPoEConfDataText = `USERCTL=yes
USERCTL=yes
BOOTPROTO=dialup
NAME=DSLppp0
DEVICE=ppp0
TYPE=xDSL
ONBOOT=yes
PIDFILE=/var/run/pppoe-adsl.pid
FIREWALL=NONE
PING=.
PPPOE_TIMEOUT=80
LCP_FAILURE=3
LCP_INTERVAL=20
CLAMPMSS=1412
CONNECT_POLL=6
CONNECT_TIMEOUT=60
DEFROUTE=yes
SYNCHRONOUS=no
PROVIDER=DSLppp0
PEERDNS=no
DEMAND=no
`

var RePPPoECmd = `
ifdown ppp0;
ifdwon ppp0;
sleep 1;
ifup ppp0;
`

type PPPoEConf struct{
	Eth	string
	User	string
	Pswd	string
	ConfText	string
	ConfFileName	string
}

func (p *PPPoEConf)Save()(error){
	p.ConfText += "ETH=" + p.Eth + "\n"
	p.ConfText += "USER=" + p.User + "\n"
	p.ConfText += "PASSWORD=" + p.Pswd + "\n"

	f, err := os.OpenFile(p.ConfFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil{
		return err
	}

	_, err = f.WriteString(p.ConfText)
	return err
}

func NewPPPoEConf(pppoeDevName string)(*PPPoEConf){
	return &PPPoEConf{
		Eth: "eth0",
		ConfText: strings.ReplaceAll(PPPoEConfDataText, "ppp0", pppoeDevName),
		ConfFileName: strings.ReplaceAll(PPPoEFConfFile, "ppp0", pppoeDevName),
	}
}

func RePPPoE(pppoeDevName string)error{
	strCmd := strings.ReplaceAll(RePPPoECmd, "ppp0", pppoeDevName)
	cmd := exec.Command("bash", "-c", strCmd)
	err := cmd.Run()
	if err != nil{
		return fmt.Errorf("RePPPoE failed, exec shell error: %s\n", err)
	}
	return nil
}