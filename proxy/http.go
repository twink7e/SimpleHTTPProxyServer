package proxy

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

// 10MiB
const HTTP_BODY_MAX_SIZE = 10485760

const MAX_BUFFER_SIZE = 8192

const HTTP_HEADER_BREAK_MARK = "\r\n"
const HTTP_HEADER_BREAK_MARK_SIZE = 2

const HTTP_HEADER_ENDMARK = HTTP_HEADER_BREAK_MARK + HTTP_HEADER_BREAK_MARK
const HTTP_HEADER_ENDMARK_SIZE = HTTP_HEADER_BREAK_MARK_SIZE * 2

const HTTP_BODY_ENDMARK = "\r\n0\r\n"
const HTTP_BODY_ENDMARK_SIZE = 5

//type Response struct {
//	Code int
//	Content string
//}
//
//func (r Response)String()string{
//	contentLen := len(r.Content)
//	return fmt.Sprintf("HTTP/1.1 %d OK\r\nContent-Length: %d\r\n\r\n%s", r.Code, contentLen, r.Content)
//}
//

type HTTPPacket struct {
	Option string
	Uri string
	Version string
	Headers map[string]string
	HeadersReadDone bool
	BodyReadDone bool
	PacketDone bool
	ContentSize int
	Buffer	[]byte
}

