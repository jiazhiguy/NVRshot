package main
import(
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"encoding/json"
	"log"
	"time"
	"bytes"
	"os"
	"errors"
	"mime/multipart"
	"path/filepath"
	"github.com/robfig/cron"
	"github.com/sparrc/go-ping"
)
var (
    HttpClient = &http.Client{
        Timeout: 3 * time.Second,
    }
)
func main() {
	urls := []string{}
    category := "3";
    host := "http://localhost:9090/message";
    var tip,clientId,topic,group string
    fmt.Print("设备ID:")
    fmt.Scanln(&clientId)
    fmt.Print("主题:")
    fmt.Scanln(&topic)
    fmt.Print("设备类型(1=NVR,2=摄像头,3=网络地址):")
    fmt.Scanln(&group)
    switch group{
	    case "1":
	    	urls =nvrShot()
	    case "2":
	    	urls =cameraShot()
	    case "3":
	    	urls =urlShot()
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
	//给终端设备发送截图指令
	go func(){
		for url := range c{
			sendParament := fmt.Sprintf("id=%s&&topic=%s&&type=%s&&cmd=%s",clientId,topic,category,url)
			log.Println(url+":开始截图")
			rsp,err := HttpPost(sendParament,host)
			if err != nil {
				log.Println(err)	
			}
			fmt.Println(rsp)
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
	type message struct{
		Message string `json "message"`
	}
    c :=make(chan string)
    go func(){
	    resp, err := http.Post(
	    	host,
	        "application/x-www-form-urlencoded",
	        strings.NewReader(option),
	    )
	    if err != nil {
	        return 
	    }
	    body, err := ioutil.ReadAll(resp.Body)
	    if err != nil {
	        return 
	    }
	    defer resp.Body.Close()
	    c<-string(body)
    }()
    select {
    case data := <-c:
    	fmt.Println(data)
    	var jsonRsp message
   		json.Unmarshal([]byte(data),&jsonRsp)
    	path :=strings.TrimSpace(strings.Split(jsonRsp.Message,":")[1])
    	p,_ :=getCurrentDirectory()
    	path =p+"/bin/public/images/"+path
    	fmt.Println(path)
    	upload(path)
    case <-time.After(10*time.Second):
    	return  "",errors.New("Timeout")
    }
    return "",nil

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
func nvrShot() []string{
	var hikaNVR,channels string
	urls := []string{}
    fmt.Print("海康NVR地址：(示例 admin:a12345678@192.168.10.100:554)")
    fmt.Scanln(&hikaNVR)
    if hikaNVR == ""{
    	exit("hikaNVR不能为空")
    }
    targetNVR := strings.Split(hikaNVR,"@")
    if len(targetNVR) !=2{
    	exit("hikaNVR参数错误")
    }
    targetUrl := strings.Split(targetNVR[1],":")[0]
    pingOk,_:= ServerPing(targetUrl)
    if !pingOk{
    	exit("NVR地址["+targetUrl+"]:不能ping通")
    }
    fmt.Print("起始截止通道：(示例 1-32 表示1到32通道)")
    fmt.Scanln(&channels)
    if channels == ""{
    	exit("通道不能为空")
    }
    chanArray := strings.Split(channels,",")
   	for _,channelGroup := range chanArray{
		chanNumArray := strings.Split(channelGroup,"-") 
		numLen :=len(chanNumArray)
		if  numLen>2 {
			exit("起始截止通道参数错误")
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
	return urls
}
func cameraShot() []string{
	var address string
	fmt.Print("输入摄像头地址(admin:abcefg12345678@192.168.1.102):")
	fmt.Scanln(&address)
	addressArray := strings.Split(address,",")
	for i,_ :=range addressArray{
		// rtsp://admin:abcefg12345678@192.168.1.102/h264/ch1/main/av_stream
		addressArray[i]=fmt.Sprintf("rtsp://%s/h264/ch1/main/av_stream",addressArray[i]) 
	}
	return addressArray
}
func urlShot()[]string{
	var address string
	fmt.Print("输入摄像头网络流地址:")
	fmt.Scanln(&address)
	addressArray := strings.Split(address,",")
	return addressArray
}
func getCurrentDirectory() (string ,error){
    dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
        return "",err
    }
    return strings.Replace(dir, "\\", "/", -1),nil
}
func upload(path string){
    url :="--------"
    params := map[string]string{
        //fmt.Sprintf("%d",17)不能直接使用"17",为什么
        "channel_id":fmt.Sprintf("%d",17),
    }
    data ,err := ioutil.ReadFile(path) // just pass the file name
    if err != nil {
        fmt.Print(err)
    }
    rsp ,err :=uploadFile(url,params,"upFile","11.png",data)
    if err!=nil{
    	 fmt.Println(err)
    }
    fmt.Println("....",rsp)
}
func uploadFile(url string, params map[string]string, nameField, fileName string, fileData []byte) ([]byte, error) {
    body := new(bytes.Buffer)
    body_writer := multipart.NewWriter(body)
    //写入文件
    formFile, err := body_writer.CreateFormFile(nameField, fileName)
    if err != nil {
        return nil, err
    }
    formFile.Write(fileData)
    // 其他参数列表写入body
    for key, val := range params {
        _ = body_writer.WriteField(key, val)
    }
    err = body_writer.Close()
    if err != nil {
        return nil, err
    }
    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", body_writer.FormDataContentType())
    req.Header.Add("X-No-Sign","yes")
    resp, err := HttpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    content, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    return content, nil
}