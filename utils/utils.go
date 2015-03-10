package utils

//package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	//	"github.com/op/go-logging"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogLevel uint32

const (
	NoticeLevel  LogLevel = 1
	FatalLevel   LogLevel = 2
	WarningLevel LogLevel = 3
	DebugLevel   LogLevel = 4
)

type LogControl struct {
	TimeGap   int64  //间隔时间，单位秒
	FileName  string //日志文件名
	FilePath  string //日志路径
	FileOut   *os.File
	FileMutex sync.Mutex //日志锁
	LogLevel  LogLevel   //当前日志级别
	LogFormat string
}

var DebugLog *LogControl
var FatalLog *LogControl
var WarningLog *LogControl
var NoticeLog *LogControl
var GlobalLogLevel LogLevel

// 传入timegap 单位为分钟
func (this *LogControl) Init(timegap int64, filename string, filepath string, loglevel LogLevel) (err error) {
	// 内部转化为秒
	this.TimeGap = timegap * 60
	this.FileName = filename
	this.FilePath = filepath
	this.LogLevel = loglevel
	switch loglevel {
	case NoticeLevel:
		this.LogFormat = "NOTICE: "
	case FatalLevel:
		this.LogFormat = "FATAL: "
	case WarningLevel:
		this.LogFormat = "WARNING: "
	case DebugLevel:
		this.LogFormat = "DEBUG: "
	}
	if this.LogLevel > GlobalLogLevel {
		return
	}
	err = this.open_file()
	if err != nil {
		return
	}
	go this.LogCut()
	return
}

func (this *LogControl) Write(format string, args ...interface{}) (err error) {
	if this.LogLevel > GlobalLogLevel {
		return
	}
	this.FileMutex.Lock()
	defer this.FileMutex.Unlock()
	err = this.check_valid()
	if err != nil {
		return err
	}
	var body string
	head := fmt.Sprintf("%s %s * ", this.LogFormat, time.Now().Format("2006-01-02 15:04:05"))
	if args != nil {
		body = fmt.Sprintf(format, args...)
	} else {
		body = format
	}
	_, err = this.FileOut.Write([]byte(head + body + "\n"))
	return
}

