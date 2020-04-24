package tcpmgr

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"time"

	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uclient"
	"camel.uangel.com/ua5g/ulib.git/uconf"

	"camel.uangel.com/ua5g/usmsf.git/common"
	"camel.uangel.com/ua5g/usmsf.git/dao"
	cdr "camel.uangel.com/ua5g/usmsf.git/implements/cdrmgr"
	"camel.uangel.com/ua5g/usmsf.git/interfaces"

	"io"

	"net"
	"net/http"
)

const protocol string = "tcp4"

const TYPE_MO = "1"
const TYPE_MT = "2"

const CDR_SUCC = "1"
const CDR_FAIL = "2"

type TcpServer struct {
	Client   net.Conn
	err      error
	redisDao dao.RedisSubDao

	cdr *cdr.CdrMgr

	msgproxy_ipaddr string

	router uclient.HTTPRouter

	cliConf common.HTTPCliConf
}

func (s *TcpServer) LoadConfig(cfg uconf.Config) (err error) {
	smsfConf := cfg.GetConfig("tcpmgr")

	if smsfConf != nil {

		s.cliConf.DialTimeout = smsfConf.GetDuration("map-client.connection.timeout", time.Second*20)
		s.cliConf.DialKeepAlive = smsfConf.GetDuration("map-client.connection.keep-alive", time.Second*20)
		s.cliConf.IdleConnTimeout = smsfConf.GetDuration("map-client.connection.expire-time", 1*time.Minute)
		s.cliConf.InsecureSkipVerify = true

		s.cliConf.MaxHeaderListSize = uint32(smsfConf.GetInt("map-client.MaxHeaderListSize", 1024000000))
		s.cliConf.StrictMaxConcurrentStreams = smsfConf.GetBoolean("StrictMaxConcurrentStreams", false)

		s.msgproxy_ipaddr = smsfConf.GetString("tcp.info", "127.0.0.1:8000")
	} else {
		return errors.New("Fail -> TcpMgr Config Load")
	}

	return nil
}

func (s *TcpServer) svchostConfig(httpcli uclient.HTTP,
	circuitBreaker uclient.HTTPCircuitBreaker,
) (err error) {
	svchost := os.Getenv("SVC_POD_HOST")

	if svchost != "" {
		s.router = uclient.HTTPRouter{
			Scheme:  "http",
			Servers: []string{svchost},
			Client:  httpcli,
			//	CircuitBreaker:   circuitBreaker,
			Random: false,
			//	RetryStatusCodes: uclient.StatusCodeSet(400, 403, 404, 502, 508),
		}
	} else {
		return errors.New("Set FAil : svc-pod host Config")
	}

	return nil

}

func (s *TcpServer) InitRedisServer(redisdaoSet *dao.RedisDaoSet) {

	svctype := os.Getenv("MY_SERVICE_TYPE")

	if svctype != "" {
		if svctype == "SIGTRAN" {
			s.redisDao = redisdaoSet.RedisSubDao
		} else if svctype == "DIAMETER" {
			loggers.InfoLogger().Comment("No Redis POD Mode")
		} else {

		}
	} else {
		loggers.ErrorLogger().Major("Fail -> Get env Service_type")
	}

}

func NewTcpServerDia(
	cfg uconf.Config,
	traceMgr interfaces.TraceMgr,
	httpcli uclient.HTTP,
	circuitBreaker uclient.HTTPCircuitBreaker,
	cdrMgr *cdr.CdrMgr,
) *TcpServer {
	var err error

	s := &TcpServer{
		cdr: cdrMgr,
	}

	err = s.LoadConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	err = s.svchostConfig(httpcli, circuitBreaker)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	s.MakeMsgProxyClientTcpInfo()
	s.ReadSigibData()

	return s
}

func NewTcpServer(
	cfg uconf.Config,
	traceMgr interfaces.TraceMgr,
	redisdaoSet *dao.RedisDaoSet,
	httpcli uclient.HTTP,
	circuitBreaker uclient.HTTPCircuitBreaker,
	cdrMgr *cdr.CdrMgr,
) *TcpServer {
	var err error

	s := &TcpServer{}

	s.InitRedisServer(redisdaoSet)

	err = s.LoadConfig(cfg)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	err = s.svchostConfig(httpcli, circuitBreaker)
	if err != nil {
		loggers.ErrorLogger().Major("%v", err)
		return nil
	}

	s.MakeMsgProxyClientTcpInfo()
	s.ReadSigibData()

	return s
}

