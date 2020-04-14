package main

import (
    "fmt"
    "log"
    "net"
	"strings"
	"runtime"
	"syscall"
	"unsafe"
	"encoding/json"
)

var (
    advapi = syscall.NewLazyDLL("Advapi32.dll")
    kernel = syscall.NewLazyDLL("Kernel32.dll")
)

type diskusage struct {
    Path  string `json:"path"`
    Total uint64 `json:"total"`
    Free  uint64 `json:"free"`
}

func usage(getDiskFreeSpaceExW *syscall.LazyProc, path string) (diskusage, error) {
    lpFreeBytesAvailable := int64(0)
    var info = diskusage{Path: path}
    diskret, _, err := getDiskFreeSpaceExW.Call(
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(info.Path))),
        uintptr(unsafe.Pointer(&lpFreeBytesAvailable)),
        uintptr(unsafe.Pointer(&(info.Total))),
        uintptr(unsafe.Pointer(&(info.Free))))
    if diskret != 0 {
        err = nil
    }
    return info, err
}

//硬盘信息
func GetDiskInfo() (infos []diskusage) {
    GetLogicalDriveStringsW := kernel.NewProc("GetLogicalDriveStringsW")
    GetDiskFreeSpaceExW := kernel.NewProc("GetDiskFreeSpaceExW")
    lpBuffer := make([]byte, 254)
    diskret, _, _ := GetLogicalDriveStringsW.Call(
        uintptr(len(lpBuffer)),
        uintptr(unsafe.Pointer(&lpBuffer[0])))
    if diskret == 0 {
        return
    }
    for _, v := range lpBuffer {
        if v >= 65 && v <= 90 {
            path := string(v) + ":"
            if path == "A:" || path == "B:" {
                continue
            }
            info, err := usage(GetDiskFreeSpaceExW, string(v)+":")
            if err != nil {
                continue
            }
            infos = append(infos, info)
        }
    }
    return infos
}

func connHandler(c net.Conn) {
    //1.conn是否有效
    if c == nil {
        log.Panic("无效的 socket 连接")
    }

    //2.新建网络数据流存储结构
    buf := make([]byte, 4096)
    //3.循环读取网络数据流
    for {
        //3.1 网络数据流读入 buffer
        cnt, err := c.Read(buf)
        //3.2 数据读尽、读取错误 关闭 socket 连接
        if cnt == 0 || err != nil {
            c.Close()
            break
        }

        //3.3 根据输入流进行逻辑处理
        //buf数据 -> 去两端空格的string
        inStr := strings.TrimSpace(string(buf[0:cnt]))
        //去除 string 内部空格
        cInputs := strings.Split(inStr, " ")
        //获取 客户端输入第一条命令
        fCommand := cInputs[0]

        fmt.Println("客户端传输->" + fCommand)

        switch fCommand {
        case "OS_info":
			c.Write([]byte("系统架构："+runtime.GOARCH+"\n"+"系统版本："+runtime.GOOS+"\n"))
			break
		case "Disk_info":
			disk := GetDiskInfo()
			for i :=0; i < len(disk); i++{
				b,_:= json.Marshal(disk[i])
				c.Write([]byte(string(b)+"\n"))
			}
			break
        default:
			c.Write([]byte("请输入正确请求\n"))
			break
        }

        //c.Close() //关闭client端的连接，telnet 被强制关闭

        fmt.Printf("来自 %v 的连接关闭\n", c.RemoteAddr())
    }
}

//开启serverSocket
func ServerSocket() {
    //1.监听端口
    server, err := net.Listen("tcp", ":8087")

    if err != nil {
        fmt.Println("开启socket服务失败")
    }

    fmt.Println("正在开启 Server ...")

    for {
        //2.接收来自 client 的连接,会阻塞
        conn, err := server.Accept()

        if err != nil {
            fmt.Println("连接出错")
        }

        //并发模式 接收来自客户端的连接请求，一个连接 建立一个 conn，服务器资源有可能耗尽 BIO模式
        go connHandler(conn)
    }

}


func main(){
	ServerSocket()
}