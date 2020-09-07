package vmess

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"github.com/jarvisgally/v2simple/common"
	"hash/fnv"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jarvisgally/v2simple/proxy"
	"golang.org/x/crypto/chacha20poly1305"
)

func init() {
	proxy.RegisterClient(Name, NewVmessClient)
}

func NewVmessClient(url *url.URL) (proxy.Client, error) {
	addr := url.Host
	uuidStr := url.User.Username()
	uuid, err := StrToUUID(uuidStr)
	if err != nil {
		return nil, err
	}
	alterIDStr, ok := url.User.Password()
	if !ok {
		alterIDStr = "4"
	}
	alterID, err := strconv.ParseUint(alterIDStr, 10, 32)
	if err != nil {
		log.Printf("parse alterId err: %v", err)
		return nil, err
	}
	query := url.Query()

	security := query.Get("security")
	if security == "" {
		security = "none"
	}

	c := &Client{addr: addr}
	user := NewUser(uuid)
	c.users = append(c.users, user)
	c.users = append(c.users, user.GenAlterIDUsers(int(alterID))...)

	c.opt = OptChunkStream
	security = strings.ToLower(security)
	switch security {
	case "aes-128-gcm":
		c.security = SecurityAES128GCM
	case "chacha20-poly1305":
		c.security = SecurityChacha20Poly1305
	case "none":
		c.security = SecurityNone
	case "":
		// NOTE: use basic format when no method specified
		c.opt = OptBasicFormat
		c.security = SecurityNone
	default:
		return nil, errors.New("unknown security type: " + security)
	}
	rand.Seed(time.Now().UnixNano())

	return c, nil
}

// Client is a vmess client
type Client struct {
	addr     string
	users    []*User
	opt      byte
	security byte
}

func (c *Client) Name() string { return Name }

func (c *Client) Addr() string { return c.addr }

func (c *Client) Handshake(underlay net.Conn, target string) (io.ReadWriter, error) {
	r := rand.Intn(len(c.users))
	conn := &ClientConn{user: c.users[r], opt: c.opt, security: c.security}
	conn.Conn = underlay
	var err error
	conn.atyp, conn.addr, conn.port, err = ParseAddr(target)
	if err != nil {
		return nil, err
	}
	randBytes := common.GetBuffer(32)
	rand.Read(randBytes)
	copy(conn.reqBodyIV[:], randBytes[:16])
	copy(conn.reqBodyKey[:], randBytes[16:32])
	common.PutBuffer(randBytes)
	conn.reqRespV = byte(rand.Intn(1 << 8))
	conn.respBodyIV = md5.Sum(conn.reqBodyIV[:])
	conn.respBodyKey = md5.Sum(conn.reqBodyKey[:])

	// Auth
	err = conn.Auth()
	if err != nil {
		return nil, err
	}

	// Request
	err = conn.Request()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// ClientConn is a connection to vmess server
type ClientConn struct {
	user     *User
	opt      byte
	security byte

	atyp byte
	addr []byte
	port uint16

	reqBodyIV   [16]byte
	reqBodyKey  [16]byte
	reqRespV    byte
	respBodyIV  [16]byte
	respBodyKey [16]byte

	net.Conn
	dataReader io.Reader
	dataWriter io.Writer
}

// Auth send auth info: HMAC("md5", UUID, UTC)
func (c *ClientConn) Auth() error {
	ts := common.GetBuffer(8)
	defer common.PutBuffer(ts)

	binary.BigEndian.PutUint64(ts, uint64(time.Now().UTC().Unix()))

	h := hmac.New(md5.New, c.user.UUID[:])
	h.Write(ts)

	_, err := c.Conn.Write(h.Sum(nil))
	return err
}

// Request sends request to server.
func (c *ClientConn) Request() error {
	buf := common.GetWriteBuffer()
	defer common.PutWriteBuffer(buf)

	// Request
	buf.WriteByte(1)           // Ver
	buf.Write(c.reqBodyIV[:])  // IV
	buf.Write(c.reqBodyKey[:]) // Key
	buf.WriteByte(c.reqRespV)  // V
	buf.WriteByte(c.opt)       // Opt

	// pLen and Sec
	paddingLen := rand.Intn(16)
	pSec := byte(paddingLen<<4) | c.security // P(4bit) and Sec(4bit)
	buf.WriteByte(pSec)

	buf.WriteByte(0)      // reserved
	buf.WriteByte(CmdTCP) // cmd

	// target
	err := binary.Write(buf, binary.BigEndian, c.port) // port
	if err != nil {
		return err
	}

	buf.WriteByte(c.atyp) // atyp
	buf.Write(c.addr)     // addr

	// padding
	if paddingLen > 0 {
		padding := common.GetBuffer(paddingLen)
		rand.Read(padding)
		buf.Write(padding)
		common.PutBuffer(padding)
	}

	// F
	fnv1a := fnv.New32a()
	_, err = fnv1a.Write(buf.Bytes())
	if err != nil {
		return err
	}
	buf.Write(fnv1a.Sum(nil))

	// log.Printf("Request Send %v", buf.Bytes())

	block, err := aes.NewCipher(c.user.CmdKey[:])
	if err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, TimestampHash(time.Now().UTC().Unix()))
	stream.XORKeyStream(buf.Bytes(), buf.Bytes())

	_, err = c.Conn.Write(buf.Bytes())

	return err
}