func (h HTTPPacket)String()string{
	data := fmt.Sprintf("%s %s %s\r\n", h.Option, h.Uri, h.Version)
	for key, value := range h.Headers{
		data += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	data += HTTP_HEADER_BREAK_MARK

	return data
}

func (h *HTTPPacket)resetBuffer(){
	h.Buffer = make([]byte, 0)
}

func (h *HTTPPacket)EatEndOfNextLine(end_idx int)([]byte){
	var data []byte
	data = h.Buffer[:end_idx]
	if len(h.Buffer) == end_idx + HTTP_HEADER_BREAK_MARK_SIZE{
		h.resetBuffer()
	}else{
		h.Buffer = h.Buffer[end_idx+HTTP_HEADER_BREAK_MARK_SIZE:]
	}
	return data
}

func (h *HTTPPacket)parseOptionUriVersion(data []byte) error{
	result := strings.Split(string(data), " ")
	if len(result) != 3{
		return fmt.Errorf("parseOptionUriVersion error length of result  is :%v", result)
	}
	h.Option, h.Uri, h.Version = result[0], result[1], result[2]
	return nil
}

func (h *HTTPPacket)parseHeader(data []byte)error{
	if h.Headers == nil{
		h.Headers = make(map[string]string)
	}
	if len(data) == 0{
		h.HeadersReadDone = true
		return nil
	}
	result := bytes.Split(data, []byte{':', ' '})
	if len(result) != 2{
		return fmt.Errorf("parseHeader error length of result is :%v. The correct answer would be 2.", result)
	}
	log.Printf("HTTP Header Parser  key: %s, value: %s.\n", result[0], result[1])
	h.Headers[string(result[0])] = string(result[1])
	return nil
}

func (h *HTTPPacket)ParseContentSize()error{
	if contentLengh, ok := h.Headers["Content-Length"]; ok{
		contentSize, err := strconv.Atoi(contentLengh)
		if err != nil{
			return err
		}
		if contentSize > HTTP_BODY_MAX_SIZE{
			return fmt.Errorf("http body max size is %v", contentSize)
		}
		h.ContentSize = contentSize
		log.Println("content length: ", contentSize)
	}else{
		h.ContentSize = 0
		log.Println("non content length")
	}
	h.PacketDone = true
	return nil
}

func (h *HTTPPacket)HasBody()bool{
	if _, ok := h.Headers["Content-Length"]; ok{
		return true
	}else{
		return false
	}
}

func (h *HTTPPacket)GetRealServerAddr()string{
	if h.Option != "CONNECT" {
		result := strings.Split(h.Headers["Host"], ":")
		if len(result) == 1{
			return fmt.Sprintf("%s:80", h.Headers["Host"])
		}else{
			return fmt.Sprintf("%s", h.Headers["Host"])
		}

	}else{
		return fmt.Sprintf("%s", h.Uri)
	}
}

func (h *HTTPPacket)Router(){
	for ; !h.HeadersReadDone && len(h.Buffer) > 0;{
		optionLen := len(h.Option)
		endIdx := bytes.Index(h.Buffer, []byte(HTTP_HEADER_BREAK_MARK))
		if endIdx == -1{
			// TODO handel error
			break
		}
		if endIdx == 0{
			if len(h.Buffer) >= endIdx+HTTP_HEADER_BREAK_MARK_SIZE{
				h.Buffer = h.Buffer[endIdx + HTTP_HEADER_BREAK_MARK_SIZE:]
			}else{
				h.resetBuffer()
			}
			h.HeadersReadDone = true
			break
		}
		buffer := h.EatEndOfNextLine(endIdx)
		if optionLen == 0{
			h.parseOptionUriVersion(buffer)
		}else{
			h.parseHeader(buffer)
		}
	}
}












type Controller struct{
	Packer HTTPPacket
	ClientConn net.Conn
	RealServerConn net.Conn
	HasCreateServerRoutine bool
}

func (c *Controller)createRealServerConn()error{
	conn, err := net.Dial("tcp", c.Packer.GetRealServerAddr())
	if err != nil{
		return fmt.Errorf("createRealServerConn failed : %s", err)
	}

	c.RealServerConn = conn
	return nil
}



func (c *Controller)TellClientTunOK(){
	data := []byte("HTTP/1.0 200 Connection Established\r\n\r\n")
	_, err := c.ClientConn.Write(data)
	if err != nil{
		log.Println("TellClientTunOK send data to client error: %v", err)
	}
}

func (c *Controller)ServerToClient(){
	defer c.RealServerConn.Close()
	for{
		buffer := make([]byte, MAX_BUFFER_SIZE)
		readSize, err := c.RealServerConn.Read(buffer)
		if err != nil{
			log.Println("ServerToClient error from server conn: ", err)
			return
		}
		log.Printf("read data from server to client size : %d", readSize)
		sendSzie, err := c.ClientConn.Write(buffer[:readSize])
		if err != nil{
			log.Println("ServerToClient error from client conn: ", err)
			return
		}
		log.Println("send data from server to client size: ", sendSzie)
	}
}


func (c *Controller)ProcessHTTPSTun()error{
	log.Println("starting to processing https tun....")
	for{
		buffer := make([]byte, MAX_BUFFER_SIZE)
		readSize, err := c.ClientConn.Read(buffer);
		if err != nil{
			return fmt.Errorf("ProcessHTTPSTun error from client conn: %s", err)
		}
		log.Printf("processing data from client to server %s")
		log.Printf("read data from client to server size : %d", readSize)
		sendSize, err := c.RealServerConn.Write(buffer[:readSize])
		if err != nil{
			return fmt.Errorf("ProcessHTTPSTun error from server conn: %s", err)
		}
		log.Printf("send data from client to server size : %d", sendSize)
	}
	return nil
}


func (c *Controller)FlashBodyToServer()error{
	if c.Packer.ContentSize < MAX_BUFFER_SIZE{
		var err error
		var readSize int
		buffer := make([]byte, c.Packer.ContentSize)
		if len(c.Packer.Buffer) < c.Packer.ContentSize{
			readSize, err = c.ClientConn.Read(buffer)
		}
		if len(c.Packer.Buffer) > 0{
			buffer = append(c.Packer.Buffer, buffer...)
			readSize = len(buffer)
			c.Packer.resetBuffer()
		}

		if err != nil{
			return fmt.Errorf("FlashBodyToServer error from client conn err: %s", err)
		}
		if _, err := c.RealServerConn.Write(buffer[:readSize]); err != nil{
			return fmt.Errorf("FlashBodyToServer error from conn: %s", err)
		}
		fmt.Println("not too bad....")
	}else{
		var readSize, bufferSize int
		var err error

		for pushdSize:=c.Packer.ContentSize; pushdSize-MAX_BUFFER_SIZE>0;pushdSize+=readSize{
			residueSize := c.Packer.ContentSize - pushdSize
			if  residueSize <= MAX_BUFFER_SIZE{
				bufferSize = residueSize
			}else{
				bufferSize = MAX_BUFFER_SIZE
			}

			buffer := make([]byte, bufferSize)
			readSize, err = c.ClientConn.Read(buffer)
			if len(c.Packer.Buffer) > 0{
				buffer = append(c.Packer.Buffer, buffer...)
				readSize = len(buffer)
				c.Packer.resetBuffer()
			}
			if err != nil{
				return fmt.Errorf("FlashBodyToServer error from client conn err: %s", err)
			}

			_, err = c.RealServerConn.Write(buffer[:readSize])
			if err != nil{
				return fmt.Errorf("FlashBodyToServer error  from realserver conn: %s", err)
			}
		}
	}

	return nil

}

func (c *Controller)ClientToServer(){
	defer c.ClientConn.Close()
	for{
		// new request. to clear packer
		if c.Packer.PacketDone{
			fmt.Println("Packet is done")
			c.Packer = HTTPPacket{}
		}

		// process headers
		if !c.Packer.HeadersReadDone{
			buffer := make([]byte, MAX_BUFFER_SIZE)
			size, err := c.ClientConn.Read(buffer)
			if err != nil{
				fmt.Println("HTTPHander error from conn err:", err)
				return
			}

			c.Packer.Buffer = append(c.Packer.Buffer, buffer[:size]...)
			c.Packer.Router()
		// process body or open real server connection. send some data to real server.
		}else{
			if c.RealServerConn == nil{
				c.Packer.ParseContentSize()
				if err := c.createRealServerConn(); err !=nil{
					fmt.Println("ClientToServer has an error: ", err)
					return
				}
			}

			// 443 or 80. not same way
			if c.Packer.Option == "CONNECT" && c.RealServerConn != nil{
				c.TellClientTunOK()
			}else{

				data := c.Packer.String()
				if _, err := c.RealServerConn.Write([]byte(data)); err != nil{
					fmt.Println("ClientToServer has an error: ", err)
					return
				}
				fmt.Println("write ok")
			}

			// start goroutine to connection real server. and send data to him.
			if !c.HasCreateServerRoutine{
				log.Printf("option: %s. get %s now start goroutine.\n", c.Packer.Option, c.Packer.Uri)
				go c.ServerToClient()
			}

			// if don't have content or not https(need to tls hands). don't need to process body.
			if c.Packer.ContentSize > 0{
				if err := c.FlashBodyToServer(); err != nil{
					fmt.Println("ClientToServer has an error: ", err)
					return
				}
			}else if c.Packer.Option == "CONNECT"{
				if err := c.ProcessHTTPSTun(); err != nil{
					fmt.Println("ClientToServer has an error: ", err)
					return
				}
			}
		}
	}
}

func NewController()Controller{
	return Controller{}
}




func HTTPHandler(conn net.Conn)error{
	defer conn.Close()

	ctl := NewController()
	ctl.ClientConn = conn

	//parse http packet to realserver
	ctl.ClientToServer()
	return nil
}

func HTTPServer(address string)error{
	l, err := net.Listen("tcp", address)
	if err != nil{
		return fmt.Errorf("ProxyServer listen error: %s", err)
	}

	defer l.Close()

	for{
		c, err := l.Accept()
		if err != nil{
			return fmt.Errorf("HTTPServer accept error: %s", err)
		}

		go HTTPHandler(c)
	}
}