func (s *TcpServer) MakeMsgProxyClientTcpInfo() {
	loggers.InfoLogger().Comment("MakeMsgProxyClientTcpInfo() Start..........")
	loggers.InfoLogger().Comment("TCP_INFO : IP/PORT[%s]", s.msgproxy_ipaddr)

	var i int
	var tcpAddr *net.TCPAddr
	var client *net.TCPConn
	var err error

	tcpAddr, err = net.ResolveTCPAddr(protocol, s.msgproxy_ipaddr)
	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		for {
			i++
			loggers.InfoLogger().Comment("Retry Connection MsgProxy Client to Server(try count : %d)", i)
			tcpAddr, err = net.ResolveTCPAddr(protocol, s.msgproxy_ipaddr)
			if err != nil {
				loggers.ErrorLogger().Major("%s", err.Error())
			} else {
				//	loggers.InfoLogger().Comment("TCP ReConnection Succ.")
				break
			}

			time.Sleep(1 * time.Second)

		}
	}

	client, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		for {
			i++
			loggers.InfoLogger().Comment("Retry Connection MsgProxy Client to Server(try count : %d)", i)
			client, err = net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				loggers.ErrorLogger().Major("%s", err.Error())
			} else {
				loggers.InfoLogger().Comment("TCP ReConnection Succ.")
				break
			}

			time.Sleep(1 * time.Second)

		}
	}
	s.Client = client
	s.err = nil
}

func (s *TcpServer) MakeHTTPMsgResp(SendData MtData) {

	now_in := time.Now()
	timestamp := fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		now_in.Year(), now_in.Month(), now_in.Day(),
		now_in.Hour(), now_in.Minute(), now_in.Second())

	supi := string(SendData.Supi[:SendData.SupiLen])
	var smsfURL string

	loggers.InfoLogger().Comment("Make And Send MO-RESPONSE DATA to SMSF-SVC-POD[%s]",
		supi)

	smsfURL = fmt.Sprintf("/svc/v1/%s/rpresp", supi)

	loggers.InfoLogger().Comment("Send To SMSF-SVC_POD : %s", smsfURL)

	hdr := http.Header{}
	//	hdr.Add("Content-Type", "multipart/related;boundary=Boundary")
	hdr.Add("Content-Type", "application/json")

	reqBody, err := MakeHttpIfMsgResp(SendData)

	loggers.InfoLogger().Comment("Send To SVC-POD From Tcpmgr : %s%s",
		s.router.Servers[0], smsfURL)
	Resp, err := s.router.SendRequest(context.Background(),
		smsfURL, "POST", hdr, reqBody, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("Fail --> Send To SMSF_SVC_POD(supi : %s)", supi)
		loggers.ErrorLogger().Major("err : %v", err)
		return
	}

	now_out := time.Now()
	doneTime := fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		now_out.Year(), now_out.Month(), now_out.Day(),
		now_out.Hour(), now_out.Minute(), now_out.Second())

	Result := fmt.Sprintf("%d", SendData.Result)
	DestSupi := ""

	s.cdr.WriteCdr(TYPE_MO, supi, DestSupi, timestamp, doneTime, Result)
	loggers.InfoLogger().Comment("Recv Response From SVC_POD: %d", Resp.StatusCode)

}

