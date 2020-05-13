node('jenkins-slave') {
    
     stage('test pipeline') {
        sh(script: """
            echo "hello"
			cat /etc/resolv.conf
			cat /etc/resolv1.conf
			curl -I http://10.96.0.10
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