func (this *LogControl) open_file() (err error) {
	this.FileOut, err = os.OpenFile(this.FilePath+"/"+this.FileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	return
}

func (this *LogControl) check_valid() (err error) {
	//这部分代码有待商榷。。。时间开销
	_, err = os.Stat(this.FilePath + "/" + this.FileName)
	if err != nil {
		err = this.open_file()
		if err != nil {
			return
		}
	}
	return
}

func (this *LogControl) LogCut() {
	var err error
	for {
		nowtime := time.Now().Unix()
		nexttime := int64(nowtime/this.TimeGap+1) * this.TimeGap
		var delta time.Duration
		delta = time.Duration(nexttime - nowtime)
		time.Sleep(time.Second * delta)
		this.FileMutex.Lock()
		//		date_format := time.Now().Truncate(time.Duration(this.TimeGap * time.Second)).Format("200601021504")
		date_now := time.Now().Unix() - this.TimeGap
		date_now = (int64(float64(date_now)+0.5*float64(this.TimeGap)) / this.TimeGap) * this.TimeGap
		date_format := time.Unix(date_now, 0).Format("200601021504")
		cutfile := this.FilePath + this.FileName + "." + date_format + "00"
		_, err = os.Stat(cutfile)
		if err != nil { // 如果文件不存在,则切割日志. 如果文件存在，则不覆盖
			err = this.check_valid()
			if err != nil {
				log.Printf("check log file fail. err[%s]", err.Error())
				os.Exit(-1)
			}
			this.FileOut.Close()
			os.Rename(this.FilePath+this.FileName, cutfile)
			err = this.open_file()
			if err != nil {
				os.Exit(-1)
			}
		}
		this.FileMutex.Unlock()
	}
	return
}

func GenSearchid(imei string) (searchid string) {

	var tmp string
	tmp = imei
	tmp += time.Now().String()
	tmp += strconv.Itoa(rand.Int())
	DebugLog.Write("searchid is %s", tmp)
	sha1_t := sha1.New()
	io.WriteString(sha1_t, tmp)
	searchid = fmt.Sprintf("%x", sha1_t.Sum(nil))
	return
}

const (
	c1 = 0xcc9e2d51
	c2 = 0x1b873593
	c3 = 0x85ebca6b
	c4 = 0xc2b2ae35
	r1 = 15
	r2 = 13
	m  = 5
	n  = 0xe6546b64
)

func Murmur3(key []byte, seed uint32) (hash uint32) {
	hash = seed
	iByte := 0
	for ; iByte+4 <= len(key); iByte += 4 {
		k := uint32(key[iByte]) | uint32(key[iByte+1])<<8 | uint32(key[iByte+2])<<16 | uint32(key[iByte+3])<<24
		k *= c1
		k = (k << r1) | (k >> (32 - r1))
		k *= c2
		hash ^= k
		hash = (hash << r2) | (hash >> (32 - r2))
		hash = hash*m + n
	}

	var remainingBytes uint32
	switch len(key) - iByte {
	case 3:
		remainingBytes += uint32(key[iByte+2]) << 16
		fallthrough
	case 2:
		remainingBytes += uint32(key[iByte+1]) << 8
		fallthrough
	case 1:
		remainingBytes += uint32(key[iByte])
		remainingBytes *= c1
		remainingBytes = (remainingBytes << r1) | (remainingBytes >> (32 - r1))
		remainingBytes = remainingBytes * c2
		hash ^= remainingBytes
	}

	hash ^= uint32(len(key))
	hash ^= hash >> 16
	hash *= c3
	hash ^= hash >> 13
	hash *= c4
	hash ^= hash >> 16

	// 出发吧，狗嬷嬷！
	return
}

func TagExist(tags []string, key string) (exist bool) {
	exist = false
	for i := 0; i < len(tags); i++ {
		if strings.EqualFold(tags[i], key) == true {
			exist = true
			return
		}
	}
	return
}

//key must plus sep . eg: ctr_model:
func GetTagInt(tags []string, key string) (ret int, exist bool) {
	exist = false
	ret = 0
	var err error
	for i := 0; i < len(tags); i++ {
		if strings.Contains(tags[i], key) == true {
			exist = true
			ret, err = strconv.Atoi(strings.Trim(tags[i], key))
			if err != nil {
				FatalLog.Write("get tag int . atoi fail . tag[%s], key[%s],err[%s]", tags[i], key, err.Error())
				exist = false
				return
			}
		}
	}
	return
}

func GetTagStr(tags []string, key string) (ret string, exist bool) {
	exist = false
	for i := 0; i < len(tags); i++ {
		if strings.Contains(tags[i], key) == true {
			exist = true
			ret = strings.TrimPrefix(tags[i], key)
			DebugLog.Write("tags get tag str is [%s], tags[%s], key[%s]", ret, tags[i], key)
			return
		}
	}
	return
}

func GenIMEIFromCookie(cookie string) (imei string) {
	ans := 0
	if len(cookie) < 14 {
		imei = ""
		return
	}
	for i := 0; i < 14; i++ {
		imei += strconv.Itoa(int(cookie[i]) % 10)
	}

	for i := 0; i < 14; i++ {
		rand := int(imei[i] - '0')
		if i%2 == 0 {
			ans += rand
		} else {
			ans += (rand*2)%10 + (rand*2)/10
		}
	}
	if ans%10 == 0 {
		imei += "0"
	} else {
		imei += strconv.Itoa(10 - (ans % 10))
	}
	return
}

func GenIDFAFromCookie(cookie string) (idfa string) {
	if len(cookie) != 32 {
		idfa = ""
		return
	}
	idfa = cookie[0:8] + "-" + cookie[8:12] + "-" + cookie[12:16] + "-" + cookie[16:20] + "-" + cookie[20:32]
	return
}