func (s *TcpServer) MakeHTTPMsgMtForSigtran(SendData MtData) {

	supi := string(SendData.Supi[:SendData.SupiLen])
	var smsfURL string

	loggers.InfoLogger().Comment("Make And Send MT(RPDATA) in oder to Send SMSF-SVC-POD")

	smsfURL = fmt.Sprintf("/map/v1/%s/rpdata", supi)

	loggers.InfoLogger().Comment("%s", smsfURL)

	hdr := http.Header{}
	hdr.Add("Content-Type", "multipart/related;boundary=Boundary")

	reqBody, err := MakeHttpIfMsgMt(SendData)

	loggers.InfoLogger().Comment("Send Message Info to SVC-POD : %s", string(reqBody))
	loggers.InfoLogger().Comment("Send To SVC-POD From Tcpmgr : %s%s", s.router.Servers[0], smsfURL)
	Resp, err := s.router.SendRequest(context.Background(), smsfURL, "POST", hdr, reqBody, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("Req err() : %s, USER : %s\n", err.Error(), supi)
		return
	}

	loggers.InfoLogger().Comment("Send Succ. HTTP MT Message -> supi %s", supi)

	if Resp.StatusCode >= 300 {
		loggers.InfoLogger().Comment("Recv ErrResp Message From SMSF_SVC_POD")

		// Get Redis Memory
		rval, rdata := s.redisDao.GetSubBySUPI(supi)
		if rval == -1 {
			loggers.ErrorLogger().Major("Dose not find Response Info in RedisDB. USER : %s", supi)
			return
		}

		SendErrData := s.MakeMtErrRespSendData(SendData, Resp.StatusCode, supi, MT_ERR, rdata)

		s.SendErrRespToMsgProxy(SendErrData)

		loggers.InfoLogger().Comment("resp Err code : %d\n", Resp.StatusCode)
		return
	}

	loggers.InfoLogger().Comment("resp code : %d, data : %s\n", Resp.StatusCode, string(Resp.Response.([]byte)))
}

func (s *TcpServer) MakeHTTPMsgMtForDiameter(SendData MtData) {

	supi := string(SendData.Supi[:SendData.SupiLen])
	var smsfURL string

	loggers.InfoLogger().Comment("Make And Send MT(RPDATA) in oder to Send SMSF-SVC-POD")

	smsfURL = fmt.Sprintf("/map/v1/%s/rpdata", supi)

	loggers.InfoLogger().Comment("%s", smsfURL)

	hdr := http.Header{}
	//	hdr.Add("Content-Type", "multipart/related;boundary=Boundary")
	hdr.Add("Content-Type", "application/json")

	reqBody, err := MakeHttpIfMsgMt(SendData)

	loggers.InfoLogger().Comment("Send To SVC-POD From Tcpmgr : %s%s", s.router.Servers[0], smsfURL)
	Resp, err := s.router.SendRequest(context.Background(), smsfURL, "POST", hdr, reqBody, 2*time.Second)
	if err != nil {
		loggers.ErrorLogger().Major("Req err() : %s, USER : %s\n", err.Error(), supi)
		return
	}

	loggers.InfoLogger().Comment("Send Succ. HTTP MT Message -> supi %s", supi)

	if Resp.StatusCode >= 300 {
		loggers.InfoLogger().Comment("Recv ErrResp Message From SMSF_SVC_POD")

		SendErrData := s.MakeMtErrRespSendDataForDiameter(SendData, Resp.StatusCode, supi, MT_ERR)

		s.SendErrRespToMsgProxy(SendErrData)

		loggers.InfoLogger().Comment("resp Err code : %d\n", Resp.StatusCode)
		return
	}

	loggers.InfoLogger().Comment("resp code : %d, data : %s\n", Resp.StatusCode, string(Resp.Response.([]byte)))
}

func (s *TcpServer) ReadSigibData() {

	notify := make(chan error)

	exec.SafeGo(func() {
		for {
			var length int
			dataLen := make([]byte, 4) // 4096 크기의 바이트 슬라이스 생성
			var err error

			//	_, err := c.Read(dataLen) // 서버에서 받은 데이터를 읽음
			io.ReadFull(s.Client, dataLen)
			if err != nil {
				s.Client.Close()
				notify <- err
				//		close(notify)
				return
			}

			length = int(binary.LittleEndian.Uint32(dataLen))

			//			if length == 540 || length == 1068 {
			data := make([]byte, length-4)

			//_, err = c.Read(data)
			io.ReadFull(s.Client, data)
			if err != nil {
				s.Client.Close()
				notify <- err
				//	close(notify)
				return
			}

			//	close(notify)

			s.DecodeMessage(data, length-4)

			//	} else {
			//		loggers.ErrorLogger().Major("Recv data is Wlong Size from msgproxy : %d", length)
			//	}
		}
	})

	exec.SafeGo(func() {
		for {
			select {
			case err := <-notify:
				if err != nil {

					s.MakeMsgProxyClientTcpInfo()
					s.ReadSigibData()
					close(notify)
					break
				}

			}
		}
	})

}

