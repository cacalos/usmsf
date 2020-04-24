package configmgr

import "net"

func getMyIp() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		loggers.ErrorLogger().Major("%s", err.Error())
		return ""
	}

	var currentIP string

	for _, address := range addrs {

		// check the address type and if it is not a loopback the display it
		// = GET LOCAL IP ADDRESS
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				currentIP = ipnet.IP.String()
				return currentIP
			}
		}
	}
	loggers.ErrorLogger().Major("GetMyIp Fail..")
	return ""
}
