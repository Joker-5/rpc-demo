package rpc_demo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"rpc_demo/codec"
	"sync"
)

const MagicNumber = 0xffff7777

// Option 承载RPC需要协商的信息
// 为实现简单，固定采用 JSON 编码 Option，后续的 header 和 body 的编码方式由 Option 中的 CodeType 指定，
// 服务端首先使用 JSON 解码 Option，然后通过 Option 的 CodeType 解码剩余的内容
type Option struct {
	MagicNumber int
	CodecType   codec.Type // 指定消息编解码格式
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.JsonType,
}

type Server struct {
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

// Accept 经典Accept函数，循环等待socket连接，开启协程进行处理
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error: ", err)
			return
		}
		go server.ServeConn(conn)
	}
}

// Accept 默认实现，方便用户调用
func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	// 根据事先约定用json反序列化获取Option实例，并进行参数后续的参数检查
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Println("rpc server: invalid magic number: ", opt.MagicNumber)
		return
	}
	// 根据Option获取编解码器
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Println("rpc server: invalid codec type: ", opt.CodecType)
		return
	}
	// 进行相关编解码业务处理
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	// 因为用go routine并发处理请求，但回复请求的报文必须要求顺序返回，
	// 所以引入锁来保证回复报文的有序性
	sending := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	// 因允许接收多个请求，因此用死循环持续监听请求，直到发生错误
	for {
		// 读取请求
		req, err := server.readRequest(cc)
		if err != nil {
			// 只有当header解析失败时才会终止循环
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 处理请求
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error: ", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO 请求参数类型未判断
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err: ", err)
	}
	return req, err
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error: ", err)
	}
}
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	// 接收到请求进行简单print
	// TODO 后续完整实现
	req.replyv = reflect.ValueOf(fmt.Sprintf("rpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
