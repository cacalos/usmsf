package rpdu

import ()

/*************************************************
Define seo
**************************************************/
// Complete return true if decoding of message completed
func (msg *Rpmessage) Complete() bool { return msg.End }

// Direction Message direction
func (msg *Rpmessage) Direction() byte { return msg.Dir }

func (msg *Rpmessage) RpMsgType() byte { return msg.RpMessageType }
func (msg *Rpmessage) RpMsgRef() byte  { return msg.RpMessageReference }

func (msg *Rpmessage) RpOrigAddr_f() string   { return msg.RpOrigAddr }
func (msg *Rpmessage) RpOrigNpi_f() byte      { return msg.RpOrigNpi }
func (msg *Rpmessage) RpOrigTon_f() byte      { return msg.RpOrigTon }
func (msg *Rpmessage) RpOrigAddrLen_f() uint8 { return msg.RpOrigAddrLen }

func (msg *Rpmessage) RpDestAddr_f() string   { return msg.RpDestAddr }
func (msg *Rpmessage) RpDestNpi_f() byte      { return msg.RpDestNpi }
func (msg *Rpmessage) RpDestTon_f() byte      { return msg.RpDestTon }
func (msg *Rpmessage) RpDestAddrLen_f() uint8 { return msg.RpDestAddrLen }
