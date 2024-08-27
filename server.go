package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户的列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	// 消息广播的channel
	Message chan string
}

// 创建一个server接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 启动服务器的接口
func (this *Server) Start() {
	// socket listen
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	defer listen.Close()

	go this.ListenMassage()

	for {
		// accept
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("listen.Accept err:", err)
			continue
		}
		// do handle
		go this.Handle(conn)
	}

}

func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

// server连接成功后处理逻辑
func (this *Server) Handle(conn net.Conn) {
	// 当前连接的业务
	fmt.Println("连接建立成功")
	user := NewUser(conn, this)
	user.Online()
	isLive := make(chan bool)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("conn.Read err:", err)
				return
			}
			msg := string(buf[:n-1])
			user.DoMessage(msg)

			isLive <- true
		}
	}()
	// 当前handel阻塞
	for {
		select {
		case <-isLive:
			// 当前用户活跃
		case <-time.After(time.Second * 10):
			// 已经超时, 将当前User强制关闭
			user.SendMsg("你被踢了")
			close(user.C)
			conn.Close()
			return
		}
	}
}

// 监听Message的goroutine， 一旦有消息就会发给所有在线的User
func (this *Server) ListenMassage() {
	for {
		msg := <-this.Message

		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}
