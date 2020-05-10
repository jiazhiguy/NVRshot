package main
import(
	// "encoding/json"
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"log"
	"time"

	"github.com/robfig/cron"
	"github.com/sparrc/go-ping"
)
func main() {
	urls := []string{}
    category := "3";
    host := "http://localhost:9090/message";
    var hikaNVR,channels,tip,clientId,topic string
    fmt.Print("设备ID:")
    fmt.Scanln(&clientId)
    fmt.Print("主题:")
    fmt.Scanln(&topic)
    fmt.Print("海康NVR地址：(示例 admin:a12345678@192.168.10.100:554)")
    fmt.Scanln(&hikaNVR)
    if hikaNVR == ""{
    	exit("hikaNVR不能为空")
    	// <-time.After(2*time.Second)
    	// return
    }
    targetNVR := strings.Split(hikaNVR,"@")
    if len(targetNVR) !=2{
    	exit("hikaNVR参数错误")
    	// <-time.After(2*time.Second)
    	// return
    }
    targetUrl := strings.Split(targetNVR[1],":")[0]
    pingOk,_:= ServerPing(targetUrl)
    if !pingOk{
    	exit("NVR地址["+targetUrl+"]:不能ping通")
    	// <-time.After(2*time.Second)
    	// return
    }
    fmt.Print("起始截止通道：(示例 1-32   表示1到32通道)")
    fmt.Scanln(&channels)
    if channels == ""{
    	exit("通道不能为空")
    	// <-time.After(2*time.Second)
    	// return
    }
    chanArray := strings.Split(channels,",")
   	for _,channelGroup := range chanArray{
		chanNumArray := strings.Split(channelGroup,"-") 
		// fmt.Println(chanNumArray)
		numLen :=len(chanNumArray)
		if  numLen>2 {
			exit("起始截止通道参数错误")
			// fmt.Println("起始截止通道参数错误")
			// <-time.After(2*time.Second)
			// panic("exit")
		}else{
			var start ,end int
			var err error
			if numLen == 1{
		        if start,err = strconv.Atoi(chanNumArray[0]) ;err != nil {
	                exit("起始截止通道参数必须为数字")
	            }
				url := fmt.Sprintf("rtsp://%s/Streaming/Channels/%s01?transportmode=unicast",hikaNVR,start)
				urls = append(urls,url)
			}else{
	          	if start,err = strconv.Atoi(chanNumArray[0]) ;err != nil {
	                exit("起始截止通道参数必须为数字")
	            }
	      		if end,err = strconv.Atoi(chanNumArray[1]) ;err != nil {
	                exit("起始截止通道参数必须为数字")
	            } 
	            i := start
	            for i<=end {
					url := fmt.Sprintf("rtsp://%s/Streaming/Channels/%d01?transportmode=unicast",hikaNVR,i)
				    urls = append(urls,url)
	            	i=i+1
	            }
			}
		}
	}
    fmt.Print("定时计划:(示例 0_0/10_*_*_*_?  表示每隔10分钟截图一次)")
    fmt.Scanln(&tip)
    if tip == ""{
    	exit("定时计划不能为空")
    	return
    } 
    tip = strings.Replace(tip,"_"," ",-1)
	cronHub := cron.New()
    cronHub.Start() 
	c := make(chan string)
	// //给终端设备发送截图指令
	go func(){
		for url := range c{
			fmt.Println("start ")
			sendParament := fmt.Sprintf("id=%s&&topic=%s&&type=%s&&cmd=%s",clientId,topic,category,url)
			log.Println(url+":开始截图")
			_,err := HttpPost(sendParament,host)
			if err != nil {
				log.Println(err)	
			}
		}
	}()
	err := cronHub.AddFunc(tip,func(){
		for _,url := range urls {
			c <- url
		} 
	})

	if err != nil{
		exit("添加定时错误:"+err.Error())
	}
	select{}
} 
func HttpPost(option,host string) (string,error){
    resp, err := http.Post(
    	host,
        "application/x-www-form-urlencoded",
        strings.NewReader(option),
    )
    if err != nil {
        return "",err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "",err
    }
    return string(body),nil
    
}
func ServerPing(target string) (bool,error)  {
	var ICMPCOUNT = 2
	var PINGTIME = time.Duration(1)
    pinger, err := ping.NewPinger(target)

    if err != nil {
        return false,err
    }
    pinger.Count = ICMPCOUNT

    pinger.Timeout = time.Duration(PINGTIME*time.Millisecond)

    pinger.SetPrivileged(true)

    pinger.Run()// blocks until finished

    stats := pinger.Statistics()
    // 有回包，就是说明IP是可用的
    if stats.PacketsRecv >= 1 {
        return true,nil
    }
    return false,nil

}
func exit(err string) {
	fmt.Println(err)
	fmt.Println("********程序2s后退出*********")
	<-time.After(2*time.Second)
	panic("exit")
}