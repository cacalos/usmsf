/**
 * @brief Configuration Data MAP PUSH & POP
 * @author parkjh
 * @file map.go
 * @data 2019-06-13
 * @version 0.1
 */
package configmgr

import (
	"strings"
)

func DecisionStoragePush(confId string, data DecisionTableInfo) {
	// The reason configid is needed is for debugging later

	for i, _ := range data.Decision {
		loggers.InfoLogger().Comment("Decision.ConfId : %s", confId)
		loggers.InfoLogger().Comment("Decision.Prefix : %s", data.Decision[i].Prefix)
		loggers.InfoLogger().Comment("Decision.Decision : %s", data.Decision[i].Decision)

		decisionMap[data.Decision[i].Prefix] = data.Decision[i]
	}

}

func DecisionStoragePop(supi string) *DecisionConf {
	var decision_pop_data DecisionConf

	prefix := strings.TrimLeft(supi, "imsi-")

	length := len(prefix)
	for i := 0; i < length; i++ {
		value, ok := decisionMap[prefix]
		if ok != true {
			prefix = prefix[0:(len(prefix) - 1)]
		} else {
			loggers.InfoLogger().Comment("DecisionPop Success.. prefix[%s],Decision[%s]", prefix, value.Decision)
			decision_pop_data = decisionMap[prefix]
			return &decision_pop_data
		}

		if i == length {
			loggers.ErrorLogger().Major("DecisionPop Not Exist.. prefix[%s]", prefix)
			return nil
		}
	}

	loggers.ErrorLogger().Major("DecisionPop Fail.. prefix[%s]", prefix)
	return nil

}

/**
 * @brief Common Configuration Map Data Push Function
 * @param confId, CommonConfiguration(PlmnId, Smsf-InstanceId...)
 * @return none
 */
func CommonStoragePush(confId string, data CommonConfiguration) {
	// The reason configid is needed is for debugging later
	loggers.InfoLogger().Comment("confId : %s", confId)
	loggers.InfoLogger().Comment("PLMN ID : %d", data.PlmnId)
	loggers.InfoLogger().Comment("SMSF-InstanceId : %s", data.SmsfInstanceId)
	loggers.InfoLogger().Comment("SMSF-MAP-Address : %s", data.SmsfMapAddress)
	loggers.InfoLogger().Comment("SMSF-Diameter-Address : %s", data.SmsfDiameterAddress)
	loggers.InfoLogger().Comment("SMSF-Point-Code : %d", data.SmsfPointCode)
	loggers.InfoLogger().Comment("SmsfSsn : %d", data.SmsfSsn)

	commonData = data
}

/**
 * @brief SmscTable Configuration Map Data Push Function
 * @param confId, SmscTableInfo(Name, Isdn, Pc, Ssn, .. , Preifx...)
 * @return none
 */
func SmscStoragePush(confId string, data SmscTableInfo) {
	// The reason configid is needed is for debugging later

	for i, _ := range data.Prefix {
		loggers.InfoLogger().Comment("=====================================[%d]", i)
		loggers.InfoLogger().Comment("confId : %s", confId)
		loggers.InfoLogger().Comment("PREFIX.Prefix : %s", data.Prefix[i].Prefix)
		loggers.InfoLogger().Comment("PREFIX.SMSC Name : %s", data.Prefix[i].SmscName)
		loggers.InfoLogger().Comment("=====================================")

		prefixMap[data.Prefix[i].Prefix] = data.Prefix[i]

	}

	for i, _ := range data.Node {
		loggers.InfoLogger().Comment("=====================================[%d]", i)
		loggers.InfoLogger().Comment("confId : %s", confId)
		loggers.InfoLogger().Comment("NODE.Name : %s", data.Node[i].Name)
		loggers.InfoLogger().Comment("NODE.Isdn : %s", data.Node[i].Isdn)
		loggers.InfoLogger().Comment("NODE.Pc : %d", data.Node[i].Pc)
		loggers.InfoLogger().Comment("NODE.Ssn : %d", data.Node[i].Ssn)
		loggers.InfoLogger().Comment("NODE.Type : %d", data.Node[i].Type)
		loggers.InfoLogger().Comment("NODE.FlowCTRL : %d", data.Node[i].FlowCtrl)
		loggers.InfoLogger().Comment("NODE.Dest_Host : %s", data.Node[i].Dest_host)
		loggers.InfoLogger().Comment("NODE.Dest_Realm : %s", data.Node[i].Dest_realm)
		loggers.InfoLogger().Comment("NODE.Desc : %s", data.Node[i].Desc)
		loggers.InfoLogger().Comment("=====================================")

		nodeMap[data.Node[i].Name] = data.Node[i]

	}

}

/**
 * @brief confid Configuration Map Data
 * @param confid
 * @return none
 */
func SetConfIdStorage(confId string) {
	configMap[count] = confId
	count++
}

/**
 * @brief confid Configuration Map Data
 * @param confid
 * @return none
 */
func GetConfIdStorage() (int, map[int]string) {
	return count, configMap
}

func SetWatchIdStorage(watchId string) {
	watchIdMap[watchcnt] = watchId
	watchcnt++
}

func GetWatchIdMap() (int, map[int]string) {
	return watchcnt, watchIdMap
}

/**
 * @brief Common Configuration Map Data Pop Function
 * @param plmnId
 * @return CommonConfiguration, string(error string)
 */

func CommonStoragePop() *CommonConfiguration {
	loggers.InfoLogger().Comment("Get SMSF Configuration Success")
	return &commonData
}

/**
* @brief Smsc Prefix Configuration Map Data Pop Function
* @param preifx
* @return SmscPrefix, string(error string)
 */
func SmscPrefixStoragePop(prefix string) *SmscPrefix {
	var prefix_pop_data SmscPrefix

	length := len(prefix)
	for i := 0; i < length; i++ {
		value, ok := prefixMap[prefix]
		if ok != true {
			prefix = prefix[0:(len(prefix) - 1)]
		} else {
			loggers.InfoLogger().Comment("SMSCPrefixPop Success.. prefix[%s],SmscName[%s]",
				prefix, value.SmscName)
			prefix_pop_data = prefixMap[prefix]
			return &prefix_pop_data
		}

		if i == length {
			loggers.ErrorLogger().Major("SMSCPrefixPop Not Exist.. prefix[%s]", prefix)
			return nil
		}
	}

	loggers.ErrorLogger().Major("SMSCPrefixPop Fail.. prefix[%s]", prefix)
	return nil

}

/**
 * @brief Smsc Node Configuration Map Data Pop Function
 * @param name(SMSC)
 * @return SmscNode, string(error string)
 */
func SmscNodeStoragePop(name string) *SmscNode {
	var node_pop_data SmscNode

	node_pop_data = nodeMap[name]

	loggers.InfoLogger().Comment("SMSCNodePop Success.. name[%s]", name)
	return &node_pop_data
}
