package pdu

import ()

/*************************************************
Define seo
**************************************************/
// Complete return true if decoding of message completed
func (msg *Cpmessage) Complete() bool { return msg.End }

// Direction Message direction
func (msg *Cpmessage) Direction() byte { return msg.Dir }

func (msg *Cpmessage) ProtocolDisc() byte    { return msg.ProtocolDiscr }
func (msg *Cpmessage) TransactionID() byte   { return msg.TransactionId }
func (msg *Cpmessage) MessageTypeInd() byte  { return msg.MessageType }
func (msg *Cpmessage) LengthIndicator() byte { return msg.LengthInd }
func (msg *Cpmessage) RpMsgType() byte       { return msg.RpMessageType }
func (msg *Cpmessage) RpMsgRef() byte        { return msg.RpMessageReference }

func (msg *Cpmessage) RpOrigAddr_f() string   { return msg.RpOrigAddr }
func (msg *Cpmessage) RpOrigNpi_f() byte      { return msg.RpOrigNpi }
func (msg *Cpmessage) RpOrigTon_f() byte      { return msg.RpOrigTon }
func (msg *Cpmessage) RpOrigAddrLen_f() uint8 { return msg.RpOrigAddrLen }

func (msg *Cpmessage) RpDestAddr_f() string   { return msg.RpDestAddr }
func (msg *Cpmessage) RpDestNpi_f() byte      { return msg.RpDestNpi }
func (msg *Cpmessage) RpDestTon_f() byte      { return msg.RpDestTon }
func (msg *Cpmessage) RpDestAddrLen_f() uint8 { return msg.RpDestAddrLen }
