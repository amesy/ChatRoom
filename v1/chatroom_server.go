// chatroom server
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const (
	LOG_DIRECTORY = "./test.log" //记录错误日志的路径.
)

var onlineConns = make(map[string]net.Conn) //存储客户端链接映射 key为链接ip:port value为连接对象conn.
var messageQueue = make(chan string, 1000) //消息队列 带缓冲的buf.
var logFile *os.File
var logger *log.Logger

var quitChan = make(chan bool)

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

//消息接收协程
func ProcessInfo(conn net.Conn) {
	buf := make([]byte, 1024)
	//协程退出时 说明客户端断开链接 所以要将当前链接从onlineConns删除掉
	defer func(conn net.Conn) {
		addr := fmt.Sprintf("%s", conn.RemoteAddr())
		delete(onlineConns, addr)
		conn.Close()

		for i := range onlineConns {
			fmt.Println("now online conns:" + i)
		}
	}(conn) //采用匿名函数的方式 调用defer

	for {
		numOfBytes, err := conn.Read(buf)
		if err != nil {
			break
		}

		if numOfBytes != 0 {
			/*结尾buf[0:numOfBytes]的原因是：numOfBytes是指接收的字节数 如果只用string(buf)
	    		可能会导致接收字符串中有其他之前接收的字符
	    	*/
			message := string(buf[0:numOfBytes])

			//将消息放入到消息队列
			messageQueue <- message
		}
	}
}

//消费者协程
func ConsumeMessage() {
	for {
		select {
		case message := <-messageQueue:
			//对消息进行解析
			doProcessMessage(message)
		case <-quitChan:
			break
		}
	}
}

//消息解析函数
func doProcessMessage(message string) {
	contents := strings.Split(message, "#")
	if len(contents) > 1 {
		addr := contents[0]
		sendMessage := strings.Join(contents[1:], "#")//这么做是为了防止 消息体也含有"#"

		addr = strings.Trim(addr, " ")

		if conn, ok := onlineConns[addr]; ok {
			_, err := conn.Write([]byte(sendMessage))
			if err != nil {
				fmt.Println("online conns send failure!")
			}
		}
	} else {
		//走到这里 说明客户端调用list命令 查看系统当前链接ip
		contents := strings.Split(message, "*")
		if strings.ToUpper(contents[1]) == "LIST" {
			var ips string = ""
			for i := range onlineConns {
				ips = ips + "|" + i
			}
			if conn, ok := onlineConns[contents[0]]; ok {
				_, err := conn.Write([]byte(ips))
				if err != nil {
					fmt.Println("online conns send failure!")
				}
			}
		}
	}
}

func main() {
	//打开日志文件
	logFile, err := os.OpenFile(LOG_DIRECTORY, os.O_RDWR|os.O_CREATE, 0)
	if err != nil {
		fmt.Println("log file create failure!")
		os.Exit(-1)
	}

	defer logFile.Close()

	//利用go自带的log 将打开文件对象生成日志文件对象
	logger = log.New(logFile, "\r\n", log.Ldate|log.Ltime|log.Llongfile)

	listen_socket, err := net.Listen("tcp", "127.0.0.1:8080")
	CheckError(err)
	defer listen_socket.Close()

	fmt.Println("Server is waiting....")

	logger.Println("I am writing the logs...")

	//开启消费者协程
	go ConsumeMessage()

	for {
		conn, err := listen_socket.Accept()
		CheckError(err)

		//将conn存储到onlineConns映射表中.
		addr := fmt.Sprintf("%s", conn.RemoteAddr())
		//如果有客户端链接 则将对应ip和链接对象以KV形式记录到onlineConns中
		onlineConns[addr] = conn
		for i := range onlineConns {
			fmt.Println(i)
		}

		//开启协程处理
		go ProcessInfo(conn)
	}
}
