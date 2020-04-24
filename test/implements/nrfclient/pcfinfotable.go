package nrfclient

import (
	"sync"

	"github.com/go-openapi/strfmt"
)

// IDInfo NF instance ID 및 Subscription ID 정보 저장 구조체
type IDInfo struct {
	InstanceID     string
	SubscriptionID string
}

// PcfInfo PCF 관리용 구조체
type PcfInfo struct {
	InstanceID          string
	SubscriptionID      string
	ValidityTime        strfmt.DateTime
	Removing            chan bool
	ValidityTimeChecker func()
}

// PcfInfoTable PCF 정보 관리 테이블 구조체
type PcfInfoTable struct {
	sync.RWMutex
	table map[string]*PcfInfo
}

// NewPcfInfoTable PcfInfoTable 생성 및 반환
func NewPcfInfoTable() *PcfInfoTable {
	return &PcfInfoTable{
		table: make(map[string]*PcfInfo),
	}
}

// Add 지정한 키 값으로 아이템 생성 및 추가
func (t *PcfInfoTable) Add(info *PcfInfo) {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.table[info.InstanceID]; !ok {
		t.table[info.InstanceID] = info
		loggers.InfoLogger().Comment("PCF(%s)'s validity time: %s", info.InstanceID, info.ValidityTime.String())
		// starting validity time checking for this PCF
		go info.ValidityTimeChecker()
	}
}

// Remove 지정한 키 값의 아이템 제거
func (t *PcfInfoTable) Remove(instanceID string) {
	t.Lock()
	defer t.Unlock()

	if pcfInfo, ok := t.table[instanceID]; ok {
		// Stopping validity time checking for this PCF
		pcfInfo.Removing <- true
		delete(t.table, instanceID)
	}
}

// Get 지정한 키 값의 PcfInfo 를 가져 옴
func (t *PcfInfoTable) Get(instanceID string) (*PcfInfo, bool) {
	t.RLock()
	defer t.RUnlock()

	info, ok := t.table[instanceID]
	return info, ok
}

// GetPcfInstanceIDs 현재 테이블에 등록된 모든 PCF instance ID 값을 반환
func (t *PcfInfoTable) GetPcfInstanceIDs() []string {
	t.RLock()
	defer t.RUnlock()

	instanceIDs := []string{}
	for id := range t.table {
		instanceIDs = append(instanceIDs, id)
	}
	return instanceIDs
}

// GetIDs 현재 테이블에 등록된 모든 NF instance ID 별 구독 ID 값을 반환
func (t *PcfInfoTable) GetIDs() []IDInfo {
	t.RLock()
	defer t.RUnlock()

	idList := []IDInfo{}
	for _, subscription := range t.table {
		idList = append(idList, IDInfo{
			InstanceID:     subscription.InstanceID,
			SubscriptionID: subscription.SubscriptionID,
		})
	}
	return idList
}
