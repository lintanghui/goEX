// Create by Linth at  N015-8-4
// 聊天室服务端
// 实现一对一 一对多聊天
package main

import (
    "encoding/binary"
    "github.com/golang/protobuf/proto"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "log"
    "net"

    "userdata"
)

const (
    LEN = 2 // 消息头len字节长度
)

var (
    userMapConn map[string]connProperty // 用户与连接的映射
)

// conn封装，添加conn属性
type connProperty struct {
    connfd   net.Conn
    userName string
}

type User struct {
    ID  string `bson:"_id,omitempty"`

    Passwd string
}

func main() {

    listern, err := net.Listen("tcp", "localhost:9090")
    checkerr(err)
    defer listern.Close()

    session, err := mgo.Dial("mongodb://192.168.1.51:27017")
    if err != nil {
        panic(err)
    }
    defer session.Close()
    session.SetMode(mgo.Monotonic, true)
    c := session.DB("db_test").C("user")

    userMapConn = make(map[string]connProperty) // 全局变量初始化
    for {
        conn, err := listern.Accept()
        defer conn.Close()
        checkerr(err)
        go login(conn, c)
    }
}

// TO DO 用户注册

// 用户登录 每一个用户映射一个conn
func login(conn net.Conn, c *mgo.Collection) {
    var (
        name      string
        passwd    string
        mode      string
        connInfor = connProperty{connfd: conn}
    )

    for {
        requestMess, err := readByte(connInfor)
        if err != nil {
            return
        }
        name = requestMess.Name // 获取用户名
        passwd = requestMess.Passwd
        mode = requestMess.Mode
        log.Println(mode)
        // 注册操作
        if mode == "B" {
            log.Println("sign in")
            err = c.Insert(&User{name, passwd})
            if err != nil {
                log.Println("sign in error", err)
            }
        } else { // 登录操作 跳出循环，开始对话
            break
        }
    }
    var result []User
    log.Println(name, passwd)

    // 登录验证
    err := c.Find(bson.M{"_id": name, "passwd": passwd}).All(&result)
    if err != nil {
        log.Println("search fail")
        return
    }
    // 返回登录信息
    // 不存在关闭连接
    if result == nil {
        log.Println("user do not exit")
        conn.Close()
        return
    } else {
        // 返回登录成功信息
        response := &userdata.Chat_Response{
            Timestamp: 0,
            Object:    "",
            Content:   name + " login successfully",
            Mode:      "",
        }
        sendContent(connInfor, response)
    }
    log.Println(result)

    log.Println(name, " login sucessfully")
    connInfor = connProperty{connfd: conn, userName: name}

    userMapConn[name] = connInfor // 用户名映射conn  通过用户名进行私聊
    modeSelect(connInfor)

}

// 读取字节流
func readByte(conn connProperty) (req *userdata.Chat_Request, err error) {
    lenth := make([]byte, LEN)
    _, err = conn.connfd.Read(lenth)
    if !checkerr(err) {
        dealerr(conn)
        return
    }

    contentLenth := binary.BigEndian.Uint16(lenth)
    data := make([]byte, contentLenth)
    _, err = conn.connfd.Read(data)
    if !checkerr(err) {
        dealerr(conn)
        return
    }
    requestMess := &userdata.Chat_Request{}
    if err = proto.Unmarshal(data, requestMess); err != nil {
        log.Println("Unmarshal err ", err)
        // TO DO 编码错误的进一步处理
    }
    return requestMess, nil
}

// 模式选择
func modeSelect(conn connProperty) {
    for {
        requestMess, err := readByte(conn)
        if err != nil {
            return
        }

        mode := requestMess.Mode
        switch mode { // 去掉换行和结束符
        case "ls":
            listUser(conn) // 列出当前在线用户
        case "all":
            talkToAll(conn, requestMess) // 广播
        case "person":
            personTalk(conn, requestMess) // 私聊

            // TO DO  群聊
        }
    }
}

// 读取内容
func readcontent(conn connProperty) (str string, err error) {
    requestMess, err := readByte(conn)
    if err != nil {
        return
    }
    content := requestMess.Content
    return content, nil
}

// 私聊
func personTalk(conn connProperty, request *userdata.Chat_Request) {
    name := request.Object
    personConn, flag := userMapConn[name]
    log.Println(personConn, flag)
    if !flag {
        // TO DO 接收离线消息
        response := &userdata.Chat_Response{
            Timestamp: request.Timestamp,
            Object:    name,
            Content:   name + " do not on line",
            Mode:      "person",
        }
        sendContent(conn, response)
        return
    }
    // 获取对话内容
    content := request.Content

    // 发送内容
    response := &userdata.Chat_Response{
        Timestamp: request.Timestamp,
        Object:    conn.userName,
        Content:   content,
        Mode:      "person",
    }
    sendContent(personConn, response)
}

// 发送内容
func sendContent(personConn connProperty, response *userdata.Chat_Response) {

    responseMess, err := proto.Marshal(response)
    if err != nil {
        // 编码失败，返回长度零
        b := make([]byte, LEN)
        binary.BigEndian.PutUint16(b, uint16(0))
        personConn.connfd.Write(b)
        return
    }
    b := make([]byte, LEN)
    binary.BigEndian.PutUint16(b, uint16(len(responseMess)))

    personConn.connfd.Write(b)            // 发送长度
    personConn.connfd.Write(responseMess) // 发送内容
}

// 列出当前在线用户
func listUser(conn connProperty) {
    for key, _ := range userMapConn {
        responese := &userdata.Chat_Response{
            Timestamp: 0,
            Object:    key,
            Content:   "",
            Mode:      "ls",
        }
        sendContent(conn, responese)
    }

}

// 向每一个连接发送信息
// TO DO 向指定的room成员发送信息
func talkToAll(conn connProperty, request *userdata.Chat_Request) {
    log.Println("talk to all ")
    content := request.Content
    if content == "quit" {
        log.Println("quit person talk")
        return
    }
    response := &userdata.Chat_Response{
        Timestamp: request.Timestamp,
        Object:    conn.userName,
        Content:   content,
        Mode:      "all",
    }
    log.Println("talk to all")
    for _, sendconn := range userMapConn {
        sendContent(sendconn, response)
        log.Println("send to ", sendconn.userName)
    }
}

func checkerr(err error) bool {
    if err != nil {

        log.Println("an error", err)
        return false
    }
    return true
}

func dealerr(conn connProperty) {
    conn.connfd.Close()
    // connPool = append(connPool[:n], connPool[:n+1]...)
    log.Println(conn.userName, "login out")
    delete(userMapConn, conn.userName) // 删除下线的用户
}
