package vmess

import (
	"bytes"
	"io"
	"log"
	"net"
	"testing"

	"github.com/jarvisgally/v2simple/proxy"
)

func TestVMess(t *testing.T) {
	url := "vmess://a684455c-b14f-11ea-bf0d-42010aaa0003@127.0.0.1:9527?alterID=4"
	server, err := proxy.ServerFromURL(url)
	if err != nil {
		return
	}
	defer server.Stop()
	client, err := proxy.ClientFromURL(url)
	if err != nil {
		return
	}

	target := "dummy.com:80"

	// 开始监听
	listener, err := net.Listen("tcp", server.Addr())
	if err != nil {
		t.Logf("can not listen on %v: %v", server.Addr(), err)
		return
	}
	go func() {
		for {
			lc, err := listener.Accept()
			if err != nil {
				t.Logf("failed in accept: %v", err)
				break
			}
			go func() {
				defer lc.Close()
				wlc, targetAddr, err := server.Handshake(lc)
				if err != nil {
					t.Logf("failed in handshake form %v: %v", server.Addr(), err)
					return
				}

				if targetAddr.String() != target {
					t.Fail()
				}

				var hello [5]byte
				io.ReadFull(wlc, hello[:])
				if !bytes.Equal(hello[:], []byte("hello")) {
					t.Fail()
				}

				wlc.Write([]byte("world"))
			}()
		}
	}()

	// 连接
	rc, _ := net.Dial("tcp", server.Addr())
	defer rc.Close()

	wrc, err := client.Handshake(rc, target)
	if err != nil {
		log.Printf("failed in handshake to %v: %v", server.Addr(), err)
		return
	}
	wrc.Write([]byte("hello"))

	var world [5]byte
	io.ReadFull(wrc, world[:])
	if !bytes.Equal(world[:], []byte("world")) {
		t.Fail()
	}
}
