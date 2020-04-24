package utils

import "encoding/json"

// GetMapKeys map의 키 값 목록을 반환
func GetMapKeys(m map[string]interface{}) []string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// ConvertJSONBytesToMap 마샬링된 JSON의 바이트 배열을 map[string]interface{} 타입으로 변환
func ConvertJSONBytesToMap(srcBytes []byte) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	err := json.Unmarshal(srcBytes, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ConvertStructToMap 구조체를 map[string]interface{} 타입으로 변환
func ConvertStructToMap(srcStruct interface{}) (map[string]interface{}, error) {
	// Struct => Bytes
	jsonBytes, err := json.Marshal(srcStruct)
	if err != nil {
		return nil, err
	}
	return ConvertJSONBytesToMap(jsonBytes)
}