// DecodeRespHeader decodes response header.
func (c *ClientConn) DecodeRespHeader() error {
	block, err := aes.NewCipher(c.respBodyKey[:])
	if err != nil {
		return err
	}

	stream := cipher.NewCFBDecrypter(block, c.respBodyIV[:])

	b := common.GetBuffer(4)
	defer common.PutBuffer(b)

	_, err = io.ReadFull(c.Conn, b)
	if err != nil {
		return err
	}

	stream.XORKeyStream(b, b)

	if b[0] != c.reqRespV {
		return errors.New("unexpected response header")
	}

	if b[2] != 0 {
		// dataLen := int32(buf[3])
		return errors.New("dynamic port is not supported now")
	}

	return nil
}

func (c *ClientConn) Write(b []byte) (n int, err error) {
	if c.dataWriter != nil {
		return c.dataWriter.Write(b)
	}

	c.dataWriter = c.Conn
	if c.opt&OptChunkStream == OptChunkStream {
		switch c.security {
		case SecurityNone:
			c.dataWriter = ChunkedWriter(c.Conn)

		case SecurityAES128GCM:
			block, _ := aes.NewCipher(c.reqBodyKey[:])
			aead, _ := cipher.NewGCM(block)
			c.dataWriter = AEADWriter(c.Conn, aead, c.reqBodyIV[:])

		case SecurityChacha20Poly1305:
			key := common.GetBuffer(32)
			t := md5.Sum(c.reqBodyKey[:])
			copy(key, t[:])
			t = md5.Sum(key[:16])
			copy(key[16:], t[:])
			aead, _ := chacha20poly1305.New(key)
			c.dataWriter = AEADWriter(c.Conn, aead, c.reqBodyIV[:])
			common.PutBuffer(key)
		}
	}

	return c.dataWriter.Write(b)
}

func (c *ClientConn) Read(b []byte) (n int, err error) {
	if c.dataReader != nil {
		return c.dataReader.Read(b)
	}

	err = c.DecodeRespHeader()
	if err != nil {
		return 0, err
	}

	c.dataReader = c.Conn
	if c.opt&OptChunkStream == OptChunkStream {
		switch c.security {
		case SecurityNone:
			c.dataReader = ChunkedReader(c.Conn)

		case SecurityAES128GCM:
			block, _ := aes.NewCipher(c.respBodyKey[:])
			aead, _ := cipher.NewGCM(block)
			c.dataReader = AEADReader(c.Conn, aead, c.respBodyIV[:])

		case SecurityChacha20Poly1305:
			key := common.GetBuffer(32)
			t := md5.Sum(c.respBodyKey[:])
			copy(key, t[:])
			t = md5.Sum(key[:16])
			copy(key[16:], t[:])
			aead, _ := chacha20poly1305.New(key)
			c.dataReader = AEADReader(c.Conn, aead, c.respBodyIV[:])
			common.PutBuffer(key)
		}
	}

	return c.dataReader.Read(b)
}
