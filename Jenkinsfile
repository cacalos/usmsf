node('jenkins-slave') {
    
     stage('test pipeline') {
        sh(script: """
            echo "hello"
			cat /etc/resolv.conf
			cat /etc/resolv1.conf
			cat /etc/resolv.conf
			sudo echo "nameserver 192.168.1.71" >> /etc/resolv.conf
			nslookup github.com
			nslookup github.com
			nslookup github.com
			nslookup github.com
			nslookup github.com
			nslookup github.com
            git clone -v https://github.com/marcel-dempers/docker-development-youtube-series.git
        """)
    }
}
