package interfaces

import "camel.uangel.com/ua5g/usmsf.git/msg5g"

// NRFClientService NRF 클라이언트 서비스 API
type NRFClientService interface {
	Start()
	Stop()
	GetBSFProfile() *msg5g.NFMgmtProfile
	SetNFStatusNotifyCallbackURI(uri string)
	AddMonitoring(pcfIntanceID string)
	RemoveMonitoring(pcfInstanceID string)
}
