// Create by Linth at  2015-8-5
// 聊天室客户端
package main

import (
    "bufio"
    "encoding/binary"
    "errors"
    "fmt"
    "github.com/golang/protobuf/proto"
    "log"
    "net"
    "os"
    "strings"
    "sync"
    "time"
    "userdata"
)

const (
    LEN = 2 // 消息头len字节长度

)

var name string // 用户名
var wg sync.WaitGroup

func main() {

    var mode string
    conn, err := net.Dial("tcp", "localhost:9090")
    if err != nil {
        log.Println("connect err", err)
    }
    // TO DO 用户注册
    for {
        fmt.Println("input A to login or B to register")
        fmt.Scanf("%s", &mode)
        switch mode {
        case "A", "B":
            login(conn, mode)
        default:
        }

    }

}

// 登录
func login(conn net.Conn, mode string) {
    // log.Println(mode)
    var passwd string
    fmt.Println("please input username and passwd")

    fmt.Scanf("%s%s", &name, &passwd)
    // 登录验证
    request := &userdata.Chat_Request{
        Timestamp: time.Now().Unix(),
        Name:      name,
        Mode:      mode,
        Object:    "",
        Content:   "",
        Passwd:    passwd,
    }
    if sendCont(conn, request) != nil {
        log.Println("login fail")
        return
    }
    // 注册操作，返回登录界面
    if mode == "B" {
        // TO DO 返回注册信息
        return
    }
    rsp, err := readCont(conn)
    if err != nil {
        log.Println("uncorrect passwd")
        return
    }

    fmt.Println(rsp.Content) // 打印成功登录信息

    wg.Add(2)
    go selectMode(conn)
    go readResponse(conn)
    wg.Wait()
}

// 读取消息体并输出
func readResponse(conn net.Conn) {
    defer wg.Done()
    for {
        dialContent, err := readCont(conn)
        if err != nil {
            conn.Close()
            os.Exit(1)
            return
        }
        mode := dialContent.Mode
        switch mode {
        case "ls":
            fmt.Println("online user :", dialContent.Object)
        case "all":
            fmt.Println(time.Unix(dialContent.Timestamp, 0), dialContent.Object)
            fmt.Println(dialContent.Content)
        case "person":
            fmt.Println(time.Unix(dialContent.Timestamp, 0), dialContent.Object)
            fmt.Println(dialContent.Content)
        }
    }
}

// 聊天模式选择
func selectMode(conn net.Conn) {
    defer wg.Done()

    fmt.Println("select mode:")
    fmt.Println("all:", "talk to everyone")
    fmt.Println("ls:", "list the user  on line")
    fmt.Println("person  target  content:", "send content to user that called target")
    fmt.Println("quit:", "exit client")
    for {
        // 拆分对话模式及内容
        inputReader := bufio.NewReader(os.Stdin)
        input, _ := inputReader.ReadString('\n')
        str := strings.Split(input, "\n")
        parameter := strings.Split(str[0], " ")

        switch parameter[0] {
        case "ls":
            listUser(conn, parameter[0])
        case "all":
            content := strings.Join(parameter[1:], " ") // 合并对话内容
            talkToAll(conn, parameter[0], content)
        case "person":
            content := strings.Join(parameter[2:], " ")
            personTalk(conn, parameter[0], parameter[1], content)
        case "quit":
            conn.Close()
            os.Exit(1)
        default:
            fmt.Println("no such mode,select again")
        }

    }

}

// 私聊
func personTalk(conn net.Conn, mode, target, content string) {
    fmt.Println("talk to person ,enter \"quit\" to exit")
    bindMess(conn, mode, target, content)
}

// 群聊
func talkToAll(conn net.Conn, mode, content string) {
    fmt.Println("talk to all ,enter \"quit\" to exit")
    bindMess(conn, mode, "", content)
}

// 绑定信息
func bindMess(conn net.Conn, mode, target, content string) {
    request := &userdata.Chat_Request{
        Timestamp: time.Now().Unix(),
        Name:      name,
        Mode:      mode,
        Object:    target,
        Content:   content,
        Passwd:    "",
    }
    sendCont(conn, request)

    for {
        inputReader := bufio.NewReader(os.Stdin)
        input, _ := inputReader.ReadString('\n')
        str := strings.Split(input, "\n")
        request := &userdata.Chat_Request{
            Timestamp: time.Now().Unix(),
            Name:      name,
            Mode:      mode,
            Object:    target,
            Content:   str[0],
            Passwd:    "",
        }
        sendCont(conn, request)
        if str[0] == "quit" {
            fmt.Println("exit dial,select mode again")
            return
        }
    }
}

// 列出在线用户
func listUser(conn net.Conn, mode string) {
    request := &userdata.Chat_Request{
        Timestamp: time.Now().Unix(),
        Name:      name,
        Mode:      mode,
        Object:    "",
        Content:   "",
        Passwd:    "",
    }

    sendCont(conn, request)

}

// 读取conn，返回消息体
func readCont(conn net.Conn) (res *userdata.Chat_Response, err error) {
    lenth := make([]byte, LEN)
    _, err = conn.Read(lenth)
    if err != nil {
        return
    }

    contentLen := binary.BigEndian.Uint16(lenth)
    if contentLen == 0 {
        err = errors.New("EOF")
        return
    }
    data := make([]byte, contentLen)
    if _, err = conn.Read(data); err != nil {
        return
    }
    response := &userdata.Chat_Response{}
    if err = proto.Unmarshal(data, response); err != nil {
        log.Println("Unmarshal err ", err)
        return
    }
    return response, nil
}

// 发送内容
func sendCont(conn net.Conn, req *userdata.Chat_Request) (err error) {
    reqMess, err := proto.Marshal(req)
    if err != nil {
        return
    }
    b := make([]byte, LEN)
    binary.BigEndian.PutUint16(b, uint16(len(reqMess)))

    conn.Write(b)       // 发送长度
    conn.Write(reqMess) // 发送内容
    return nil
}