func (s *TcpServer) DecodeMessage(data []byte, length int) {
	SendData := ConvertSendToSvcPodMsg(data, length)

	svctype := os.Getenv("MY_SERVICE_TYPE")

	if SendData.Type == MO_RESP {
		s.MakeHTTPMsgResp(SendData)
	} else if SendData.Type == MT_MSG {
		if svctype != "" {
			if svctype == "SIGTRAN" {
				s.InsRedisMtMsg(SendData)
				s.MakeHTTPMsgMtForSigtran(SendData)
			} else if svctype == "DIAMETER" {
				s.MakeHTTPMsgMtForDiameter(SendData)
			} else {
				loggers.ErrorLogger().Major("Invaild svcType")

			}
		} else {
			loggers.ErrorLogger().Major("Fail -> Get Env")

		}
	} else {
		loggers.ErrorLogger().Major("Invalid Type : [%d]", SendData.Type)
	}

}

func (s *TcpServer) SendErrRespToMsgProxy(input MtData) error {
	var err error

	//defer client.Close() // main 함수가 끝나기 직전에 TCP 연결을 닫음
	var offset int
	var i int
	Len := make([]byte, 4)
	causecode := make([]byte, 4)

	binary.LittleEndian.PutUint32(Len, uint32(input.TotalLen+4))

	buf := make([]byte, input.TotalLen+4)

	copy(buf[0:3], Len[0:3])
	offset = offset + 4

	buf[offset] = input.Type
	offset++

	binary.LittleEndian.PutUint32(causecode, uint32(input.CauseCode))
	copy(buf[offset:], causecode[0:3])
	offset = offset + 4

	buf[offset] = byte(input.SupiLen)
	offset++
	for i = 0; i < int(input.SupiLen); i++ {
		buf[offset+i] = input.Supi[i]
	}
	offset = offset + i

	buf[offset] = byte(input.Diag_id_len)
	offset++
	copy(buf[offset:offset+input.Diag_id_len], input.Diag_id[:input.Diag_id_len])
	offset = offset + int(input.Diag_id_len)

	binary.LittleEndian.PutUint32(Len, uint32(input.Acn))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	binary.LittleEndian.PutUint32(Len, uint32(input.Prov_id))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	binary.LittleEndian.PutUint32(Len, uint32(input.Inv_id))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	binary.LittleEndian.PutUint32(Len, uint32(input.Hop_id))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	binary.LittleEndian.PutUint32(Len, uint32(input.End_id))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	binary.LittleEndian.PutUint32(Len, uint32(input.Peer_id))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	buf[offset] = byte(input.Orig_realm_len)
	offset++
	copy(buf[offset:], input.Orig_realm[:input.Orig_realm_len])
	offset = offset + int(input.Orig_realm_len)

	buf[offset] = byte(input.Orig_host_len)
	offset++
	copy(buf[offset:], input.Orig_host[:input.Orig_host_len])
	offset = offset + int(input.Orig_host_len)

	buf[offset] = byte(input.Smsc_node_len)
	offset++
	copy(buf[offset:], input.Smsc_node[:input.Smsc_node_len])
	offset = offset + int(input.Smsc_node_len)

	binary.LittleEndian.PutUint32(Len, uint32(input.Session_id_len))
	copy(buf[offset:offset+3], Len[0:3])
	offset = offset + 4

	copy(buf[offset:], input.Session_id[:input.Session_id_len])
	offset = offset + int(input.Session_id_len)

	buf[offset] = byte(input.ContentDataLen)
	offset++
	copy(buf[offset:], input.ContentData[:input.ContentDataLen])
	offset = offset + int(input.ContentDataLen)

	_, err = s.Client.Write(buf) // 서버로 데이터를 보냄
	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		return err
	}

	return err
}
